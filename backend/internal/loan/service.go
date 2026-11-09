package loan

import (
	"context"
	"credit-union/backend/internal/auth"
	"credit-union/backend/internal/platform/config"
	"credit-union/backend/internal/platform/database"
	"credit-union/backend/internal/platform/fault"
	"encoding/json"
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

// List returns the existing bounded staff-facing projection.
func (s *Service) List(ctx context.Context) ([]Application, error) { return s.store.List(ctx) }
