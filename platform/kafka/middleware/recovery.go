package middleware

import (
	"context"

	"go.uber.org/zap"

	"github.com/you-humble/rocket-maintenance/platform/kafka"
)

type ErrorLogger interface {
	Error(ctx context.Context, msg string, fields ...zap.Field)
}

func Recovery(logger ErrorLogger) kafka.Middleware {
	return func(next kafka.MessageHandler) kafka.MessageHandler {
		return func(ctx context.Context, msg kafka.Message) error {
			defer func() {
				if r := recover(); r != nil {
					logger.Error(ctx, "Recovered from panic in message processing", zap.Any("error", r))
				}
			}()
			return next(ctx, msg)
		}
	}
}
