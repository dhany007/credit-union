package database

import (
	"context"
	"credit-union/backend/internal/platform/fault"
	"time"

	"github.com/jackc/pgx/v5"
)

// LockSavingsWriter must precede member, registration and branch locks.
// The exclusive form is reserved for immutable daily interest closing.
func LockSavingsWriter(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock_shared(101,1)")
	return err
}

// PostingTime assigns a database timestamp only after the writer's locks.
func PostingTime(ctx context.Context, tx pgx.Tx) (time.Time, error) {
	var now time.Time
	var closed bool
	err := tx.QueryRow(ctx, `SELECT clock_timestamp()`).Scan(&now)
	if err != nil {
		return now, err
	}
	err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM interest_days
 WHERE business_date>=($1::timestamptz AT TIME ZONE 'Asia/Jakarta')::date)`, now).Scan(&closed)
	if err != nil {
		return now, err
	}
	if closed {
		return now, fault.New(fault.Unavailable, "Savings business date is already closed")
	}
	return now, nil
}
