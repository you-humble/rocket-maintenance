package service

import (
	"context"
	"fmt"
	"time"

	"github.com/you-humble/rocket-maintenance/assembly/internal/model"
	"github.com/you-humble/rocket-maintenance/platform/kafka"
	"github.com/you-humble/rocket-maintenance/platform/logger"
)

const assemblyDelay = 10 * time.Second

type KafkaConverter interface {
	PaidOrderToModel([]byte) (model.PaidOrder, error)
	AssembledShipToPayload(model.AssembledShip) ([]byte, error)
}

type service struct {
	consumer kafka.Consumer
	producer kafka.Producer
	conv     KafkaConverter
	delay    time.Duration
	newTimer func(time.Duration) *time.Timer
}

func NewAssemblyService(
	consumer kafka.Consumer,
	producer kafka.Producer,
	conv KafkaConverter,
) *service {
	return &service{
		consumer: consumer,
		producer: producer,
		conv:     conv,
		delay:    assemblyDelay,
		newTimer: time.NewTimer,
	}
}

func (s *service) RunOrderPaidConsume(ctx context.Context) error {
	logger.Info(ctx, "Starting paid order consumer")

	if err := s.consumer.Consume(ctx, s.paidOrderHandler); err != nil {
		logger.Error(ctx, "Consume from order.paid topic error", logger.ErrorF(err))
		return err
	}

	return nil
}

func (s *service) paidOrderHandler(ctx context.Context, msg kafka.Message) error {
	event, err := s.conv.PaidOrderToModel(msg.Value)
	if err != nil {
		logger.Error(ctx, "Failed to decode PaidOrderRecord", logger.ErrorF(err))
		return fmt.Errorf("converter paid_order_to_model error: %w", err)
	}

	timer := s.newTimer(s.delay)
	defer timer.Stop()

	start := time.Now()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
	}

	logger.Info(ctx, "Processing message",
		logger.String("topic", msg.Topic),
		logger.Any("partition", msg.Partition),
		logger.Any("offset", msg.Offset),
		logger.String("event_uuid", event.EventID.String()),
		logger.String("order_uuid", event.OrderID.String()),
		logger.String("user_uuid", event.UserID.String()),
		logger.String("payment_method", event.PaymentMethod),
		logger.String("transaction_uuid", event.TransactionID.String()),
	)

	if err := s.sendAssembledShip(ctx, event, time.Since(start)); err != nil {
		logger.Error(ctx, "Failed to send AssembledShipRecord", logger.ErrorF(err))
		return err
	}
	return nil
}

func (s *service) sendAssembledShip(
	ctx context.Context,
	event model.PaidOrder,
	buildTime time.Duration,
) error {
	payload, err := s.conv.AssembledShipToPayload(model.AssembledShip{
		EventID:   event.EventID,
		OrderID:   event.OrderID,
		UserID:    event.UserID,
		BuildTime: buildTime,
	})
	if err != nil {
		return fmt.Errorf("converter assembled_ship_to_proto error: %w", err)
	}

	if err := s.producer.Send(ctx, event.OrderID[:], payload); err != nil {
		return fmt.Errorf("produce to order.assembled topic error: %w", err)
	}
	return nil
}
