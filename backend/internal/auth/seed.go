package auth

import (
	"context"
	"credit-union/backend/internal/platform/database"
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

const minimumSeedPasswordLength = 12

// Seed creates development branches and users without replacing existing credentials.
func Seed(ctx context.Context, db *pgxpool.Pool, password string) error {
	if len(password) < minimumSeedPasswordLength {
		return errors.New("DEV_SEED_PASSWORD must have at least 12 characters")
	}
	hash, e := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if e != nil {
		return e
	}
	tx, e := db.Begin(ctx)
	if e != nil {
		return e
	}
	defer database.Rollback(tx)
	if _, e = tx.Exec(ctx, `
 INSERT INTO branches(id,name) VALUES(1,'KCP Pusat'),(2,'KCP Utara')
 ON CONFLICT DO NOTHING`); e != nil {
		return e
	}
	for _, u := range []struct {
		username, name, role string
		branch               int
	}{
		{"teller.a", "Teller Pusat", "teller", 1},
		{"teller.b", "Teller Utara", "teller", 2},
		{"supervisor", "Supervisor", "supervisor", 1},
	} {
		if _, e = tx.Exec(ctx, `
 INSERT INTO users(username,name,password_hash,role,branch_id)
 VALUES($1,$2,$3,$4,$5) ON CONFLICT(username) DO NOTHING`,
			u.username, u.name, string(hash), u.role, u.branch); e != nil {
			return e
		}
	}
	return tx.Commit(ctx)
}
