package opconsumer

import (
	"context"
	"fmt"

	"github.com/you-humble/rocket-maintenance/notification/internal/model"
	"github.com/you-humble/rocket-maintenance/platform/kafka"
	"github.com/you-humble/rocket-maintenance/platform/logger"
)

type PaidOrderConverter interface {
	PaidOrderToModel(data []byte) (model.PaidOrder, error)
}

type OrderPaidNotifier interface {
	NotifyPaidOrder(ctx context.Context, event model.PaidOrder) error
}

type ordPaidConsumer struct {
	consumer kafka.Consumer
	conv     PaidOrderConverter
	svc      OrderPaidNotifier
}

func NewOrderPaidConsumer(
	consumer kafka.Consumer,
	conv PaidOrderConverter,
	svc OrderPaidNotifier,
) *ordPaidConsumer {
	return &ordPaidConsumer{
		consumer: consumer,
		conv:     conv,
		svc:      svc,
	}
}

func (s *ordPaidConsumer) RunOrderPaidConsume(ctx context.Context) error {
	logger.Info(ctx, "Starting order paid consumer")

	if err := s.consumer.Consume(ctx, s.orderAssembledHandler); err != nil {
		logger.Error(ctx, "Consume from order.paid topic error", logger.ErrorF(err))
		return err
	}

	return nil
}

func (s *ordPaidConsumer) orderAssembledHandler(ctx context.Context, msg kafka.Message) error {
	event, err := s.conv.PaidOrderToModel(msg.Value)
	if err != nil {
		logger.Error(ctx, "Failed to decode PaidOrderRecord", logger.ErrorF(err))
		return fmt.Errorf("converter paid_order_to_model error: %w", err)
	}

	if err := s.svc.NotifyPaidOrder(ctx, event); err != nil {
		logger.Error(ctx, "Failed to notify about OrderPaid", logger.ErrorF(err))
		return err
	}

	return nil
}
