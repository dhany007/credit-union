package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"testing"
	"time"
)

func TestInterestPolicyContract(t *testing.T) {
	f := newFixture(t)
	tomorrow := time.Now().In(time.FixedZone("Jakarta", 7*60*60)).AddDate(0, 0, 1).Format(time.DateOnly)
	payload := fmt.Sprintf(
		`{"effective_date":"%s",
"base_rate_bps":300,"bands":[{"min_balance":1000000,"max_balance":null,"rate_bps":450}],
"reason":"Initial approved savings rates","expected_latest_id":null}`,
		tomorrow,
	)
	requireStatus(
		t,
		f.call("teller.a", "POST", "/api/interest/policies", "interest-policy-0001", payload),
		http.StatusForbidden,
	)
	first := f.call("supervisor", "POST", "/api/interest/policies", "interest-policy-0001", payload)
	requireStatus(t, first, http.StatusOK)
	replay := f.call("supervisor", "POST", "/api/interest/policies", "interest-policy-0001", payload)
	requireStatus(t, replay, http.StatusOK)
	var original, repeated any
	if err := json.Unmarshal(first.Body.Bytes(), &original); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(replay.Body.Bytes(), &repeated); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(original, repeated) {
		t.Fatal("policy replay changed")
	}
	requireStatus(
		t,
		f.call("supervisor", "POST", "/api/interest/policies", "interest-policy-0002", payload),
		http.StatusConflict,
	)
	requireStatus(t, f.call("supervisor", "GET", "/api/interest/policies", "", ""), http.StatusOK)
	requireStatus(t, f.call("supervisor", "GET", "/api/interest/status", "", ""), http.StatusOK)
	var count int
	if err := f.db.QueryRow(context.Background(), "SELECT count(*) FROM interest_policies").
		Scan(&count); err != nil ||
		count != 1 {
		t.Fatalf("versions %d %v", count, err)
	}
}
