package httpapi

import (
	"bytes"
	"log/slog"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestLoggingAndRecoveryDoNotExposeInputs(t *testing.T) {
	var logs bytes.Buffer
	router := New("http://localhost:3000", slog.New(slog.NewJSONHandler(&logs, nil)))
	router.GET("/search", func(c *gin.Context) { c.Status(204) })
	router.GET("/panic", func(_ *gin.Context) { panic("private-credential-value") })
	for _, target := range []string{"/search?nik=1234567890123456", "/panic"} {
		request := httptest.NewRequest("GET", target, nil)
		request.Header.Set("Authorization", "Bearer private-token")
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		if target == "/panic" && (response.Code != 500 || strings.Contains(response.Body.String(), "private")) {
			t.Fatal("unsafe panic response")
		}
	}
	for _, secret := range []string{"1234567890123456", "private-token", "private-credential-value"} {
		if strings.Contains(logs.String(), secret) {
			t.Fatalf("log leaked %s", secret)
		}
	}
	if !strings.Contains(logs.String(), `"route":"/search"`) {
		t.Fatal("missing route template")
	}
}
