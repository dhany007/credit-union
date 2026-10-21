package main

import (
	"context"
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
	if err = database.Migrate(ctx, pool); err != nil {
		log.Print("migration failed")
		return 1
	}
	log.Print("Schema ready")
	return 0
}
