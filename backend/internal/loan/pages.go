package loan

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"

	"credit-union/backend/internal/platform/database"
	"credit-union/backend/internal/platform/fault"
)

const pageSize = 20
const maxCursorBytes = 1024

// Page contains bounded results; cursors are tied to the exact endpoint/filter scope.
type Page struct {
	Items      []json.RawMessage `json:"items"`
	NextCursor string            `json:"next_cursor"`
}
type pageCursor struct {
	ID      int64  `json:"id"`
	Scope   string `json:"scope"`
	Version int    `json:"v"`
}

func parseCursor(raw, scope string) (int64, error) {
	if raw == "" {
		return 0, nil
	}
	var c pageCursor
	b, err := base64.RawURLEncoding.DecodeString(raw)
	if len(raw) > maxCursorBytes || err != nil || json.Unmarshal(b, &c) != nil || c.ID < 1 || c.Scope != scope ||
		c.Version != 1 {
		return 0, fault.New(fault.Invalid, "Invalid pagination cursor; apply the filters again")
	}
	return c.ID, nil
}
func queryScope(path string, values url.Values) string {
	copyValues := url.Values{}
	for key, value := range values {
		if key != "cursor" {
			copyValues[key] = value
		}
	}
	return database.Hash(path + "?" + copyValues.Encode())
}
func (s *Service) page(ctx context.Context, sql string, args []any, raw, scope string, ascending bool) (Page, error) {
	out := Page{Items: []json.RawMessage{}}
	before, err := parseCursor(raw, scope)
	if err != nil {
		return out, err
	}
	args = append(args, before)
	operator, order := "<", "DESC"
	if ascending {
		operator, order = ">", "ASC"
	}
	sql = fmt.Sprintf("SELECT q.id,q.data FROM (%s) q WHERE ($%d::bigint=0 OR q.id %s $%d) ORDER BY q.id %s LIMIT 21",
		sql, len(args), operator, len(args), order)
	rows, err := s.store.pool.Query(ctx, sql, args...)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	var last int64
	for rows.Next() {
		var id int64
		var data json.RawMessage
		if err = rows.Scan(&id, &data); err != nil {
			return out, err
		}
		if len(out.Items) == pageSize {
			cursor, err := json.Marshal(pageCursor{ID: last, Scope: scope, Version: 1})
			if err != nil {
				return out, err
			}
			out.NextCursor = base64.RawURLEncoding.EncodeToString(cursor)
			break
		}
		last = id
		out.Items = append(out.Items, data)
	}
	return out, rows.Err()
}
func parseID(raw string) (int64, error) {
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id < 1 {
		return 0, fault.New(fault.Invalid, "Invalid loan resource ID")
	}
	return id, nil
}

// Policies returns immutable policy history with bounded cursor pagination.
func (s *Service) Policies(ctx context.Context, values url.Values) (Page, error) {
	for k, entries := range values {
		if k != "cursor" || len(entries) != 1 {
			return Page{}, fault.New(fault.Invalid, "Invalid policy filter")
		}
	}
	return s.page(ctx, `SELECT id,jsonb_build_object('id',id,'effective_date',effective_date,
 'base_rate_bps',base_rate_bps,'bands',bands,'reason',reason,'created_by',created_by,'created_at',
created_at) data
 FROM loan_interest_policies`, nil, values.Get("cursor"), queryScope("loan-policies", values), false)
}
