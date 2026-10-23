package main

import (
	"context"
	"credit-union/backend/internal/interest"
	"credit-union/backend/internal/platform/app"
	"credit-union/backend/internal/platform/config"
	"credit-union/backend/internal/platform/database"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	shutdownTimeout   = 10 * time.Second
	readHeaderTimeout = 5 * time.Second
	readTimeout       = 15 * time.Second
	writeTimeout      = 20 * time.Second
	idleTimeout       = 60 * time.Second
)

func main() { os.Exit(run()) }
func run() int {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg, err := config.Load(os.Getenv)
	if err != nil {
		logger.Error("invalid configuration")
		return 1
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	startup, cancel := context.WithTimeout(ctx, shutdownTimeout)
	pool, err := database.Open(startup, cfg.DatabaseURL)
	cancel()
	if err != nil {
		logger.Error("database unavailable", "operation", "database.open")
		return 1
	}
	defer pool.Close()
	interestService := interest.NewService(pool, database.NewRunner(pool), cfg.InterestEnabled)
	jobDone := make(chan struct{})
	jobCtx, stopJob := context.WithCancel(ctx)
	defer stopJob()
	go func() {
		defer close(jobDone)
		interestService.Run(jobCtx, logger)
	}()
	defer func() { stopJob(); <-jobDone }()
	gin.SetMode(gin.ReleaseMode)
	server := &http.Server{
		Addr:              cfg.Address,
		Handler:           app.NewWithInterest(pool, cfg, logger, interestService),
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
	}
	result := make(chan error, 1)
	go func() { result <- server.ListenAndServe() }()
	logger.Info("api listening", "address", cfg.Address)
	exit := 0
	select {
	case <-ctx.Done():
	case err = <-result:
		if !errors.Is(err, http.ErrServerClosed) {
			logger.Error("api listener failed")
			exit = 1
		}
	}
	shutdown, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err = server.Shutdown(shutdown); err != nil {
		logger.Error("api shutdown failed")
		_ = server.Close()
		exit = 1
	}
	return exit
}
