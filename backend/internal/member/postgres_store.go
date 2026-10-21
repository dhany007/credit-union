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

// Insert allocates the account in the caller-owned registration transaction.
func (s *PostgresStore) Insert(
	ctx context.Context,
	tx pgx.Tx,
	input Input,
	branchID int,
) (id int64, accountNumber string, err error) {
	err = tx.QueryRow(ctx, `
        INSERT INTO members(nik,name,phone,address,home_branch_id)
        VALUES($1,$2,$3,$4,$5)
        RETURNING id,account_number
    `, input.NIK, input.Name, input.Phone, input.Address, branchID).Scan(&id, &accountNumber)
	return id, accountNumber, err
}

const memberProjection = `
    SELECT m.id,m.account_number,m.name,right(m.nik,4),m.phone,m.address,m.home_branch_id,b.name,m.balance
    FROM members m
    JOIN branches b ON b.id=m.home_branch_id
`

func scanMember(row pgx.Row) (Member, error) {
	var item Member
	var suffix string
	if err := row.Scan(&item.ID, &item.AccountNumber, &item.Name, &suffix, &item.Phone, &item.Address,
		&item.HomeBranchID, &item.HomeBranch, &item.Balance); err != nil {
		return Member{}, err
	}
	item.Number = fmt.Sprintf("CU-%06d", item.ID)
	item.NIKMasked = "************" + suffix
	return item, nil
}

// List returns the bounded projection, also matching full account numbers.
func (s *PostgresStore) List(ctx context.Context, search string) ([]Member, error) {
	rows, err := s.pool.Query(ctx, memberProjection+`
        WHERE $1='' OR m.name ILIKE '%'||$1||'%' OR m.nik=$1
            OR ('CU-'||lpad(m.id::text,6,'0')) ILIKE '%'||$1||'%'
            OR m.account_number=$1
        ORDER BY m.id DESC LIMIT 100
    `, search)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Member{}
	for rows.Next() {
		item, scanErr := scanMember(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

// FindByAccountNumber uses the unique index for cooperative-wide exact lookup.
func (s *PostgresStore) FindByAccountNumber(ctx context.Context, accountNumber string) (Member, error) {
	return scanMember(s.pool.QueryRow(ctx, memberProjection+` WHERE m.account_number=$1`, accountNumber))
}
