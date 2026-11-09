package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"
)

func TestLoanAuditAndPricingGuards(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	var missing int
	err := f.db.QueryRow(ctx, `SELECT count(*) FROM information_schema.tables t
 WHERE t.table_schema=current_schema() AND t.table_type='BASE TABLE'
 AND EXISTS(SELECT 1 FROM unnest(ARRAY['is_active','created_by','created_at','updated_by','updated_at',
'deleted_by','deleted_at']) c
 WHERE NOT EXISTS(SELECT 1 FROM information_schema.columns x WHERE x.table_schema=t.table_schema
 AND x.table_name=t.table_name AND x.column_name=c))`).Scan(&missing)
	if err != nil || missing != 0 {
		t.Fatalf("audit fields missing on %d tables: %v", missing, err)
	}
	body := fmt.Sprintf(
		`{"amount":1000000,"tenor_months":12,"expected_disbursement_date":"%s"}`,
		time.Now().AddDate(0, 0, 1).Format(time.DateOnly),
	)
	requireStatus(t, f.call("teller.a", "POST", "/api/loans/preview", "", body), 422)
	tomorrow := time.Now().In(time.FixedZone("Jakarta", 7*60*60)).AddDate(0, 0, 1).Format(time.DateOnly)
	policy := fmt.Sprintf(
		`{"effective_date":"%s","base_rate_bps":1200,"bands":[{"min_amount":1000000,"max_amount":10000000,
"rate_bps":1000}],"reason":"Synthetic local tariff","expected_latest_id":null}`,
		tomorrow,
	)
	requireStatus(t, f.call("teller.a", "POST", "/api/loan-interest/policies", "policy-key-teller-01", policy), 403)
	w := f.call("supervisor", "POST", "/api/loan-interest/policies", "policy-key-create-01", policy)
	requireStatus(t, w, 200)
	requireStatus(t, f.call("supervisor", "POST", "/api/loan-interest/policies", "policy-key-create-01", policy), 200)
	requireStatus(t, f.call("supervisor", "POST", "/api/loan-interest/policies", "policy-key-create-02", policy), 409)
	// Fixture-only date adjustment makes the prospective policy effective for contract tests.
	if _, err := f.db.Exec(ctx, "UPDATE loan_interest_policies SET effective_date=CURRENT_DATE"); err != nil {
		t.Fatal(err)
	}
	w = f.call("teller.a", "POST", "/api/loans/preview", "", body)
	requireStatus(t, w, 200)
	var preview struct {
		Rate         int       `json:"annual_rate_bps"`
		Total        int64     `json:"total_principal"`
		CalculatedAt time.Time `json:"calculated_at"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &preview); err != nil {
		t.Fatal(err)
	}
	if delta := time.Since(preview.CalculatedAt); delta < -time.Minute || delta > time.Minute {
		t.Fatal("preview timestamp is not an actual instant")
	}
	if preview.Rate != 1000 || preview.Total != 1000000 {
		t.Fatalf("bad pricing %s", w.Body.String())
	}
	var count int
	if err := f.db.QueryRow(ctx, "SELECT count(*) FROM loans").Scan(&count); err != nil || count != 0 {
		t.Fatal("preview created a loan")
	}
}
