package app

import (
	"context"
	"fmt"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	invclient "github.com/you-humble/rocket-maintenance/order/internal/client/grpc/inventory/v1"
	pmtclient "github.com/you-humble/rocket-maintenance/order/internal/client/grpc/payment/v1"
	"github.com/you-humble/rocket-maintenance/order/internal/config"
	repository "github.com/you-humble/rocket-maintenance/order/internal/repository/order"
	service "github.com/you-humble/rocket-maintenance/order/internal/service/order"
	thttp "github.com/you-humble/rocket-maintenance/order/internal/transport/http/order/v1"
	"github.com/you-humble/rocket-maintenance/platform/closer"
	"github.com/you-humble/rocket-maintenance/platform/db/migrator"
	orderv1 "github.com/you-humble/rocket-maintenance/shared/pkg/openapi/order/v1"
	inventorypbv1 "github.com/you-humble/rocket-maintenance/shared/pkg/proto/inventory/v1"
	paymentpbv1 "github.com/you-humble/rocket-maintenance/shared/pkg/proto/payment/v1"
)

type di struct {
	inventoryClient service.InventoryClient
	paymentClient   service.PaymentClient

	dbPool     *pgxpool.Pool
	migrator   *migrator.Migrator
	repository service.OrderRepository

	service thttp.OrderService
	handler orderv1.Handler

	router *chi.Mux
}

func NewDI() *di { return &di{} }

func (d *di) InventoryClient(ctx context.Context) service.InventoryClient {
	if d.inventoryClient == nil {
		cfg := config.C()

		invConn, err := grpc.NewClient(
			cfg.Inventory.Address(),
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
		if err != nil {
			panic(fmt.Sprintf("failed to connect to inventory service %s: %v",
				cfg.Inventory.Address(), err),
			)
		}

		closer.AddNamed("Inventory Service",
			func(ctx context.Context) error {
				return invConn.Close()
			})

		grpcInventoryClient := inventorypbv1.NewInventoryServiceClient(invConn)
		d.inventoryClient = invclient.NewClient(grpcInventoryClient)
	}

	return d.inventoryClient
}

func (d *di) PaymentClient(ctx context.Context) service.PaymentClient {
	if d.paymentClient == nil {
		cfg := config.C()

		payConn, err := grpc.NewClient(
			cfg.Payment.Address(),
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
		if err != nil {
			panic(fmt.Sprintf("failed to connect to payment service %s: %v",
				cfg.Payment.Address(), err),
			)
		}

		closer.AddNamed("Payment Service",
			func(ctx context.Context) error {
				return payConn.Close()
			})

		grpcPaymentClient := paymentpbv1.NewPaymentServiceClient(payConn)
		d.paymentClient = pmtclient.NewClient(grpcPaymentClient)
	}

	return d.paymentClient
}

func (d *di) DBPool(ctx context.Context) *pgxpool.Pool {
	if d.dbPool == nil {

		pool, err := pgxpool.New(ctx, config.C().Postgres.DSN())
		if err != nil {
			panic(fmt.Sprintf("failed to create pg pool: %v\n", err))
		}

		closer.AddNamed("PGX Pool",
			func(ctx context.Context) error {
				pool.Close()
				return nil
			})

		if err := pool.Ping(ctx); err != nil {
			panic(fmt.Sprintf("failed to ping db: %v\n", err))
		}

		d.dbPool = pool
	}

	return d.dbPool
}

func (d *di) Migrator(ctx context.Context) *migrator.Migrator {
	if d.migrator == nil {
		d.migrator = migrator.NewMigrator(
			stdlib.OpenDBFromPool(d.DBPool(ctx)),
			config.C().Postgres.MigrationDirectory(),
		)

		closer.AddNamed("Migrator",
			func(ctx context.Context) error {
				return d.migrator.Close()
			})
	}

	return d.migrator
}

func (d *di) OrderRepository(ctx context.Context) service.OrderRepository {
	if d.repository == nil {
		d.repository = repository.NewOrderRepository(d.DBPool(ctx))
	}

	return d.repository
}

func (d *di) OrderService(ctx context.Context) thttp.OrderService {
	if d.service == nil {
		d.service = service.NewOrderService(
			d.OrderRepository(ctx),
			d.InventoryClient(ctx),
			d.PaymentClient(ctx),
			config.C().Server.BDEReadTimeout(),
			config.C().Server.DBWriteTimeout(),
		)
	}

	return d.service
}

func (d *di) OrderHandler(ctx context.Context) orderv1.Handler {
	if d.handler == nil {
		d.handler = thttp.NewOrderHandler(d.OrderService(ctx))
	}

	return d.handler
}

func (d *di) Router(_ context.Context) *chi.Mux {
	if d.router == nil {
		d.router = chi.NewRouter()

	}

	return d.router
}
