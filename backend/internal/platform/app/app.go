// Package app is the composition root. Domains own their handlers, services, and stores.
package app

import (
	"credit-union/backend/internal/auth"
	"credit-union/backend/internal/cash"
	"credit-union/backend/internal/loan"
	"credit-union/backend/internal/member"
	"credit-union/backend/internal/platform/config"
	"credit-union/backend/internal/platform/database"
	"credit-union/backend/internal/platform/httpapi"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// New constructs the configured component with explicit dependencies.
func New(pool *pgxpool.Pool, cfg config.Config, logger *slog.Logger) *gin.Engine {
	router := httpapi.New(cfg.Origin, logger)
	runner := database.NewRunner(pool)
	authService := auth.NewService(auth.NewPostgresStore(pool))
	guard := auth.NewHandler(authService, cfg.SecureCookies)
	cashService := cash.NewService(cash.NewPostgresStore(pool), runner)
	memberService := member.NewService(member.NewPostgresStore(pool), runner, cashService)
	loanService := loan.NewService(loan.NewPostgresStore(pool), runner, cfg.ApprovalLimit)
	routes := router.Group("/api")
	guard.Register(routes)
	member.RegisterRoutes(routes, memberService, guard)
	cash.RegisterRoutes(routes, cashService, guard)
	loan.RegisterRoutes(routes, loanService, guard)
	httpapi.GET(routes, "/health", func(c *gin.Context) {
		if err := pool.Ping(c.Request.Context()); err != nil {
			httpapi.JSON(c, http.StatusServiceUnavailable, map[string]string{"error": "Database unavailable"})
			return
		}
		httpapi.JSON(c, http.StatusOK, map[string]string{"status": "ok"})
	})
	httpapi.GET(routes, "/meta", guard.Require(), func(c *gin.Context) {
		branches, err := cashService.Branches(c.Request.Context())
		if err != nil {
			httpapi.Error(c, err)
			return
		}
		available, err := cashService.BranchCash(c.Request.Context(), auth.ActorFrom(c).BranchID)
		if err != nil {
			httpapi.Error(c, err)
			return
		}
		httpapi.JSON(c, http.StatusOK, map[string]any{"branches": branches, "branch_cash": available,
			"approval_limit": cfg.ApprovalLimit, "approval_product": "regular",
			"approval_risk": "low", "disbursement_enabled": false})
	})
	return router
}
