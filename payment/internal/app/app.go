package app

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/you-humble/rocket-maintenance/payment/internal/config"
	service "github.com/you-humble/rocket-maintenance/payment/internal/service/payment"
	"github.com/you-humble/rocket-maintenance/payment/internal/transport/grpc/interceptors"
	transport "github.com/you-humble/rocket-maintenance/payment/internal/transport/grpc/payment/v1"
	paymentpbv1 "github.com/you-humble/rocket-maintenance/shared/pkg/proto/payment/v1"
)

func Run(ctx context.Context, srvCfg config.Server) error {
	const op string = "payment"

	lis, err := net.Listen("tcp", srvCfg.Address())
	if err != nil {
		log.Printf("%s: failed to listen: %v\n", op, err)
		return err
	}
	defer func() {
		if cerr := lis.Close(); cerr != nil {
			log.Printf("%s: failed to close listener: %v\n", op, cerr)
		}
	}()

	svc := service.NewPaymentService()
	handler := transport.NewPaymentHandler(svc)

	s := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			interceptors.UnaryLogging(),
			interceptors.RejectNilRequest(),
		),
	)
	paymentpbv1.RegisterPaymentServiceServer(s, handler)

	reflection.Register(s)

	errCh := make(chan error, 1)
	go func() {
		log.Printf("🚀 gRPC server listening on %s\n", srvCfg.Address())
		if err := s.Serve(lis); err != nil {
			errCh <- err
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-ctx.Done():
		log.Println("🛑 context cancelled, stopping gRPC server")
	case sig := <-quit:
		log.Printf("🛑 received signal %s, stopping gRPC server", sig)
	case err := <-errCh:
		log.Printf("❌ gRPC server error: %v", err)
	}

	log.Println("🛑 Shutting down gRPC server...")
	s.GracefulStop()
	log.Println("✅ Server stopped")
	return nil
}
