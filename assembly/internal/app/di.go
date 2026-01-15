package app

import (
	"context"
	"fmt"

	"github.com/IBM/sarama"

	"github.com/you-humble/rocket-maintenance/assembly/internal/config"
	"github.com/you-humble/rocket-maintenance/assembly/internal/converter"
	service "github.com/you-humble/rocket-maintenance/assembly/internal/service/assembly"
	"github.com/you-humble/rocket-maintenance/platform/closer"
	"github.com/you-humble/rocket-maintenance/platform/kafka"
	"github.com/you-humble/rocket-maintenance/platform/kafka/consumer"
	"github.com/you-humble/rocket-maintenance/platform/kafka/middleware"
	"github.com/you-humble/rocket-maintenance/platform/kafka/producer"
	"github.com/you-humble/rocket-maintenance/platform/logger"
)

type AssemblyService interface {
	Run(ctx context.Context) error
}

type di struct {
	consumerGroup     sarama.ConsumerGroup
	orderPaidConsumer kafka.Consumer

	syncProducer           sarama.SyncProducer
	orderAseembledProducer kafka.Producer

	conv service.KafkaConverter

	service AssemblyService
}

func NewDI() *di { return &di{} }

func (d *di) ConsumerGroup(ctx context.Context) sarama.ConsumerGroup {
	if d.consumerGroup == nil {
		cfg := config.C()

		consumerGroup, err := sarama.NewConsumerGroup(
			cfg.Kafka.Brokers(),
			cfg.Kafka.ConsumerGroupID(),
			cfg.Kafka.OrderPaidConsumerConfig(),
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

func (d *di) OrderPaidConsumer(ctx context.Context) kafka.Consumer {
	if d.orderPaidConsumer == nil {
		d.orderPaidConsumer = consumer.NewConsumer(
			d.ConsumerGroup(ctx),
			[]string{
				config.C().Kafka.OrderPaidTopic(),
			},
			logger.L(),
			middleware.Logging(logger.L()),
		)
	}

	return d.orderPaidConsumer
}

func (d *di) SyncProducer(ctx context.Context) sarama.SyncProducer {
	if d.syncProducer == nil {
		cfg := config.C()

		p, err := sarama.NewSyncProducer(
			cfg.Kafka.Brokers(),
			cfg.Kafka.OrderAssembledProducerConfig(),
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

func (d *di) OrderAssembledProducer(ctx context.Context) kafka.Producer {
	if d.orderAseembledProducer == nil {
		d.orderAseembledProducer = producer.NewProducer(
			d.SyncProducer(ctx),
			config.C().Kafka.OrderAssembledTopic(),
			logger.L(),
		)
	}

	return d.orderAseembledProducer
}

func (d *di) KafkaConverter(ctx context.Context) service.KafkaConverter {
	if d.conv == nil {
		d.conv = converter.NewKafkaCoverter()
	}

	return d.conv
}

func (d *di) AssemblyService(ctx context.Context) AssemblyService {
	if d.consumerGroup == nil {
		d.service = service.NewAssemblyService(
			d.OrderPaidConsumer(ctx),
			d.OrderAssembledProducer(ctx),
			d.KafkaConverter(ctx),
		)
	}

	return d.service
}
