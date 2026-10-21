package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"sync"
	"testing"

	"credit-union/backend/internal/platform/database"
)

type balanceLedgerRow struct {
	Reference string `json:"reference"`
	Kind      string `json:"kind"`
	Amount    int64  `json:"amount"`
	Before    *int64 `json:"balance_before"`
	After     *int64 `json:"balance_after"`
}

func ledgerRows(t *testing.T, w *httptest.ResponseRecorder) []balanceLedgerRow {
	t.Helper()
	requireStatus(t, w, 200)
	var rows []balanceLedgerRow
	if err := json.Unmarshal(w.Body.Bytes(), &rows); err != nil {
		t.Fatal(err)
	}
	return rows
}
func assertLedger(t *testing.T, rows []balanceLedgerRow, want map[int64][2]int64) {
	t.Helper()
	if len(rows) != len(want) {
		t.Fatalf("want %d ledger entries, got %d", len(want), len(rows))
	}
	for _, row := range rows {
		expected, ok := want[row.Amount]
		if !ok || row.Before == nil || row.After == nil || *row.Before != expected[0] || *row.After != expected[1] {
			t.Fatalf(
				"wrong snapshots for %s amount %d: before=%v after=%v",
				row.Kind,
				row.Amount,
				row.Before,
				row.After,
			)
		}
	}
}
func TestMemberBalanceLedgerPostingAndReads(t *testing.T) {
	f := newFixture(t)
	id := f.member(t, "3271010101960041", "teller.a")
	want := map[int64][2]int64{50000: {0, 50000}, 25000: {50000, 75000}, 10000: {75000, 65000}}
	first := f.postCash("teller.b", "ledger-cross-deposit-01", "deposit", id, 25000)
	requireStatus(t, first, 200)
	requireStatus(t, f.postCash("teller.b", "ledger-cross-deposit-01", "deposit", id, 25000), 200)
	requireStatus(t, f.postCash("teller.a", "ledger-withdrawal-001", "withdrawal", id, 10000), 200)
	requireStatus(t, f.postCash("teller.a", "ledger-rejected-0001", "withdrawal", id, 100000), 422)
	paths := []string{fmt.Sprintf("/api/members/%d/transactions", id),
		"/api/transactions?account_number=1000000001"}
	for _, path := range paths {
		response := f.call("supervisor", "GET", path, "", "")
		assertLedger(t, ledgerRows(t, response), want)
		var raw []map[string]json.RawMessage
		if err := json.Unmarshal(response.Body.Bytes(), &raw); err != nil {
			t.Fatal(err)
		}
		if string(raw[len(raw)-1]["balance_before"]) != "0" {
			t.Fatal("opening zero must be included in JSON")
		}
	}
	p := f.call("teller.a", "GET", "/api/transactions/search?period=last_30_days&account_number=1000000001", "", "")
	requireStatus(t, p, 200)
	var envelope struct {
		Items []balanceLedgerRow `json:"items"`
	}
	if err := json.Unmarshal(p.Body.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	assertLedger(t, envelope.Items, want)
	if _, err := f.db.Exec(
		context.Background(),
		`CREATE FUNCTION reject_ledger_audit() RETURNS trigger LANGUAGE plpgsql AS $$
 BEGIN RAISE EXCEPTION 'synthetic ledger rollback'; END $$;
 CREATE TRIGGER reject_ledger_audit BEFORE INSERT ON audit_events
 FOR EACH ROW EXECUTE FUNCTION reject_ledger_audit()`,
	); err != nil {
		t.Fatal(err)
	}
	requireStatus(t, f.postCash("teller.a", "ledger-failed-audit-01", "deposit", id, 123), 500)
	var balance, count, journalCount int64
	if err := f.db.QueryRow(context.Background(), `SELECT balance,(SELECT count(*) FROM transactions),
 (SELECT count(*) FROM journal_lines) FROM members WHERE id=$1`, id).Scan(&balance, &count, &journalCount); err != nil {
		t.Fatal(err)
	}
	if balance != 65000 || count != 3 || journalCount != 6 {
		t.Fatal("retry/rejection/rollback changed ledger or balance")
	}
}
func TestMemberBalanceLedgerConcurrentChain(t *testing.T) {
	f := newFixture(t)
	id := f.member(t, "3271010101960042", "teller.a")
	const count = 8
	statuses := make(chan int, count)
	var wg sync.WaitGroup
	for i := range count {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			actor := "teller.a"
			if i%2 == 1 {
				actor = "teller.b"
			}
			statuses <- f.postCash(actor, fmt.Sprintf("ledger-concurrent-%04d", i), "deposit", id, 1000).Code
		}(i)
	}
	wg.Wait()
	close(statuses)
	for status := range statuses {
		if status != 200 {
			t.Fatal(status)
		}
	}
	rows, err := f.db.Query(context.Background(), `SELECT balance_before,balance_after FROM transactions
 WHERE member_id=$1 ORDER BY balance_before`, id)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	expected := int64(0)
	total := 0
	for rows.Next() {
		var before, after *int64
		if err = rows.Scan(&before, &after); err != nil {
			t.Fatal(err)
		}
		if before == nil || after == nil || *before != expected {
			t.Fatal("ledger chain has a gap or missing snapshot")
		}
		expected = *after
		total++
	}
	if err = rows.Err(); err != nil {
		t.Fatal(err)
	}
	if total != count+1 || expected != 58000 {
		t.Fatal("concurrent ledger total incorrect")
	}
}
func TestMemberBalanceLedgerMigration(t *testing.T) {
	f := newFixtureWithMigration(t, false)
	ctx := context.Background()
	if _, err := f.db.Exec(ctx, `INSERT INTO members(nik,name,phone,address,home_branch_id,balance)
 VALUES('3271010101960043','Legacy ledger member','00000000','Synthetic fixture',1,50000);
 INSERT INTO transactions(id,member_id,branch_id,actor_id,kind,amount)
 SELECT 'TX-legacy-ledger',m.id,1,u.id,'deposit',50000 FROM members m CROSS JOIN users u
 WHERE u.username='teller.a';
 INSERT INTO journal_lines(transaction_id,line_no,account,debit,credit)
 VALUES('TX-legacy-ledger',1,'cash',50000,0),('TX-legacy-ledger',2,'member_savings',0,50000)`); err != nil {
		t.Fatal(err)
	}
	if err := database.Migrate(ctx, f.db); err != nil {
		t.Fatal(err)
	}
	if err := database.Migrate(ctx, f.db); err != nil {
		t.Fatal(err)
	}
	var before, after *int64
	var balance, debits, credits int64
	row := f.db.QueryRow(ctx, `SELECT t.balance_before,t.balance_after,m.balance,
 (SELECT sum(debit) FROM journal_lines),(SELECT sum(credit) FROM journal_lines)
 FROM transactions t JOIN members m ON m.id=t.member_id`)
	if err := row.Scan(&before, &after, &balance, &debits, &credits); err != nil {
		t.Fatal(err)
	}
	if before != nil || after != nil || balance != 50000 || debits != 50000 || credits != 50000 {
		t.Fatal("migration invented snapshots or changed funds")
	}
	rows := ledgerRows(t, f.call("teller.a", "GET", "/api/transactions", "", ""))
	if len(rows) != 1 || rows[0].Before != nil || rows[0].After != nil {
		t.Fatal("unknown historical snapshots must remain null")
	}
	for _, sql := range []string{
		`UPDATE transactions SET balance_before=0`,
		`UPDATE transactions SET balance_before=-1,balance_after=49999`,
		`UPDATE transactions SET balance_before=0,balance_after=1`,
	} {
		if _, err := f.db.Exec(ctx, sql); err == nil {
			t.Fatal("accepted invalid snapshot pair")
		}
	}
}
