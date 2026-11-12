package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
)

func TestMemberDirectoryPagination(t *testing.T) {
	f := newFixture(t)
	_, err := f.db.Exec(context.Background(), `INSERT INTO members(nik,name,phone,address,home_branch_id)
 SELECT lpad((770000+i)::text,16,'0'),'Pagination member '||i,'000000000','Synthetic pagination fixture',1
 FROM generate_series(1,150) i`)
	if err != nil {
		t.Fatal(err)
	}
	first := f.call("teller.a", "GET", "/api/members?search=Pagination%20member", "", "")
	requireStatus(t, first, 200)
	var rows []struct {
		ID int64 `json:"id"`
	}
	if err = json.Unmarshal(first.Body.Bytes(), &rows); err != nil {
		t.Fatal(err)
	}
	if len(rows) != 100 {
		t.Fatalf("legacy page length: %d", len(rows))
	}
	last := rows[len(rows)-1].ID
	next := f.call(
		"teller.a", "GET",
		fmt.Sprintf("/api/members?search=Pagination%%20member&before_id=%d", last),
		"", "",
	)
	requireStatus(t, next, 200)
	var rest []struct {
		ID int64 `json:"id"`
	}
	if err = json.Unmarshal(next.Body.Bytes(), &rest); err != nil {
		t.Fatal(err)
	}
	if len(rest) != 50 {
		t.Fatalf("remaining filtered members: got %d, want 50", len(rest))
	}
	seen := map[int64]bool{}
	for _, row := range append(rows, rest...) {
		if seen[row.ID] {
			t.Fatal("duplicate across pages")
		}
		seen[row.ID] = true
	}
	for _, raw := range []string{"0", "-1", "broken"} {
		requireStatus(t, f.call("teller.a", "GET", "/api/members?before_id="+raw, "", ""), 400)
	}
}
