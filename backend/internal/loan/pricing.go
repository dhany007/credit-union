package loan

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"credit-union/backend/internal/auth"
	"credit-union/backend/internal/platform/config"
	"credit-union/backend/internal/platform/database"
	"credit-union/backend/internal/platform/fault"

	"github.com/jackc/pgx/v5"
)

const minLoanRateBPS = 300
const maxPolicyBands = 100
const jakartaOffsetSeconds = 7 * 60 * 60

var jakartaLocation = time.FixedZone("Asia/Jakarta", jakartaOffsetSeconds)

const policyLock = 102
const policyReasonMin = 5
const policyReasonMax = 500

// Band selects one annual rate for the entire approved principal.
type Band struct {
	Min  int64  `json:"min_amount"`
	Max  *int64 `json:"max_amount"`
	Rate int    `json:"rate_bps"`
}

// Policy is an immutable dated loan tariff, separate from savings interest.
type Policy struct {
	ID               int64  `json:"id"`
	EffectiveDate    string `json:"effective_date"`
	BaseRate         int    `json:"base_rate_bps"`
	Bands            []Band `json:"bands"`
	Reason           string `json:"reason"`
	ExpectedLatestID *int64 `json:"expected_latest_id,omitempty"`
}

func (p Policy) rate(amount int64) int {
	for _, band := range p.Bands {
		if amount >= band.Min && (band.Max == nil || amount < *band.Max) {
			return band.Rate
		}
	}
	return p.BaseRate
}
func (p Policy) validate() error {
	if p.BaseRate < minLoanRateBPS || p.BaseRate > basisPoints || len(strings.TrimSpace(p.Reason)) < policyReasonMin ||
		len(p.Reason) > policyReasonMax || p.Bands == nil || len(p.Bands) > maxPolicyBands {
		return fault.New(
			fault.Invalid,
			"Loan base rate must be at least 3% p.a.; provide a valid bands array and reason",
		)
	}
	for i, b := range p.Bands {
		if b.Min < 0 || b.Min > config.MaxAmount || b.Rate < minLoanRateBPS || b.Rate > basisPoints ||
			(b.Max != nil && (*b.Max <= b.Min || *b.Max > config.MaxAmount)) {
			return fault.New(fault.Invalid, "Loan band rate must be at least 3% p.a. with valid amount bounds")
		}
		for _, other := range p.Bands[:i] {
			if (b.Max == nil || other.Min < *b.Max) && (other.Max == nil || b.Min < *other.Max) {
				return fault.New(fault.Invalid, "Loan interest bands overlap")
			}
		}
	}
	return nil
}

type queryer interface {
	QueryRow(context.Context, string, ...any) pgx.Row
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

func dbTime(ctx context.Context, q queryer) (time.Time, error) {
	var now time.Time
	err := q.QueryRow(ctx, "SELECT clock_timestamp()").Scan(&now)
	return now.In(jakartaLocation), err
}
func resolvePolicy(ctx context.Context, q queryer, now time.Time) (Policy, error) {
	var p Policy
	var bands []byte
	err := q.QueryRow(ctx, `SELECT id,effective_date::text,base_rate_bps,bands,reason FROM loan_interest_policies
 WHERE effective_date<=$1::date ORDER BY effective_date DESC,id DESC LIMIT 1`, now.Format(time.DateOnly)).
		Scan(&p.ID, &p.EffectiveDate, &p.BaseRate, &bands, &p.Reason)
	if errors.Is(err, pgx.ErrNoRows) {
		return p, fault.New(fault.Rejected, "Configure an effective loan interest policy first")
	}
	if err != nil {
		return p, fmt.Errorf("resolve loan tariff: %w", err)
	}
	if err = json.Unmarshal(bands, &p.Bands); err != nil {
		return p, fmt.Errorf("decode loan tariff: %w", err)
	}
	if err = p.validate(); err != nil {
		return p, fmt.Errorf("stored loan tariff is invalid: %w", err)
	}
	return p, nil
}

// CreatePolicy persists one prospective policy with conflict and durable replay protection.
func (s *Service) CreatePolicy(
	ctx context.Context,
	actor auth.Actor,
	cmd database.Command,
	p Policy,
) (json.RawMessage, error) {
	if actor.Role != "supervisor" {
		return nil, fault.New(fault.Forbidden, "Supervisor access required")
	}
	if p.ID != 0 {
		return nil, fault.New(fault.Invalid, "Policy ID is server assigned")
	}
	if err := p.validate(); err != nil {
		return nil, err
	}
	date, err := time.Parse(time.DateOnly, p.EffectiveDate)
	if err != nil {
		return nil, fault.New(fault.Invalid, "Invalid policy effective date")
	}
	return s.runner.Run(ctx, cmd, p, func(tx pgx.Tx) (any, error) {
		if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock($1,1)", policyLock); err != nil {
			return nil, err
		}
		now, err := dbTime(ctx, tx)
		if err != nil {
			return nil, err
		}
		today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
		if !date.After(today) {
			return nil, fault.New(fault.Rejected, "Loan policy must take effect tomorrow or later")
		}
		var latest *int64
		if err = tx.QueryRow(ctx, "SELECT max(id) FROM loan_interest_policies").Scan(&latest); err != nil {
			return nil, err
		}
		if (latest == nil) != (p.ExpectedLatestID == nil) || (latest != nil && *latest != *p.ExpectedLatestID) {
			return nil, fault.New(fault.Conflict, "Loan policy changed; reload before saving")
		}
		bands, err := json.Marshal(p.Bands)
		if err != nil {
			return nil, err
		}
		if err = tx.QueryRow(ctx, `INSERT INTO loan_interest_policies(effective_date,base_rate_bps,bands,reason)
  VALUES($1,$2,$3,$4) RETURNING id`, p.EffectiveDate, p.BaseRate, bands, p.Reason).Scan(&p.ID); err != nil {
			return nil, err
		}
		if err = database.Audit(ctx, tx, actor.ID, "loan.policy", fmt.Sprint(p.ID)); err != nil {
			return nil, err
		}
		return p, nil
	})
}

// PreviewInput is a non-authoritative estimate request.
type PreviewInput struct {
	Amount                   int64  `json:"amount"`
	Tenor                    int    `json:"tenor_months"`
	ExpectedDisbursementDate string `json:"expected_disbursement_date"`
}

// Preview contains the same calculator output used by financial schedule creation.
type Preview struct {
	Schedule
	Amount       int64     `json:"amount"`
	Tenor        int       `json:"tenor_months"`
	PolicyID     int64     `json:"policy_id"`
	Rate         int       `json:"annual_rate_bps"`
	Method       string    `json:"interest_method"`
	Date         string    `json:"expected_disbursement_date"`
	TimeZone     string    `json:"time_zone"`
	CalculatedAt time.Time `json:"calculated_at"`
	Estimate     bool      `json:"is_estimate"`
	MatchedBand  *Band     `json:"matched_band"`
	BaseFallback bool      `json:"base_rate_fallback"`
}

// Preview calculates without persisting application, financial or replay rows.
func (s *Service) Preview(ctx context.Context, in PreviewInput) (Preview, error) {
	var out Preview
	if in.Amount < 1 || in.Amount > config.MaxAmount || in.Tenor < 1 || in.Tenor > maxTenor {
		return out, fault.New(fault.Invalid, "Invalid loan principal or tenor")
	}
	date, err := time.Parse(time.DateOnly, in.ExpectedDisbursementDate)
	if err != nil {
		return out, fault.New(fault.Invalid, "Invalid expected disbursement date")
	}
	tx, err := s.store.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return out, err
	}
	defer database.Rollback(tx)
	now, err := dbTime(ctx, tx)
	if err != nil {
		return out, err
	}
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	if date.Before(today) || date.After(today.AddDate(1, 0, 0)) {
		return out, fault.New(fault.Invalid, "Expected payout date must be within the next year")
	}
	p, err := resolvePolicy(ctx, tx, now)
	if err != nil {
		return out, err
	}
	schedule, err := CalculateSchedule(in.Amount, p.rate(in.Amount), in.Tenor, date)
	if err != nil {
		return out, err
	}
	out = Preview{
		Schedule:     schedule,
		Amount:       in.Amount,
		Tenor:        in.Tenor,
		PolicyID:     p.ID,
		Rate:         p.rate(in.Amount),
		Method:       "annuity",
		Date:         in.ExpectedDisbursementDate,
		TimeZone:     "Asia/Jakarta",
		CalculatedAt: now,
		Estimate:     true,
		BaseFallback: true,
	}
	for _, band := range p.Bands {
		if in.Amount >= band.Min && (band.Max == nil || in.Amount < *band.Max) {
			b := band
			out.MatchedBand = &b
			out.BaseFallback = false
			break
		}
	}
	return out, nil
}
