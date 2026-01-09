package service

import (
	"context"
	"log"

	"github.com/google/uuid"

	"github.com/you-humble/rocket-maintenance/payment/internal/model"
)

type service struct{}

func NewPaymentService() *service {
	return &service{}
}

func (s *service) PayOrder(ctx context.Context, params model.PayOrderParams) (*model.PayOrderResult, error) {
	if err := params.Validate(); err != nil {
		return nil, err
	}

	txID := uuid.NewString()

	log.Printf(
		"payment succeeded, transaction_uuid=%s order_uuid=%s user_uuid=%s method=%d",
		txID, params.OrderID, params.UserID, params.Method,
	)

	return &model.PayOrderResult{TransactionUUID: txID}, nil
}
