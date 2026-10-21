package database

import (
	"bytes"
	"context"
	"credit-union/backend/internal/platform/fault"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Command identity is supplied by the authenticated HTTP adapter, never the body.
type Command struct {
	ActorID           int64
	Key, Method, Path string
}

// Runner owns persistent idempotency and the command transaction boundary.
type Runner struct{ pool *pgxpool.Pool }

// NewRunner constructs a command runner over the shared PostgreSQL pool.
func NewRunner(pool *pgxpool.Pool) *Runner { return &Runner{pool: pool} }

// Hash returns the SHA-256 representation used by persisted sessions and requests.
func Hash(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

// Run owns the only transaction. fingerprintInput must retain legacy field order and normalization.
func (r *Runner) Run(
	ctx context.Context, cmd Command, fingerprintInput any, work func(pgx.Tx) (any, error),
) (json.RawMessage, error) {
	if len(cmd.Key) < 16 || len(cmd.Key) > 100 {
		return nil, fault.New(fault.Invalid, "A valid transaction reference is required")
	}
	input, err := json.Marshal(fingerprintInput)
	if err != nil {
		return nil, fmt.Errorf("encode command: %w", err)
	}
	fp := Hash(cmd.Method + cmd.Path + string(input))
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin command: %w", err)
	}
	defer Rollback(tx)
	_, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`,
		fmt.Sprintf("request:%d:%s", cmd.ActorID, cmd.Key))
	if err != nil {
		return nil, fmt.Errorf("lock command: %w", err)
	}
	var storedFP string
	var stored json.RawMessage
	err = tx.QueryRow(ctx, `
 SELECT fingerprint,response FROM requests
 WHERE user_id=$1 AND request_key=$2`, cmd.ActorID, cmd.Key).Scan(&storedFP, &stored)
	if err == nil {
		if fp != storedFP {
			return nil, fault.New(fault.Conflict, "Reference already used for different data")
		}
		var compact bytes.Buffer
		if err = json.Compact(&compact, stored); err != nil {
			return nil, fmt.Errorf("read command result: %w", err)
		}
		return json.RawMessage(compact.Bytes()), nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("find command: %w", err)
	}
	result, err := work(tx)
	if err != nil {
		return nil, err
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("encode result: %w", err)
	}
	_, err = tx.Exec(ctx, `
 INSERT INTO requests(user_id,request_key,fingerprint,response)
 VALUES($1,$2,$3,$4)`, cmd.ActorID, cmd.Key, fp, encoded)
	if err != nil {
		return nil, fmt.Errorf("save command result: %w", err)
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit command: %w", err)
	}
	return json.RawMessage(encoded), nil
}
