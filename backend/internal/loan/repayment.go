package loan

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"credit-union/backend/internal/auth"
	"credit-union/backend/internal/cash"
	"credit-union/backend/internal/platform/config"
	"credit-union/backend/internal/platform/database"
	"credit-union/backend/internal/platform/fault"

	"github.com/jackc/pgx/v5"
)

// RepaymentInput binds allocation to the reviewed loan revision and schedule.
type RepaymentInput struct {
	Amount    int64  `json:"amount"`
	Revision  int64  `json:"expected_payment_revision"`
	VersionID int64  `json:"expected_schedule_version_id"`
	Note      string `json:"note"`
}

// Allocation connects an actual receipt to one contractual installment.
type Allocation struct {
	ScheduleID int64 `json:"schedule_id"`
	Number     int   `json:"installment_number"`
	Principal  int64 `json:"principal_amount"`
	Interest   int64 `json:"interest_amount"`
}

// RepaymentPreview separates the actual remaining principal from contractual payable.
type RepaymentPreview struct {
	Amount             int64        `json:"amount"`
	Revision           int64        `json:"expected_payment_revision"`
	VersionID          int64        `json:"current_schedule_version_id"`
	OutstandingPayable int64        `json:"outstanding_payable"`
	PrincipalBefore    int64        `json:"principal_before"`
	PrincipalAfter     int64        `json:"principal_after"`
	Allocations        []Allocation `json:"allocations"`
}
type accountState struct {
	MemberID            int64
	Responsible         int
	Principal           int64
	Status              string
	Revision, VersionID int64
}

func loadAccount(ctx context.Context, tx pgx.Tx, id int64, lock bool) (accountState, error) {
	var out accountState
	sql := `SELECT l.member_id,l.responsible_branch_id,l.principal_amount,l.status,l.payment_revision,v.id
 FROM loans l JOIN loan_schedule_versions v ON v.loan_id=l.id AND v.is_current_version
 WHERE l.id=$1`
	if lock {
		sql += " FOR UPDATE OF l"
	}
	err := tx.QueryRow(ctx, sql, id).
		Scan(&out.MemberID, &out.Responsible, &out.Principal, &out.Status, &out.Revision, &out.VersionID)
	if errors.Is(err, pgx.ErrNoRows) {
		return out, fault.New(fault.NotFound, "Loan account not found")
	}
	return out, err
}

func repaymentPlan(
	ctx context.Context,
	tx pgx.Tx,
	id int64,
	item accountState,
	amount int64,
) (RepaymentPreview, error) {
	out := RepaymentPreview{
		Amount:      amount,
		Revision:    item.Revision,
		VersionID:   item.VersionID,
		Allocations: []Allocation{},
	}
	if amount < 1 || amount > config.MaxAmount {
		return out, fault.New(fault.Invalid, "Invalid repayment amount")
	}
	if item.Status != "active" {
		return out, fault.New(fault.Rejected, "Loan is not active")
	}
	var paid int64
	if err := tx.QueryRow(ctx, `
 SELECT COALESCE(sum(principal_amount),0) FROM loan_repayment_allocations WHERE loan_id=$1`, id).
		Scan(&paid); err != nil {
		return out, err
	}
	out.PrincipalBefore = item.Principal - paid
	rows, err := tx.Query(ctx, `
 SELECT s.id,s.installment_number,
 s.principal_due-COALESCE(sum(a.principal_amount),0),s.interest_due-COALESCE(sum(a.interest_amount),0)
 FROM loan_schedules s LEFT JOIN loan_repayment_allocations a ON a.schedule_id=s.id
 WHERE s.schedule_version_id=$1 GROUP BY s.id ORDER BY s.installment_number`, item.VersionID)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	remaining := amount
	var principalTotal int64
	for rows.Next() {
		var row Allocation
		var principal, interest int64
		if err = rows.Scan(&row.ScheduleID, &row.Number, &principal, &interest); err != nil {
			return out, err
		}
		if principal < 0 || interest < 0 {
			return out, fmt.Errorf("negative scheduled loan component")
		}
		out.OutstandingPayable += principal + interest
		row.Interest = min(remaining, interest)
		remaining -= row.Interest
		row.Principal = min(remaining, principal)
		remaining -= row.Principal
		if row.Interest+row.Principal > 0 {
			out.Allocations = append(out.Allocations, row)
			principalTotal += row.Principal
		}
	}
	if err = rows.Err(); err != nil {
		return out, err
	}
	if remaining != 0 {
		return out, fault.New(fault.Rejected, "Repayment exceeds remaining scheduled payable")
	}
	out.PrincipalAfter = out.PrincipalBefore - principalTotal
	return out, nil
}

// RepaymentPreview calculates oldest-first allocations without persisting a receipt.
func (s *Service) RepaymentPreview(ctx context.Context, id, amount int64) (RepaymentPreview, error) {
	var out RepaymentPreview
	tx, err := s.store.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return out, err
	}
	defer database.Rollback(tx)
	item, err := loadAccount(ctx, tx, id, false)
	if err != nil {
		return out, err
	}
	return repaymentPlan(ctx, tx, id, item, amount)
}

// Repay posts the receipt, allocation, cash/receivable/income journal and loan revision atomically.
func (s *Service) Repay(
	ctx context.Context,
	actor auth.Actor,
	cmd database.Command,
	id int64,
	in RepaymentInput,
) (json.RawMessage, error) {
	if actor.Role != "teller" {
		return nil, fault.New(fault.Forbidden, "Teller access required")
	}
	if len(in.Note) > policyReasonMax {
		return nil, fault.New(fault.Invalid, "Repayment note is too long")
	}
	return s.runner.Run(ctx, cmd, in, func(tx pgx.Tx) (any, error) {
		item, err := loadAccount(ctx, tx, id, true)
		if err != nil {
			return nil, err
		}
		if item.Revision != in.Revision || item.VersionID != in.VersionID {
			return nil, fault.New(fault.Conflict, "Payment terms changed; review allocation again")
		}
		if err = lockEligible(ctx, tx, item.MemberID, item.Responsible); err != nil {
			return nil, err
		}
		plan, err := repaymentPlan(ctx, tx, id, item, in.Amount)
		if err != nil {
			return nil, err
		}
		cs := cash.NewPostgresStore(s.store.pool)
		if err = cs.LockBranch(ctx, tx, actor.BranchID); err != nil {
			return nil, err
		}
		ref, err := newReference()
		if err != nil {
			return nil, err
		}
		var now time.Time
		if err = tx.QueryRow(ctx, "SELECT clock_timestamp()").Scan(&now); err != nil {
			return nil, err
		}
		if _, err = tx.Exec(ctx, `
 INSERT INTO transactions(id,member_id,branch_id,actor_id,kind,amount,note,created_at)
   VALUES($1,$2,$3,$4,'loan_repayment',$5,$6,$7)`,
			ref, item.MemberID, actor.BranchID, actor.ID, in.Amount, in.Note, now); err != nil {
			return nil, err
		}
		var receiptID int64
		var number string
		if err = tx.QueryRow(ctx, `
 INSERT INTO loan_repayments(loan_id,transaction_id,servicing_branch_id,amount,paid_at,note)
   VALUES($1,$2,$3,$4,$5,$6) RETURNING id,repayment_number`, id, ref, actor.BranchID, in.Amount, now, in.Note).
			Scan(&receiptID, &number); err != nil {
			return nil, err
		}
		var principal, interest int64
		for _, a := range plan.Allocations {
			if _, err = tx.Exec(
				ctx,
				`INSERT INTO loan_repayment_allocations(loan_id,repayment_id,schedule_id,principal_amount,interest_amount)
   VALUES($1,$2,$3,$4,$5)`,
				id,
				receiptID,
				a.ScheduleID,
				a.Principal,
				a.Interest,
			); err != nil {
				return nil, err
			}
			principal += a.Principal
			interest += a.Interest
		}
		if _, err = tx.Exec(ctx, `
 INSERT INTO journal_lines(transaction_id,line_no,account,debit,credit)
   VALUES($1,1,'cash',$2,0)`, ref, in.Amount); err != nil {
			return nil, err
		}
		line := 2
		for _, entry := range []struct {
			account string
			amount  int64
		}{{"loan_principal_receivable", principal}, {"loan_interest_income", interest}} {
			if entry.amount == 0 {
				continue
			}
			if _, err = tx.Exec(ctx, `
 INSERT INTO journal_lines(transaction_id,line_no,account,debit,credit)
    VALUES($1,$2,$3,0,$4)`, ref, line, entry.account, entry.amount); err != nil {
				return nil, err
			}
			line++
		}
		status := "active"
		var closedAt any
		if plan.OutstandingPayable == in.Amount {
			status = "paid_off"
			closedAt = now
		}
		if _, err = tx.Exec(
			ctx,
			"UPDATE loans SET status=$2,closed_at=$3,payment_revision=payment_revision+1 WHERE id=$1",
			id,
			status,
			closedAt,
		); err != nil {
			return nil, err
		}
		if err = database.Audit(ctx, tx, actor.ID, "loan.repay", fmt.Sprint(receiptID)); err != nil {
			return nil, err
		}
		return map[string]any{
			"repayment_id":     receiptID,
			"repayment_number": number,
			"reference":        ref,
			"amount":           in.Amount,
			"allocations":      plan.Allocations,
			"principal_before": plan.PrincipalBefore,
			"principal_after":  plan.PrincipalAfter,
			"paid_at":          now,
			"status":           status,
			"payment_revision": item.Revision + 1,
		}, nil
	})
}
