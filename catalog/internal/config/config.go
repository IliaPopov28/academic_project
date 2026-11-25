package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL string
	GRPCAddr string
}

func Load() (Config, error) {
	_ = godotenv.Load()

	cfg := Config {
		DatabaseURL: os.Getenv("DATABASE_URL"),
		GRPCAddr: os.Getenv("GRPC_ADDR"),
	}

	var missing []string
	if cfg.DatabaseURL == "" {
		missing = append(missing, "DATABASE_URL")
	}
	if cfg.GRPCAddr == "" {
		missing = append(missing, "GRPC_ADDR")
	}

	if len(missing) > 0 {
		return cfg, fmt.Errorf("missing required environment variables: %v", missing)
	}

	return cfg, nil
}