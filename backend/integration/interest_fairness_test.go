package integration

import (
	"context"
	"testing"
)

func TestInterestFailedAccountsDoNotStarveHealthyAccount(t *testing.T) {
	f := newFixture(t)
	id := f.member(t, "3271010101900805", "teller.a")
	svc, start := interestHistory(t, f, id, 300)
	ctx := context.Background()
	// The first 100 accounts cannot accept credit; account 101 remains healthy.
	if _, err := f.db.Exec(ctx, `UPDATE members SET balance=1000000000000 WHERE id=$1`, id); err != nil {
		t.Fatal(err)
	}
	if _, err := f.db.Exec(ctx, `INSERT INTO members(nik,name,phone,address,home_branch_id,balance)
 SELECT lpad((800000+n)::text,16,'0'),'Isolated cap fixture','00000000','Synthetic',1,
 CASE WHEN n=100 THEN 50000 ELSE 1000000000000 END FROM generate_series(1,100) n`); err != nil {
		t.Fatal(err)
	}
	if _, err := f.db.Exec(ctx, `INSERT INTO transactions(id,member_id,branch_id,actor_id,kind,amount,created_at)
 SELECT 'CAP-'||id,id,1,$1,'deposit',balance,$2::date AT TIME ZONE 'Asia/Jakarta'
 FROM members WHERE id<>$3`, f.ids["teller.a"], start.Format("2006-01-02"), id); err != nil {
		t.Fatal(err)
	}
	for range 3 {
		// Persistent cap failures are expected, but must not monopolize the batch.
		_ = svc.CatchUp(ctx)
	}
	var credited int
	if err := f.db.QueryRow(ctx, `SELECT count(*) FROM interest_monthly WHERE credit_amount>0`).
		Scan(&credited); err != nil {
		t.Fatal(err)
	}
	if credited != 1 {
		t.Fatalf("healthy account starved: credited %d", credited)
	}
}
