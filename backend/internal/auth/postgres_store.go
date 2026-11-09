package auth

import (
	"context"
	"credit-union/backend/internal/platform/database"
	"errors"
	"strconv"

	"github.com/jackc/pgx/v5"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresStore persists auth data through PostgreSQL.
type PostgresStore struct{ pool *pgxpool.Pool }

// NewPostgresStore constructs the auth PostgreSQL store.
func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore { return &PostgresStore{pool: pool} }

// Credentials loads the user ID and password hash for authentication.
func (s *PostgresStore) Credentials(ctx context.Context, username string) (int64, string, error) {
	var id int64
	var hash string
	err := s.pool.QueryRow(ctx, `
 SELECT u.id,u.password_hash FROM users u JOIN branches b ON b.id=u.branch_id WHERE u.username=$1 AND
u.is_active AND u.deleted_at IS NULL AND b.is_active AND b.deleted_at IS NULL`, username).
		Scan(&id, &hash)
	return id, hash, err
}

// Current resolves an unexpired session into its staff identity.
func (s *PostgresStore) Current(ctx context.Context, token string) (Actor, error) {
	var actor Actor
	err := s.pool.QueryRow(ctx, `
 SELECT u.id,u.username,u.name,u.role,u.branch_id,b.name
 FROM sessions s JOIN users u ON u.id=s.user_id JOIN branches b ON b.id=u.branch_id
 WHERE s.token_hash=$1 AND s.expires_at>now() AND s.is_active AND s.deleted_at IS NULL AND u.is_active
AND u.deleted_at IS NULL AND b.is_active AND b.deleted_at IS NULL`, database.Hash(token)).Scan(
		&actor.ID, &actor.Username, &actor.Name, &actor.Role, &actor.BranchID, &actor.BranchName)
	return actor, err
}

// CreateSession rotates session material within one transaction.
func (s *PostgresStore) CreateSession(ctx context.Context, id int64, token, oldToken string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer database.Rollback(tx)
	if _, err = tx.Exec(ctx, `UPDATE sessions SET is_active=false WHERE expires_at<now() AND is_active`); err != nil {
		return err
	}
	var active bool
	if err = tx.QueryRow(ctx, `
 SELECT u.is_active AND u.deleted_at IS NULL AND b.is_active AND b.deleted_at IS NULL
 FROM users u JOIN branches b ON b.id=u.branch_id WHERE u.id=$1 FOR SHARE OF u,b`, id).Scan(&active); err != nil {
		return err
	}
	if !active {
		return pgx.ErrNoRows
	}
	if _, err = tx.Exec(ctx, "SELECT set_config('cu.actor_id',$1,true)", strconv.FormatInt(id, 10)); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `
 INSERT INTO sessions(token_hash,user_id,expires_at,created_by)
 VALUES($1,$2,now()+interval '8 hours',$2)`, database.Hash(token), id); err != nil {
		return err
	}
	if oldToken != "" {
		if _, err = tx.Exec(
			ctx,
			`UPDATE sessions SET is_active=false WHERE token_hash=$1`,
			database.Hash(oldToken),
		); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// DeleteSession invalidates the specified session token.
func (s *PostgresStore) DeleteSession(ctx context.Context, token string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer database.Rollback(tx)
	var id int64
	err = tx.QueryRow(ctx, "SELECT user_id FROM sessions WHERE token_hash=$1", database.Hash(token)).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, "SELECT set_config('cu.actor_id',$1,true)", strconv.FormatInt(id, 10)); err != nil {
		return err
	}
	if _, err = tx.Exec(
		ctx,
		"UPDATE sessions SET is_active=false WHERE token_hash=$1",
		database.Hash(token),
	); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
