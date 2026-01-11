package app

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/you-humble/rocket-maintenance/inventory/internal/config"
	repository "github.com/you-humble/rocket-maintenance/inventory/internal/repository/part"
	service "github.com/you-humble/rocket-maintenance/inventory/internal/service/part"
	"github.com/you-humble/rocket-maintenance/inventory/internal/transport/grpc/interceptors"
	tgrpc "github.com/you-humble/rocket-maintenance/inventory/internal/transport/grpc/inventory/v1"
	"github.com/you-humble/rocket-maintenance/platform/closer"
	"github.com/you-humble/rocket-maintenance/platform/grpc/health"
	inventorypbv1 "github.com/you-humble/rocket-maintenance/shared/pkg/proto/inventory/v1"
)

type PartRepository interface {
	service.PartRepository
	repository.BatchCreator
}

type di struct {
	mongo      *mongo.Client
	collection *mongo.Collection

	repository PartRepository
	service    tgrpc.InventoryService
	handler    inventorypbv1.InventoryServiceServer

	server *grpc.Server
}

func NewDI() *di { return &di{} }

func (d *di) MongoDB(ctx context.Context) *mongo.Client {
	if d.mongo == nil {
		cfg := config.C()

		mongoClient, err := mongo.Connect(
			options.Client().ApplyURI(cfg.Mongo.DSN()),
		)
		if err != nil {
			panic(fmt.Sprintf("failed to create mongodb client: %v\n", err))
		}
		closer.AddNamed("Mongo Client",
			func(ctx context.Context) error {
				return mongoClient.Disconnect(ctx)
			})

		if err := mongoClient.Ping(ctx, readpref.Primary()); err != nil {
			panic(fmt.Sprintf("failed to ping database: %v\n", err))
		}

		d.mongo = mongoClient
	}

	return d.mongo
}

func (d *di) PartsCollection(ctx context.Context) *mongo.Collection {
	if d.collection == nil {
		d.collection = d.MongoDB(ctx).
			Database(config.C().Mongo.DatabaseName()).
			Collection(config.C().Mongo.PartsCollection())

		if err := ensurePartIndexes(ctx, d.collection); err != nil {
			panic(fmt.Sprintf("failed to ensure indexes: %v\n", err))
		}
	}

	return d.collection
}

func (d *di) PartsRepository(ctx context.Context) PartRepository {
	if d.repository == nil {
		d.repository = repository.NewPartRepository(d.PartsCollection(ctx))
	}

	return d.repository
}

func (d *di) InventoryService(ctx context.Context) tgrpc.InventoryService {
	if d.service == nil {
		d.service = service.NewInventoryService(
			d.PartsRepository(ctx),
			config.C().Server.BDEReadTimeout(),
		)
	}

	return d.service
}

func (d *di) InventoryHandler(ctx context.Context) inventorypbv1.InventoryServiceServer {
	if d.handler == nil {
		d.handler = tgrpc.NewInventoryHandler(d.InventoryService(ctx))
	}

	return d.handler
}

func (d *di) Server(ctx context.Context) *grpc.Server {
	if d.server == nil {
		d.server = grpc.NewServer(
			grpc.UnaryInterceptor(interceptors.UnaryLogging()),
		)
		inventorypbv1.RegisterInventoryServiceServer(d.server, d.InventoryHandler(ctx))

		reflection.Register(d.server)

		health.RegisterService(d.server)
	}

	return d.server
}

func ensurePartIndexes(ctx context.Context, coll *mongo.Collection) error {
	_, err := coll.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "name", Value: 1}}},
		{Keys: bson.D{{Key: "category", Value: 1}}},
		{Keys: bson.D{{Key: "manufacturer.country_norm", Value: 1}}},
		{Keys: bson.D{{Key: "tags", Value: 1}}},
	}, options.CreateIndexes())

	return err
}
