package cash

import (
	"context"
	"credit-union/backend/internal/auth"
	"credit-union/backend/internal/platform/config"
	"credit-union/backend/internal/platform/database"
	"credit-union/backend/internal/platform/fault"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

const referenceBytes = 10

// Service owns cash use-case rules and transaction orchestration.
type Service struct {
	store  *PostgresStore
	runner *database.Runner
}

// NewService wires the explicit cash service dependencies.
func NewService(store *PostgresStore, runner *database.Runner) *Service {
	return &Service{store: store, runner: runner}
}

// Execute validates and atomically posts a cash command.
func (s *Service) Execute(
	ctx context.Context, actor auth.Actor, cmd database.Command, input Input,
) (json.RawMessage, error) {
	if actor.Role != "teller" {
		return nil, fault.New(fault.Forbidden, "Your role cannot perform this action")
	}
	if input.MemberID < 1 || input.Amount < 1 || input.Amount > config.MaxAmount ||
		(input.Kind != "deposit" && input.Kind != "withdrawal") || len(input.Note) > 500 {
		return nil, fault.New(fault.Invalid, "Invalid member, amount, or transaction type")
	}
	return s.runner.Run(ctx, cmd, fingerprintInput(input), func(tx pgx.Tx) (any, error) {
		ref, err := s.postInTx(ctx, tx, actor, input)
		return Result{Reference: ref}, err
	})
}

// DepositInTx participates in registration's transaction; it never begins or commits one.
func (s *Service) DepositInTx(ctx context.Context, tx pgx.Tx, actor auth.Actor,
	memberID, amount int64, note string) (string, error) {
	return s.postInTx(ctx, tx, actor, Input{MemberID: memberID, Amount: amount, Kind: "deposit", Note: note})
}
func (s *Service) postInTx(ctx context.Context, tx pgx.Tx, actor auth.Actor, input Input) (string, error) {
	balance, err := s.store.MemberBalance(ctx, tx, input.MemberID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", fault.New(fault.NotFound, "Member not found")
	}
	if err != nil {
		return "", fmt.Errorf("lock member balance: %w", err)
	}
	if err = s.store.LockBranch(ctx, tx, actor.BranchID); err != nil {
		return "", fmt.Errorf("lock branch cash: %w", err)
	}
	delta := input.Amount
	if input.Kind == "withdrawal" {
		if balance < input.Amount {
			return "", fault.New(fault.Rejected, "Insufficient member balance")
		}
		available, err := s.store.CashInTx(ctx, tx, actor.BranchID)
		if err != nil {
			return "", err
		}
		if available < input.Amount {
			return "", fault.New(fault.Rejected, "Insufficient cash at this KCP")
		}
		delta = -input.Amount
	}
	if balance+delta > config.MaxAmount {
		return "", fault.New(fault.Rejected, "Balance exceeds the supported limit")
	}
	bytes := make([]byte, referenceBytes)
	if _, err = rand.Read(bytes); err != nil {
		return "", fmt.Errorf("generate cash reference: %w", err)
	}
	ref := "TX-" + hex.EncodeToString(bytes)
	if err = s.store.Post(
		ctx, tx, input.MemberID, actor.ID, actor.BranchID, input.Amount, input.Kind, input.Note, ref,
	); err != nil {
		return "", fmt.Errorf("post cash: %w", err)
	}
	return ref, database.Audit(ctx, tx, actor.ID, input.Kind, ref)
}

// History lists the latest transactions for a member.
func (s *Service) History(ctx context.Context, id int64) ([]Transaction, error) {
	return s.store.History(ctx, id)
}

// Branches lists the configured service branches.
func (s *Service) Branches(ctx context.Context) ([]Branch, error) { return s.store.Branches(ctx) }

// BranchCash returns committed cash at the requested KCP.
func (s *Service) BranchCash(ctx context.Context, id int) (int64, error) {
	return s.store.BranchCash(ctx, id)
}
