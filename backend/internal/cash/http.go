package cash

import (
	"credit-union/backend/internal/auth"
	"credit-union/backend/internal/platform/fault"
	"credit-union/backend/internal/platform/httpapi"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes registers cash routes with the staff authorization guards.
func RegisterRoutes(r gin.IRouter, service *Service, guard *auth.Handler) {
	r.POST("/cash", guard.Require("teller"), func(c *gin.Context) {
		var input Input
		if err := httpapi.Decode(c, &input); err != nil {
			httpapi.Error(c, err)
			return
		}
		actor := auth.ActorFrom(c)
		result, err := service.Execute(c.Request.Context(), actor, httpapi.Command(c, actor.ID), input)
		if err != nil {
			httpapi.Error(c, err)
			return
		}
		httpapi.JSON(c, http.StatusOK, result)
	})
	httpapi.GET(r, "/members/:id/transactions", guard.Require(), func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			httpapi.Error(c, fault.New(fault.Invalid, "Invalid member"))
			return
		}
		rows, err := service.History(c.Request.Context(), id)
		if err != nil {
			httpapi.Error(c, err)
			return
		}
		httpapi.JSON(c, http.StatusOK, rows)
	})
}
