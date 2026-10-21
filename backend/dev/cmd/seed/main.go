package main

import (
	"context"
	"credit-union/backend/internal/auth"
	"credit-union/backend/internal/platform/config"
	"credit-union/backend/internal/platform/database"
	"log"
	"os"
	"time"
)

const commandTimeout = 30 * time.Second

func main() { os.Exit(run()) }
func run() int {
	cfg, err := config.Load(os.Getenv)
	if err != nil {
		log.Print("invalid configuration")
		return 1
	}
	ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
	defer cancel()
	pool, err := database.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Print("database unavailable")
		return 1
	}
	defer pool.Close()
	if err = auth.Seed(ctx, pool, os.Getenv("DEV_SEED_PASSWORD")); err != nil {
		log.Print("development seed failed; check local password and schema")
		return 1
	}
	log.Print("Development accounts ready")
	return 0
}
