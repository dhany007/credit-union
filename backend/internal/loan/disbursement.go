package loan

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"credit-union/backend/internal/auth"
	"credit-union/backend/internal/cash"
	"credit-union/backend/internal/platform/database"
	"credit-union/backend/internal/platform/fault"

	"github.com/jackc/pgx/v5"
)

const loanReferenceBytes = 10

// PayoutPreviewInput binds the reviewed approval.
type PayoutPreviewInput struct {
	Revision int64 `json:"expected_approval_revision"`
}

// PayoutInput contains expectations and staff attestations, never authoritative pricing.
type PayoutInput struct {
	PayoutPreviewInput
	PolicyID         int64  `json:"expected_policy_id"`
	Rate             int    `json:"expected_rate_bps"`
	Principal        int64  `json:"expected_principal"`
	Tenor            int    `json:"expected_tenor_months"`
	Date             string `json:"expected_payout_date"`
	Recipient        bool   `json:"recipient_verified"`
	Agreement        bool   `json:"agreement_signed"`
	Conditions       bool   `json:"conditions_verified"`
	VerificationNote string `json:"verification_note"`
	Note             string `json:"note"`
}

// PayoutPreview uses approved pricing and the actual current business date.
type PayoutPreview struct {
	Schedule
	Revision      int64  `json:"approval_revision"`
	PolicyID      int64  `json:"policy_id"`
	Rate          int    `json:"annual_rate_bps"`
	Principal     int64  `json:"amount"`
	Tenor         int    `json:"tenor_months"`
	Date          string `json:"payout_date"`
	BranchID      int    `json:"servicing_branch_id"`
	AvailableCash int64  `json:"available_cash"`
	Eligible      bool   `json:"payout_eligible"`
}

func newReference() (string, error) {
	b := make([]byte, loanReferenceBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "TX-" + hex.EncodeToString(b), nil
}
func (s *Service) payoutTerms(ctx context.Context, tx pgx.Tx, id int64, revision int64) (LockedApplication, error) {
	item, err := s.store.Lock(ctx, tx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return item, fault.New(fault.NotFound, "Application not found")
	}
	if err != nil {
		return item, err
	}
	var exists bool
	if err = tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM loans WHERE application_id=$1)", id).
		Scan(&exists); err != nil {
		return item, err
	}
	if exists {
		return item, fault.New(fault.Conflict, "Application has already been disbursed")
	}
	if item.Status != "approved" || item.ApprovedAmount == nil || item.ApprovedTenor == nil ||
		item.PolicyID == nil || item.Rate == nil || item.DecisionBy == nil || item.DecidedAt == nil {
		return item, fault.New(fault.Rejected, "A complete priced approval is required before disbursement")
	}
	if revision != item.Revision {
		return item, fault.New(fault.Conflict, "Approval revision changed")
	}
	if s.limit <= 0 || *item.ApprovedAmount > s.limit {
		return item, fault.New(fault.Rejected, "Current payout authority is not configured")
	}
	return item, lockEligible(ctx, tx, item.MemberID, item.Responsible)
}

// PayoutPreview checks the same conditions and schedule as payout without saving anything.
func (s *Service) PayoutPreview(
	ctx context.Context,
	actor auth.Actor,
	id int64,
	in PayoutPreviewInput,
) (PayoutPreview, error) {
	var out PayoutPreview
	if actor.Role != "supervisor" {
		return out, fault.New(fault.Forbidden, "Supervisor access required")
	}
	tx, err := s.store.pool.Begin(ctx)
	if err != nil {
		return out, err
	}
	defer database.Rollback(tx)
	if _, err = tx.Exec(ctx, "SET LOCAL lock_timeout='5s'; SET LOCAL statement_timeout='15s'"); err != nil {
		return out, err
	}
	item, err := s.payoutTerms(ctx, tx, id, in.Revision)
	if err != nil {
		return out, err
	}
	cs := cash.NewPostgresStore(s.store.pool)
	if err = cs.LockBranch(ctx, tx, actor.BranchID); err != nil {
		return out, err
	}
	available, err := cs.CashInTx(ctx, tx, actor.BranchID)
	if err != nil {
		return out, err
	}
	now, err := dbTime(ctx, tx)
	if err != nil {
		return out, err
	}
	schedule, err := CalculateSchedule(*item.ApprovedAmount, *item.Rate, *item.ApprovedTenor, now)
	if err != nil {
		return out, err
	}
	return PayoutPreview{
		Schedule:      schedule,
		Revision:      item.Revision,
		PolicyID:      *item.PolicyID,
		Rate:          *item.Rate,
		Principal:     *item.ApprovedAmount,
		Tenor:         *item.ApprovedTenor,
		Date:          now.Format(time.DateOnly),
		BranchID:      actor.BranchID,
		AvailableCash: available,
		Eligible:      available >= *item.ApprovedAmount,
	}, nil
}

// Disburse atomically creates one loan, initial schedule, cash journal, audit and replay result.
func (s *Service) Disburse(
	ctx context.Context,
	actor auth.Actor,
	cmd database.Command,
	id int64,
	in PayoutInput,
) (json.RawMessage, error) {
	if actor.Role != "supervisor" {
		return nil, fault.New(fault.Forbidden, "Supervisor access required")
	}
	if !in.Recipient || !in.Agreement || !in.Conditions || len(strings.TrimSpace(in.VerificationNote)) < 5 ||
		len(in.VerificationNote) > 500 || len(in.Note) > 500 {
		return nil, fault.New(fault.Invalid, "Payout attestations and verification note are required")
	}
	return s.runner.Run(ctx, cmd, in, func(tx pgx.Tx) (any, error) {
		item, err := s.payoutTerms(ctx, tx, id, in.Revision)
		if err != nil {
			return nil, err
		}
		if *item.PolicyID != in.PolicyID || *item.Rate != in.Rate || *item.ApprovedAmount != in.Principal ||
			*item.ApprovedTenor != in.Tenor {
			return nil, fault.New(fault.Conflict, "Payout terms changed; review again")
		}
		cs := cash.NewPostgresStore(s.store.pool)
		if err = cs.LockBranch(ctx, tx, actor.BranchID); err != nil {
			return nil, err
		}
		available, err := cs.CashInTx(ctx, tx, actor.BranchID)
		if err != nil {
			return nil, err
		}
		if available < in.Principal {
			return nil, fault.New(fault.Rejected, "Insufficient cash at this KCP")
		}
		now, err := dbTime(ctx, tx)
		if err != nil {
			return nil, err
		}
		if now.Format(time.DateOnly) != in.Date {
			return nil, fault.New(fault.Conflict, "Payout date changed; review the schedule again")
		}
		schedule, err := CalculateSchedule(in.Principal, in.Rate, in.Tenor, now)
		if err != nil {
			return nil, err
		}
		ref, err := newReference()
		if err != nil {
			return nil, err
		}
		posted := now
		if _, err = tx.Exec(ctx, `
 INSERT INTO transactions(id,member_id,branch_id,actor_id,kind,amount,note,created_at)
  VALUES($1,$2,$3,$4,'loan_disbursement',$5,$6,$7)`,
			ref, item.MemberID, actor.BranchID, actor.ID, in.Principal, in.Note, posted); err != nil {
			return nil, err
		}
		var loanID, versionID int64
		var number string
		err = tx.QueryRow(
			ctx,
			`INSERT INTO loans(application_id,member_id,responsible_branch_id,disbursing_branch_id,
   principal_amount,disbursement_transaction_id,disbursed_at,verification_note,recipient_verified,
agreement_signed,conditions_verified)
   VALUES($1,$2,$3,$4,$5,$6,$7,$8,true,true,true) RETURNING id,loan_number`,
			id,
			item.MemberID,
			item.Responsible,
			actor.BranchID,
			in.Principal,
			ref,
			posted,
			in.VerificationNote,
		).Scan(&loanID, &number)
		if err != nil {
			return nil, err
		}
		err = tx.QueryRow(ctx, `
 INSERT INTO loan_schedule_versions(loan_id,version_number,is_current_version,version_type,effective_date,
   starting_principal,loan_interest_policy_id,interest_rate_bps,tenor_months,first_due_date,status,
reason,approved_by,approved_at)
   VALUES($1,1,true,'initial',$2,$3,$4,$5,$6,$7,'active','Initial approved schedule',$8,$9) RETURNING id`,
			loanID,
			in.Date,
			in.Principal,
			in.PolicyID,
			in.Rate,
			in.Tenor,
			schedule.Installments[0].DueDate,
			*item.DecisionBy,
			*item.DecidedAt,
		).
			Scan(&versionID)
		if err != nil {
			return nil, err
		}
		for _, row := range schedule.Installments {
			if _, err = tx.Exec(
				ctx,
				`INSERT INTO loan_schedules(loan_id,schedule_version_id,installment_number,due_date,principal_due,
interest_due)
    VALUES($1,$2,$3,$4,$5,$6)`,
				loanID,
				versionID,
				row.Number,
				row.DueDate,
				row.PrincipalDue,
				row.InterestDue,
			); err != nil {
				return nil, err
			}
		}
		if _, err = tx.Exec(ctx, `
 INSERT INTO journal_lines(transaction_id,line_no,account,debit,credit)
   VALUES($1,1,'loan_principal_receivable',$2,0),($1,2,'cash',0,$2)`, ref, in.Principal); err != nil {
			return nil, err
		}
		if err = database.Audit(ctx, tx, actor.ID, "loan.disburse", fmt.Sprint(loanID)); err != nil {
			return nil, err
		}
		return map[string]any{"loan_id": loanID, "loan_number": number, "application_id": id, "reference": ref,
			"amount": in.Principal, "disbursed_at": posted, "initial_schedule_version_id": versionID}, nil
	})
}
