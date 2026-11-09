package integration

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"credit-union/backend/internal/platform/database"
)

func TestAuditMigrationWithPopulatedInterest(t *testing.T) {
	f := newFixtureWithMigration(t, false)
	ctx := context.Background()
	for _, name := range []string{
		"0002_savings_account_number.sql",
		"0003_member_balance_snapshots.sql",
		"0004_savings_interest.sql",
	} {
		//nolint:gosec // Migration fixture names come only from the fixed allowlist above.
		sql, err := os.ReadFile(filepath.Join("..", "internal", "platform", "database", "migrations", name))
		if err != nil {
			t.Fatal(err)
		}
		if _, err = f.db.Exec(ctx, string(sql)); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := f.db.Exec(
		ctx,
		`CREATE TABLE schema_migrations(version integer PRIMARY KEY,applied_at timestamptz NOT NULL DEFAULT now());
 INSERT INTO schema_migrations(version) VALUES(1),(2),(3),(4);
 INSERT INTO members(nik,name,phone,address,home_branch_id) 
 VALUES('3271010101908812','Legacy interest member','08123456789','Synthetic address',1);
 INSERT INTO interest_policies(effective_date,base_rate_bps,bands,created_by,reason)
 SELECT CURRENT_DATE-1,300,'[]',id,'Synthetic historical interest' FROM users WHERE username='supervisor';
 INSERT INTO interest_days(business_date,eligible_account_count,policy_id)
 SELECT CURRENT_DATE-1,1,id FROM interest_policies;
 INSERT INTO interest_daily(member_id,business_date,policy_id,closing_balance,rate_bps,numerator)
 SELECT m.id,CURRENT_DATE-1,p.id,0,300,0 FROM members m CROSS JOIN interest_policies p;`,
	); err != nil {
		t.Fatal(err)
	}
	if err := database.Migrate(ctx, f.db); err != nil {
		t.Fatal(err)
	}
	var known bool
	if err := f.db.QueryRow(ctx, "SELECT created_at=calculated_at FROM interest_daily").
		Scan(&known); err != nil ||
		!known {
		t.Fatal("historical interest audit timestamp was not preserved", err)
	}
}
