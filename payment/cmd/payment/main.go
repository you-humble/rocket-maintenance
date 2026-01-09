package main

import (
	"context"
	"log"

	"github.com/you-humble/rocket-maintenance/payment/internal/app"
)

const GRPCAddr = "0.0.0.0:50052"

func main() {
	ctx := context.Background()

	cfg := app.Config{GRPCAddr: GRPCAddr}

	if err := app.Run(ctx, cfg); err != nil {
		log.Fatalf("❌😵‍💫 payment server stopped with error: %v", err)
	}
}
