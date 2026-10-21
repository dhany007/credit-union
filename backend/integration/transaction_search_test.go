package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

type transactionPage struct {
	Items      []listedCashTransaction `json:"items"`
	NextCursor string                  `json:"next_cursor"`
	StartDate  string                  `json:"start_date"`
	EndDate    string                  `json:"end_date"`
	TimeZone   string                  `json:"time_zone"`
}

func decodeTransactionPage(t *testing.T, w *httptest.ResponseRecorder) transactionPage {
	t.Helper()
	requireStatus(t, w, 200)
	var p transactionPage
	if err := json.Unmarshal(w.Body.Bytes(), &p); err != nil {
		t.Fatal(err)
	}
	if p.Items == nil || p.TimeZone != "Asia/Jakarta" {
		t.Fatal("invalid envelope")
	}
	return p
}
func TestTransactionDateSearch(t *testing.T) {
	f := newFixture(t)
	first := f.member(t, "3271010101960021", "teller.a")
	second := f.member(t, "3271010101960022", "teller.b")
	ctx := context.Background()
	_, err := f.db.Exec(ctx, `UPDATE transactions SET created_at=CASE WHEN member_id=$1 THEN
 '2026-02-28 17:00:00+00'::timestamptz ELSE '2026-03-01 17:00:00+00'::timestamptz END`, first)
	if err != nil {
		t.Fatal(err)
	}
	base := "/api/transactions/search?period=custom&start_date=2026-03-01&end_date=2026-03-01"
	p := decodeTransactionPage(t, f.call("teller.a", "GET", base, "", ""))
	if len(p.Items) != 1 || p.Items[0].MemberID != first || p.StartDate != "2026-03-01" || p.EndDate != "2026-03-01" {
		t.Fatal("Jakarta boundary mismatch")
	}
	p = decodeTransactionPage(t, f.call("supervisor", "GET", base+"&account_number=1000000002", "", ""))
	if len(p.Items) != 0 {
		t.Fatal("combined filter mismatch")
	}
	_, err = f.db.Exec(
		ctx,
		`UPDATE transactions SET created_at='2026-03-01 16:59:59.999999+00' WHERE member_id=$1`,
		second,
	)
	if err != nil {
		t.Fatal(err)
	}
	p = decodeTransactionPage(t, f.call("teller.b", "GET", base, "", ""))
	if len(p.Items) != 2 {
		t.Fatal("end date not inclusive")
	}
	for _, q := range []string{
		"period=invalid", "period=custom", "period=custom&start_date=2026-03-02&end_date=2026-03-01",
		"period=today&end_date=2026-03-01", "cursor=invalid", "account_number=123",
	} {
		requireStatus(t, f.call("teller.a", "GET", "/api/transactions/search?"+q, "", ""), 400)
	}
	requireStatus(t, f.call("", "GET", base, "", ""), 401)
	if len(decodeTransactions(t, f.call("teller.a", "GET", "/api/transactions", "", ""))) != 2 {
		t.Fatal("legacy changed")
	}
}
func TestTransactionSearchPagination(t *testing.T) {
	f := newFixture(t)
	id := f.member(t, "3271010101960023", "teller.a")
	if _, err := f.db.Exec(
		context.Background(),
		`UPDATE transactions SET created_at='2026-01-01 00:00:00+00'`,
	); err != nil {
		t.Fatal(err)
	}
	_, err := f.db.Exec(context.Background(), `
 INSERT INTO transactions(id,member_id,branch_id,actor_id,kind,amount,note,created_at)
 SELECT 'TX-page-'||lpad(n::text,4,'0'),$1,1,u.id,'deposit',1,'Synthetic pagination fixture',
 '2026-01-01 00:00:00+00'::timestamptz FROM generate_series(1,120) n CROSS JOIN users u
 WHERE u.username='teller.a'`, id)
	if err != nil {
		t.Fatal(err)
	}
	base := "/api/transactions/search?period=custom&start_date=2026-01-01&end_date=2026-01-01"
	cursor := ""
	seen := map[string]bool{}
	previous := ""
	for page := 0; page < 3; page++ {
		p := decodeTransactionPage(t, f.call("teller.a", "GET", base+"&cursor="+url.QueryEscape(cursor), "", ""))
		want := 50
		if page == 2 {
			want = 21
		}
		if len(p.Items) != want {
			t.Fatalf("page %d: want %d got %d", page, want, len(p.Items))
		}
		for _, row := range p.Items {
			if seen[row.Reference] || (previous != "" && previous <= row.Reference) {
				t.Fatal("duplicated or unordered row")
			}
			seen[row.Reference] = true
			previous = row.Reference
		}
		if page == 0 {
			paths := []string{base + "&account_number=1000000001",
				strings.Replace(base, "end_date=2026-01-01", "end_date=2026-01-02", 1)}
			for _, path := range paths {
				requireStatus(t, f.call("teller.a", "GET", path+"&cursor="+url.QueryEscape(p.NextCursor), "", ""), 400)
			}
		}
		cursor = p.NextCursor
		if (page < 2) == (cursor == "") {
			t.Fatal("incorrect has-more marker")
		}
	}
	if len(seen) != 121 {
		t.Fatal(fmt.Sprint("incomplete pagination: ", len(seen)))
	}
}

func TestTransactionSearchPageBoundaryAndFailure(t *testing.T) {
	f := newFixture(t)
	id := f.member(t, "3271010101960024", "teller.a")
	ctx := context.Background()
	if _, err := f.db.Exec(ctx, `UPDATE transactions SET created_at='2026-01-01 00:00:00+00'`); err != nil {
		t.Fatal(err)
	}
	insert := func(first, last int) {
		t.Helper()
		_, err := f.db.Exec(ctx, `INSERT INTO transactions(id,member_id,branch_id,actor_id,kind,amount,note,created_at)
  SELECT 'TX-bound-'||lpad(n::text,4,'0'),$1,1,u.id,'deposit',1,'Synthetic boundary fixture',
  '2026-01-01 00:00:00+00'::timestamptz FROM generate_series($2::int,$3::int) n CROSS JOIN users u
  WHERE u.username='teller.a'`, id, first, last)
		if err != nil {
			t.Fatal(err)
		}
	}
	path := "/api/transactions/search?period=custom&start_date=2026-01-01&end_date=2026-01-01"
	insert(1, 49)
	p := decodeTransactionPage(t, f.call("teller.a", "GET", path, "", ""))
	if len(p.Items) != 50 || p.NextCursor != "" {
		t.Fatal("exactly 50 must be final page")
	}
	insert(50, 50)
	p = decodeTransactionPage(t, f.call("teller.a", "GET", path, "", ""))
	if len(p.Items) != 50 || p.NextCursor == "" {
		t.Fatal("51 must offer next page")
	}
	next := decodeTransactionPage(t, f.call("teller.a", "GET", path+"&cursor="+url.QueryEscape(p.NextCursor), "", ""))
	if len(next.Items) != 1 || next.NextCursor != "" {
		t.Fatal("lookahead row was skipped")
	}
	empty := decodeTransactionPage(t, f.call("supervisor", "GET", path+"&account_number=9999999998", "", ""))
	if len(empty.Items) != 0 || empty.NextCursor != "" {
		t.Fatal("unknown account must be empty")
	}
	if _, err := f.db.Exec(ctx, `ALTER TABLE transactions RENAME TO unavailable_search`); err != nil {
		t.Fatal(err)
	}
	failed := f.call("teller.a", "GET", path, "", "")
	requireStatus(t, failed, 500)
	if strings.Contains(failed.Body.String(), "relation") || strings.Contains(failed.Body.String(), "SELECT") {
		t.Fatal("SQL leaked")
	}
}
