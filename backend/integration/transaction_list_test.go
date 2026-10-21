package integration

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"
)

type listedCashTransaction struct {
	Reference     string    `json:"reference"`
	Kind          string    `json:"kind"`
	Amount        int64     `json:"amount"`
	Note          string    `json:"note"`
	CreatedAt     time.Time `json:"created_at"`
	Branch        string    `json:"branch"`
	Actor         string    `json:"actor"`
	MemberID      int64     `json:"member_id"`
	MemberName    string    `json:"member_name"`
	MemberNumber  string    `json:"member_number"`
	AccountNumber string    `json:"account_number"`
}

func decodeTransactions(t *testing.T, w *httptest.ResponseRecorder) []listedCashTransaction {
	t.Helper()
	requireStatus(t, w, 200)
	var rows []listedCashTransaction
	if err := json.Unmarshal(w.Body.Bytes(), &rows); err != nil {
		t.Fatal(err)
	}
	if rows == nil {
		t.Fatal("transaction endpoint must return an array, not null")
	}
	return rows
}

func TestTransactionListingAndAccountFilter(t *testing.T) {
	f := newFixture(t)
	first := f.member(t, "3271010101960001", "teller.a")
	second := f.member(t, "3271010101960002", "teller.b")
	ctx := context.Background()
	if _, err := f.db.Exec(
		ctx,
		`UPDATE members SET name=CASE WHEN id=$1 THEN 'First Account' ELSE 'Second Account' END`,
		first,
	); err != nil {
		t.Fatal(err)
	}
	requireStatus(t, f.postCash("teller.b", "list-cross-deposit-001", "deposit", first, 25000), 200)
	requireStatus(t, f.postCash("teller.a", "list-withdrawal-0001", "withdrawal", first, 5000), 200)
	snapshot := func() [6]int64 {
		t.Helper()
		var result [6]int64
		err := f.db.QueryRow(ctx, `
            SELECT (SELECT coalesce(sum(balance),0) FROM members),
                (SELECT count(*) FROM transactions),
                (SELECT count(*) FROM journal_lines),
                (SELECT coalesce(sum(debit),0) FROM journal_lines),
                (SELECT coalesce(sum(credit),0) FROM journal_lines),
                (SELECT count(*) FROM requests)
        `).Scan(&result[0], &result[1], &result[2], &result[3], &result[4], &result[5])
		if err != nil {
			t.Fatal(err)
		}
		return result
	}
	before := snapshot()
	allResponse := f.call("teller.a", "GET", "/api/transactions", "", "")
	all := decodeTransactions(t, allResponse)
	if len(all) != 4 {
		t.Fatalf("want two opening deposits and two cash postings, got %d", len(all))
	}
	if strings.Contains(allResponse.Body.String(), `"nik"`) || strings.Contains(allResponse.Body.String(), `"phone"`) {
		t.Fatal("listing exposed unrelated member identity fields")
	}
	for _, row := range all {
		if row.Reference == "" || row.CreatedAt.IsZero() || row.MemberNumber == "" || row.AccountNumber == "" {
			t.Fatal("missing transaction attribution")
		}
		if row.MemberID == first && (row.AccountNumber != "1000000001" || row.MemberName != "First Account") {
			t.Fatal("wrong first-account attribution")
		}
		if row.MemberID == second && (row.AccountNumber != "1000000002" || row.MemberName != "Second Account") {
			t.Fatal("wrong second-account attribution")
		}
	}
	filtered := decodeTransactions(t, f.call("teller.b", "GET", "/api/transactions?account_number=1000000001", "", ""))
	if len(filtered) != 3 {
		t.Fatalf("want only the first account's three postings, got %d", len(filtered))
	}
	deposits, withdrawals := 0, 0
	for _, row := range filtered {
		if row.MemberID != first {
			t.Fatal("account filter leaked another member's transactions")
		}
		if row.Kind == "deposit" {
			deposits++
		}
		if row.Kind == "withdrawal" {
			withdrawals++
			if row.Amount != 5000 {
				t.Fatal("wrong withdrawal amount")
			}
		}
		if row.Amount == 25000 && (row.Branch != "KCP Utara" || row.Actor != "Teller Utara") {
			t.Fatal("wrong servicing KCP/actor")
		}
	}
	if deposits != 2 || withdrawals != 1 {
		t.Fatal("opening deposit or cash direction missing")
	}
	trimmed := decodeTransactions(
		t,
		f.call("supervisor", "GET", "/api/transactions?account_number="+url.QueryEscape(" 1000000001 "), "", ""),
	)
	if !reflect.DeepEqual(filtered, trimmed) {
		t.Fatal("supervisor/trimmed filter changed transaction scope")
	}
	if len(decodeTransactions(t, f.call("teller.b", "GET", "/api/transactions", "", ""))) != 4 {
		t.Fatal("cross-KCP listing was restricted")
	}
	unknown := f.call("teller.a", "GET", "/api/transactions?account_number=9999999998", "", "")
	if len(decodeTransactions(t, unknown)) != 0 || strings.TrimSpace(unknown.Body.String()) != "[]" {
		t.Fatal("unknown account must return an empty array")
	}
	for _, invalid := range []string{"100000", "10000000010", "abcdefghij", "0000000000", "1000000001' OR 1=1"} {
		requireStatus(
			t,
			f.call("teller.a", "GET", "/api/transactions?account_number="+url.QueryEscape(invalid), "", ""),
			400,
		)
	}
	requireStatus(t, f.call("", "GET", "/api/transactions", "", ""), 401)
	requireStatus(t, f.call("", "GET", "/api/transactions?account_number=1000000001", "", ""), 401)
	if snapshot() != before {
		t.Fatal("read-only listing changed balances, journals or idempotency records")
	}
}

func TestTransactionListingLatest100AndStableTies(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	_, err := f.db.Exec(ctx, `
        INSERT INTO members(id,nik,name,phone,address,home_branch_id,balance)
            OVERRIDING SYSTEM VALUE
            VALUES(1000001,'3271010101960011','Large Member ID','00000000','Synthetic fixture',1,101);
        INSERT INTO transactions(id,member_id,branch_id,actor_id,kind,amount,note,created_at)
            SELECT 'TX-list-'||lpad(n::text,4,'0'),1000001,1,u.id,'deposit',1,'Fixture',
                '2026-01-01 00:00:00+00'::timestamptz
            FROM generate_series(1,101) AS n CROSS JOIN users u WHERE u.username='teller.a';
        INSERT INTO journal_lines(transaction_id,line_no,account,debit,credit)
            SELECT id,1,'cash',1,0 FROM transactions;
        INSERT INTO journal_lines(transaction_id,line_no,account,debit,credit)
            SELECT id,2,'member_savings',0,1 FROM transactions;
    `)
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/api/transactions", "/api/transactions?account_number=1000000001"} {
		rows := decodeTransactions(t, f.call("teller.a", "GET", path, "", ""))
		if len(rows) != 100 || rows[0].Reference != "TX-list-0101" || rows[99].Reference != "TX-list-0002" {
			t.Fatal("latest-100 limit or stable timestamp tie ordering is incorrect")
		}
		for i, row := range rows {
			if row.MemberNumber != "CU-1000001" {
				t.Fatal("member number was truncated")
			}
			if i > 0 && rows[i-1].Reference <= row.Reference {
				t.Fatal("reference tie-breaker is not descending")
			}
		}
	}
}

func TestTransactionListingEmptyAndSafeFailure(t *testing.T) {
	f := newFixture(t)
	for _, path := range []string{"/api/transactions", "/api/transactions?account_number=1000000001"} {
		if len(decodeTransactions(t, f.call("supervisor", "GET", path, "", ""))) != 0 {
			t.Fatal("new database must have no transactions")
		}
	}
	if _, err := f.db.Exec(context.Background(), `ALTER TABLE transactions RENAME TO unavailable_history`); err != nil {
		t.Fatal(err)
	}
	response := f.call("teller.a", "GET", "/api/transactions", "", "")
	requireStatus(t, response, 500)
	if strings.Contains(response.Body.String(), "relation") || strings.Contains(response.Body.String(), "SELECT") {
		t.Fatal("database error details leaked")
	}
}
