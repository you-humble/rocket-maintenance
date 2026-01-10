package main

import (
	"context"
	"log"

	"github.com/you-humble/rocket-maintenance/inventory/internal/app"
)

const (
	GRPCAddr = "0.0.0.0:50051"
	MongoDSN = "mongodb://inventory-service-user:inventory-service-password@mongo-inventory:27017/inventory-service?authSource=admin"
)

func main() {
	ctx := context.Background()

	cfg := app.Config{GRPCAddr: GRPCAddr, MongoDSN: MongoDSN}

	if err := app.Run(ctx, cfg); err != nil {
		log.Fatalf("❌😵‍💫 inventory server stopped with error: %v", err)
	}
}
