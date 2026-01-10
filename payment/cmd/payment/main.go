package main

import (
	"context"
	"log"

	"github.com/you-humble/rocket-maintenance/payment/internal/app"
	"github.com/you-humble/rocket-maintenance/payment/internal/config"
)

const GRPCAddr = "0.0.0.0:50052"

func main() {
	ctx := context.Background()

	if err := config.Load(); err != nil {
		log.Fatal(err)
	}
	cfg := config.C()

	if err := app.Run(ctx, cfg.Server); err != nil {
		log.Fatalf("❌😵‍💫 payment server stopped with error: %v", err)
	}
}
