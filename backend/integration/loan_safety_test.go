package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"credit-union/backend/internal/platform/database"
)

func postLoanJSON(t *testing.T, f *fixture, actor, path, key string, input any) *httptestResponse {
	t.Helper()
	b, err := json.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}
	w := f.call(actor, "POST", path, key, string(b))
	return &httptestResponse{Code: w.Code, Body: w.Body.String()}
}

type httptestResponse struct {
	Code int
	Body string
}

func requireLoanCode(t *testing.T, w *httptestResponse, code int) {
	t.Helper()
	if w.Code != code {
		t.Fatalf("want %d got %d: %s", code, w.Code, w.Body)
	}
}
func approvedLoanCase(t *testing.T) (*fixture, int64, int64, map[string]any) {
	t.Helper()
	f := newFixture(t)
	ctx := context.Background()
	member := f.member(t, "3271010101908811", "teller.a")
	requireStatus(t, f.postCash("teller.a", "safety-cash-fund-01", "deposit", member, 4000000), http.StatusOK)
	var policyID int64
	if err := f.db.QueryRow(ctx, `
 INSERT INTO loan_interest_policies(effective_date,base_rate_bps,bands,reason)
 VALUES(CURRENT_DATE,1200,'[]','Synthetic safety tariff') RETURNING id`).Scan(&policyID); err != nil {
		t.Fatal(err)
	}
	w := postLoanJSON(t, f, "teller.b", "/api/loans", "safety-application-01", map[string]any{
		"member_id":             member,
		"amount":                1000000,
		"tenor_months":          12,
		"purpose":               "Synthetic safety loan",
		"risk":                  "low",
		"responsible_branch_id": 1,
	})
	requireLoanCode(t, w, http.StatusOK)
	var app struct {
		ID int64 `json:"id"`
	}
	if err := json.Unmarshal([]byte(w.Body), &app); err != nil {
		t.Fatal(err)
	}
	decision := map[string]any{
		"decision":                   "approved",
		"reason":                     "Synthetic reviewed approval",
		"approved_amount":            1000000,
		"approved_tenor_months":      12,
		"expected_policy_id":         policyID,
		"expected_rate_bps":          1200,
		"expected_approval_revision": 0,
	}
	requireLoanCode(
		t,
		postLoanJSON(t, f, "supervisor", fmt.Sprintf("/api/loans/%d/decision", app.ID), "safety-approval-01", decision),
		http.StatusOK,
	)
	today := time.Now().In(time.FixedZone("Jakarta", 7*60*60)).Format(time.DateOnly)
	input := map[string]any{
		"expected_approval_revision": 1,
		"expected_policy_id":         policyID,
		"expected_rate_bps":          1200,
		"expected_principal":         1000000,
		"expected_tenor_months":      12,
		"expected_payout_date":       today,
		"recipient_verified":         true,
		"agreement_signed":           true,
		"conditions_verified":        true,
		"verification_note":          "Synthetic identity and agreement checked",
	}
	return f, app.ID, member, input
}

func TestLoanConcurrencyAndRollback(t *testing.T) {
	f, applicationID, member, input := approvedLoanCase(t)
	ctx := context.Background()
	path := fmt.Sprintf("/api/loan-applications/%d/disburse", applicationID)
	codes := make(chan int, 2)
	var wg sync.WaitGroup
	for i := range 2 {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			w := postLoanJSON(t, f, "supervisor", path, fmt.Sprintf("payout-race-key-%04d", i), input)
			codes <- w.Code
		}(i)
	}
	wg.Wait()
	close(codes)
	ok, conflict := 0, 0
	for code := range codes {
		if code == 200 {
			ok++
		}
		if code == 409 {
			conflict++
		}
	}
	if ok != 1 || conflict != 1 {
		t.Fatalf("payout outcomes %d/%d", ok, conflict)
	}
	var loanID, versionID int64
	if err := f.db.QueryRow(ctx, "SELECT l.id,v.id FROM loans l JOIN loan_schedule_versions v ON v.loan_id=l.id").
		Scan(&loanID, &versionID); err != nil {
		t.Fatal(err)
	}
	paymentPath := fmt.Sprintf("/api/loan-accounts/%d/repayments", loanID)
	payment := map[string]any{
		"amount":                       50000,
		"expected_payment_revision":    1,
		"expected_schedule_version_id": versionID,
	}
	if _, err := f.db.Exec(ctx, `CREATE FUNCTION fail_loan_journal() RETURNS trigger LANGUAGE plpgsql AS $$
 BEGIN RAISE EXCEPTION 'test journal failure'; END $$;
 CREATE TRIGGER fail_loan_journal BEFORE INSERT ON journal_lines FOR EACH ROW EXECUTE FUNCTION
fail_loan_journal()`); err != nil {
		t.Fatal(err)
	}
	requireLoanCode(t, postLoanJSON(t, f, "supervisor", paymentPath, "rollback-payment-key01", payment), 500)
	var receipts, allocations, revision int64
	if err := f.db.QueryRow(ctx, `SELECT (SELECT count(*) FROM loan_repayments),
 (SELECT count(*) FROM loan_repayment_allocations),payment_revision FROM loans WHERE id=$1`, loanID).
		Scan(&receipts, &allocations, &revision); err != nil {
		t.Fatal(err)
	}
	if receipts != 0 || allocations != 0 || revision != 1 {
		t.Fatal("partial failed payment")
	}
	if _, err := f.db.Exec(
		ctx,
		"DROP TRIGGER fail_loan_journal ON journal_lines; DROP FUNCTION fail_loan_journal()",
	); err != nil {
		t.Fatal(err)
	}
	codes = make(chan int, 2)
	for i := range 2 {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			codes <- postLoanJSON(t, f, "supervisor", paymentPath, fmt.Sprintf("pay-race-key-%05d", i), payment).Code
		}(i)
	}
	wg.Wait()
	close(codes)
	ok, conflict = 0, 0
	for code := range codes {
		if code == 200 {
			ok++
		}
		if code == 409 {
			conflict++
		}
	}
	if ok != 1 || conflict != 1 {
		t.Fatalf("payment outcomes %d/%d", ok, conflict)
	}
	var due int64
	if err := f.db.QueryRow(ctx, `
 SELECT (SELECT sum(principal_due+interest_due) FROM loan_schedules WHERE loan_id=$1)-
 (SELECT sum(amount) FROM loan_repayments WHERE loan_id=$1)`, loanID).Scan(&due); err != nil {
		t.Fatal(err)
	}
	payment["amount"] = due
	payment["expected_payment_revision"] = 2
	requireLoanCode(t, postLoanJSON(t, f, "supervisor", paymentPath, "payoff-key-0000001", payment), 200)
	requireLoanCode(t, postLoanJSON(t, f, "supervisor", paymentPath, "payoff-key-0000001", payment), 200)
	var state string
	if err := f.db.QueryRow(ctx, "SELECT status FROM loans WHERE id=$1", loanID).
		Scan(&state); err != nil ||
		state != "paid_off" {
		t.Fatal("loan not paid off", err)
	}
	history := f.call("teller.a", "GET", fmt.Sprintf("/api/members/%d/transactions", member), "", "")
	requireStatus(t, history, 200)
	var rows []json.RawMessage
	if err := json.Unmarshal(history.Body.Bytes(), &rows); err != nil || len(rows) != 2 {
		t.Fatal("loan flows leaked into savings history", err)
	}
	report, err := os.ReadFile("../../docs/testing/sql/phase-1-reconciliation.sql")
	if err != nil {
		t.Fatal(err)
	}
	sql := strings.TrimPrefix(string(report), "\\set ON_ERROR_STOP on\n")
	if _, err = f.db.Exec(ctx, sql); err != nil {
		t.Fatal("loan-aware recovery reconciliation failed", err)
	}

}

func TestLoanMinimumTariffAPIAndSessionAudit(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	tomorrow := time.Now().In(time.FixedZone("Jakarta", 7*60*60)).AddDate(0, 0, 1).Format(time.DateOnly)
	base := fmt.Sprintf(
		`{"effective_date":"%s","base_rate_bps":300,"bands":[],"reason":"Synthetic minimum tariff"}`,
		tomorrow,
	)
	for i, body := range []string{
		strings.Replace(base, "\"base_rate_bps\":300,", "", 1),
		strings.Replace(base, ":300", ":null", 1),
		strings.Replace(base, ":300", ":299", 1),
		strings.Replace(base, "\"bands\":[]", "\"bands\":null", 1),
		strings.Replace(base, "\"bands\":[]", `"bands":[{"min_amount":0,"max_amount":null}]`, 1),
	} {
		requireStatus(
			t,
			f.call("supervisor", "POST", "/api/loan-interest/policies", fmt.Sprintf("invalid-policy-key%03d", i), body),
			400,
		)
	}
	var policies, requests int
	if err := f.db.QueryRow(ctx, "SELECT (SELECT count(*) FROM loan_interest_policies),(SELECT count(*) FROM requests)").
		Scan(&policies, &requests); err != nil ||
		policies != 0 ||
		requests != 0 {
		t.Fatal("invalid tariff persisted data", err)
	}
	requireStatus(t, f.call("supervisor", "POST", "/api/loan-interest/policies", "minimum-policy-key01", base), 200)
	if _, err := f.db.Exec(ctx, `
 INSERT INTO loan_interest_policies(effective_date,base_rate_bps,bands,reason)
 VALUES(CURRENT_DATE,299,'[]','Rejected direct tariff')`); err == nil {
		t.Fatal("database accepted below-minimum tariff")
	}
	requireStatus(t, f.call("teller.a", "POST", "/api/logout", "", `{}`), 200)
	var updater *int64
	if err := f.db.QueryRow(ctx, "SELECT updated_by FROM sessions WHERE token_hash=$1",
		database.Hash(f.cookies["teller.a"]),
	).
		Scan(&updater); err != nil {
		t.Fatal(err)
	}
	if updater == nil || *updater != f.ids["teller.a"] {
		t.Fatal("logout lost known actor")
	}
}
