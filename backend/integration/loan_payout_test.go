package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
)

func TestLoanPayoutAndRepayment(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	member := f.member(t, "3271010101908801", "teller.b")
	requireStatus(t, f.postCash("teller.a", "loan-fund-cash-0001", "deposit", member, 2000000), 200)
	_, err := f.db.Exec(ctx, `
 INSERT INTO loan_interest_policies(effective_date,base_rate_bps,bands,reason)
 VALUES(CURRENT_DATE,1200,'[]','Synthetic test policy')`)
	if err != nil {
		t.Fatal(err)
	}
	var policyID int64
	if err = f.db.QueryRow(ctx, "SELECT id FROM loan_interest_policies").Scan(&policyID); err != nil {
		t.Fatal(err)
	}
	w := f.call(
		"teller.b",
		"POST",
		"/api/loans",
		"phase3-application-0001",
		fmt.Sprintf(
			`{"member_id":%d,"amount":1000000,"tenor_months":12,"purpose":"Synthetic business loan","risk":"low",
"responsible_branch_id":1}`,
			member,
		),
	)
	requireStatus(t, w, 200)
	var app struct {
		ID int64 `json:"id"`
	}
	if err = json.Unmarshal(w.Body.Bytes(), &app); err != nil {
		t.Fatal(err)
	}
	decision := fmt.Sprintf(`{"decision":"approved","reason":"Synthetic terms verified","approved_amount":1000000,
 "approved_tenor_months":12,"expected_policy_id":%d,"expected_rate_bps":1200,
"expected_approval_revision":0}`, policyID)
	requireStatus(
		t,
		f.call("supervisor", "POST", fmt.Sprintf("/api/loans/%d/decision", app.ID), "phase3-decision-0001", decision),
		200,
	)
	path := fmt.Sprintf("/api/loan-applications/%d", app.ID)
	requireStatus(
		t,
		f.call("teller.a", "POST", path+"/disbursement-preview", "", `{"expected_approval_revision":1}`),
		403,
	)
	w = f.call("supervisor", "POST", path+"/disbursement-preview", "", `{"expected_approval_revision":1}`)
	requireStatus(t, w, 200)
	var preview struct {
		Date string `json:"payout_date"`
	}
	if err = json.Unmarshal(w.Body.Bytes(), &preview); err != nil {
		t.Fatal(err)
	}
	payout := fmt.Sprintf(`{"expected_approval_revision":1,"expected_policy_id":%d,"expected_rate_bps":1200,
 "expected_principal":1000000,"expected_tenor_months":12,"expected_payout_date":"%s",
 "recipient_verified":true,"agreement_signed":true,"conditions_verified":true,
 "verification_note":"Identity and signed agreement verified","note":"Synthetic payout"}`, policyID, preview.Date)
	for _, status := range []string{"submitted", "rejected"} {
		if _, err = f.db.Exec(ctx, "UPDATE loan_applications SET status=$1 WHERE id=$2", status, app.ID); err != nil {
			t.Fatal(err)
		}
		requireStatus(
			t,
			f.call("supervisor", "POST", path+"/disbursement-preview", "", `{"expected_approval_revision":1}`),
			422,
		)
		requireStatus(t, f.call("supervisor", "POST", path+"/disburse", "blocked-status-"+status, payout), 422)
	}
	if _, err = f.db.Exec(ctx, "UPDATE loan_applications SET status='approved' WHERE id=$1", app.ID); err != nil {
		t.Fatal(err)
	}
	requireStatus(t, f.call("teller.a", "POST", path+"/disburse", "phase3-payout-wrong", payout), 403)
	w = f.call("supervisor", "POST", path+"/disburse", "phase3-payout-key-01", payout)
	requireStatus(t, w, 200)
	var paid struct {
		ID        int64  `json:"loan_id"`
		Reference string `json:"reference"`
	}
	if err = json.Unmarshal(w.Body.Bytes(), &paid); err != nil {
		t.Fatal(err)
	}
	replay := f.call("supervisor", "POST", path+"/disburse", "phase3-payout-key-01", payout)
	requireStatus(t, replay, 200)
	requireStatus(t, f.call("supervisor", "POST", path+"/disburse", "phase3-payout-key-02", payout), 409)
	loanPath := fmt.Sprintf("/api/loan-accounts/%d", paid.ID)
	requireStatus(t, f.call("teller.a", "GET", loanPath, "", ""), 200)
	requireStatus(t, f.call("teller.a", "GET", loanPath+"/installments", "", ""), 200)
	requireStatus(t, f.call("teller.a", "GET", loanPath+"/schedule-versions", "", ""), 200)
	// The same Supervisor services repayment from another KCP without changing loan ownership.
	if _, err = f.db.Exec(ctx, "UPDATE users SET branch_id=2 WHERE id=$1", f.ids["supervisor"]); err != nil {
		t.Fatal(err)
	}
	requireStatus(t, f.call("teller.b", "POST", loanPath+"/repayment-preview", "", `{"amount":50000}`), 403)
	requireStatus(t, f.call("teller.b", "POST", loanPath+"/repayments", "teller-blocked-pay01", `{}`), 403)
	w = f.call("supervisor", "POST", loanPath+"/repayment-preview", "", `{"amount":50000}`)
	requireStatus(t, w, 200)
	var plan struct {
		Version int64 `json:"current_schedule_version_id"`
	}
	if err = json.Unmarshal(w.Body.Bytes(), &plan); err != nil {
		t.Fatal(err)
	}
	payment := fmt.Sprintf(
		`{"amount":50000,"expected_payment_revision":1,"expected_schedule_version_id":%d,
"note":"Partial at another KCP"}`,
		plan.Version,
	)
	w = f.call("supervisor", "POST", loanPath+"/repayments", "phase3-payment-key01", payment)
	requireStatus(t, w, 200)
	requireStatus(t, f.call("supervisor", "POST", loanPath+"/repayments", "phase3-payment-key01", payment), 200)
	requireStatus(t, f.call("supervisor", "POST", loanPath+"/repayments", "phase3-payment-key02", payment), 409)
	requireStatus(t, f.call("supervisor", "POST", loanPath+"/repayment-preview", "", `{"amount":999999999}`), 422)
	requireStatus(t, f.call("teller.a", "GET", loanPath+"/repayments", "", ""), 200)
	var balance, principal, interest, unbalanced int64
	if err = f.db.QueryRow(ctx, "SELECT balance FROM members WHERE id=$1", member).Scan(&balance); err != nil {
		t.Fatal(err)
	}
	if balance != 2050000 {
		t.Fatalf("loan mutated savings %d", balance)
	}
	if err = f.db.QueryRow(ctx, "SELECT sum(principal_amount),sum(interest_amount) FROM loan_repayment_allocations").
		Scan(&principal, &interest); err != nil {
		t.Fatal(err)
	}
	if principal != 40000 || interest != 10000 {
		t.Fatalf("allocation %d/%d", principal, interest)
	}
	if err = f.db.QueryRow(ctx, `
 SELECT count(*) FROM (SELECT transaction_id FROM journal_lines GROUP BY transaction_id HAVING
sum(debit)<>sum(credit)) x`).
		Scan(&unbalanced); err != nil ||
		unbalanced != 0 {
		t.Fatal("unbalanced journal", err)
	}
	var responsible, servicing int
	if err = f.db.QueryRow(ctx, `SELECT l.responsible_branch_id,r.servicing_branch_id
 FROM loans l JOIN loan_repayments r ON r.loan_id=l.id`).
		Scan(&responsible, &servicing); err != nil {
		t.Fatal(err)
	}
	if responsible != 1 || servicing != 2 {
		t.Fatal("KCP ownership changed")
	}
	var inactive int
	if _, err = f.db.Exec(ctx, "UPDATE members SET is_active=false WHERE id=$1", member); err != nil {
		t.Fatal(err)
	}
	requireStatus(t, f.postCash("teller.a", "phase3-inactive-cash", "deposit", member, 10), 404)
	requireStatus(t, f.call("teller.a", "GET", loanPath, "", ""), 200)
	if err = f.db.QueryRow(ctx, "SELECT count(*) FROM loan_schedule_versions WHERE is_current_version AND is_active").
		Scan(&inactive); err != nil ||
		inactive != 1 {
		t.Fatal("current version")
	}
}
