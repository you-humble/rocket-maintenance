package main

import (
	"context"
	"os/signal"
	"syscall"

	"github.com/you-humble/rocket-maintenance/inventory/internal/app"
	"github.com/you-humble/rocket-maintenance/platform/logger"
)

const (
	GRPCAddr = "0.0.0.0:50051"
	MongoDSN = "mongodb://inventory-service-user:inventory-service-password@mongo-inventory:27017/inventory-service?authSource=admin"
)

func main() {
	ctx, quit := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT, syscall.SIGTERM,
	)
	defer quit()

	a, err := app.New(ctx)
	if err != nil {
		logger.Error(ctx,
			"❌ Failed to create an application",
			logger.ErrorF(err),
		)
		return
	}

	if err := a.Run(ctx); err != nil {
		logger.Error(ctx, "❌ Inventory server error", logger.ErrorF(err))
	}
}
