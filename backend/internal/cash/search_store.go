package cash

import (
	"context"
	"fmt"
)

// Search fetches one bounded page plus a lookahead row. Every filter value is parameterized.
func (s *PostgresStore) Search(ctx context.Context, f searchFilter) ([]ListedTransaction, error) {
	var beforeTime any
	beforeRef := ""
	if f.Before != nil {
		beforeTime = f.Before.CreatedAt
		beforeRef = f.Before.Reference
	}
	rows, err := s.pool.Query(ctx, `
 SELECT t.id,t.kind,t.amount,t.note,t.created_at,b.name,u.name,m.id,m.name,m.account_number
 FROM transactions t
 JOIN members m ON m.id=t.member_id
 JOIN branches b ON b.id=t.branch_id
 JOIN users u ON u.id=t.actor_id
 WHERE t.created_at >= $1 AND t.created_at < $2
 AND ($3::text='' OR m.account_number=$3)
 AND ($4::timestamptz IS NULL OR (t.created_at,t.id)<($4,$5::text))
 ORDER BY t.created_at DESC,t.id DESC LIMIT $6`,
		f.Start, f.End, f.AccountNumber, beforeTime, beforeRef, searchPageSize+1)
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
