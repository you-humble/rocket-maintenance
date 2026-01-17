package middleware

import (
	"context"

	"go.uber.org/zap"

	"github.com/you-humble/rocket-maintenance/platform/kafka"
)

type InfoLogger interface {
	Info(ctx context.Context, msg string, fields ...zap.Field)
}

func Logging(logger InfoLogger) kafka.Middleware {
	return func(next kafka.MessageHandler) kafka.MessageHandler {
		return func(ctx context.Context, msg kafka.Message) error {
			logger.Info(ctx, "Kafka msg received", zap.String("topic", msg.Topic))
			return next(ctx, msg)
		}
	}
}
