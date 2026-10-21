package integration

import (
	"credit-union/backend/internal/platform/app"
	"credit-union/backend/internal/platform/config"

	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestTransportBaseline(t *testing.T) {
	cfg := config.Config{Origin: "http://localhost:3000"}
	handler := app.New(nil, cfg, slog.New(slog.NewTextHandler(io.Discard, nil)))
	tests := []struct {
		name, method, path, body string
		want                     int
	}{
		{"unknown route", "GET", "/api/not-found", "", 404},
		{"wrong method", "PUT", "/api/members", "", 405},
		{"session required", "GET", "/api/members", "", 401},
		{"unknown fields", "POST", "/api/login", `{"username":"x","password":"x","extra":1}`, 400},
		{"trailing JSON", "POST", "/api/login", `{"username":"x","password":"x"}{}`, 400},
		{"oversized", "POST", "/api/login", `{"username":"` + strings.Repeat("x", 17000) + `"}`, 400},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body))
			r.Header.Set("Origin", cfg.Origin)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, r)
			if w.Code != tt.want {
				t.Fatalf("want %d got %d: %s", tt.want, w.Code, w.Body.String())
			}
			if tt.want == 405 && !strings.Contains(w.Header().Get("Allow"), "GET") {
				t.Fatal("missing Allow header")
			}
		})
	}
	server := httptest.NewServer(handler)
	defer server.Close()
	response, err := http.Head(server.URL + "/api/members")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := response.Body.Close(); err != nil {
			t.Error(err)
		}
	}()
	data, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != 401 || len(data) != 0 {
		t.Fatal("HEAD must enforce auth without a body")
	}
}

func TestAuthDependencyFailure(t *testing.T) {
	pool, err := pgxpool.New(context.Background(), "postgres:///credit_union?host=/var/run/postgresql")
	if err != nil {
		t.Fatal(err)
	}
	pool.Close()
	cfg := config.Config{Origin: "http://localhost:3000"}
	handler := app.New(pool, cfg, slog.New(slog.NewTextHandler(io.Discard, nil)))
	for _, path := range []string{"/api/me", "/api/login"} {
		t.Run(path, func(t *testing.T) {
			method := "GET"
			input := ""
			if path == "/api/login" {
				method = "POST"
				input = `{"username":"someone","password":"not-a-real-password"}`
			}
			r := httptest.NewRequest(method, path, strings.NewReader(input))
			r.Header.Set("Origin", cfg.Origin)
			r.AddCookie(
				&http.Cookie{
					Name:     "cu_session",
					Value:    "synthetic-test-token",
					Secure:   true,
					HttpOnly: true,
					SameSite: http.SameSiteStrictMode,
				},
			)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, r)
			if w.Code != 503 {
				t.Fatalf("database failure must be 503, got %d: %s", w.Code, w.Body.String())
			}
		})
	}
}
