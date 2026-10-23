package interest

import (
	"credit-union/backend/internal/auth"
	"credit-union/backend/internal/platform/httpapi"
	"net/http"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes exposes only Supervisor policy/status operations.
func RegisterRoutes(r gin.IRouter, s *Service, guard *auth.Handler) {
	httpapi.GET(r, "/interest/policies", guard.Require("supervisor"), func(c *gin.Context) {
		items, err := s.Policies(c.Request.Context())
		if err != nil {
			httpapi.Error(c, err)
			return
		}
		httpapi.JSON(c, http.StatusOK, map[string]any{"items": items})
	})
	r.POST("/interest/policies", guard.Require("supervisor"), func(c *gin.Context) {
		var in PolicyInput
		if err := httpapi.Decode(c, &in); err != nil {
			httpapi.Error(c, err)
			return
		}
		actor := auth.ActorFrom(c)
		result, err := s.CreatePolicy(c.Request.Context(), actor, httpapi.Command(c, actor.ID), in)
		if err != nil {
			httpapi.Error(c, err)
			return
		}
		httpapi.JSON(c, http.StatusOK, result)
	})
	httpapi.GET(r, "/interest/status", guard.Require("supervisor"), func(c *gin.Context) {
		result, err := s.Status(c.Request.Context())
		if err != nil {
			httpapi.Error(c, err)
			return
		}
		httpapi.JSON(c, http.StatusOK, result)
	})
}
