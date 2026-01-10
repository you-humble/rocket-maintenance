package app

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
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
	MongoDSN string
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

	mongoClient, err := mongo.Connect(
		options.Client().ApplyURI(cfg.MongoDSN),
	)
	if err != nil {
		return err
	}
	defer func() {
		if merr := mongoClient.Disconnect(ctx); merr != nil {
			log.Printf("%s: failed to disconnect mongo client: %v\n", op, merr)
		}
	}()

	if err := mongoClient.Ping(ctx, readpref.Primary()); err != nil {
		log.Printf("failed to ping database: %v\n", err)
		return err
	}

	collection := mongoClient.Database("inventory-service").Collection("parts")
	if err := EnsurePartIndexes(ctx, collection); err != nil {
		return err
	}

	repo := repository.NewPartRepository(collection)

	if err := repository.PartsBootstrap(ctx, repo); err != nil {
		return err
	}

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

func EnsurePartIndexes(ctx context.Context, coll *mongo.Collection) error {
	_, err := coll.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "name", Value: 1}}},
		{Keys: bson.D{{Key: "category", Value: 1}}},
		{Keys: bson.D{{Key: "manufacturer.country_norm", Value: 1}}},
		{Keys: bson.D{{Key: "tags", Value: 1}}},
	}, options.CreateIndexes())

	return err
}
