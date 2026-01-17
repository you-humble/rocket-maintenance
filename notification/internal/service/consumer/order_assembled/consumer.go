package oaconsumer

import (
	"context"
	"fmt"

	"github.com/you-humble/rocket-maintenance/notification/internal/model"
	"github.com/you-humble/rocket-maintenance/platform/kafka"
	"github.com/you-humble/rocket-maintenance/platform/logger"
)

type AssembledShipConverter interface {
	AssembledShipToModel(data []byte) (model.AssembledShip, error)
}

type ShipAssembledNotifier interface {
	NotifyShipAssembled(ctx context.Context, event model.AssembledShip) error
}

type ordAssembledConsumer struct {
	consumer kafka.Consumer
	conv     AssembledShipConverter
	svc      ShipAssembledNotifier
}

func NewOrderAssembledConsumer(
	consumer kafka.Consumer,
	conv AssembledShipConverter,
	svc ShipAssembledNotifier,
) *ordAssembledConsumer {
	return &ordAssembledConsumer{
		consumer: consumer,
		conv:     conv,
		svc:      svc,
	}
}

func (s *ordAssembledConsumer) RunOrderAssembledConsume(ctx context.Context) error {
	logger.Info(ctx, "Starting order assembled consumer")

	if err := s.consumer.Consume(ctx, s.orderAssembledHandler); err != nil {
		logger.Error(ctx, "Consume from order.assembled topic error", logger.ErrorF(err))
		return err
	}

	return nil
}

func (s *ordAssembledConsumer) orderAssembledHandler(ctx context.Context, msg kafka.Message) error {
	event, err := s.conv.AssembledShipToModel(msg.Value)
	if err != nil {
		logger.Error(ctx, "Failed to decode AssembledShipRecord", logger.ErrorF(err))
		return fmt.Errorf("converter assembled_ship_to_model error: %w", err)
	}

	if err := s.svc.NotifyShipAssembled(ctx, event); err != nil {
		logger.Error(ctx, "Failed to notify about AssembledShip", logger.ErrorF(err))
		return err
	}

	return nil
}
