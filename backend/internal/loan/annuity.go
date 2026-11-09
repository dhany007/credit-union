package loan

import (
	"fmt"
	"math/big"
	"time"

	"credit-union/backend/internal/platform/config"
	"credit-union/backend/internal/platform/fault"
)

const (
	halfUpDivisor = 2
	monthsPerYear = 12
	basisPoints   = 10000
	maxTenor      = 120
)

// Installment is a contractual projection, independent of actual receipt timing.
type Installment struct {
	Number           int    `json:"installment_number"`
	DueDate          string `json:"due_date"`
	OpeningPrincipal int64  `json:"opening_principal"`
	PrincipalDue     int64  `json:"principal_due"`
	InterestDue      int64  `json:"interest_due"`
	TotalDue         int64  `json:"total_due"`
	ClosingPrincipal int64  `json:"closing_principal"`
}

// Schedule contains rounded whole-IDR rows and their reconciled totals.
type Schedule struct {
	RegularInstallmentAmount int64         `json:"regular_installment_amount"`
	TotalPrincipal           int64         `json:"total_principal"`
	TotalInterest            int64         `json:"total_interest"`
	TotalPayable             int64         `json:"total_payable"`
	Installments             []Installment `json:"installments"`
}

// CalculateSchedule uses exact rationals, monthly nominal interest and half-up IDR rounding.
func CalculateSchedule(principal int64, rate, tenor int, date time.Time) (Schedule, error) {
	out := Schedule{Installments: []Installment{}}
	if principal < 1 || principal > config.MaxAmount || rate < 0 || rate > basisPoints || tenor < 1 ||
		tenor > maxTenor {
		return out, fault.New(fault.Invalid, "Invalid loan principal, annual rate or tenor")
	}
	r := big.NewRat(int64(rate), basisPoints*monthsPerYear)
	payment := new(big.Rat).SetFrac(big.NewInt(principal), big.NewInt(int64(tenor)))
	if rate > 0 {
		power := big.NewRat(1, 1)
		factor := new(big.Rat).Add(big.NewRat(1, 1), r)
		for range tenor {
			power.Mul(power, factor)
		}
		payment.Mul(big.NewRat(principal, 1), r)
		payment.Mul(payment, power)
		payment.Quo(payment, new(big.Rat).Sub(power, big.NewRat(1, 1)))
	}
	regular, err := roundMoney(payment)
	if err != nil {
		return out, err
	}
	out.RegularInstallmentAmount = regular
	remaining := principal
	for k := 1; k <= tenor; k++ {
		interest, err := roundMoney(new(big.Rat).Mul(big.NewRat(remaining, 1), r))
		if err != nil {
			return Schedule{}, err
		}
		portion := regular - interest
		if k == tenor {
			portion = remaining
		}
		if portion <= 0 || portion > remaining {
			return Schedule{}, fault.New(fault.Rejected, "Principal is too small for the selected tenor and rounding")
		}
		row := Installment{
			Number:           k,
			DueDate:          dueDate(date, k),
			OpeningPrincipal: remaining,
			PrincipalDue:     portion,
			InterestDue:      interest,
			TotalDue:         portion + interest,
			ClosingPrincipal: remaining - portion,
		}
		out.Installments = append(out.Installments, row)
		remaining -= portion
		out.TotalPrincipal += portion
		out.TotalInterest += interest
	}
	out.TotalPayable = out.TotalPrincipal + out.TotalInterest
	return out, nil
}

func roundMoney(value *big.Rat) (int64, error) {
	if value.Sign() < 0 {
		return 0, fmt.Errorf("negative money calculation")
	}
	n := new(big.Int).Mul(value.Num(), big.NewInt(halfUpDivisor))
	n.Add(n, value.Denom())
	n.Quo(n, new(big.Int).Mul(value.Denom(), big.NewInt(halfUpDivisor)))
	if !n.IsInt64() {
		return 0, fault.New(fault.Rejected, "Loan calculation exceeds the supported limit")
	}
	return n.Int64(), nil
}

func dueDate(anchor time.Time, months int) string {
	start := time.Date(anchor.Year(), anchor.Month()+time.Month(months), 1, 0, 0, 0, 0, time.UTC)
	last := start.AddDate(0, 1, -1).Day()
	day := min(anchor.Day(), last)
	return time.Date(start.Year(), start.Month(), day, 0, 0, 0, 0, time.UTC).Format(time.DateOnly)
}
