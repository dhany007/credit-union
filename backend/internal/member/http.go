package member

import (
	"credit-union/backend/internal/auth"
	"credit-union/backend/internal/platform/httpapi"
	"net/http"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes registers member routes with the staff authorization guards.
func RegisterRoutes(r gin.IRouter, service *Service, guard *auth.Handler) {
	httpapi.GET(r, "/members/by-account/:account_number", guard.Require(), func(c *gin.Context) {
		item, err := service.FindByAccountNumber(c.Request.Context(), c.Param("account_number"))
		if err != nil {
			httpapi.Error(c, err)
			return
		}
		httpapi.JSON(c, http.StatusOK, item)
	})
	httpapi.GET(r, "/members", guard.Require(), func(c *gin.Context) {
		rows, err := service.List(c.Request.Context(), c.Query("search"))
		if err != nil {
			httpapi.Error(c, err)
			return
		}
		httpapi.JSON(c, http.StatusOK, rows)
	})
	r.POST("/members", guard.Require("teller"), func(c *gin.Context) {
		var input Input
		if err := httpapi.Decode(c, &input); err != nil {
			httpapi.Error(c, err)
			return
		}
		actor := auth.ActorFrom(c)
		result, err := service.Register(c.Request.Context(), actor, httpapi.Command(c, actor.ID), input)
		if err != nil {
			httpapi.Error(c, err)
			return
		}
		httpapi.JSON(c, http.StatusOK, result)
	})
}
