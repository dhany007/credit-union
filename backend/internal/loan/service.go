package loan

import (
	"context"
	"credit-union/backend/internal/auth"
	"credit-union/backend/internal/platform/config"
	"credit-union/backend/internal/platform/database"
	"credit-union/backend/internal/platform/fault"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
)

// Service owns loan use-case rules and transaction orchestration.
type Service struct {
	store  *PostgresStore
	runner *database.Runner
	limit  int64
}

// NewService wires the explicit loan service dependencies.
func NewService(store *PostgresStore, runner *database.Runner, limit int64) *Service {
	return &Service{store: store, runner: runner, limit: limit}
}

// Apply creates an application without creating principal or moving money.
func (s *Service) Apply(
	ctx context.Context, actor auth.Actor, cmd database.Command, input Input,
) (json.RawMessage, error) {
	if actor.Role != "teller" {
		return nil, fault.New(fault.Forbidden, "Your role cannot perform this action")
	}
	if input.MemberID < 1 || input.Amount < 1 || input.Amount > config.MaxAmount || input.Tenor < 1 ||
		input.Tenor > 120 ||
		len(strings.TrimSpace(input.Purpose)) < 5 ||
		len(input.Purpose) > 1000 ||
		(input.Risk != "low" && input.Risk != "medium" && input.Risk != "high") {
		return nil, fault.New(fault.Invalid, "Check loan amount, tenor, purpose, and risk")
	}
	return s.runner.Run(ctx, cmd, fingerprintInput(input), func(tx pgx.Tx) (any, error) {
		exists, err := s.store.RelationshipsExist(ctx, tx, input)
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, fault.New(fault.Invalid, "Invalid member or responsible KCP")
		}
		id, err := s.store.Insert(ctx, tx, actor, input)
		if err != nil {
			return nil, fmt.Errorf("insert application: %w", err)
		}
		if err = database.Audit(ctx, tx, actor.ID, "loan.submit", strconv.FormatInt(id, 10)); err != nil {
			return nil, err
		}
		return map[string]any{"id": id, "status": "submitted"}, nil
	})
}

// Decide records one permitted application decision.
func (s *Service) Decide(
	ctx context.Context, actor auth.Actor, cmd database.Command, id int64, input Decision,
) (json.RawMessage, error) {
	if actor.Role != "supervisor" {
		return nil, fault.New(fault.Forbidden, "Your role cannot perform this action")
	}
	if (input.Decision != "approved" && input.Decision != "rejected") ||
		len(strings.TrimSpace(input.Reason)) < 5 || len(input.Reason) > 1000 {
		return nil, fault.New(fault.Invalid, "A decision and reason of at least 5 characters are required")
	}
	return s.runner.Run(ctx, cmd, fingerprintDecision(input), func(tx pgx.Tx) (any, error) {
		item, err := s.store.Lock(ctx, tx, id)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fault.New(fault.NotFound, "Application not found")
		}
		if err != nil {
			return nil, fmt.Errorf("lock application: %w", err)
		}
		if item.Status != "submitted" {
			return nil, fault.New(fault.Conflict, "Application already decided")
		}
		if item.Creator == actor.ID {
			return nil, fault.New(fault.Forbidden, "You cannot decide your own application input")
		}
		authorized := s.limit > 0 && item.Amount <= s.limit && item.Product == "regular" && item.Risk == "low"
		if input.Decision == "approved" && !authorized {
			return nil, fault.New(fault.Rejected,
				"No approval authority matches this amount, product, and risk. Configure policy first")
		}
		if err = s.store.Decide(ctx, tx, id, actor.ID, s.limit, input); err != nil {
			return nil, fmt.Errorf("save decision: %w", err)
		}
		if err = database.Audit(ctx, tx, actor.ID, "loan."+input.Decision, strconv.FormatInt(id, 10)); err != nil {
			return nil, err
		}
		return map[string]any{"id": id, "status": input.Decision}, nil
	})
}

// List returns the existing bounded staff-facing projection.
func (s *Service) List(ctx context.Context) ([]Application, error) { return s.store.List(ctx) }
