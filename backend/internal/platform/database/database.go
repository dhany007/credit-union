package database

import (
	"context"
	_ "embed"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Schema is the unchanged initial PostgreSQL schema.
//
//go:embed schema.sql
var Schema string

// Open constructs and verifies the shared PostgreSQL pool.
func Open(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("configure database: %w", err)
	}
	if err = pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("connect database: %w", err)
	}
	return pool, nil
}

// Migrate applies the explicit idempotent initial schema.
func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, Schema)
	return err
}

// Rollback uses a bounded cleanup context even when the request was cancelled.
func Rollback(tx pgx.Tx) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_ = tx.Rollback(ctx)
}

// Audit records a business event in the caller-owned transaction.
func Audit(ctx context.Context, tx pgx.Tx, actorID int64, action, entity string) error {
	_, err := tx.Exec(ctx, `
 INSERT INTO audit_events(actor_id, action, entity_id)
 VALUES($1, $2, $3)`, actorID, action, entity)
	if err != nil {
		return fmt.Errorf("record audit: %w", err)
	}
	return nil
}
