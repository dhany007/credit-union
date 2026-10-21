package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"path"
	"strings"
	"sync/atomic"
	"time"

	"credit-union/backend/internal/platform/database"
	"credit-union/backend/internal/platform/fault"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgconn"
)

const requestTimeout = 15 * time.Second

const maxBodyBytes = 16 * 1024

var requestSequence atomic.Uint64

// Decode decodes exactly one bounded JSON object and rejects unknown fields.
func Decode(c *gin.Context, into any) error {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBodyBytes)
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(into); err != nil {
		return fault.New(fault.Invalid, "Invalid request data")
	}
	if decoder.Decode(&struct{}{}) != io.EOF {
		return fault.New(fault.Invalid, "Only one JSON object is allowed")
	}
	return nil
}

// JSON writes the existing JSON response contract.
func JSON(c *gin.Context, status int, value any) {
	c.Header("Content-Type", "application/json")
	c.Status(status)
	if err := json.NewEncoder(c.Writer).Encode(value); err != nil {
		c.Abort()
	}
}

// Error is a safe application failure with an optional underlying cause.
func Error(c *gin.Context, err error) {
	status, message := http.StatusInternalServerError, "Operation failed. Please retry with the same reference."
	var appErr *fault.Error
	if errors.As(err, &appErr) {
		statuses := map[fault.Kind]int{
			fault.Invalid:      http.StatusBadRequest,
			fault.Unauthorized: http.StatusUnauthorized,
			fault.Forbidden:    http.StatusForbidden,
			fault.NotFound:     http.StatusNotFound,
			fault.Conflict:     http.StatusConflict,
			fault.Rejected:     http.StatusUnprocessableEntity,
			fault.Unavailable:  http.StatusServiceUnavailable,
		}
		if code, ok := statuses[appErr.Kind]; ok {
			status, message = code, appErr.Message
		}
	} else {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			status, message = 409, "Data already exists. Check the NIK or request reference."
		}
	}
	if status >= http.StatusInternalServerError {
		c.Set("error_class", "dependency_or_internal")
	}
	JSON(c, status, map[string]string{"error": message})
	c.Abort()
}

// Command captures authenticated request identity for durable replay.
func Command(c *gin.Context, actorID int64) database.Command {
	return database.Command{ActorID: actorID, Key: c.GetHeader("Idempotency-Key"),
		Method: c.Request.Method, Path: c.Request.URL.Path}
}

// GET registers equivalent GET and HEAD handlers.
func GET(r gin.IRoutes, route string, handlers ...gin.HandlerFunc) {
	r.GET(route, handlers...)
	r.HEAD(route, handlers...)
}

// New constructs the configured component with explicit dependencies.
func New(origin string, logger *slog.Logger) *gin.Engine {
	if logger == nil {
		logger = slog.Default()
	}
	r := gin.New()
	r.HandleMethodNotAllowed = true
	r.RedirectTrailingSlash = false
	r.RedirectFixedPath = false
	_ = r.SetTrustedProxies(nil)
	r.Use(func(c *gin.Context) {
		requestCtx, cancel := context.WithTimeout(c.Request.Context(), requestTimeout)
		defer cancel()
		c.Request = c.Request.WithContext(requestCtx)
		start := time.Now()
		id := requestSequence.Add(1)
		c.Set("request_id", id)
		defer func() {
			if recover() != nil {
				if !c.Writer.Written() {
					JSON(
						c,
						http.StatusInternalServerError,
						map[string]string{"error": "Operation failed. Please retry with the same reference."},
					)
				}
				c.Abort()
				c.Set("error_class", "panic")
			}
			template := c.FullPath()
			if template == "" {
				template = "unmatched"
			}
			class, _ := c.Get("error_class")
			logger.Info("http request", "request_id", id, "method", c.Request.Method, "route", template,
				"status", c.Writer.Status(), "duration_ms", time.Since(start).Milliseconds(), "error_class", class)
		}()
		c.Header("Cache-Control", "no-store")
		c.Header("X-Content-Type-Options", "nosniff")
		if c.Request.Method != "GET" && c.Request.Method != "HEAD" && c.GetHeader("Origin") != origin {
			Error(c, fault.New(fault.Forbidden, "Invalid request origin"))
			return
		}
		// Match ServeMux's path cleaning before route dispatch, without touching query contents.
		clean := path.Clean(c.Request.URL.Path)
		if strings.HasSuffix(c.Request.URL.Path, "/") && clean != "/" {
			clean += "/"
		}
		if clean != c.Request.URL.Path {
			target := *c.Request.URL
			target.Path = clean
			target.RawPath = ""
			http.Redirect(c.Writer, c.Request, target.String(), http.StatusMovedPermanently)
			c.Abort()
			return
		}
		c.Next()
	})
	r.NoRoute(func(c *gin.Context) { http.NotFound(c.Writer, c.Request) })
	r.NoMethod(func(c *gin.Context) {
		methods := []string{}
		for _, route := range r.Routes() {
			if sameRoute(route.Path, c.Request.URL.Path) {
				methods = append(methods, route.Method)
			}
		}
		c.Header("Allow", strings.Join(methods, ", "))
		http.Error(c.Writer, "Method Not Allowed", http.StatusMethodNotAllowed)
	})
	return r
}
func sameRoute(pattern, value string) bool {
	p, v := strings.Split(pattern, "/"), strings.Split(value, "/")
	if len(p) != len(v) {
		return false
	}
	for i := range p {
		if !strings.HasPrefix(p[i], ":") && p[i] != v[i] {
			return false
		}
	}
	return true
}
