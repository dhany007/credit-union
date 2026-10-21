package integration

import (
	"context"
	"credit-union/backend/internal/auth"
	"credit-union/backend/internal/platform/app"
	appconfig "credit-union/backend/internal/platform/config"
	"credit-union/backend/internal/platform/database"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestMVP(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set TEST_DATABASE_URL to run PostgreSQL integration tests")
	}
	ctx := context.Background()
	admin, e := pgxpool.New(ctx, dsn)
	if e != nil {
		t.Fatal(e)
	}
	defer admin.Close()
	schemaName := fmt.Sprintf("cu_test_%d", time.Now().UnixNano())
	if _, e = admin.Exec(ctx, "CREATE SCHEMA "+schemaName); e != nil {
		t.Fatal(e)
	}
	defer func() {
		if _, err := admin.Exec(ctx, "DROP SCHEMA "+schemaName+" CASCADE"); err != nil {
			t.Error(err)
		}
	}()
	config, e := pgxpool.ParseConfig(dsn)
	if e != nil {
		t.Fatal(e)
	}
	config.ConnConfig.RuntimeParams["search_path"] = schemaName
	db, e := pgxpool.NewWithConfig(ctx, config)
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	if e = database.Migrate(ctx, db); e != nil {
		t.Fatal(e)
	}
	if e = auth.Seed(ctx, db, "test-password-12345"); e != nil {
		t.Fatal(e)
	}
	cfg := appconfig.Config{Origin: "http://localhost:3000", ApprovalLimit: 10000000}
	handler := app.New(db, cfg, slog.New(slog.NewTextHandler(io.Discard, nil)))
	cookies := map[string]string{}
	for _, name := range []string{"teller.a", "teller.b", "supervisor"} {
		var id int64
		if e = db.QueryRow(ctx, "SELECT id FROM users WHERE username=$1", name).Scan(&id); e != nil {
			t.Fatal(e)
		}
		tok := token()
		cookies[name] = tok
		if _, e = db.Exec(
			ctx,
			"INSERT INTO sessions(token_hash,user_id,expires_at) VALUES($1,$2,now()+interval '1 hour')",
			database.Hash(tok),
			id,
		); e != nil {
			t.Fatal(e)
		}
	}
	call := func(name, method, path, key, payload string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, strings.NewReader(payload))
		r.Header.Set("Origin", cfg.Origin)
		r.Header.Set("Content-Type", "application/json")
		if key != "" {
			r.Header.Set("Idempotency-Key", key)
		}
		if name != "" {
			r.AddCookie(
				&http.Cookie{
					Name:     "cu_session",
					Value:    cookies[name],
					Secure:   true,
					HttpOnly: true,
					SameSite: http.SameSiteStrictMode,
				},
			)
		}
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		return w
	}
	expect := func(w *httptest.ResponseRecorder, status int) {
		t.Helper()
		if w.Code != status {
			t.Fatalf("want HTTP %d got %d: %s", status, w.Code, w.Body.String())
		}
	}
	registration := `{"nik":"3271010101900001","name":"Test Member","phone":"08123456789","address":"Jalan Pengujian 1"}`
	w := call("teller.a", "POST", "/api/members", "registration-key-0001", registration)
	expect(w, 200)
	var registered struct {
		ID        int64  `json:"id"`
		Reference string `json:"reference"`
	}
	if e = json.Unmarshal(w.Body.Bytes(), &registered); e != nil {
		t.Fatal(e)
	}
	id := registered.ID
	t.Run("registration retry and cooperative uniqueness", func(t *testing.T) {
		repeat := call("teller.a", "POST", "/api/members", "registration-key-0001", registration)
		expect(repeat, 200)
		var original, retried any
		if err := json.Unmarshal(w.Body.Bytes(), &original); err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(repeat.Body.Bytes(), &retried); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(original, retried) {
			t.Fatal("retry must return original response")
		}
		expect(call("teller.b", "POST", "/api/members", "registration-key-0002", registration), 409)
		var balance, count int64
		if err := db.QueryRow(ctx, "SELECT balance FROM members WHERE id=$1", id).Scan(&balance); err != nil {
			t.Fatal(err)
		}
		if err := db.QueryRow(ctx, "SELECT count(*) FROM transactions WHERE member_id=$1", id).
			Scan(&count); err != nil {
			t.Fatal(err)
		}
		if balance != 50000 || count != 1 {
			t.Fatal("duplicate registration changed funds")
		}
	})

	t.Run("concurrent registration creates one membership", func(t *testing.T) {
		payload := `{"nik":"3271010101900002","name":"Concurrent Member","phone":"08123456789","address":"Test address only"}`
		codes := make(chan int, 2)
		var wg sync.WaitGroup
		for i, name := range []string{"teller.a", "teller.b"} {
			wg.Add(1)
			go func(i int, name string) {
				defer wg.Done()
				codes <- call(name, "POST", "/api/members", fmt.Sprintf("concurrent-register-%d", i), payload).Code
			}(i, name)
		}
		wg.Wait()
		close(codes)
		successful, duplicate := 0, 0
		for code := range codes {
			if code == 200 {
				successful++
			}
			if code == 409 {
				duplicate++
			}
		}
		if successful != 1 || duplicate != 1 {
			t.Fatalf("expected one membership and one duplicate; got %d/%d", successful, duplicate)
		}
		var count int
		if e := db.QueryRow(ctx, `SELECT count(*) FROM members WHERE nik='3271010101900002'`).
			Scan(&count); e != nil ||
			count != 1 {
			t.Fatal("concurrent identity uniqueness failed")
		}
	})
	t.Run("cross branch deposits and idempotency", func(t *testing.T) {
		input := fmt.Sprintf(`{"member_id":%d,"amount":100000,"kind":"deposit","note":"cross KCP"}`, id)
		first := call("teller.b", "POST", "/api/cash", "deposit-key-000001", input)
		expect(first, 200)
		repeat := call("teller.b", "POST", "/api/cash", "deposit-key-000001", input)
		expect(repeat, 200)
		if first.Body.String() != repeat.Body.String() {
			t.Fatal("duplicate result differs")
		}
		expect(
			call("teller.b", "POST", "/api/cash", "deposit-key-000001", strings.Replace(input, "100000", "100001", 1)),
			409,
		)
		var balance int64
		var home int
		if err := db.QueryRow(ctx, "SELECT balance,home_branch_id FROM members WHERE id=$1", id).
			Scan(&balance, &home); err != nil {
			t.Fatal(err)
		}
		if balance != 150000 || home != 1 {
			t.Fatal("cross branch deposit changed home or double credited")
		}
	})
	t.Run("concurrent withdrawals cannot overdraw", func(t *testing.T) {
		codes := make(chan int, 2)
		var wg sync.WaitGroup
		for i := 0; i < 2; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				w := call(
					"teller.b",
					"POST",
					"/api/cash",
					fmt.Sprintf("withdrawal-key-%03d", i),
					fmt.Sprintf(`{"member_id":%d,"amount":80000,"kind":"withdrawal","note":""}`, id),
				)
				codes <- w.Code
			}(i)
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
			t.Fatalf("want one success and one blocked, got %d/%d", ok, blocked)
		}
		var balance int64
		if err := db.QueryRow(ctx, "SELECT balance FROM members WHERE id=$1", id).Scan(&balance); err != nil {
			t.Fatal(err)
		}
		if balance != 70000 {
			t.Fatal("incorrect balance after concurrent withdrawals")
		}
	})
	t.Run("posting failure rolls back balances and records", func(t *testing.T) {
		_, e = db.Exec(
			ctx,
			`CREATE FUNCTION reject_journal() RETURNS trigger LANGUAGE plpgsql AS $$
 BEGIN RAISE EXCEPTION 'simulated journal failure'; END $$;
 CREATE TRIGGER reject_journal BEFORE INSERT ON journal_lines
 FOR EACH ROW EXECUTE FUNCTION reject_journal()`,
		)
		if e != nil {
			t.Fatal(e)
		}
		expect(
			call(
				"teller.a",
				"POST",
				"/api/cash",
				"failure-key-000001",
				fmt.Sprintf(`{"member_id":%d,"amount":500,"kind":"deposit","note":""}`, id),
			),
			500,
		)
		var balance int64
		if err := db.QueryRow(ctx, "SELECT balance FROM members WHERE id=$1", id).Scan(&balance); err != nil {
			t.Fatal(err)
		}
		if balance != 70000 {
			t.Fatal("partial balance update")
		}
		if _, e = db.Exec(
			ctx,
			"DROP TRIGGER reject_journal ON journal_lines; DROP FUNCTION reject_journal()",
		); e != nil {
			t.Fatal(e)
		}
	})
	t.Run("roles origin and exact amounts", func(_ *testing.T) {
		input := fmt.Sprintf(`{"member_id":%d,"amount":100,"kind":"deposit","note":""}`, id)
		expect(call("supervisor", "POST", "/api/cash", "forbidden-key-0001", input), 403)
		expect(call("", "GET", "/api/members", "", ""), 401)
		expect(
			call(
				"teller.a",
				"POST",
				"/api/cash",
				"invalid-amount-key",
				strings.Replace(input, `"amount":100`, `"amount":0.5`, 1),
			),
			400,
		)
		r := httptest.NewRequest("POST", "/api/cash", strings.NewReader(input))
		r.AddCookie(
			&http.Cookie{
				Name:     "cu_session",
				Value:    cookies["teller.a"],
				Secure:   true,
				HttpOnly: true,
				SameSite: http.SameSiteStrictMode,
			},
		)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, r)
		expect(rec, 403)
	})
	t.Run("approval follows amount product risk and moves no cash", func(t *testing.T) {
		create := func(amount int, risk string, key string) int64 {
			t.Helper()
			rec := call(
				"teller.b",
				"POST",
				"/api/loans",
				key,
				fmt.Sprintf(
					`{"member_id":%d,"amount":%d,"tenor_months":12,"purpose":"Business equipment",`+
						`"risk":"%s","responsible_branch_id":1}`,
					id,
					amount,
					risk,
				),
			)
			expect(rec, 200)
			var l struct {
				ID int64 `json:"id"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &l); err != nil {
				t.Fatal(err)
			}
			return l.ID
		}
		loan := create(1000000, "low", "loan-key-00000001")
		path := fmt.Sprintf("/api/loans/%d/decision", loan)
		decision := `{"decision":"approved","reason":"Documents and authority verified"}`
		expect(call("teller.a", "POST", path, "decision-key-0001", decision), 403)
		cfg.ApprovalLimit = 0
		handler = app.New(db, cfg, slog.New(slog.NewTextHandler(io.Discard, nil)))
		expect(call("supervisor", "POST", path, "decision-key-0002", decision), 422)
		cfg.ApprovalLimit = 10000000
		handler = app.New(db, cfg, slog.New(slog.NewTextHandler(io.Discard, nil)))
		expect(call("supervisor", "POST", path, "decision-key-0003", decision), 200)
		expect(call("supervisor", "POST", path, "decision-key-0003", decision), 200)
		expect(call("supervisor", "POST", path, "decision-key-0004", decision), 409)
		high := create(1000000, "high", "loan-key-00000002")
		expect(
			call("supervisor", "POST", fmt.Sprintf("/api/loans/%d/decision", high), "decision-key-0005", decision),
			422,
		)
		large := create(10000001, "low", "loan-key-00000003")
		expect(
			call("supervisor", "POST", fmt.Sprintf("/api/loans/%d/decision", large), "decision-key-0006", decision),
			422,
		)
		expect(call("teller.a", "POST", fmt.Sprintf("/api/loans/%d/disburse", loan), "disburse-key-0001", `{}`), 404)
		var balance, unbalanced int64
		if err := db.QueryRow(ctx, "SELECT balance FROM members WHERE id=$1", id).Scan(&balance); err != nil {
			t.Fatal(err)
		}
		if balance != 70000 {
			t.Fatal("approval moved savings")
		}
		if err := db.QueryRow(ctx, `SELECT count(*) FROM (
 SELECT transaction_id FROM journal_lines GROUP BY transaction_id
 HAVING sum(debit)<>sum(credit)) q`).
			Scan(&unbalanced); err != nil {
			t.Fatal(err)
		}
		if unbalanced != 0 {
			t.Fatal("unbalanced journal")
		}
	})
	t.Run("password login and logout invalidate session", func(_ *testing.T) {
		expect(call("", "POST", "/api/login", "", `{"username":"teller.a","password":"incorrect"}`), 401)
		rec := call("", "POST", "/api/login", "", `{"username":"teller.a","password":"test-password-12345"}`)
		expect(rec, 200)
		cookies["teller.a"] = rec.Result().Cookies()[0].Value
		expect(call("teller.a", "GET", "/api/me", "", ""), 200)
		expect(call("teller.a", "POST", "/api/logout", "", `{}`), 200)
		expect(call("teller.a", "GET", "/api/me", "", ""), 401)
	})
}

func token() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}
