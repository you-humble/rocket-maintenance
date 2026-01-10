package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	invclient "github.com/you-humble/rocket-maintenance/order/internal/client/grpc/inventory/v1"
	pmtclient "github.com/you-humble/rocket-maintenance/order/internal/client/grpc/payment/v1"
	"github.com/you-humble/rocket-maintenance/order/internal/migrator"
	repository "github.com/you-humble/rocket-maintenance/order/internal/repository/order"
	service "github.com/you-humble/rocket-maintenance/order/internal/service/order"
	thttp "github.com/you-humble/rocket-maintenance/order/internal/transport/http/order/v1"
	orderv1 "github.com/you-humble/rocket-maintenance/shared/pkg/openapi/order/v1"
	inventorypbv1 "github.com/you-humble/rocket-maintenance/shared/pkg/proto/inventory/v1"
	paymentpbv1 "github.com/you-humble/rocket-maintenance/shared/pkg/proto/payment/v1"
)

const (
	httpAddr          = "0.0.0.0:8080"
	inventoryGRPCAddr = "inventory:50051"
	paymentGRPCAddr   = "payment:50052"
	readHeaderTimeout = 5 * time.Second
	shutdownTimeout   = 10 * time.Second
	pgDsn             = "postgres://order-service-user:order-service-password@postgres-order:5432/order-service?sslmode=disable"
	migrationsDir     = "./migrations"
)

func main() {
	// Inventory
	invConn, err := grpc.NewClient(
		inventoryGRPCAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Printf("failed to connect to inventory service %s: %v\n", inventoryGRPCAddr, err)
		return
	}
	defer func() {
		if cerr := invConn.Close(); cerr != nil {
			log.Printf("failed to close inventory service connect: %v", cerr)
		}
	}()

	grpcInventoryClient := inventorypbv1.NewInventoryServiceClient(invConn)
	inventoryClient := invclient.NewClient(grpcInventoryClient)

	// Payment
	payConn, err := grpc.NewClient(
		paymentGRPCAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Printf("failed to connect to payment service %s: %v\n", paymentGRPCAddr, err)
		return
	}
	defer func() {
		if cerr := payConn.Close(); cerr != nil {
			log.Printf("failed to close payment service connect: %v", cerr)
		}
	}()

	grpcPaymentClient := paymentpbv1.NewPaymentServiceClient(payConn)
	paymentClient := pmtclient.NewClient(grpcPaymentClient)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// DB pool
	pool, err := pgxpool.New(ctx, pgDsn)
	if err != nil {
		log.Printf("failed to create pg pool: %v\n", err)
		return
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Printf("failed to ping db: %v\n", err)
		return
	}

	// Migrations
	m := migrator.NewMigrator(stdlib.OpenDBFromPool(pool), migrationsDir)
	defer func() {
		if cerr := m.Close(); cerr != nil {
			log.Printf("failed to close migrator db: %v\n", cerr)
		}
	}()

	defer func() {
		if dberr := m.Close(); err != nil {
			log.Printf("failed to close migrator db connect: %v", dberr)
		}
	}()

	if err := m.Up(); err != nil {
		log.Printf("failed to apply migrations: %v\n", err)
		return
	}

	repo := repository.NewOrderRepository(pool)
	service := service.NewOrderService(repo, inventoryClient, paymentClient)

	handler := thttp.NewOrderHandler(service)

	orderServer, err := orderv1.NewServer(handler)
	if err != nil {
		log.Printf("failed to create a new server: %v\n", err)
		return
	}

	r := chi.NewRouter()

	r.Use(
		middleware.Recoverer,
		middleware.Logger,
	)

	r.Mount("/", orderServer)

	server := &http.Server{
		Addr:              httpAddr,
		Handler:           r,
		ReadHeaderTimeout: readHeaderTimeout,
	}

	go func() {
		log.Printf("🚀 order server listening on %s", httpAddr)
		err := server.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("❌ order server error: %v\n", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("🛑 Server shutdown...")

	sdCtx, sdCancel := context.WithTimeout(ctx, shutdownTimeout)
	defer sdCancel()

	if err := server.Shutdown(sdCtx); err != nil {
		log.Printf("❌ Error during server shutdown: %v\n", err)
		log.Println("❌😵‍💫 Server stopped")
		return
	}

	log.Println("✅ Server stopped")
}
