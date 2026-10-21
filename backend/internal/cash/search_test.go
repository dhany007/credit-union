package cash

import (
	"testing"
	"time"
)

func TestSearchDateRanges(t *testing.T) {
	now := time.Date(2026, 3, 1, 18, 0, 0, 0, time.UTC) // March 2 in Jakarta.
	cases := []struct{ period, start, end string }{
		{"", "2026-03-02", "2026-03-02"},
		{"today", "2026-03-02", "2026-03-02"},
		{"last_7_days", "2026-02-24", "2026-03-02"},
		{"last_30_days", "2026-02-01", "2026-03-02"},
	}
	for _, tc := range cases {
		t.Run(tc.period, func(t *testing.T) {
			f, err := resolveSearch(SearchInput{Period: tc.period}, now)
			if err != nil {
				t.Fatal(err)
			}
			if f.Start.Format(time.DateOnly) != tc.start || f.End.AddDate(0, 0, -1).Format(time.DateOnly) != tc.end {
				t.Fatalf("wrong inclusive range: %v %v", f.Start, f.End)
			}
			_, offset := f.Start.Zone()
			if offset != 7*60*60 || f.Start.Hour() != 0 || f.End.Hour() != 0 {
				t.Fatal("not Jakarta midnights")
			}
		})
	}
	leap, err := resolveSearch(SearchInput{Period: "custom", StartDate: "2024-02-29", EndDate: "2024-02-29"}, now)
	if err != nil || leap.End.Sub(leap.Start) != 24*time.Hour {
		t.Fatalf("leap single-day: %v", err)
	}
	for _, in := range []SearchInput{
		{Period: "last_month"}, {Period: "custom"},
		{Period: "custom", StartDate: "2026-02-29", EndDate: "2026-03-01"},
		{Period: "custom", StartDate: "2026-03-02", EndDate: "2026-03-01"},
		{Period: "custom", StartDate: "2026-3-1", EndDate: "2026-03-02"},
		{Period: "custom", StartDate: "0000-01-01", EndDate: "2026-03-02"},
		{Period: "today", StartDate: "2026-03-01"},
		{AccountNumber: "123"}, {Cursor: "invalid"},
	} {
		if _, err := resolveSearch(in, now); err == nil {
			t.Fatalf("accepted invalid query %+v", in)
		}
	}
}
