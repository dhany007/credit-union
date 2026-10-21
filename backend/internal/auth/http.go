package auth

import (
	"credit-union/backend/internal/platform/fault"
	"credit-union/backend/internal/platform/httpapi"
	"net"
	"net/http"

	"github.com/gin-gonic/gin"
)

const cookieName = "cu_session"
const sessionSeconds = 8 * 60 * 60
const actorKey = "authenticated_actor"

// Handler adapts staff authentication to Gin and session cookies.
type Handler struct {
	service *Service
	secure  bool
}

// NewHandler constructs the authentication HTTP adapter.
func NewHandler(service *Service, secure bool) *Handler {
	return &Handler{service: service, secure: secure}
}

// Register registers login, logout, and current-staff endpoints.
func (h *Handler) Register(r gin.IRouter) {
	r.POST("/login", h.login)
	r.POST("/logout", h.Require(), h.logout)
	httpapi.GET(r, "/me", h.Require(), func(c *gin.Context) { httpapi.JSON(c, http.StatusOK, ActorFrom(c)) })
}

// ActorFrom returns the identity established by session middleware.
func ActorFrom(c *gin.Context) Actor { v, _ := c.Get(actorKey); actor, _ := v.(Actor); return actor }

// Require rejects missing sessions and roles before invoking business handlers.
func (h *Handler) Require(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw, err := c.Cookie(cookieName)
		if err != nil {
			httpapi.Error(c, fault.New(fault.Unauthorized, "Please sign in"))
			return
		}
		actor, err := h.service.Current(c.Request.Context(), raw)
		if err != nil {
			httpapi.Error(c, err)
			return
		}
		if len(roles) > 0 {
			allowed := false
			for _, role := range roles {
				if actor.Role == role {
					allowed = true
				}
			}
			if !allowed {
				httpapi.Error(c, fault.New(fault.Forbidden, "Your role cannot perform this action"))
				return
			}
		}
		c.Set(actorKey, actor)
		c.Next()
	}
}
func (h *Handler) login(c *gin.Context) {
	ip, _, _ := net.SplitHostPort(c.Request.RemoteAddr)
	if err := h.service.CheckAttempt(ip); err != nil {
		httpapi.JSON(c, http.StatusTooManyRequests, map[string]string{"error": err.Error()})
		return
	}
	var input LoginInput
	if err := httpapi.Decode(c, &input); err != nil {
		httpapi.Error(c, err)
		return
	}
	old, _ := c.Cookie(cookieName)
	raw, err := h.service.Login(c.Request.Context(), input, old)
	if err != nil {
		httpapi.Error(c, err)
		return
	}
	// #nosec G124 -- Secure follows deployment config; HttpOnly and Strict are always set.
	http.SetCookie(c.Writer, &http.Cookie{Name: cookieName, Value: raw, Path: "/", HttpOnly: true,
		Secure: h.secure, SameSite: http.SameSiteStrictMode, MaxAge: sessionSeconds})
	httpapi.JSON(c, http.StatusOK, map[string]bool{"ok": true})
}
func (h *Handler) logout(c *gin.Context) {
	raw, _ := c.Cookie(cookieName)
	if err := h.service.Logout(c.Request.Context(), raw); err != nil {
		httpapi.Error(c, err)
		return
	}
	// #nosec G124 -- Secure follows deployment config; HttpOnly and Strict are always set.
	http.SetCookie(c.Writer, &http.Cookie{Name: cookieName, Path: "/", HttpOnly: true,
		Secure: h.secure, SameSite: http.SameSiteStrictMode, MaxAge: -1})
	httpapi.JSON(c, http.StatusOK, map[string]bool{"ok": true})
}
