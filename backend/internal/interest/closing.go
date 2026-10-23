package interest

import (
	"context"
	"fmt"
	"time"

	"credit-union/backend/internal/platform/database"

	"github.com/jackc/pgx/v5"
)

// closeDay persists the entire date and its completion marker atomically.
func (s *Service) closeDay(ctx context.Context, date time.Time) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer database.Rollback(tx)
	if err = setTimeouts(ctx, tx); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, "SELECT pg_advisory_xact_lock(101,1)"); err != nil {
		return err
	}
	var now time.Time
	if err = tx.QueryRow(ctx, "SELECT clock_timestamp()").Scan(&now); err != nil {
		return err
	}
	if date.After(dueDate(now)) {
		return fmt.Errorf("interest date is not complete")
	}
	var done bool
	if err = tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM interest_days WHERE business_date=$1)",
		date.Format(time.DateOnly)).
		Scan(&done); err != nil {
		return err
	}
	if done {
		return nil
	}
	policy, err := policyFor(ctx, tx, date)
	if err != nil {
		return err
	}
	cutoff := date.AddDate(0, 0, 1)
	rows, err := tx.Query(ctx, `SELECT m.id,m.balance-COALESCE((
 SELECT sum(CASE WHEN t.kind='withdrawal' THEN -t.amount ELSE t.amount END)
 FROM transactions t WHERE t.member_id=m.id AND t.created_at >= $1),0)
 FROM members m WHERE EXISTS(
 SELECT 1 FROM transactions opening WHERE opening.member_id=m.id AND opening.kind='deposit' AND opening.created_at<$1)
 ORDER BY m.id`, cutoff)
	if err != nil {
		return err
	}
	type closing struct{ id, balance int64 }
	balances := []closing{}
	for rows.Next() {
		var item closing
		if err = rows.Scan(&item.id, &item.balance); err != nil {
			rows.Close()
			return err
		}
		if item.balance < 0 {
			rows.Close()
			return fmt.Errorf("negative interest closing balance")
		}
		balances = append(balances, item)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	for _, item := range balances {
		rate := policy.Rate(item.balance)
		if _, err = tx.Exec(
			ctx,
			`INSERT INTO interest_daily(member_id,business_date,policy_id,closing_balance,rate_bps,numerator)
  VALUES($1,$2,$3,$4,$5,$6)`,
			item.id,
			date.Format(time.DateOnly),
			policy.ID,
			item.balance,
			rate,
			Numerator(item.balance, rate).String(),
		); err != nil {
			return err
		}
	}
	if _, err = tx.Exec(ctx, `INSERT INTO interest_days(business_date,eligible_account_count,policy_id)
 VALUES($1,$2,$3)`, date.Format(time.DateOnly), len(balances), policy.ID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
func setTimeouts(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, "SET LOCAL lock_timeout='5s'; SET LOCAL statement_timeout='15s'")
	return err
}
