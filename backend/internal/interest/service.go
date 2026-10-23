package interest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"credit-union/backend/internal/auth"
	"credit-union/backend/internal/platform/database"
	"credit-union/backend/internal/platform/fault"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Service owns policy configuration and recoverable interest processing.
type Service struct {
	cursorPeriod string
	cursorMember int64
	jobOperation string
	jobPeriod    string
	jobCompleted int
	jobFailed    int

	pool        *pgxpool.Pool
	runner      *database.Runner
	enabled     bool
	mu          sync.RWMutex
	lastFailure string
	lastSuccess *time.Time
}

// NewService wires the existing database and staff command runner.
func NewService(pool *pgxpool.Pool, runner *database.Runner, enabled bool) *Service {
	return &Service{pool: pool, runner: runner, enabled: enabled}
}

// CreatePolicy saves an audited prospective policy, with durable staff replay.
func (s *Service) CreatePolicy(
	ctx context.Context,
	actor auth.Actor,
	cmd database.Command,
	in PolicyInput,
) (json.RawMessage, error) {
	if actor.Role != "supervisor" {
		return nil, fault.New(fault.Forbidden, "Your role cannot perform this action")
	}
	in.Reason = strings.TrimSpace(in.Reason)
	p := Policy{
		EffectiveDate: in.EffectiveDate,
		BaseRateBPS:   in.BaseRateBPS,
		Bands:         in.Bands,
		Reason:        in.Reason,
		CreatedBy:     actor.ID,
	}
	if p.Bands == nil {
		p.Bands = []Band{}
		in.Bands = p.Bands
	}
	date, err := time.ParseInLocation(time.DateOnly, in.EffectiveDate, jakarta)
	if err != nil || len(in.Reason) < 1 || len(in.Reason) > 500 {
		return nil, fault.New(fault.Invalid, "Invalid savings interest policy")
	}
	if err = p.Validate(); err != nil {
		return nil, err
	}
	return s.runner.Run(ctx, cmd, in, func(tx pgx.Tx) (any, error) {
		if err := database.LockSavingsWriter(ctx, tx); err != nil {
			return nil, err
		}
		if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock(101,2)"); err != nil {
			return nil, err
		}
		var latest int64
		var start *time.Time
		if err := tx.QueryRow(ctx, "SELECT COALESCE(max(id),0),min(effective_date) FROM interest_policies").
			Scan(&latest, &start); err != nil {
			return nil, err
		}
		if (in.ExpectedLatestID == nil && latest != 0) ||
			(in.ExpectedLatestID != nil && *in.ExpectedLatestID != latest) {
			return nil, fault.New(fault.Conflict, "Savings interest policy changed. Reload and try again")
		}
		var now time.Time
		if err := tx.QueryRow(ctx, "SELECT clock_timestamp()").Scan(&now); err != nil {
			return nil, err
		}
		if date.Before(businessDate(now).AddDate(0, 0, 1)) || (start != nil && date.Before(businessDate(*start))) {
			return nil, fault.New(
				fault.Rejected,
				"Interest policy must take effect tomorrow or later without changing activation",
			)
		}
		bands, err := json.Marshal(p.Bands)
		if err != nil {
			return nil, err
		}
		err = tx.QueryRow(ctx, `INSERT INTO interest_policies(effective_date,base_rate_bps,bands,reason,created_by)
  VALUES($1,$2,$3,$4,$5) RETURNING id,created_at`,
			p.EffectiveDate, p.BaseRateBPS, bands, p.Reason, actor.ID).Scan(&p.ID, &p.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("create interest policy: %w", err)
		}
		if err = database.Audit(ctx, tx, actor.ID, "interest_policy", fmt.Sprint(p.ID)); err != nil {
			return nil, err
		}
		return p, nil
	})
}

// Policies returns the immutable configuration history, newest first.
func (s *Service) Policies(ctx context.Context) ([]Policy, error) {
	rows, err := s.pool.Query(ctx, `SELECT id,effective_date::text,base_rate_bps,bands,reason,created_by,created_at
 FROM interest_policies ORDER BY id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Policy{}
	for rows.Next() {
		var p Policy
		var bands []byte
		if err = rows.Scan(
			&p.ID,
			&p.EffectiveDate,
			&p.BaseRateBPS,
			&bands,
			&p.Reason,
			&p.CreatedBy,
			&p.CreatedAt,
		); err != nil {
			return nil, err
		}
		if err = json.Unmarshal(bands, &p.Bands); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}
func policyFor(ctx context.Context, tx pgx.Tx, date time.Time) (Policy, error) {
	var p Policy
	var bands []byte
	err := tx.QueryRow(ctx, `SELECT id,base_rate_bps,bands FROM interest_policies
 WHERE effective_date<=$1 ORDER BY effective_date DESC,id DESC LIMIT 1`, date.Format(time.DateOnly)).
		Scan(&p.ID, &p.BaseRateBPS, &bands)
	if errors.Is(err, pgx.ErrNoRows) {
		return p, fault.New(fault.Rejected, "No savings interest policy covers this date")
	}
	if err != nil {
		return p, err
	}
	if err = json.Unmarshal(bands, &p.Bands); err != nil {
		return p, err
	}
	return p, p.Validate()
}

// Status is the small operational projection exposed to Supervisors.
type Status struct {
	Enabled           bool       `json:"enabled"`
	StartDate         *string    `json:"start_date"`
	LastCompletedDate *string    `json:"last_completed_date"`
	OutstandingMonths []string   `json:"outstanding_months"`
	LastSuccessAt     *time.Time `json:"last_success_at"`
	FailureClass      string     `json:"failure_class"`
}

// Status returns durable progress plus the current process's last job result.
func (s *Service) Status(ctx context.Context) (Status, error) {
	out := Status{Enabled: s.enabled, OutstandingMonths: []string{}}
	if err := s.pool.QueryRow(ctx, `SELECT (SELECT min(effective_date)::text FROM interest_policies),
 (SELECT max(business_date)::text FROM interest_days)`).Scan(&out.StartDate, &out.LastCompletedDate); err != nil {
		return out, err
	}
	months, err := s.pendingMonths(ctx)
	if err != nil {
		return out, err
	}
	out.OutstandingMonths = months
	s.mu.RLock()
	out.LastSuccessAt = s.lastSuccess
	out.FailureClass = s.lastFailure
	s.mu.RUnlock()
	return out, nil
}
