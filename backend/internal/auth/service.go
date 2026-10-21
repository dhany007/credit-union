package auth

import (
	"context"
	"credit-union/backend/internal/platform/fault"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

const attemptPruneThreshold = 10000

const loginAttemptLimit = 20
const loginWindow = 15 * time.Minute
const tokenBytes = 32

type attempt struct {
	count int
	until time.Time
}

// Service owns auth use-case rules and transaction orchestration.
type Service struct {
	store    *PostgresStore
	mu       sync.Mutex
	attempts map[string]attempt
	now      func() time.Time
}

// NewService wires the explicit auth service dependencies.
func NewService(store *PostgresStore) *Service {
	return &Service{store: store, attempts: make(map[string]attempt), now: time.Now}
}

// CheckAttempt enforces the existing per-peer login attempt budget.
func (s *Service) CheckAttempt(ip string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.now()
	v := s.attempts[ip]
	if now.After(v.until) {
		v = attempt{until: now.Add(loginWindow)}
	}
	v.count++
	s.attempts[ip] = v
	if len(s.attempts) > attemptPruneThreshold {
		for k, v := range s.attempts {
			if now.After(v.until) {
				delete(s.attempts, k)
			}
		}
	}
	if v.count > loginAttemptLimit {
		return fault.New(fault.Rejected, "Too many sign-in attempts. Try again later")
	}
	return nil
}

// Login validates credentials and creates a fresh session.
func (s *Service) Login(ctx context.Context, input LoginInput, oldToken string) (string, error) {
	id, hash, err := s.store.Credentials(ctx, input.Username)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", fault.New(fault.Unauthorized, "Incorrect username or password")
	}
	if err != nil {
		return "", fault.Wrap(fault.Unavailable, "Authentication service unavailable",
			fmt.Errorf("load login credentials: %w", err))
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(input.Password)) != nil {
		return "", fault.New(fault.Unauthorized, "Incorrect username or password")
	}
	bytes := make([]byte, tokenBytes)
	if _, err = rand.Read(bytes); err != nil {
		return "", fmt.Errorf("generate session: %w", err)
	}
	raw := hex.EncodeToString(bytes)
	if err = s.store.CreateSession(ctx, id, raw, oldToken); err != nil {
		return "", fmt.Errorf("create session: %w", err)
	}
	return raw, nil
}

// Current resolves an unexpired session into its staff identity.
func (s *Service) Current(ctx context.Context, raw string) (Actor, error) {
	actor, err := s.store.Current(ctx, raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return Actor{}, fault.New(fault.Unauthorized, "Session expired. Please sign in")
	}
	if err != nil {
		return Actor{}, fault.Wrap(fault.Unavailable, "Authentication service unavailable",
			fmt.Errorf("load session: %w", err))
	}
	return actor, nil
}

// Logout invalidates the current session.
func (s *Service) Logout(ctx context.Context, raw string) error {
	if err := s.store.DeleteSession(ctx, raw); err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}
