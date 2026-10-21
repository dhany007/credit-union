package cash

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresStore persists cash data through PostgreSQL.
type PostgresStore struct{ pool *pgxpool.Pool }

// NewPostgresStore constructs the cash PostgreSQL store.
func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore { return &PostgresStore{pool: pool} }

// MemberBalance locks and reads a member balance inside the caller transaction.
func (s *PostgresStore) MemberBalance(ctx context.Context, tx pgx.Tx, id int64) (int64, error) {
	var balance int64
	err := tx.QueryRow(ctx, `SELECT balance FROM members WHERE id=$1 FOR UPDATE`, id).Scan(&balance)
	return balance, err
}

// LockBranch serializes cash writers at one servicing KCP.
func (s *PostgresStore) LockBranch(ctx context.Context, tx pgx.Tx, id int) error {
	_, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(100,$1)`, id)
	return err
}

const branchCashSQL = `
 SELECT COALESCE(sum(j.debit-j.credit),0)
 FROM journal_lines j JOIN transactions t ON t.id=j.transaction_id
 WHERE j.account='cash' AND t.branch_id=$1`

// CashInTx reads committed branch cash while its transaction lock is held.
func (s *PostgresStore) CashInTx(ctx context.Context, tx pgx.Tx, id int) (int64, error) {
	var amount int64
	err := tx.QueryRow(ctx, branchCashSQL, id).Scan(&amount)
	return amount, err
}

// BranchCash returns committed cash at the requested KCP.
func (s *PostgresStore) BranchCash(ctx context.Context, id int) (int64, error) {
	var amount int64
	err := s.pool.QueryRow(ctx, branchCashSQL, id).Scan(&amount)
	return amount, err
}

// Post writes the balance, transaction, and journal without committing.
func (s *PostgresStore) Post(ctx context.Context, tx pgx.Tx, memberID, actorID int64, branchID int,
	amount int64, kind, note, reference string) error {
	delta := amount
	debit, credit := "cash", "member_savings"
	if kind == "withdrawal" {
		delta = -amount
		debit, credit = credit, debit
	}
	if _, err := tx.Exec(ctx, `UPDATE members SET balance=balance+$1 WHERE id=$2`, delta, memberID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
 INSERT INTO transactions(id,member_id,branch_id,actor_id,kind,amount,note)
 VALUES($1,$2,$3,$4,$5,$6,$7)`, reference, memberID, branchID, actorID, kind, amount, note); err != nil {
		return err
	}
	_, err := tx.Exec(ctx, `
 INSERT INTO journal_lines(transaction_id,line_no,account,debit,credit)
 VALUES($1,1,$2,$4,0),($1,2,$3,0,$4)`, reference, debit, credit, amount)
	if err != nil {
		return fmt.Errorf("write cash journal: %w", err)
	}
	return nil
}

// History lists the latest transactions for a member.
func (s *PostgresStore) History(ctx context.Context, id int64) ([]Transaction, error) {
	rows, err := s.pool.Query(ctx, `
 SELECT t.id,t.kind,t.amount,t.note,t.created_at,b.name,u.name
 FROM transactions t JOIN branches b ON b.id=t.branch_id JOIN users u ON u.id=t.actor_id
 WHERE member_id=$1 ORDER BY t.created_at DESC LIMIT 100`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Transaction{}
	for rows.Next() {
		var item Transaction
		if err = rows.Scan(&item.Reference, &item.Kind, &item.Amount,
			&item.Note, &item.CreatedAt, &item.Branch, &item.Actor); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

// Branches lists the configured service branches.
func (s *PostgresStore) Branches(ctx context.Context) ([]Branch, error) {
	rows, err := s.pool.Query(ctx, `SELECT id,name FROM branches ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Branch{}
	for rows.Next() {
		var item Branch
		if err = rows.Scan(&item.ID, &item.Name); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

// List returns the latest 100 rows in a deterministic order. Filter values stay parameterized.
func (s *PostgresStore) List(ctx context.Context, accountNumber string) ([]ListedTransaction, error) {
	query := `
        SELECT t.id,t.kind,t.amount,t.note,t.created_at,b.name,u.name,
            m.id,m.name,m.account_number
        FROM transactions t
        JOIN members m ON m.id=t.member_id
        JOIN branches b ON b.id=t.branch_id
        JOIN users u ON u.id=t.actor_id
    `
	var args []any
	if accountNumber != "" {
		query += ` WHERE m.account_number=$1`
		args = append(args, accountNumber)
	}
	query += ` ORDER BY t.created_at DESC,t.id DESC LIMIT 100`
	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ListedTransaction{}
	for rows.Next() {
		var item ListedTransaction
		if err = rows.Scan(&item.Reference, &item.Kind, &item.Amount, &item.Note, &item.CreatedAt,
			&item.Branch, &item.Actor, &item.MemberID, &item.MemberName, &item.AccountNumber); err != nil {
			return nil, err
		}
		item.MemberNumber = fmt.Sprintf("CU-%06d", item.MemberID)
		out = append(out, item)
	}
	return out, rows.Err()
}
