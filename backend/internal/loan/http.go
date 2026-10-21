package loan

import (
	"credit-union/backend/internal/auth"
	"credit-union/backend/internal/platform/fault"
	"credit-union/backend/internal/platform/httpapi"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes registers loan routes with the staff authorization guards.
func RegisterRoutes(r gin.IRouter, service *Service, guard *auth.Handler) {
	httpapi.GET(r, "/loans", guard.Require(), func(c *gin.Context) {
		rows, err := service.List(c.Request.Context())
		if err != nil {
			httpapi.Error(c, err)
			return
		}
		httpapi.JSON(c, http.StatusOK, rows)
	})
	r.POST("/loans", guard.Require("teller"), func(c *gin.Context) {
		var input Input
		if err := httpapi.Decode(c, &input); err != nil {
			httpapi.Error(c, err)
			return
		}
		actor := auth.ActorFrom(c)
		result, err := service.Apply(c.Request.Context(), actor, httpapi.Command(c, actor.ID), input)
		if err != nil {
			httpapi.Error(c, err)
			return
		}
		httpapi.JSON(c, http.StatusOK, result)
	})
	r.POST("/loans/:id/decision", guard.Require("supervisor"), func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			httpapi.Error(c, fault.New(fault.Invalid, "Invalid application"))
			return
		}
		var input Decision
		if err = httpapi.Decode(c, &input); err != nil {
			httpapi.Error(c, err)
			return
		}
		actor := auth.ActorFrom(c)
		result, err := service.Decide(c.Request.Context(), actor, httpapi.Command(c, actor.ID), id, input)
		if err != nil {
			httpapi.Error(c, err)
			return
		}
		httpapi.JSON(c, http.StatusOK, result)
	})
}
