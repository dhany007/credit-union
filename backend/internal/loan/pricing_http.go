package loan

import (
	"credit-union/backend/internal/auth"
	"credit-union/backend/internal/platform/httpapi"
	"net/http"

	"github.com/gin-gonic/gin"
)

func registerPricingRoutes(r gin.IRouter, s *Service, guard *auth.Handler) {
	r.POST("/loans/preview", guard.Require("teller", "supervisor"), func(c *gin.Context) {
		var in PreviewInput
		if err := httpapi.Decode(c, &in); err != nil {
			httpapi.Error(c, err)
			return
		}
		result, err := s.Preview(c.Request.Context(), in)
		if err != nil {
			httpapi.Error(c, err)
			return
		}
		httpapi.JSON(c, http.StatusOK, result)
	})
	httpapi.GET(r, "/loan-interest/policies", guard.Require("supervisor"), func(c *gin.Context) {
		result, err := s.Policies(c.Request.Context(), c.Request.URL.Query())
		if err != nil {
			httpapi.Error(c, err)
			return
		}
		httpapi.JSON(c, http.StatusOK, result)
	})
	r.POST("/loan-interest/policies", guard.Require("supervisor"), func(c *gin.Context) {
		var in Policy
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
}
