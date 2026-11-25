package main

import (
	"context"
	"log"

	"github.com/foodly/catalog/internal/config"
	"github.com/foodly/catalog/internal/postgres"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	ctx := context.Background()

	if err := postgres.EnsureDatabase(ctx, cfg.DatabaseURL); err != nil {
		log.Fatal(err)
	}
	if err := postgres.Migrate(cfg.DatabaseURL); err != nil {
		log.Fatal(err)
	}

	pool, err := postgres.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	log.Println("catalog: migrations applied, pool ready")
}