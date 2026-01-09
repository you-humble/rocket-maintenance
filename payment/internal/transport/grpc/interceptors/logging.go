package interceptors

import (
	"context"
	"log"
	"path"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

func UnaryLogging() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		method := path.Base(info.FullMethod)
		start := time.Now()

		resp, err := handler(ctx, req)

		d := time.Since(start)
		if err != nil {
			st, _ := status.FromError(err)
			log.Printf("grpc: method=%s code=%s dur=%s err=%v", method, st.Code(), d, err)
			return resp, err
		}

		log.Printf("grpc: method=%s code=OK dur=%s", method, d)
		return resp, nil
	}
}
