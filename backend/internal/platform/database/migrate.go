package database

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/0002_savings_account_number.sql
var accountNumberMigration string

const (
	initialSchemaVersion = 1
	accountNumberVersion = 2
)

// Migrate applies ordered migrations atomically, including the existing initial schema.
// The short local backfill requires a maintenance window; allocated numbers are never reset.
func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin migration: %w", err)
	}
	defer Rollback(tx)
	if _, err = tx.Exec(ctx, `SET LOCAL lock_timeout = '5s'`); err != nil {
		return fmt.Errorf("set migration lock timeout: %w", err)
	}
	if _, err = tx.Exec(ctx, `
        SELECT pg_advisory_xact_lock(hashtextextended(current_schema() || '.credit_union.migrate',0))
    `); err != nil {
		return fmt.Errorf("lock migration: %w", err)
	}
	if _, err = tx.Exec(ctx, `
        CREATE TABLE IF NOT EXISTS schema_migrations (
            version integer PRIMARY KEY,
            applied_at timestamptz NOT NULL DEFAULT now()
        )
    `); err != nil {
		return fmt.Errorf("prepare migration ledger: %w", err)
	}
	migrations := []struct {
		version int
		sql     string
	}{
		{initialSchemaVersion, Schema},
		{accountNumberVersion, accountNumberMigration},
	}
	for _, migration := range migrations {
		var applied bool
		if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version=$1)`, migration.version).
			Scan(&applied); err != nil {
			return fmt.Errorf("read migration %d: %w", migration.version, err)
		}
		if applied {
			continue
		}
		if _, err = tx.Exec(ctx, migration.sql); err != nil {
			return fmt.Errorf("apply migration %d: %w", migration.version, err)
		}
		if _, err = tx.Exec(ctx, `INSERT INTO schema_migrations(version) VALUES($1)`, migration.version); err != nil {
			return fmt.Errorf("record migration %d: %w", migration.version, err)
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit migration: %w", err)
	}
	return nil
}
