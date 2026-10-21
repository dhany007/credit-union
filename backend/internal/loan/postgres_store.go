package loan

import (
	"context"
	"credit-union/backend/internal/auth"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresStore persists loan data through PostgreSQL.
type PostgresStore struct{ pool *pgxpool.Pool }

// NewPostgresStore constructs the loan PostgreSQL store.
func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore { return &PostgresStore{pool: pool} }

// RelationshipsExist checks the referenced member and responsible KCP.
func (s *PostgresStore) RelationshipsExist(ctx context.Context, tx pgx.Tx, input Input) (bool, error) {
	var exists bool
	err := tx.QueryRow(ctx, `
 SELECT EXISTS(SELECT 1 FROM members WHERE id=$1)
 AND EXISTS(SELECT 1 FROM branches WHERE id=$2)`, input.MemberID, input.Responsible).Scan(&exists)
	return exists, err
}

// Insert inserts a record using the caller-owned transaction.
func (s *PostgresStore) Insert(ctx context.Context, tx pgx.Tx, actor auth.Actor, input Input) (int64, error) {
	var id int64
	err := tx.QueryRow(ctx, `
 INSERT INTO loans(member_id,receiving_branch_id,responsible_branch_id,created_by,amount,tenor_months,purpose,risk)
 VALUES($1,$2,$3,$4,$5,$6,$7,$8) RETURNING id`, input.MemberID, actor.BranchID, input.Responsible,
		actor.ID, input.Amount, input.Tenor, input.Purpose, input.Risk).Scan(&id)
	return id, err
}

// Lock locks an application before its decision is evaluated.
func (s *PostgresStore) Lock(ctx context.Context, tx pgx.Tx, id int64) (LockedApplication, error) {
	var item LockedApplication
	err := tx.QueryRow(ctx, `
 SELECT amount,created_by,product,risk,status FROM loans WHERE id=$1 FOR UPDATE`, id).Scan(
		&item.Amount, &item.Creator, &item.Product, &item.Risk, &item.Status)
	return item, err
}

// Decide records one permitted application decision.
func (s *PostgresStore) Decide(ctx context.Context, tx pgx.Tx, id, actorID, limit int64, input Decision) error {
	_, err := tx.Exec(ctx, `
 UPDATE loans SET status=$1,decision_by=$2,decision_reason=$3,authority_limit=$4,decided_at=now()
 WHERE id=$5`, input.Decision, actorID, input.Reason, limit, id)
	return err
}

// List returns the existing bounded staff-facing projection.
func (s *PostgresStore) List(ctx context.Context) ([]Application, error) {
	rows, err := s.pool.Query(ctx, `
 SELECT l.id,l.member_id,m.name,l.amount,l.tenor_months,l.purpose,l.risk,l.status,
 b.name,l.responsible_branch_id,rb.name,l.created_at,COALESCE(l.decision_reason,''),COALESCE(du.name,'')
 FROM loans l JOIN members m ON m.id=l.member_id
 JOIN branches b ON b.id=l.receiving_branch_id JOIN branches rb ON rb.id=l.responsible_branch_id
 LEFT JOIN users du ON du.id=l.decision_by
 ORDER BY l.id DESC LIMIT 100`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Application{}
	for rows.Next() {
		var item Application
		if err = rows.Scan(&item.ID, &item.MemberID, &item.MemberName, &item.Amount, &item.Tenor, &item.Purpose,
			&item.Risk, &item.Status, &item.ReceivingBranch, &item.ResponsibleID, &item.ResponsibleBranch,
			&item.CreatedAt, &item.DecisionReason, &item.DecisionBy); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}
