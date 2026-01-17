package app

import (
	"context"
	"fmt"

	"github.com/IBM/sarama"
	"github.com/go-telegram/bot"

	tgclient "github.com/you-humble/rocket-maintenance/notification/internal/client/http/telegram"
	"github.com/you-humble/rocket-maintenance/notification/internal/config"
	converter "github.com/you-humble/rocket-maintenance/notification/internal/converter/kafka"
	oaconsumer "github.com/you-humble/rocket-maintenance/notification/internal/service/consumer/order_assembled"
	opconsumer "github.com/you-humble/rocket-maintenance/notification/internal/service/consumer/order_paid"
	service "github.com/you-humble/rocket-maintenance/notification/internal/service/telegram"
	"github.com/you-humble/rocket-maintenance/platform/closer"
	"github.com/you-humble/rocket-maintenance/platform/kafka"
	"github.com/you-humble/rocket-maintenance/platform/kafka/consumer"
	"github.com/you-humble/rocket-maintenance/platform/kafka/middleware"
	"github.com/you-humble/rocket-maintenance/platform/logger"
)

type TelegramService interface {
	oaconsumer.ShipAssembledNotifier
	opconsumer.OrderPaidNotifier
	AddChatID(ctx context.Context, chatID int64)
}

type OrderPaidConsumer interface {
	RunOrderPaidConsume(ctx context.Context) error
}

type OrderAssembledConsumer interface {
	RunOrderAssembledConsume(ctx context.Context) error
}

type Converter interface {
	opconsumer.PaidOrderConverter
	oaconsumer.AssembledShipConverter
}

type di struct {
	converter Converter

	orderPaidConsumerGroup sarama.ConsumerGroup
	orderPaidKafkaConsumer kafka.Consumer
	orderPaidConsumer      OrderPaidConsumer

	orderAseembledConsumerGroup sarama.ConsumerGroup
	orderAseembledKafkaConsumer kafka.Consumer
	orderAseembledConsumer      OrderAssembledConsumer

	tgBot     *bot.Bot
	tgClient  service.MessageSender
	tgService TelegramService
}

func NewDI() *di { return &di{} }

func (d *di) KafkaConverter(ctx context.Context) Converter {
	if d.converter == nil {
		d.converter = converter.NewKafkaCoverter()
	}

	return d.converter
}

func (d *di) OrderPaidConsumerGroup(ctx context.Context) sarama.ConsumerGroup {
	if d.orderPaidConsumerGroup == nil {
		cfg := config.C()

		consumerGroup, err := sarama.NewConsumerGroup(
			cfg.Kafka.Brokers(),
			cfg.Kafka.OrderPaidConsumerGroupID(),
			cfg.Kafka.OrderPaidConsumerConfig(),
		)
		if err != nil {
			panic(fmt.Sprintf("failed to create order.paid consumer group: %s\n", err.Error()))
		}
		closer.AddNamed("Kafka order.paid consumer group", func(ctx context.Context) error {
			return consumerGroup.Close()
		})

		d.orderPaidConsumerGroup = consumerGroup
	}

	return d.orderPaidConsumerGroup
}

func (d *di) OrderPaidKafkaConsumer(ctx context.Context) kafka.Consumer {
	if d.orderPaidKafkaConsumer == nil {
		d.orderPaidKafkaConsumer = consumer.NewConsumer(
			d.OrderPaidConsumerGroup(ctx),
			[]string{
				config.C().Kafka.OrderPaidTopic(),
			},
			logger.L(),
			middleware.Recovery(logger.L()),
			middleware.Logging(logger.L()),
		)
	}

	return d.orderPaidKafkaConsumer
}

func (d *di) OrderPaidConsumer(ctx context.Context) OrderPaidConsumer {
	if d.orderPaidConsumer == nil {
		d.orderPaidConsumer = opconsumer.NewOrderPaidConsumer(
			d.OrderPaidKafkaConsumer(ctx),
			d.KafkaConverter(ctx),
			d.TelegramService(ctx),
		)
	}

	return d.orderPaidConsumer
}

func (d *di) OrderAssembledConsumerGroup(ctx context.Context) sarama.ConsumerGroup {
	if d.orderAseembledConsumerGroup == nil {
		cfg := config.C()

		consumerGroup, err := sarama.NewConsumerGroup(
			cfg.Kafka.Brokers(),
			cfg.Kafka.OrderAssembledConsumerGroupID(),
			cfg.Kafka.OrderAssembledConsumerConfig(),
		)
		if err != nil {
			panic(fmt.Sprintf("failed to create order.assembled consumer group: %s\n", err.Error()))
		}
		closer.AddNamed("Kafka order.assembled consumer group", func(ctx context.Context) error {
			return consumerGroup.Close()
		})

		d.orderAseembledConsumerGroup = consumerGroup
	}

	return d.orderAseembledConsumerGroup
}

func (d *di) OrderAssembledKafkaConsumer(ctx context.Context) kafka.Consumer {
	if d.orderAseembledKafkaConsumer == nil {
		d.orderAseembledKafkaConsumer = consumer.NewConsumer(
			d.OrderAssembledConsumerGroup(ctx),
			[]string{
				config.C().Kafka.OrderAssembledTopic(),
			},
			logger.L(),
			middleware.Recovery(logger.L()),
			middleware.Logging(logger.L()),
		)
	}

	return d.orderAseembledKafkaConsumer
}

func (d *di) OrderAssembledConsumer(ctx context.Context) OrderAssembledConsumer {
	if d.orderAseembledConsumer == nil {
		d.orderAseembledConsumer = oaconsumer.NewOrderAssembledConsumer(
			d.OrderAssembledKafkaConsumer(ctx),
			d.KafkaConverter(ctx),
			d.TelegramService(ctx),
		)
	}

	return d.orderAseembledConsumer
}

func (d *di) TelegramBot(ctx context.Context) *bot.Bot {
	if d.tgBot == nil {
		b, err := bot.New(config.C().Telegram.BotToken())
		if err != nil {
			panic(fmt.Sprintf("failed to create telegram bot: %s\n", err.Error()))
		}
		closer.AddNamed("Telegram Bot", func(ctx context.Context) error {
			_, err := b.Close(ctx)
			return err
		})

		d.tgBot = b
	}

	return d.tgBot
}

func (d *di) TelegramClient(ctx context.Context) service.MessageSender {
	if d.tgClient == nil {
		d.tgClient = tgclient.NewClient(d.TelegramBot(ctx))
	}

	return d.tgClient
}

func (d *di) TelegramService(ctx context.Context) TelegramService {
	if d.tgService == nil {
		d.tgService = service.NewTgService(
			d.TelegramClient(ctx),
		)
	}

	return d.tgService
}
