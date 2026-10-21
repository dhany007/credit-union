package integration

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"credit-union/backend/internal/auth"
	"credit-union/backend/internal/platform/app"
	"credit-union/backend/internal/platform/config"
	"credit-union/backend/internal/platform/database"

	"github.com/jackc/pgx/v5/pgxpool"
)

type fixture struct {
	db      *pgxpool.Pool
	handler http.Handler
	cookies map[string]string
	ids     map[string]int64
}

func newFixture(t *testing.T) *fixture {
	return newFixtureWithMigration(t, true)
}

func newFixtureWithMigration(t *testing.T, migrate bool) *fixture {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("requires isolated PostgreSQL integration database")
	}
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	schemaName := fmt.Sprintf("cu_test_%d", time.Now().UnixNano())
	if _, err = admin.Exec(ctx, "CREATE SCHEMA "+schemaName); err != nil {
		admin.Close()
		t.Fatal(err)
	}
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schemaName
	cfg.ConnConfig.RuntimeParams["application_name"] = schemaName
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		pool.Close()
		if _, err := admin.Exec(context.Background(), "DROP SCHEMA "+schemaName+" CASCADE"); err != nil {
			t.Error(err)
		}
		admin.Close()
	})
	if migrate {
		err = database.Migrate(ctx, pool)
	} else {
		_, err = pool.Exec(ctx, database.Schema)
	}
	if err != nil {
		t.Fatal(err)
	}
	if err = auth.Seed(ctx, pool, "isolated-test-password"); err != nil {
		t.Fatal(err)
	}
	f := &fixture{db: pool, cookies: map[string]string{}, ids: map[string]int64{}}
	for _, name := range []string{"teller.a", "teller.b", "supervisor"} {
		var id int64
		if err = pool.QueryRow(ctx, "SELECT id FROM users WHERE username=$1", name).Scan(&id); err != nil {
			t.Fatal(err)
		}
		raw := token()
		f.cookies[name] = raw
		f.ids[name] = id
		if _, err = pool.Exec(
			ctx,
			`INSERT INTO sessions(token_hash,user_id,expires_at) VALUES($1,$2,now()+interval '1 hour')`,
			database.Hash(raw),
			id,
		); err != nil {
			t.Fatal(err)
		}
	}
	f.handler = app.New(pool, config.Config{Origin: "http://localhost:3000", ApprovalLimit: 10000000},
		slog.New(slog.NewTextHandler(io.Discard, nil)))
	return f
}
func (f *fixture) call(actor, method, path, key, payload string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, path, strings.NewReader(payload))
	r.Header.Set("Origin", "http://localhost:3000")
	r.Header.Set("Idempotency-Key", key)
	r.AddCookie(
		&http.Cookie{
			Name:     "cu_session",
			Value:    f.cookies[actor],
			Secure:   true,
			HttpOnly: true,
			SameSite: http.SameSiteStrictMode,
		},
	)
	w := httptest.NewRecorder()
	f.handler.ServeHTTP(w, r)
	return w
}
func requireStatus(t *testing.T, w *httptest.ResponseRecorder, status int) {
	t.Helper()
	if w.Code != status {
		t.Fatalf("want %d got %d: %s", status, w.Code, w.Body.String())
	}
}
func (f *fixture) member(t *testing.T, nik, actor string) int64 {
	t.Helper()
	w := f.call(actor, "POST", "/api/members", "registration-"+nik,
		fmt.Sprintf(`{"nik":"%s","name":"Synthetic Member","phone":"08123456789","address":"Test address only"}`, nik))
	requireStatus(t, w, 200)
	var result struct {
		ID int64 `json:"id"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	return result.ID
}
func (f *fixture) postCash(actor, key, kind string, memberID, amount int64) *httptest.ResponseRecorder {
	return f.call(
		actor,
		"POST",
		"/api/cash",
		key,
		fmt.Sprintf(`{"member_id":%d,"amount":%d,"kind":"%s","note":""}`, memberID, amount, kind),
	)
}
func TestPersistedLegacyReplay(t *testing.T) {
	f := newFixture(t)
	cases := []struct{ name, actor, path, input, canonical, result string }{
		{
			"registration",
			"teller.a",
			"/api/members",
			`{"nik":" 3271010101900001 ","name":" Test Member ","phone":" 08123456 ","address":" Test address "}`,
			`{"nik":"3271010101900001","name":"Test Member","phone":"08123456","address":"Test address"}`,
			`{"id":42,"number":"CU-000042","reference":"legacy-opening","balance":50000}`,
		},
		{"cash", "teller.a", "/api/cash", `{"member_id":42,"amount":100,"kind":"deposit","note":" old note "}`,
			`{"member_id":42,"amount":100,"kind":"deposit","note":" old note "}`, `{"reference":"legacy-deposit"}`},
		{
			"application",
			"teller.a",
			"/api/loans",
			`{"member_id":42,"amount":100000,"tenor_months":12,` +
				`"purpose":" original purpose ","risk":"low","responsible_branch_id":1}`,
			`{"member_id":42,"amount":100000,"tenor_months":12,` +
				`"purpose":" original purpose ","risk":"low","responsible_branch_id":1}`,
			`{"id":42,"status":"submitted"}`,
		},
		{"decision", "supervisor", "/api/loans/42/decision", `{"decision":"approved","reason":" original reason "}`,
			`{"decision":"approved","reason":" original reason "}`, `{"id":42,"status":"approved"}`},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			key := "legacy-replay-" + tt.name
			hash := sha256.Sum256([]byte("POST" + tt.path + tt.canonical))
			_, err := f.db.Exec(
				context.Background(),
				`INSERT INTO requests(user_id,request_key,fingerprint,response) VALUES($1,$2,$3,$4)`,
				f.ids[tt.actor],
				key,
				hex.EncodeToString(hash[:]),
				[]byte(tt.result),
			)
			if err != nil {
				t.Fatal(err)
			}
			w := f.call(tt.actor, "POST", tt.path, key, tt.input)
			requireStatus(t, w, 200)
			var got, want any
			if err = json.Unmarshal(w.Body.Bytes(), &got); err != nil {
				t.Fatal(err)
			}
			if err = json.Unmarshal([]byte(tt.result), &want); err != nil {
				t.Fatal(err)
			}
			gotJSON, _ := json.Marshal(got)
			wantJSON, _ := json.Marshal(want)
			if !bytes.Equal(gotJSON, wantJSON) {
				t.Fatal("stored result changed")
			}
		})
	}
	var count int
	if err := f.db.QueryRow(context.Background(), `SELECT count(*) FROM members`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatal("legacy replays executed business work")
	}
}
func TestSharedBranchCashAndConcurrentRetry(t *testing.T) {
	f := newFixture(t)
	first := f.member(t, "3271010101900011", "teller.a")
	second := f.member(t, "3271010101900012", "teller.a")
	requireStatus(t, f.postCash("teller.b", "fund-north-branch-001", "deposit", first, 50000), 200)
	// Both members have enough savings individually; branch B can pay only once.
	codes := make(chan int, 2)
	var wg sync.WaitGroup
	for i, id := range []int64{first, second} {
		wg.Add(1)
		go func(i int, id int64) {
			defer wg.Done()
			codes <- f.postCash("teller.b", fmt.Sprintf("shared-cash-key-%d", i), "withdrawal", id, 40000).Code
		}(i, id)
	}
	wg.Wait()
	close(codes)
	ok, blocked := 0, 0
	for code := range codes {
		if code == 200 {
			ok++
		}
		if code == 422 {
			blocked++
		}
	}
	if ok != 1 || blocked != 1 {
		t.Fatalf("branch cash protection failed: %d/%d", ok, blocked)
	}
	before := queryInt(t, f.db, `SELECT balance FROM members WHERE id=$1`, first)
	codes = make(chan int, 4)
	for range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			codes <- f.postCash("teller.a", "concurrent-retry-0001", "deposit", first, 1000).Code
		}()
	}
	wg.Wait()
	close(codes)
	for code := range codes {
		if code != 200 {
			t.Fatalf("retry failed: %d", code)
		}
	}
	if after := queryInt(t, f.db, `SELECT balance FROM members WHERE id=$1`, first); after != before+1000 {
		t.Fatal("retry posted more than once")
	}
	assertReconciled(t, f.db)
}
func TestCancelledLockAndAtomicFailures(t *testing.T) {
	f := newFixture(t)
	id := f.member(t, "3271010101900021", "teller.a")
	tx, err := f.db.Begin(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer database.Rollback(tx)
	if _, err = tx.Exec(context.Background(), `SELECT id FROM members WHERE id=$1 FOR UPDATE`, id); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	payload := fmt.Sprintf(`{"member_id":%d,"amount":500,"kind":"deposit","note":""}`, id)
	r := httptest.NewRequest("POST", "/api/cash", strings.NewReader(payload)).WithContext(ctx)
	r.Header.Set("Origin", "http://localhost:3000")
	r.Header.Set("Idempotency-Key", "cancelled-request-key")
	r.AddCookie(
		&http.Cookie{
			Name:     "cu_session",
			Value:    f.cookies["teller.a"],
			Secure:   true,
			HttpOnly: true,
			SameSite: http.SameSiteStrictMode,
		},
	)

	finished := make(chan int, 1)
	go func() { w := httptest.NewRecorder(); f.handler.ServeHTTP(w, r); finished <- w.Code }()
	waitCtx, stop := context.WithTimeout(context.Background(), 3*time.Second)
	defer stop()
	ticker := time.NewTicker(5 * time.Millisecond)
	defer ticker.Stop()
	for {
		var count int
		err = f.db.QueryRow(waitCtx, `SELECT count(*) FROM pg_stat_activity
 WHERE application_name=current_setting('application_name')
 AND wait_event_type='Lock'`).
			Scan(&count)
		if err != nil {
			t.Fatal(err)
		}
		if count > 0 {
			break
		}
		select {
		case <-waitCtx.Done():
			t.Fatal("request never reached database lock")
		case <-ticker.C:
		}
	}
	cancel()
	select {
	case status := <-finished:
		if status == 200 {
			t.Fatal("cancelled request succeeded")
		}
	case <-waitCtx.Done():
		t.Fatal("cancelled lock wait did not stop")
	}
	database.Rollback(tx)
	for _, table := range []string{"audit_events", "requests"} {
		t.Run(table, func(t *testing.T) {
			// Fixed, test-owned allowlist; no external identifiers reach this SQL.
			_, err := f.db.Exec(
				context.Background(),
				`CREATE FUNCTION reject_effect() RETURNS trigger LANGUAGE plpgsql AS $$
 BEGIN RAISE EXCEPTION 'synthetic failure'; END $$`,
			)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = f.db.Exec(
				context.Background(),
				`CREATE TRIGGER reject_effect BEFORE INSERT ON `+table+` FOR EACH ROW EXECUTE FUNCTION reject_effect()`,
			); err != nil {
				t.Fatal(err)
			}
			before := queryInt(t, f.db, `SELECT balance FROM members WHERE id=$1`, id)
			requireStatus(t, f.postCash("teller.a", "failure-rollback-"+table, "deposit", id, 500), 500)
			if queryInt(t, f.db, `SELECT balance FROM members WHERE id=$1`, id) != before {
				t.Fatal("partial cash effect")
			}
			if _, err = f.db.Exec(
				context.Background(),
				`DROP TRIGGER reject_effect ON `+table+`; DROP FUNCTION reject_effect()`,
			); err != nil {
				t.Fatal(err)
			}
		})
	}
	assertReconciled(t, f.db)
}
func TestConcurrentLoanDecisions(t *testing.T) {
	f := newFixture(t)
	id := f.member(t, "3271010101900031", "teller.a")
	w := f.call(
		"teller.a",
		"POST",
		"/api/loans",
		"decision-fixture-key",
		fmt.Sprintf(
			`{"member_id":%d,"amount":100000,"tenor_months":12,`+
				`"purpose":"Synthetic request","risk":"low","responsible_branch_id":2}`,
			id,
		),
	)
	requireStatus(t, w, 200)
	var result struct {
		ID int64 `json:"id"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	codes := make(chan int, 2)
	var wg sync.WaitGroup
	for _, decision := range []string{"approved", "rejected"} {
		wg.Add(1)
		go func(value string) {
			defer wg.Done()
			path := fmt.Sprintf("/api/loans/%d/decision", result.ID)
			payload := fmt.Sprintf(`{"decision":"%s","reason":"Synthetic verification"}`, value)
			response := f.call("supervisor", "POST", path, "concurrent-decision-"+value, payload)
			codes <- response.Code
		}(decision)
	}
	wg.Wait()
	close(codes)
	ok, conflict := 0, 0
	for code := range codes {
		if code == 200 {
			ok++
		}
		if code == 409 {
			conflict++
		}
	}
	if ok != 1 || conflict != 1 {
		t.Fatalf("want one decision and conflict, got %d/%d", ok, conflict)
	}
	if queryInt(t, f.db, `SELECT count(*) FROM transactions`) != 1 {
		t.Fatal("decision moved cash")
	}
	assertReconciled(t, f.db)
}
func queryInt(t *testing.T, pool *pgxpool.Pool, sql string, args ...any) int64 {
	t.Helper()
	var n int64
	if err := pool.QueryRow(context.Background(), sql, args...).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}
func assertReconciled(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	if queryInt(
		t,
		pool,
		`SELECT count(*) FROM (
 SELECT transaction_id FROM journal_lines GROUP BY transaction_id
 HAVING sum(debit)<>sum(credit)) q`,
	) != 0 {
		t.Fatal("unbalanced journal")
	}
	if queryInt(
		t,
		pool,
		`SELECT count(*) FROM members m WHERE m.balance<>(
 SELECT COALESCE(sum(j.credit-j.debit),0) FROM journal_lines j
 JOIN transactions t ON t.id=j.transaction_id
 WHERE t.member_id=m.id AND j.account='member_savings')`,
	) != 0 {
		t.Fatal("savings do not reconcile")
	}
}
