package main

import (
	"context"
	"log"

	"github.com/you-humble/rocket-maintenance/inventory/internal/app"
	"github.com/you-humble/rocket-maintenance/inventory/internal/config"
)

const (
	GRPCAddr = "0.0.0.0:50051"
	MongoDSN = "mongodb://inventory-service-user:inventory-service-password@mongo-inventory:27017/inventory-service?authSource=admin"
)

func main() {
	ctx := context.Background()

	if err := config.Load(); err != nil {
		log.Fatal(err)
	}
	cfg := config.C()

	if err := app.Run(ctx, cfg.Server, cfg.Mongo); err != nil {
		log.Fatalf("❌😵‍💫 inventory server stopped with error: %v", err)
	}
}
