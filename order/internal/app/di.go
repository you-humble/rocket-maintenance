package app

import (
	"context"
	"fmt"

	"github.com/IBM/sarama"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	invclient "github.com/you-humble/rocket-maintenance/order/internal/client/grpc/inventory/v1"
	pmtclient "github.com/you-humble/rocket-maintenance/order/internal/client/grpc/payment/v1"
	"github.com/you-humble/rocket-maintenance/order/internal/config"
	"github.com/you-humble/rocket-maintenance/order/internal/converter"
	"github.com/you-humble/rocket-maintenance/order/internal/model"
	repository "github.com/you-humble/rocket-maintenance/order/internal/repository/order"
	ordconsumer "github.com/you-humble/rocket-maintenance/order/internal/service/consumer/order"
	service "github.com/you-humble/rocket-maintenance/order/internal/service/order"
	ordproducer "github.com/you-humble/rocket-maintenance/order/internal/service/producer/order"
	thttp "github.com/you-humble/rocket-maintenance/order/internal/transport/http/order/v1"
	"github.com/you-humble/rocket-maintenance/platform/closer"
	"github.com/you-humble/rocket-maintenance/platform/db/migrator"
	"github.com/you-humble/rocket-maintenance/platform/kafka"
	"github.com/you-humble/rocket-maintenance/platform/kafka/consumer"
	"github.com/you-humble/rocket-maintenance/platform/kafka/middleware"
	"github.com/you-humble/rocket-maintenance/platform/kafka/producer"
	"github.com/you-humble/rocket-maintenance/platform/logger"
	orderv1 "github.com/you-humble/rocket-maintenance/shared/pkg/openapi/order/v1"
	inventorypbv1 "github.com/you-humble/rocket-maintenance/shared/pkg/proto/inventory/v1"
	paymentpbv1 "github.com/you-humble/rocket-maintenance/shared/pkg/proto/payment/v1"
)

type Converter interface {
	AssembledShipToModel(data []byte) (model.AssembledShip, error)
	PaidOrderToModel(m model.PaidOrder) ([]byte, error)
}

type OrderConsumer interface {
	RunShipAssembledConsume(ctx context.Context) error
}

type OrderService interface {
	thttp.OrderService
	ordconsumer.Service
}

type di struct {
	inventoryClient service.InventoryClient
	paymentClient   service.PaymentClient

	dbPool     *pgxpool.Pool
	migrator   *migrator.Migrator
	repository service.OrderRepository

	consumerGroup          sarama.ConsumerGroup
	orderAssembledConsumer kafka.Consumer
	orderConsumer          OrderConsumer

	syncProducer      sarama.SyncProducer
	orderPaidProducer kafka.Producer
	orderProducer     service.OrderPaidSender

	conv Converter

	service OrderService
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

func (d *di) KafkaConverter(ctx context.Context) Converter {
	if d.conv == nil {
		d.conv = converter.NewKafkaCoverter()
	}

	return d.conv
}

func (d *di) ConsumerGroup(ctx context.Context) sarama.ConsumerGroup {
	if d.consumerGroup == nil {
		cfg := config.C()

		consumerGroup, err := sarama.NewConsumerGroup(
			cfg.Kafka.Brokers(),
			cfg.Kafka.ConsumerGroupID(),
			cfg.Kafka.OrderAssembledConsumerConfig(),
		)
		if err != nil {
			panic(fmt.Sprintf("failed to create consumer group: %s\n", err.Error()))
		}
		closer.AddNamed("Kafka consumer group", func(ctx context.Context) error {
			return d.consumerGroup.Close()
		})

		d.consumerGroup = consumerGroup
	}

	return d.consumerGroup
}

func (d *di) OrderAssembledConsumer(ctx context.Context) kafka.Consumer {
	if d.orderAssembledConsumer == nil {
		d.orderAssembledConsumer = consumer.NewConsumer(
			d.ConsumerGroup(ctx),
			[]string{
				config.C().Kafka.OrderAssembledTopic(),
			},
			logger.L(),
			middleware.Recovery(logger.L()),
			middleware.Logging(logger.L()),
		)
	}

	return d.orderAssembledConsumer
}

func (d *di) OrderConsumer(ctx context.Context) OrderConsumer {
	if d.orderConsumer == nil {
		d.orderConsumer = ordconsumer.NewOrderConsumer(
			d.OrderAssembledConsumer(ctx),
			d.KafkaConverter(ctx),
			d.OrderService(ctx),
		)
	}

	return d.orderConsumer
}

func (d *di) SyncProducer(ctx context.Context) sarama.SyncProducer {
	if d.syncProducer == nil {
		cfg := config.C()

		p, err := sarama.NewSyncProducer(
			cfg.Kafka.Brokers(),
			cfg.Kafka.OrderPaidProducerConfig(),
		)
		if err != nil {
			panic(fmt.Sprintf("failed to create sync producer: %s\n", err.Error()))
		}
		closer.AddNamed("Kafka sync producer", func(ctx context.Context) error {
			return p.Close()
		})

		d.syncProducer = p
	}

	return d.syncProducer
}

func (d *di) OrderPaidProducer(ctx context.Context) kafka.Producer {
	if d.orderPaidProducer == nil {
		d.orderPaidProducer = producer.NewProducer(
			d.SyncProducer(ctx),
			config.C().Kafka.OrderPaidTopic(),
			logger.L(),
		)
	}

	return d.orderPaidProducer
}

func (d *di) OrderProducer(ctx context.Context) service.OrderPaidSender {
	if d.orderProducer == nil {
		d.orderProducer = ordproducer.NewOrderProducer(
			d.OrderPaidProducer(ctx),
			d.KafkaConverter(ctx),
		)
	}

	return d.orderProducer
}

func (d *di) OrderService(ctx context.Context) OrderService {
	if d.service == nil {
		d.service = service.NewOrderService(
			d.OrderRepository(ctx),
			d.InventoryClient(ctx),
			d.PaymentClient(ctx),
			d.OrderProducer(ctx),
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
