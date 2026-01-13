package interceptors

import (
	"context"
	"path"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/status"

	"github.com/you-humble/rocket-maintenance/platform/logger"
)

func UnaryLogging() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		method := path.Base(info.FullMethod)
		start := time.Now()

		resp, err := handler(ctx, req)

		log := logger.With(logger.String("method", method))

		d := time.Since(start)
		if err != nil {
			st, _ := status.FromError(err)
			log.Error(ctx, "grpc",
				logger.String("code", st.Code().String()),
				logger.Duration("dur", d),
				logger.ErrorF(err),
			)
			return resp, err
		}

		log.Info(ctx, "grpc",
			logger.String("code", "OK"),
			logger.Duration("dur", d),
		)
		return resp, nil
	}
}
