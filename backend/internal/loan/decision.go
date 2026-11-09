package loan

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"credit-union/backend/internal/auth"
	"credit-union/backend/internal/platform/database"
	"credit-union/backend/internal/platform/fault"

	"github.com/jackc/pgx/v5"
)

// Decide preserves legacy unpriced approvals and binds reviewed pricing for payout-capable approvals.
func (s *Service) Decide(
	ctx context.Context,
	actor auth.Actor,
	cmd database.Command,
	id int64,
	in Decision,
) (json.RawMessage, error) {
	if actor.Role != "supervisor" {
		return nil, fault.New(fault.Forbidden, "Supervisor access required")
	}
	if (in.Decision != "approved" && in.Decision != "rejected") || len(strings.TrimSpace(in.Reason)) < 5 ||
		len(in.Reason) > 1000 {
		return nil, fault.New(fault.Invalid, "A decision and reason of at least 5 characters are required")
	}
	priced := in.ApprovedAmount != 0 || in.ApprovedTenor != 0 || in.ExpectedPolicyID != 0 || in.ExpectedRate != nil
	if in.Decision == "rejected" && priced {
		return nil, fault.New(fault.Invalid, "Rejection cannot set loan pricing")
	}
	return s.runner.Run(ctx, cmd, fingerprintDecision(in), func(tx pgx.Tx) (any, error) {
		item, err := s.store.Lock(ctx, tx, id)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fault.New(fault.NotFound, "Application not found")
		}
		if err != nil {
			return nil, err
		}
		completion := item.Status == "approved" && item.ApprovedAmount == nil && priced
		if item.Status != "submitted" && !completion {
			return nil, fault.New(fault.Conflict, "Application already decided")
		}
		if item.Creator == actor.ID {
			return nil, fault.New(fault.Forbidden, "You cannot decide your own application input")
		}
		amount := item.Amount
		if priced {
			amount = in.ApprovedAmount
		}
		if in.Decision == "approved" &&
			(s.limit <= 0 || amount > s.limit || item.Product != "regular" || item.Risk != "low") {
			return nil, fault.New(
				fault.Rejected,
				"No approval authority matches this amount, product, and risk. Configure policy first",
			)
		}
		if priced {
			if amount < 1 || amount > item.Amount || in.ApprovedTenor < 1 || in.ApprovedTenor > item.Tenor ||
				in.ExpectedRate == nil || in.ExpectedPolicyID < 1 {
				return nil, fault.New(fault.Invalid, "Approved terms and reviewed policy are required")
			}
			if in.ExpectedRevision != item.Revision {
				return nil, fault.New(fault.Conflict, "Approval revision changed")
			}
			if err = lockEligible(ctx, tx, item.MemberID, item.Responsible); err != nil {
				return nil, err
			}
			now, err := dbTime(ctx, tx)
			if err != nil {
				return nil, err
			}
			p, err := resolvePolicy(ctx, tx, now)
			if err != nil {
				return nil, err
			}
			rate := p.rate(amount)
			if p.ID != in.ExpectedPolicyID || rate != *in.ExpectedRate {
				return nil, fault.New(fault.Conflict, "Loan pricing changed; review a new preview")
			}
			if _, err = CalculateSchedule(amount, rate, in.ApprovedTenor, now); err != nil {
				return nil, err
			}
			tag, updateErr := tx.Exec(
				ctx,
				`UPDATE loan_applications SET status='approved',decision_by=$2,decision_reason=$3,
    authority_limit=$4,decided_at=$9,approved_amount=$5,approved_tenor_months=$6,
    loan_interest_policy_id=$7,approved_interest_rate_bps=$8,approval_revision=approval_revision+1
    WHERE id=$1 AND NOT EXISTS(SELECT 1 FROM loans WHERE application_id=$1)`,
				id,
				actor.ID,
				in.Reason,
				s.limit,
				amount,
				in.ApprovedTenor,
				p.ID,
				rate,
				now,
			)
			if updateErr != nil {
				return nil, updateErr
			}
			if tag.RowsAffected() != 1 {
				return nil, fault.New(fault.Conflict, "Application has already been disbursed")
			}
		} else if err = s.store.Decide(ctx, tx, id, actor.ID, s.limit, in); err != nil {
			return nil, err
		}
		if err = database.Audit(ctx, tx, actor.ID, "loan."+in.Decision, fmt.Sprint(id)); err != nil {
			return nil, err
		}
		if priced {
			return map[string]any{"id": id, "status": in.Decision, "approval_revision": item.Revision + 1}, nil
		}
		return map[string]any{"id": id, "status": in.Decision}, nil
	})
}

func lockEligible(ctx context.Context, tx pgx.Tx, memberID int64, responsible int) error {
	var active bool
	err := tx.QueryRow(ctx, "SELECT is_active AND deleted_at IS NULL FROM members WHERE id=$1 FOR SHARE", memberID).
		Scan(&active)
	if errors.Is(err, pgx.ErrNoRows) {
		return fault.New(fault.NotFound, "Member not found")
	}
	if err != nil {
		return err
	}
	if !active {
		return fault.New(fault.Rejected, "Member is inactive")
	}
	err = tx.QueryRow(ctx, "SELECT is_active AND deleted_at IS NULL FROM branches WHERE id=$1 FOR SHARE", responsible).
		Scan(&active)
	if err != nil {
		return err
	}
	if !active {
		return fault.New(fault.Rejected, "Responsible KCP is inactive")
	}
	return nil
}
