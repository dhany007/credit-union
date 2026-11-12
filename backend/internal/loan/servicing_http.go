package loan

import (
	"credit-union/backend/internal/auth"
	"credit-union/backend/internal/platform/httpapi"
	"net/http"

	"github.com/gin-gonic/gin"
)

func routeID(c *gin.Context) (int64, bool) {
	id, err := parseID(c.Param("id"))
	if err != nil {
		httpapi.Error(c, err)
		return 0, false
	}
	return id, true
}
func send(c *gin.Context, result any, err error) {
	if err != nil {
		httpapi.Error(c, err)
		return
	}
	httpapi.JSON(c, http.StatusOK, result)
}
func registerAccountRoutes(r gin.IRouter, s *Service, g *auth.Handler) {
	httpapi.GET(r, "/loan-applications", g.Require("teller", "supervisor"), func(c *gin.Context) {
		out, err := s.ApplicationPage(c.Request.Context(), c.Request.URL.Query())
		send(c, out, err)
	})
	httpapi.GET(r, "/loan-accounts", g.Require("teller", "supervisor"), func(c *gin.Context) {
		out, err := s.AccountPage(c.Request.Context(), c.Request.URL.Query())
		send(c, out, err)
	})
	for _, application := range []bool{true, false} {
		path := "/loan-accounts/:id"
		if application {
			path = "/loan-applications/:id"
		}
		httpapi.GET(r, path, g.Require("teller", "supervisor"), func(c *gin.Context) {
			id, ok := routeID(c)
			if !ok {
				return
			}
			out, err := s.Detail(c.Request.Context(), id, application)
			send(c, out, err)
		})
	}
	r.POST("/loan-applications/:id/disbursement-preview", g.Require("supervisor"), func(c *gin.Context) {
		id, ok := routeID(c)
		if !ok {
			return
		}
		var in PayoutPreviewInput
		if err := httpapi.Decode(c, &in); err != nil {
			httpapi.Error(c, err)
			return
		}
		out, err := s.PayoutPreview(c.Request.Context(), auth.ActorFrom(c), id, in)
		send(c, out, err)
	})
	r.POST("/loan-applications/:id/disburse", g.Require("supervisor"), func(c *gin.Context) {
		id, ok := routeID(c)
		if !ok {
			return
		}
		var in PayoutInput
		if err := httpapi.Decode(c, &in); err != nil {
			httpapi.Error(c, err)
			return
		}
		actor := auth.ActorFrom(c)
		out, err := s.Disburse(c.Request.Context(), actor, httpapi.Command(c, actor.ID), id, in)
		send(c, out, err)
	})
	for _, kind := range []string{"installments", "schedule-versions", "repayments"} {
		httpapi.GET(r, "/loan-accounts/:id/"+kind, g.Require("teller", "supervisor"), func(c *gin.Context) {
			id, ok := routeID(c)
			if !ok {
				return
			}
			out, err := s.History(c.Request.Context(), id, kind, c.Request.URL.Query())
			send(c, out, err)
		})
	}
	r.POST("/loan-accounts/:id/repayment-preview", g.Require("teller"), func(c *gin.Context) {
		id, ok := routeID(c)
		if !ok {
			return
		}
		var in struct {
			Amount          int64 `json:"amount"`
			NextInstallment bool  `json:"next_installment"`
		}
		if err := httpapi.Decode(c, &in); err != nil {
			httpapi.Error(c, err)
			return
		}
		out, err := s.RepaymentPreview(c.Request.Context(), id, in.Amount, in.NextInstallment)
		send(c, out, err)
	})
	r.POST("/loan-accounts/:id/repayments", g.Require("teller"), func(c *gin.Context) {
		id, ok := routeID(c)
		if !ok {
			return
		}
		var in RepaymentInput
		if err := httpapi.Decode(c, &in); err != nil {
			httpapi.Error(c, err)
			return
		}
		actor := auth.ActorFrom(c)
		out, err := s.Repay(c.Request.Context(), actor, httpapi.Command(c, actor.ID), id, in)
		send(c, out, err)
	})
}
