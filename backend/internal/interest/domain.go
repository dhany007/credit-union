// Package interest calculates and posts annual savings interest.
package interest

import (
	"credit-union/backend/internal/platform/config"
	"credit-union/backend/internal/platform/fault"
	"math/big"
	"time"
)

const (
	halfDenominator    = 1825000
	jakartaOffsetHours = 7
	businessDay        = 24 * time.Hour
	basisPoints        = 10000
	yearDays           = 365
	denominator        = basisPoints * yearDays
	closeDelay         = 5 * time.Minute
)

var jakarta = time.FixedZone("Asia/Jakarta", int((jakartaOffsetHours * time.Hour).Seconds()))

// Band applies one annual rate to the entire eligible balance.
type Band struct {
	MinBalance int64  `json:"min_balance"`
	MaxBalance *int64 `json:"max_balance"`
	RateBPS    int    `json:"rate_bps"`
}

// Policy is a complete, dated, immutable rate version.
type Policy struct {
	ID            int64     `json:"id"`
	EffectiveDate string    `json:"effective_date"`
	BaseRateBPS   int       `json:"base_rate_bps"`
	Bands         []Band    `json:"bands"`
	Reason        string    `json:"reason"`
	CreatedBy     int64     `json:"created_by"`
	CreatedAt     time.Time `json:"created_at"`
}

// PolicyInput carries the expected version to avoid concurrent lost updates.
type PolicyInput struct {
	EffectiveDate    string `json:"effective_date"`
	BaseRateBPS      int    `json:"base_rate_bps"`
	Bands            []Band `json:"bands"`
	Reason           string `json:"reason"`
	ExpectedLatestID *int64 `json:"expected_latest_id"`
}

// Validate checks sorted non-overlapping half-open bands and exact rates.
func (p Policy) Validate() error {
	if p.BaseRateBPS < 0 || p.BaseRateBPS > basisPoints || len(p.Bands) > 20 {
		return fault.New(fault.Invalid, "Invalid savings interest policy")
	}
	for i, b := range p.Bands {
		if b.MinBalance < 0 || b.MinBalance > config.MaxAmount || b.RateBPS < 0 || b.RateBPS > basisPoints ||
			(b.MaxBalance != nil && (*b.MaxBalance <= b.MinBalance || *b.MaxBalance > config.MaxAmount)) {
			return fault.New(fault.Invalid, "Invalid savings interest bands")
		}
		if i > 0 && (p.Bands[i-1].MaxBalance == nil || *p.Bands[i-1].MaxBalance > b.MinBalance) {
			return fault.New(fault.Invalid, "Savings interest bands overlap or are not ordered")
		}
	}
	return nil
}

// Rate selects a whole-balance band or the explicit global fallback.
func (p Policy) Rate(balance int64) int {
	for _, b := range p.Bands {
		if balance >= b.MinBalance && (b.MaxBalance == nil || balance < *b.MaxBalance) {
			return b.RateBPS
		}
	}
	return p.BaseRateBPS
}

// Numerator keeps daily amounts exact, without daily rounding.
func Numerator(balance int64, rate int) *big.Int {
	return new(big.Int).Mul(big.NewInt(balance), big.NewInt(int64(rate)))
}

// RoundCredit rounds a monthly exact numerator half up to whole IDR.
func RoundCredit(n *big.Int) (int64, error) {
	if n.Sign() < 0 {
		return 0, fault.New(fault.Rejected, "Invalid interest calculation")
	}
	rounded := new(big.Int).Add(n, big.NewInt(halfDenominator))
	rounded.Quo(rounded, big.NewInt(denominator))
	if !rounded.IsInt64() || rounded.Int64() > config.MaxAmount {
		return 0, fault.New(fault.Rejected, "Balance exceeds the supported limit")
	}
	return rounded.Int64(), nil
}
func businessDate(now time.Time) time.Time {
	local := now.In(jakarta)
	return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, jakarta)
}
func dueDate(now time.Time) time.Time {
	return businessDate(now.In(jakarta).Add(-closeDelay)).AddDate(0, 0, -1)
}
