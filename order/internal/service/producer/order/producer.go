package ordproducer

import (
	"context"
	"fmt"

	"github.com/you-humble/rocket-maintenance/order/internal/model"
	"github.com/you-humble/rocket-maintenance/platform/kafka"
)

type Converter interface {
	PaidOrderToModel(m model.PaidOrder) ([]byte, error)
}

type service struct {
	producer kafka.Producer
	conv     Converter
}

func NewOrderProducer(producer kafka.Producer, conv Converter) *service {
	return &service{producer: producer, conv: conv}
}

func (s *service) SendOrderPaid(ctx context.Context, event model.PaidOrder) error {
	payload, err := s.conv.PaidOrderToModel(event)
	if err != nil {
		return fmt.Errorf("converter paid_order_to_proto error: %w", err)
	}

	if err := s.producer.Send(ctx, event.OrderID[:], payload); err != nil {
		return fmt.Errorf("producer to order.paid topic error: %w", err)
	}

	return nil
}
