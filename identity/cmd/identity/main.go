package main

import (
	"context"
	"log"
	"net"

	identityv1 "github.com/foodly/contracts/gen/go/identity/v1"
	"github.com/foodly/identity/internal/config"
	grpcsrv "github.com/foodly/identity/internal/grpc"
	"github.com/foodly/identity/internal/postgres"
	"github.com/foodly/identity/internal/user"
	"google.golang.org/grpc"
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

	users, err := user.NewService(user.NewRepository(pool), []byte(cfg.JWTSecret))
	if err != nil {
		log.Fatal(err)
	}

	lis, err := net.Listen("tcp", cfg.GRPCAddr)
	if err != nil {
		log.Fatal(err)
	}

	srv := grpc.NewServer()
	identityv1.RegisterIdentityServiceServer(srv, grpcsrv.New(users))

	log.Println("identity: listening on", cfg.GRPCAddr)
	if err := srv.Serve(lis); err != nil {
		log.Fatal(err)
	}
}
