package interest

import (
	"math/big"
	"testing"
	"time"
)

func TestWholeBalanceRateAndRounding(t *testing.T) {
	upper := func(n int64) *int64 { return &n }
	p := Policy{BaseRateBPS: 300, Bands: []Band{
		{MinBalance: 0, MaxBalance: upper(1000000), RateBPS: 300},
		{MinBalance: 1000000, MaxBalance: upper(10000000), RateBPS: 400},
		{MinBalance: 10000000, MaxBalance: upper(100000000), RateBPS: 450},
		{MinBalance: 100000000, RateBPS: 500},
	}}
	if err := p.Validate(); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		balance int64
		rate    int
	}{{0, 300}, {999999, 300}, {1000000, 400}, {10000000, 450}, {100000000, 500}} {
		if got := p.Rate(c.balance); got != c.rate {
			t.Fatalf("balance %d: rate %d", c.balance, got)
		}
	}
	n := new(big.Int).Mul(Numerator(15000000, 450), big.NewInt(30))
	amount, err := RoundCredit(n)
	if err != nil || amount != 55479 {
		t.Fatalf("amount %d: %v", amount, err)
	}
	for _, c := range []struct {
		numerator int64
		want      int64
	}{{0, 0}, {1824999, 0}, {1825000, 1}, {5475000, 2}} {
		got, err := RoundCredit(big.NewInt(c.numerator))
		if err != nil || got != c.want {
			t.Fatalf("round %d: %d %v", c.numerator, got, err)
		}
	}
	if _, err := RoundCredit(new(big.Int).Exp(big.NewInt(10), big.NewInt(40), nil)); err == nil {
		t.Fatal("overflow accepted")
	}
}
func TestPolicyValidation(t *testing.T) {
	upperBound := int64(100)
	for _, p := range []Policy{
		{BaseRateBPS: -1}, {BaseRateBPS: 10001},
		{Bands: []Band{{MinBalance: -1}}},
		{Bands: []Band{{MinBalance: 100, MaxBalance: &upperBound}}},
		{Bands: []Band{{MinBalance: 0}, {MinBalance: 100}}},
		{Bands: []Band{{MinBalance: 100, RateBPS: 10001}}},
	} {
		if p.Validate() == nil {
			t.Fatalf("invalid policy accepted: %+v", p)
		}
	}
	p := Policy{BaseRateBPS: 300, Bands: []Band{{MinBalance: 1000, RateBPS: 450}}}
	if p.Validate() != nil || p.Rate(999) != 300 || p.Rate(1000) != 450 {
		t.Fatal("gap fallback")
	}
}
func TestDueDateAndLeapYear(t *testing.T) {
	for _, c := range []struct{ utc, want string }{
		{"2028-03-01T17:04:59Z", "2028-02-29"},
		{"2028-03-01T17:05:00Z", "2028-03-01"},
	} {
		now, err := time.Parse(time.RFC3339, c.utc)
		if err != nil {
			t.Fatal(err)
		}
		if got := dueDate(now).Format(time.DateOnly); got != c.want {
			t.Fatalf("got %s want %s", got, c.want)
		}
	}
	n := new(big.Int).Mul(Numerator(1000000, 300), big.NewInt(366))
	got, err := RoundCredit(n)
	if err != nil || got != 30082 {
		t.Fatalf("fixed365 leap year: %d %v", got, err)
	}
}
