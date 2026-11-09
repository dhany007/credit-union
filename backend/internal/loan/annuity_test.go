package loan

import (
	"testing"
	"time"
)

func TestAnnuitySchedule(t *testing.T) {
	date := time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC)
	s, err := CalculateSchedule(12000000, 1200, 12, date)
	if err != nil {
		t.Fatal(err)
	}
	if s.RegularInstallmentAmount != 1066185 || len(s.Installments) != 12 {
		t.Fatalf("unexpected schedule %+v", s)
	}
	first := s.Installments[0]
	if first.PrincipalDue != 946185 || first.InterestDue != 120000 || first.DueDate != "2026-02-28" {
		t.Fatalf("first %+v", first)
	}
	if s.Installments[1].DueDate != "2026-03-31" {
		t.Fatal("month end drift")
	}
	var principal, interest int64
	for _, row := range s.Installments {
		if row.PrincipalDue <= 0 || row.ClosingPrincipal < 0 || row.TotalDue != row.PrincipalDue+row.InterestDue {
			t.Fatalf("invalid row %+v", row)
		}
		principal += row.PrincipalDue
		interest += row.InterestDue
	}
	if principal != 12000000 || s.TotalPrincipal != principal || s.TotalInterest != interest ||
		s.TotalPayable != principal+interest || s.Installments[11].ClosingPrincipal != 0 {
		t.Fatal("schedule does not reconcile")
	}
}
func TestZeroRateAndInvalidSchedule(t *testing.T) {
	s, err := CalculateSchedule(100, 0, 3, time.Date(2028, 1, 31, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if s.Installments[0].DueDate != "2028-02-29" || s.Installments[2].PrincipalDue != 34 || s.TotalInterest != 0 {
		t.Fatalf("%+v", s)
	}
	for _, x := range []struct {
		p    int64
		r, n int
	}{{0, 1200, 12}, {1, 1200, 120}, {100, -1, 12}, {100, 10001, 12}, {100, 1200, 0}, {100, 1200, 121}} {
		if _, err := CalculateSchedule(x.p, x.r, x.n, time.Now()); err == nil {
			t.Fatalf("accepted %+v", x)
		}
	}
}
