package ordconsumer

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/you-humble/rocket-maintenance/order/internal/model"
	"github.com/you-humble/rocket-maintenance/platform/kafka"
	"github.com/you-humble/rocket-maintenance/platform/logger"
)

type Converter interface {
	AssembledShipToModel(data []byte) (model.AssembledShip, error)
}

type Service interface {
	Complete(ctx context.Context, ordID uuid.UUID) error
}

type service struct {
	consumer kafka.Consumer
	conv     Converter
	svc      Service
}

func NewOrderConsumer(
	consumer kafka.Consumer,
	conv Converter,
	svc Service,
) *service {
	return &service{consumer: consumer, conv: conv, svc: svc}
}

func (s *service) RunShipAssembledConsume(ctx context.Context) error {
	logger.Info(ctx, "Starting ship assembled consumer")

	if err := s.consumer.Consume(ctx, s.shipAssembledHandler); err != nil {
		logger.Error(ctx, "Consume from order.assembled topic error", logger.ErrorF(err))
		return err
	}

	return nil
}

func (s *service) shipAssembledHandler(ctx context.Context, msg kafka.Message) error {
	payload, err := s.conv.AssembledShipToModel(msg.Value)
	if err != nil {
		logger.Error(ctx, "Failed to decode AssembledShipRecord", logger.ErrorF(err))
		return fmt.Errorf("converter assembled_ship_to_model error: %w", err)
	}

	if err := s.svc.Complete(ctx, payload.OrderID); err != nil {
		logger.Error(ctx, "consumer.CompleteOrder", logger.ErrorF(err))
		return err
	}

	return nil
}
