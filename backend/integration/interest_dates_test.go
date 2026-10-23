package integration

import (
	"context"
	"testing"
	"time"
)

func TestInterestPartialMonthAndDatedRates(t *testing.T) {
	f := newFixture(t)
	id := f.member(t, "3271010101900806", "teller.a")
	svc, start := interestHistory(t, f, id, 300)
	ctx := context.Background()
	activation := start.AddDate(0, 0, 19).Format(time.DateOnly)
	opening := start.AddDate(0, 0, 22).Format(time.DateOnly)
	change := start.AddDate(0, 0, 24).Format(time.DateOnly)
	if _, err := f.db.Exec(ctx, "UPDATE interest_policies SET effective_date=$1", activation); err != nil {
		t.Fatal(err)
	}
	if _, err := f.db.Exec(
		ctx,
		"UPDATE transactions SET created_at=$1::date AT TIME ZONE 'Asia/Jakarta'",
		opening,
	); err != nil {
		t.Fatal(err)
	}
	if _, err := f.db.Exec(ctx, `INSERT INTO interest_policies(effective_date,base_rate_bps,bands,reason,created_by)
 VALUES($1,400,'[]','Isolated dated rate change',$2)`, change, f.ids["supervisor"]); err != nil {
		t.Fatal(err)
	}
	if err := svc.CatchUp(ctx); err != nil {
		t.Fatal(err)
	}
	if err := svc.CatchUp(ctx); err != nil {
		t.Fatal(err)
	}
	var days int
	var amount int64
	var creditedDate time.Time
	if err := f.db.QueryRow(ctx, `SELECT eligible_days,credit_amount,posted_at
 FROM interest_monthly WHERE member_id=$1`, id).
		Scan(&days, &amount, &creditedDate); err != nil {
		t.Fatal(err)
	}
	monthDays := int(start.AddDate(0, 1, 0).Sub(start) / (24 * time.Hour))
	wantDays := monthDays - 22
	wantAmount := (50000*(2*300+int64(wantDays-2)*400) + 1825000) / 3650000
	if days != wantDays || amount != wantAmount {
		t.Fatalf("partial month days=%d amount=%d want=%d/%d", days, amount, wantDays, wantAmount)
	}
	var wrongRates int
	if err := f.db.QueryRow(ctx, `SELECT count(*) FROM interest_daily WHERE business_date<$1::date
 AND (business_date<$2::date AND rate_bps<>300 OR business_date>=$2::date AND rate_bps<>400)`,
		start.AddDate(0, 1, 0).Format(time.DateOnly), change).Scan(&wrongRates); err != nil || wrongRates != 0 {
		t.Fatalf("historical rates: %d %v", wrongRates, err)
	}
	var retroactive int
	if err := f.db.QueryRow(ctx, `SELECT count(*) FROM interest_daily WHERE member_id=$1
 AND business_date<($2::timestamptz AT TIME ZONE 'Asia/Jakarta')::date AND closing_balance<>50000`,
		id, creditedDate).Scan(&retroactive); err != nil || retroactive != 0 {
		t.Fatalf("late credit backdated: %d %v", retroactive, err)
	}
}
