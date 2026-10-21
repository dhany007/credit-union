package auth

import (
	"context"
	"credit-union/backend/internal/platform/database"

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
	err := s.pool.QueryRow(ctx, `SELECT id,password_hash FROM users WHERE username=$1`, username).Scan(&id, &hash)
	return id, hash, err
}

// Current resolves an unexpired session into its staff identity.
func (s *PostgresStore) Current(ctx context.Context, token string) (Actor, error) {
	var actor Actor
	err := s.pool.QueryRow(ctx, `
 SELECT u.id,u.username,u.name,u.role,u.branch_id,b.name
 FROM sessions s JOIN users u ON u.id=s.user_id JOIN branches b ON b.id=u.branch_id
 WHERE s.token_hash=$1 AND s.expires_at>now()`, database.Hash(token)).Scan(
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
	if _, err = tx.Exec(ctx, `DELETE FROM sessions WHERE expires_at<now()`); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `
 INSERT INTO sessions(token_hash,user_id,expires_at)
 VALUES($1,$2,now()+interval '8 hours')`, database.Hash(token), id); err != nil {
		return err
	}
	if oldToken != "" {
		if _, err = tx.Exec(ctx, `DELETE FROM sessions WHERE token_hash=$1`, database.Hash(oldToken)); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// DeleteSession invalidates the specified session token.
func (s *PostgresStore) DeleteSession(ctx context.Context, token string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM sessions WHERE token_hash=$1`, database.Hash(token))
	return err
}
