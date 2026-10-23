package integration

import (
	"context"
	"testing"
	"time"

	"credit-union/backend/internal/platform/database"
)

func TestInterestDetectsMissingCompletedDayRows(t *testing.T) {
	f := newFixture(t)
	id := f.member(t, "3271010101900803", "teller.a")
	svc, _ := interestHistory(t, f, id, 300)
	ctx := context.Background()
	if _, err := f.db.Exec(ctx, `CREATE FUNCTION fail_interest() RETURNS trigger LANGUAGE plpgsql AS $$
 BEGIN RAISE EXCEPTION 'injected failure'; END $$;
 CREATE TRIGGER fail_interest BEFORE INSERT ON interest_monthly
 FOR EACH ROW EXECUTE FUNCTION fail_interest()`); err != nil {
		t.Fatal(err)
	}
	if err := svc.CatchUp(ctx); err == nil {
		t.Fatal("expected settlement failure")
	}
	if _, err := f.db.Exec(ctx, `DROP TRIGGER fail_interest ON interest_monthly; DROP FUNCTION fail_interest();
 DELETE FROM interest_daily`); err != nil {
		t.Fatal(err)
	}
	if err := svc.CatchUp(ctx); err == nil {
		t.Fatal("completed dates with missing account calculations silently accepted")
	}
}
func TestSavingsWriterBarrierAndClosedDate(t *testing.T) {
	f := newFixture(t)
	id := f.member(t, "3271010101900804", "teller.a")
	ctx := context.Background()
	tx, err := f.db.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Rollback(tx)
	if _, err = tx.Exec(ctx, "SELECT pg_advisory_xact_lock(101,1)"); err != nil {
		t.Fatal(err)
	}
	blocked, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()
	writer, err := f.db.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Rollback(writer)
	if err = database.LockSavingsWriter(blocked, writer); err == nil {
		t.Fatal("writer crossed exclusive close")
	}
	if err = tx.Rollback(ctx); err != nil {
		t.Fatal(err)
	}
	// Simulate a completed date beyond the DB clock, requiring a safe refusal.
	if _, err = f.db.Exec(ctx, `INSERT INTO interest_policies(effective_date,base_rate_bps,bands,reason,created_by)
 VALUES(current_date,300,'[]','Isolated clock regression',$1)`, f.ids["supervisor"]); err != nil {
		t.Fatal(err)
	}
	if _, err = f.db.Exec(ctx, `INSERT INTO interest_days(business_date,eligible_account_count,policy_id)
 SELECT (clock_timestamp() AT TIME ZONE 'Asia/Jakarta')::date,0,id FROM interest_policies`); err != nil {
		t.Fatal(err)
	}
	requireStatus(t, f.postCash("teller.a", "closed-date-deposit-0001", "deposit", id, 1000), 503)
	var balance int64
	if err = f.db.QueryRow(ctx, "SELECT balance FROM members WHERE id=$1", id).
		Scan(&balance); err != nil ||
		balance != 50000 {
		t.Fatalf("balance=%d %v", balance, err)
	}
}
