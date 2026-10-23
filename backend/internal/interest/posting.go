package interest

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"credit-union/backend/internal/platform/config"
	"credit-union/backend/internal/platform/database"
	"credit-union/backend/internal/platform/fault"
)

// settle posts one complete account/month, or returns its existing result.
const decimalBase = 10

func (s *Service) settle(ctx context.Context, memberID int64, period time.Time) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer database.Rollback(tx)
	if err = setTimeouts(ctx, tx); err != nil {
		return err
	}
	if err = database.LockSavingsWriter(ctx, tx); err != nil {
		return err
	}
	var balance int64
	var branch int
	if err = tx.QueryRow(ctx, "SELECT balance,home_branch_id FROM members WHERE id=$1 FOR UPDATE", memberID).
		Scan(&balance, &branch); err != nil {
		return err
	}
	var done bool
	if err = tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM interest_monthly WHERE member_id=$1 AND period_start=$2)",
		memberID, period.Format(time.DateOnly)).Scan(&done); err != nil {
		return err
	}
	if done {
		return nil
	}
	now, err := database.PostingTime(ctx, tx)
	if err != nil {
		return err
	}
	end := period.AddDate(0, 1, 0)
	if !end.Before(businessDate(now).AddDate(0, 0, 1)) {
		return fault.New(fault.Rejected, "Interest month is not complete")
	}
	var start time.Time
	err = tx.QueryRow(ctx, `SELECT GREATEST($2::date,
 (SELECT min(effective_date) FROM interest_policies),
 (SELECT (min(created_at) AT TIME ZONE 'Asia/Jakarta')::date FROM transactions WHERE member_id=$1 AND kind='deposit'))`,
		memberID, period.Format(time.DateOnly)).Scan(&start)
	if err != nil {
		return err
	}
	expected := int(end.Sub(businessDate(start)) / businessDay)
	if expected < 1 {
		return fault.New(fault.Rejected, "No eligible interest days")
	}
	var count int
	var total string
	err = tx.QueryRow(ctx, `SELECT count(*),COALESCE(sum(d.numerator),0)::text FROM interest_daily d
 JOIN interest_days c ON c.business_date=d.business_date AND c.policy_id=d.policy_id
 WHERE d.member_id=$1 AND d.business_date >= $2 AND d.business_date < $3`,
		memberID, start.Format(time.DateOnly), end.Format(time.DateOnly)).Scan(&count, &total)
	if err != nil {
		return err
	}
	if count != expected {
		return fault.New(fault.Rejected, "Interest daily calculation is incomplete")
	}
	numerator, ok := new(big.Int).SetString(total, decimalBase)
	if !ok {
		return fmt.Errorf("invalid monthly interest numerator")
	}
	amount, err := RoundCredit(numerator)
	if err != nil {
		return err
	}
	if amount > config.MaxAmount-balance {
		return fault.New(fault.Rejected, "Balance exceeds the supported limit")
	}
	var reference *string
	if amount > 0 {
		ref := fmt.Sprintf("INT-%d-%s", memberID, period.Format("200601"))
		reference = &ref
		if _, err = tx.Exec(ctx, "UPDATE members SET balance=balance+$1 WHERE id=$2", amount, memberID); err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, `INSERT INTO transactions
  (id,member_id,branch_id,actor_id,kind,amount,note,created_at,balance_before,balance_after)
  VALUES($1,$2,$3,NULL,'interest_credit',$4,$5,$6,$7,$8)`,
			ref, memberID, branch, amount, period.Format("2006-01"), now, balance, balance+amount); err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, `INSERT INTO journal_lines(transaction_id,line_no,account,debit,credit)
  VALUES($1,1,'savings_interest_expense',$2,0),($1,2,'member_savings',0,$2)`, ref, amount); err != nil {
			return err
		}
	}
	if _, err = tx.Exec(ctx, `INSERT INTO interest_monthly
 (member_id,period_start,eligible_days,numerator_total,credit_amount,transaction_id,posted_at)
 VALUES($1,$2,$3,$4,$5,$6,$7)`, memberID, period.Format(time.DateOnly), count, total, amount, reference, now,
	); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO audit_events(actor_id,action,entity_id,details)
 VALUES(NULL,'interest_settlement',$1,jsonb_build_object('origin','system','period',$2::text,'reference',$3::text))`,
		fmt.Sprint(memberID), period.Format("2006-01"), reference); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
