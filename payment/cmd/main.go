package main

import (
	"context"
	"errors"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

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

	s := grpc.NewServer()

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
