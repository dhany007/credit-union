package interest

import (
	"context"
	"credit-union/backend/internal/platform/fault"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"credit-union/backend/internal/platform/database"
)

const (
	jobInterval           = time.Minute
	jobTimeout            = 50 * time.Second
	maxDatesPerTick       = 31
	maxSettlementsPerTick = 100
)

// Run owns the cancellable background loop; callers must join it before closing the pool.
func (s *Service) Run(ctx context.Context, logger *slog.Logger) {
	if !s.enabled {
		return
	}
	ticker := time.NewTicker(jobInterval)
	defer ticker.Stop()
	for {
		started := time.Now()
		attempt, cancel := context.WithTimeout(ctx, jobTimeout)
		err := s.CatchUp(attempt)
		cancel()
		s.mu.Lock()
		if err != nil {
			s.lastFailure = failureClass(err)
		} else {
			now := time.Now()
			s.lastSuccess = &now
			s.lastFailure = ""
		}
		operation, period, completed, failed := s.jobOperation, s.jobPeriod, s.jobCompleted, s.jobFailed
		s.mu.Unlock()
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			logger.Error("interest processing failed", "failure_class", failureClass(err),
				"operation", operation, "period", period, "completed", completed, "failed", failed,
				"duration_ms", time.Since(started).Milliseconds())
		} else {
			logger.Info("interest processing completed", "duration_ms", time.Since(started).Milliseconds(),
				"operation", operation, "period", period, "completed", completed, "failed", failed)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// CatchUp processes bounded due work. It is used by the internal scheduler and isolated tests.
func (s *Service) CatchUp(ctx context.Context) error {
	operation, period := "", ""
	completed, failed := 0, 0
	defer func() {
		s.mu.Lock()
		s.jobOperation = operation
		s.jobPeriod = period
		s.jobCompleted = completed
		s.jobFailed = failed
		s.mu.Unlock()
	}()
	// A transaction-scoped scheduler lock reserves one connection and releases on every return.
	owner, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer database.Rollback(owner)
	var acquired bool
	if err = owner.QueryRow(ctx, "SELECT pg_try_advisory_xact_lock(101,3)").Scan(&acquired); err != nil {
		return err
	}
	if !acquired {
		return nil
	}
	var start, last *time.Time
	var now time.Time
	if err = s.pool.QueryRow(ctx, `SELECT (SELECT min(effective_date) FROM interest_policies),
 (SELECT max(business_date) FROM interest_days),clock_timestamp()`).Scan(&start, &last, &now); err != nil {
		return err
	}
	if start == nil {
		return nil
	}
	var inconsistent bool
	if err = s.pool.QueryRow(ctx, `SELECT EXISTS(
 SELECT 1 FROM interest_days c WHERE c.eligible_account_count<>
 (SELECT count(*) FROM interest_daily d WHERE d.business_date=c.business_date AND d.policy_id=c.policy_id))
 OR EXISTS(SELECT 1 FROM generate_series($1::date,$2::date,interval '1 day') d
 WHERE NOT EXISTS(SELECT 1 FROM interest_days c WHERE c.business_date=d::date))`,
		start, last).Scan(&inconsistent); err != nil {
		return err
	}
	if inconsistent {
		return fault.New(fault.Rejected, "Interest daily calculation is incomplete")
	}
	next := businessDate(*start)
	if last != nil {
		next = businessDate(*last).AddDate(0, 0, 1)
	}
	for n := 0; n < maxDatesPerTick && !next.After(dueDate(now)); n++ {
		operation = "close_day"
		period = next.Format(time.DateOnly)
		if err = s.closeDay(ctx, next); err != nil {
			return fmt.Errorf("close interest date: %w", err)
		}
		completed++
		next = next.AddDate(0, 0, 1)
	}
	s.mu.RLock()
	cursorPeriod, cursorMember := s.cursorPeriod, s.cursorMember
	s.mu.RUnlock()
	rows, err := s.pool.Query(ctx, `WITH pending AS (
 SELECT d.member_id,min(date_trunc('month',d.business_date)::date) AS period
 FROM interest_daily d WHERE d.business_date<date_trunc('month',clock_timestamp() AT TIME ZONE 'Asia/Jakarta')::date
 AND NOT EXISTS(SELECT 1 FROM interest_monthly m WHERE m.member_id=d.member_id
 AND m.period_start=date_trunc('month',d.business_date)::date)
 GROUP BY d.member_id)
 SELECT member_id,period FROM pending WHERE ($1::text='' OR (period,member_id)>(NULLIF($1,'')::date,$2)
 OR NOT EXISTS(SELECT 1 FROM pending WHERE (period,member_id)>(NULLIF($1,'')::date,$2)))
 ORDER BY period,member_id LIMIT $3`, cursorPeriod, cursorMember, maxSettlementsPerTick)
	if err != nil {
		return err
	}
	type pending struct {
		member int64
		period time.Time
	}
	work := []pending{}
	for rows.Next() {
		var p pending
		if err = rows.Scan(&p.member, &p.period); err != nil {
			rows.Close()
			return err
		}
		work = append(work, p)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	failures := []error{}
	if len(work) == 0 {
		s.mu.Lock()
		s.cursorPeriod = ""
		s.cursorMember = 0
		s.mu.Unlock()
	}
	for _, p := range work {
		operation = "settle_month"
		period = p.period.Format("2006-01")
		s.mu.Lock()
		s.cursorPeriod = p.period.Format(time.DateOnly)
		s.cursorMember = p.member
		s.mu.Unlock()
		if err = s.settle(ctx, p.member, businessDate(p.period)); err != nil {
			failed++
			failures = append(failures, err)
		}
		if err == nil {
			completed++
		}
		if ctx.Err() != nil {
			break
		}
	}
	return errors.Join(failures...)
}
func (s *Service) pendingMonths(ctx context.Context) ([]string, error) {
	rows, err := s.pool.Query(ctx, `SELECT DISTINCT to_char(d.business_date,'YYYY-MM')
 FROM interest_daily d WHERE d.business_date<date_trunc('month',clock_timestamp() AT TIME ZONE 'Asia/Jakarta')::date
 AND NOT EXISTS(SELECT 1 FROM interest_monthly m WHERE m.member_id=d.member_id
 AND m.period_start=date_trunc('month',d.business_date)::date)
 ORDER BY 1`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var month string
		if err = rows.Scan(&month); err != nil {
			return nil, err
		}
		out = append(out, month)
	}
	return out, rows.Err()
}

// failureClass preserves useful operational context without logging SQL or identities.
func failureClass(err error) string {
	if errors.Is(err, context.Canceled) {
		return "canceled"
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "timeout"
	}
	var domainErr *fault.Error
	if errors.As(err, &domainErr) {
		return "incomplete_or_rejected"
	}
	var dbErr *pgconn.PgError
	if errors.As(err, &dbErr) {
		switch dbErr.Code {
		case "55P03":
			return "lock_timeout"
		case "57014":
			return "statement_timeout"
		}
	}
	return "database_or_internal"
}
