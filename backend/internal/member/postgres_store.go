package member

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresStore persists member data through PostgreSQL.
type PostgresStore struct{ pool *pgxpool.Pool }

// NewPostgresStore constructs the member PostgreSQL store.
func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore { return &PostgresStore{pool: pool} }

// Insert inserts a record using the caller-owned transaction.
func (s *PostgresStore) Insert(ctx context.Context, tx pgx.Tx, input Input, branchID int) (int64, error) {
	var id int64
	err := tx.QueryRow(ctx, `
 INSERT INTO members(nik,name,phone,address,home_branch_id)
 VALUES($1,$2,$3,$4,$5) RETURNING id`, input.NIK, input.Name, input.Phone, input.Address, branchID).Scan(&id)
	return id, err
}

// List returns the existing bounded staff-facing projection.
func (s *PostgresStore) List(ctx context.Context, search string) ([]Member, error) {
	rows, err := s.pool.Query(ctx, `
 SELECT m.id,m.name,right(m.nik,4),m.phone,m.address,m.home_branch_id,b.name,m.balance
 FROM members m JOIN branches b ON b.id=m.home_branch_id
 WHERE $1='' OR m.name ILIKE '%'||$1||'%' OR m.nik=$1
   OR ('CU-'||lpad(m.id::text,6,'0')) ILIKE '%'||$1||'%'
 ORDER BY m.id DESC LIMIT 100`, search)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Member{}
	for rows.Next() {
		var item Member
		var suffix string
		if err = rows.Scan(&item.ID, &item.Name, &suffix, &item.Phone, &item.Address, &item.HomeBranchID,
			&item.HomeBranch, &item.Balance); err != nil {
			return nil, err
		}
		item.Number = fmt.Sprintf("CU-%06d", item.ID)
		item.NIKMasked = "************" + suffix
		out = append(out, item)
	}
	return out, rows.Err()
}
