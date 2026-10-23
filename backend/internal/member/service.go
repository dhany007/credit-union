package member

import (
	"context"
	"credit-union/backend/internal/auth"
	"credit-union/backend/internal/platform/database"
	"credit-union/backend/internal/platform/fault"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
)

const maximumSearchBytes = 100

const openingDeposit int64 = 50000

var nikPattern = regexp.MustCompile(`^[0-9]{16}$`)

// DepositPoster is the registration-side capability for posting its opening deposit.
type DepositPoster interface {
	DepositInTx(ctx context.Context, tx pgx.Tx, actor auth.Actor, memberID, amount int64, note string) (string, error)
}

// Service owns member use-case rules and transaction orchestration.
type Service struct {
	store  *PostgresStore
	runner *database.Runner
	poster DepositPoster
}

// NewService wires the explicit member service dependencies.
func NewService(store *PostgresStore, runner *database.Runner, poster DepositPoster) *Service {
	return &Service{store: store, runner: runner, poster: poster}
}

// Register creates a member and opening deposit in a single command transaction.
func (s *Service) Register(
	ctx context.Context, actor auth.Actor, cmd database.Command, input Input,
) (json.RawMessage, error) {
	if actor.Role != "teller" {
		return nil, fault.New(fault.Forbidden, "Your role cannot perform this action")
	}
	input.NIK = strings.TrimSpace(input.NIK)
	input.Name = strings.TrimSpace(input.Name)
	input.Phone = strings.TrimSpace(input.Phone)
	input.Address = strings.TrimSpace(input.Address)
	if !nikPattern.MatchString(input.NIK) || input.NIK == "0000000000000000" || len(input.Name) < 2 ||
		len(input.Name) > 120 || len(input.Phone) < 6 || len(input.Phone) > 30 ||
		len(input.Address) < 5 || len(input.Address) > 500 {
		return nil, fault.New(fault.Invalid, "Check NIK, name, phone, and address")
	}
	return s.runner.Run(ctx, cmd, fingerprintInput(input), func(tx pgx.Tx) (any, error) {
		if err := database.LockSavingsWriter(ctx, tx); err != nil {
			return nil, err
		}
		id, accountNumber, err := s.store.Insert(ctx, tx, input, actor.BranchID)
		if err != nil {
			return nil, fmt.Errorf("insert member: %w", err)
		}
		ref, err := s.poster.DepositInTx(ctx, tx, actor, id, openingDeposit, "Opening savings deposit")
		if err != nil {
			return nil, err
		}
		if err = database.Audit(ctx, tx, actor.ID, "member.register", strconv.FormatInt(id, 10)); err != nil {
			return nil, err
		}
		// Old persisted replay results remain untouched; new registrations include the account.
		return map[string]any{
			"id": id, "number": fmt.Sprintf("CU-%06d", id), "account_number": accountNumber,
			"reference": ref, "balance": openingDeposit,
		}, nil
	})
}

// List returns the existing bounded staff-facing projection.
func (s *Service) List(ctx context.Context, search string) ([]Member, error) {
	search = strings.TrimSpace(search)
	if len(search) > maximumSearchBytes {
		return nil, fault.New(fault.Invalid, "Search is too long")
	}
	return s.store.List(ctx, search)
}

var accountNumberPattern = regexp.MustCompile(`^[1-9][0-9]{9}$`)

// FindByAccountNumber resolves the current account owner across all KCPs.
func (s *Service) FindByAccountNumber(ctx context.Context, number string) (Member, error) {
	number = strings.TrimSpace(number)
	if !accountNumberPattern.MatchString(number) {
		return Member{}, fault.New(fault.Invalid, "Invalid account number. Enter exactly 10 digits")
	}
	item, err := s.store.FindByAccountNumber(ctx, number)
	if errors.Is(err, pgx.ErrNoRows) {
		return Member{}, fault.New(fault.NotFound, "Savings account not found")
	}
	if err != nil {
		return Member{}, fmt.Errorf("lookup savings account: %w", err)
	}
	return item, nil
}
