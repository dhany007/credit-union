package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"testing"
)

func TestLoanListPaginationScope(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	id := f.member(t, "3271010101908813", "teller.a")
	_, err := f.db.Exec(ctx, `INSERT INTO loan_applications(member_id,receiving_branch_id,responsible_branch_id,
 created_by,amount,tenor_months,purpose,risk)
 SELECT $1,1,1,$2,1000000,12,'Synthetic paginated application','low' FROM generate_series(1,25)`, id, f.ids["teller.a"])
	if err != nil {
		t.Fatal(err)
	}
	w := f.call("teller.a", "GET", "/api/loan-applications", "", "")
	requireStatus(t, w, 200)
	var first struct {
		Items  []json.RawMessage `json:"items"`
		Cursor string            `json:"next_cursor"`
	}
	if err = json.Unmarshal(w.Body.Bytes(), &first); err != nil {
		t.Fatal(err)
	}
	if len(first.Items) != 20 || first.Cursor == "" {
		t.Fatal("unbounded application page")
	}
	next := f.call("teller.a", "GET", "/api/loan-applications?cursor="+url.QueryEscape(first.Cursor), "", "")
	requireStatus(t, next, 200)
	var second struct {
		Items  []json.RawMessage `json:"items"`
		Cursor string            `json:"next_cursor"`
	}
	if err = json.Unmarshal(next.Body.Bytes(), &second); err != nil {
		t.Fatal(err)
	}
	if len(second.Items) != 5 || second.Cursor != "" {
		t.Fatal("incorrect final application page")
	}
	requireStatus(
		t,
		f.call(
			"teller.a",
			"GET",
			"/api/loan-applications?status=approved&cursor="+url.QueryEscape(first.Cursor),
			"",
			"",
		),
		400,
	)
	requireStatus(t, f.call("teller.a", "GET", "/api/loan-accounts?cursor="+url.QueryEscape(first.Cursor), "", ""), 400)
	requireStatus(t, f.call("teller.a", "GET", "/api/loan-applications?account_number=broken", "", ""), 400)
	requireStatus(
		t,
		f.call(
			"teller.a",
			"POST",
			"/api/loans/preview",
			"",
			`{"amount":0,"tenor_months":12,"expected_disbursement_date":"2099-01-01"}`,
		),
		400,
	)
	legacy := f.call("teller.a", "GET", "/api/loans", "", "")
	requireStatus(t, legacy, 200)
	var rows []json.RawMessage
	if err = json.Unmarshal(legacy.Body.Bytes(), &rows); err != nil || len(rows) != 25 {
		t.Fatal(fmt.Sprint("legacy API changed: ", err))
	}
}
