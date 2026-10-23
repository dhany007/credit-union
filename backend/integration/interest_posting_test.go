package integration

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"credit-union/backend/internal/interest"
	"credit-union/backend/internal/platform/database"
)

func interestHistory(t *testing.T, f *fixture, member int64, rate int) (*interest.Service, time.Time) {
	t.Helper()
	ctx := context.Background()
	var start time.Time
	if err := f.db.QueryRow(ctx, `SELECT
 (date_trunc('month',clock_timestamp() AT TIME ZONE 'Asia/Jakarta')-interval '1 month')::date`).
		Scan(&start); err != nil {
		t.Fatal(err)
	}
	date := start.Format(time.DateOnly)
	if _, err := f.db.Exec(
		ctx,
		"UPDATE transactions SET created_at=$1::date AT TIME ZONE 'Asia/Jakarta' WHERE member_id=$2",
		date,
		member,
	); err != nil {
		t.Fatal(err)
	}
	if _, err := f.db.Exec(ctx, `INSERT INTO interest_policies(effective_date,base_rate_bps,bands,reason,created_by)
 VALUES($1,$2,'[]','Isolated historical test',$3)`, date, rate, f.ids["supervisor"]); err != nil {
		t.Fatal(err)
	}
	return interest.NewService(f.db, database.NewRunner(f.db), false), start
}
func TestInterestMonthlyCreditAndConcurrentRetry(t *testing.T) {
	f := newFixture(t)
	id := f.member(t, "3271010101900801", "teller.a")
	svc, start := interestHistory(t, f, id, 300)
	ctx := context.Background()
	var cashBefore int64
	if err := f.db.QueryRow(ctx, "SELECT sum(debit-credit) FROM journal_lines WHERE account='cash'").
		Scan(&cashBefore); err != nil {
		t.Fatal(err)
	}
	errors := make(chan error, 2)
	var wg sync.WaitGroup
	for range 2 {
		wg.Add(1)
		go func() { defer wg.Done(); errors <- svc.CatchUp(ctx) }()
	}
	wg.Wait()
	close(errors)
	for err := range errors {
		if err != nil {
			t.Fatal(err)
		}
	}
	if err := svc.CatchUp(ctx); err != nil {
		t.Fatal(err)
	}
	days := start.AddDate(0, 1, 0).Sub(start).Hours() / 24
	expected := (50000*300*int64(days) + 1825000) / 3650000
	var balance, amount, cashAfter int64
	var count int
	var actor *int64
	if err := f.db.QueryRow(ctx, "SELECT balance FROM members WHERE id=$1", id).Scan(&balance); err != nil {
		t.Fatal(err)
	}
	if err := f.db.QueryRow(ctx, `SELECT count(*),COALESCE(sum(amount),0) FROM transactions WHERE kind='interest_credit'`).
		Scan(&count, &amount); err != nil {
		t.Fatal(err)
	}
	if err := f.db.QueryRow(ctx, "SELECT actor_id FROM transactions WHERE kind='interest_credit'").
		Scan(&actor); err != nil {
		t.Fatal(err)
	}
	if err := f.db.QueryRow(ctx, "SELECT sum(debit-credit) FROM journal_lines WHERE account='cash'").
		Scan(&cashAfter); err != nil {
		t.Fatal(err)
	}
	if count != 1 || amount != expected || balance != 50000+expected || cashBefore != cashAfter || actor != nil {
		t.Fatalf(
			"count=%d amount=%d expected=%d balance=%d cash=%d/%d actor=%v",
			count,
			amount,
			expected,
			balance,
			cashBefore,
			cashAfter,
			actor,
		)
	}
	paths := []string{
		"/api/transactions",
		fmt.Sprintf("/api/members/%d/transactions", id),
		"/api/transactions/search?period=today",
	}
	for _, path := range paths {
		w := f.call("teller.a", "GET", path, "", "")
		requireStatus(t, w, 200)
		if !strings.Contains(w.Body.String(), `"kind":"interest_credit"`) ||
			!strings.Contains(w.Body.String(), `"actor":"System"`) {
			t.Fatalf("missing system credit %s: %s", path, w.Body)
		}
	}
	requireStatus(t, f.postCash("teller.a", "cannot-forge-interest-01", "interest_credit", id, 100), 400)
	var bad int
	if err := f.db.QueryRow(ctx, `SELECT count(*) FROM (SELECT transaction_id FROM journal_lines GROUP BY transaction_id
 HAVING sum(debit)<>sum(credit)) x`).Scan(&bad); err != nil || bad != 0 {
		t.Fatalf("journal %d %v", bad, err)
	}
}
func TestInterestFailureRollsBackAndZeroSettles(t *testing.T) {
	for _, rate := range []int{0, 300} {
		t.Run(fmt.Sprint(rate), func(t *testing.T) {
			f := newFixture(t)
			id := f.member(t, "3271010101900802", "teller.a")
			svc, _ := interestHistory(t, f, id, rate)
			ctx := context.Background()
			if rate > 0 {
				_, err := f.db.Exec(ctx, `CREATE FUNCTION fail_interest() RETURNS trigger LANGUAGE plpgsql AS $$
    BEGIN IF NEW.account='savings_interest_expense' THEN RAISE EXCEPTION 'injected failure'; END IF; RETURN NEW; END $$;
    CREATE TRIGGER fail_interest BEFORE INSERT ON journal_lines FOR EACH ROW EXECUTE FUNCTION fail_interest()`)
				if err != nil {
					t.Fatal(err)
				}
				if err = svc.CatchUp(ctx); err == nil {
					t.Fatal("expected injected failure")
				}
				var balance int64
				var count int
				if err = f.db.QueryRow(ctx, "SELECT balance,(SELECT count(*) FROM interest_monthly) FROM members WHERE id=$1", id).
					Scan(&balance, &count); err != nil {
					t.Fatal(err)
				}
				if balance != 50000 || count != 0 {
					t.Fatal("partial monthly posting")
				}
				if _, err = f.db.Exec(
					ctx,
					"DROP TRIGGER fail_interest ON journal_lines; DROP FUNCTION fail_interest()",
				); err != nil {
					t.Fatal(err)
				}
			}
			if err := svc.CatchUp(ctx); err != nil {
				t.Fatal(err)
			}
			var rows, credits int
			if err := f.db.QueryRow(ctx, `SELECT count(*),
 (SELECT count(*) FROM transactions WHERE kind='interest_credit') FROM interest_monthly`).
				Scan(&rows, &credits); err != nil {
				t.Fatal(err)
			}
			if rows != 1 || (rate == 0 && credits != 0) || (rate > 0 && credits != 1) {
				t.Fatalf("settlements %d credits %d", rows, credits)
			}
		})
	}
}
