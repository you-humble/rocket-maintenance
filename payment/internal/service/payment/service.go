package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/you-humble/rocket-maintenance/payment/internal/model"
	"github.com/you-humble/rocket-maintenance/platform/logger"
)

type service struct{}

func NewPaymentService() *service {
	return &service{}
}

func (s *service) PayOrder(ctx context.Context, params model.PayOrderParams) (*model.PayOrderResult, error) {
	const op = "payment.service.PayOrder"
	log := logger.With(
		logger.String("order_id", params.OrderID),
		logger.String("user_id", params.UserID),
		logger.String("payment_method", params.Method.String()),
	)

	if err := params.Validate(); err != nil {
		log.Error(ctx, "validation failed", logger.ErrorF(err))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	txID := uuid.NewString()

	log.Info(ctx, "payment succeeded", logger.String("transaction_id", txID))
	return &model.PayOrderResult{TransactionUUID: txID}, nil
}
