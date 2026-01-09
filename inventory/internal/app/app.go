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

	repository "github.com/you-humble/rocket-maintenance/inventory/internal/repository/part"
	service "github.com/you-humble/rocket-maintenance/inventory/internal/service/part"
	"github.com/you-humble/rocket-maintenance/inventory/internal/transport/grpc/interceptors"
	transport "github.com/you-humble/rocket-maintenance/inventory/internal/transport/grpc/inventory/v1"
	inventorypbv1 "github.com/you-humble/rocket-maintenance/shared/pkg/proto/inventory/v1"
)

type Config struct {
	GRPCAddr string
}

func Run(ctx context.Context, cfg Config) error {
	const op string = "inventory"

	lis, err := net.Listen("tcp", cfg.GRPCAddr)
	if err != nil {
		log.Printf("%s: failed to listen: %v\n", op, err)
		return err
	}
	defer func() {
		if cerr := lis.Close(); cerr != nil {
			log.Printf("%s: failed to close listener: %v\n", op, cerr)
		}
	}()

	repo := repository.NewPartRepository()
	svc := service.NewInventoryService(repo)
	handler := transport.NewInventoryHandler(svc)

	s := grpc.NewServer(
		grpc.UnaryInterceptor(interceptors.UnaryLogging()),
	)
	inventorypbv1.RegisterInventoryServiceServer(s, handler)

	reflection.Register(s)

	errCh := make(chan error, 1)
	go func() {
		log.Printf("🚀 gRPC server listening on %s\n", cfg.GRPCAddr)
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
