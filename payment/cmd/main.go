package main

import (
	"context"
	"errors"
	"log"
	"net"
	"os"
	"os/signal"
	"path"
	"syscall"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"

	paymentpbv1 "github.com/you-humble/rocket-maintenance/shared/pkg/proto/payment/v1"
)

const GRPCAddr = "127.0.0.1:50052"

type PaymentServer struct {
	paymentpbv1.UnimplementedPaymentServiceServer
}

func NewPaymentServiceServer() *PaymentServer {
	return &PaymentServer{}
}

func (s *PaymentServer) PayOrder(
	ctx context.Context,
	req *paymentpbv1.PayOrderRequest,
) (*paymentpbv1.PayOrderResponse, error) {
	if err := validatePayOrderRequest(req); err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid order")
	}

	transactionID := uuid.NewString()

	log.Printf(
		"Payment succeeded, transaction_uuid: %s, order_uuid=%s, user_uuid=%s, payment_method=%s",
		transactionID,
		req.GetOrderUuid(),
		req.GetUserUuid(),
		req.GetPaymentMethod().String(),
	)

	return &paymentpbv1.PayOrderResponse{
		TransactionUuid: transactionID,
	}, nil
}

func InterceptorLogger() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (resp any, err error) {
		method := path.Base(info.FullMethod)

		start := time.Now()

		resp, err = handler(ctx, req)

		dur := time.Since(start)

		if err != nil {
			st, _ := status.FromError(err)
			log.Printf(
				"ERROR | finished gRPC method %s with code %s: %v (took: %v)\n",
				method, st.Code(), err, dur,
			)
		} else {
			log.Printf(
				"INFO | finished gRPC method %s succesfully (took: %v)\n",
				method, dur,
			)
		}

		return resp, err
	}
}

func InterceptorValidate() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		if req == nil {
			return nil, errors.New("request is nil")
		}

		payOrdReq, ok := req.(*paymentpbv1.PayOrderRequest)
		if !ok {
			return nil, errors.New("wrong request type")
		}

		if err := validatePayOrderRequest(payOrdReq); err != nil {
			return nil, err
		}

		return handler(ctx, req)
	}
}

func validatePayOrderRequest(req *paymentpbv1.PayOrderRequest) error {
	if req == nil {
		return errors.New("request is nil")
	}
	if req.GetOrderUuid() == "" {
		return errors.New("order_uuid is required")
	}
	if req.GetUserUuid() == "" {
		return errors.New("user_uuid is required")
	}
	if req.GetPaymentMethod() == paymentpbv1.PaymentMethod_PAYMENT_METHOD_UNKNOWN {
		return errors.New("payment_method must not be PAYMENT_METHOD_UNKNOWN")
	}

	return nil
}

func main() {
	const op string = "payment"

	lis, err := net.Listen("tcp", GRPCAddr) //nolint:gosec // bind to all interfaces is OK in this demo
	if err != nil {
		log.Printf("%s: failed to listen: %v\n", op, err)
		return
	}
	defer func() {
		if cerr := lis.Close(); cerr != nil {
			log.Printf("%s: failed to close listener: %v\n", op, cerr)
		}
	}()

	s := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			InterceptorLogger(),
			InterceptorValidate(),
		),
	)

	paymentpbv1.RegisterPaymentServiceServer(s, NewPaymentServiceServer())

	reflection.Register(s)

	errCh := make(chan error, 1)
	go func() {
		log.Printf("🚀 gRPC server listening on %s\n", GRPCAddr)
		if err := s.Serve(lis); err != nil {
			errCh <- err
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-quit:
		log.Printf("🛑 received signal %s, stopping gRPC server", sig)
	case err := <-errCh:
		log.Printf("❌ gRPC server error: %v", err)
	}

	log.Println("🛑 Shutting down gRPC server...")
	s.GracefulStop()
	log.Println("✅ Server stopped")
}
