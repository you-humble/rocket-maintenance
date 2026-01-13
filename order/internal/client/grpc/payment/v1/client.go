package pmtclient

import (
	"context"

	"github.com/you-humble/rocket-maintenance/order/internal/client/converter"
	"github.com/you-humble/rocket-maintenance/order/internal/model"
	paymentpbv1 "github.com/you-humble/rocket-maintenance/shared/pkg/proto/payment/v1"
)

type client struct {
	grpc paymentpbv1.PaymentServiceClient
}

func NewClient(grpc paymentpbv1.PaymentServiceClient) *client {
	return &client{grpc: grpc}
}

func (c *client) PayOrder(ctx context.Context, params model.PayOrderParams) (string, error) {
	paid, err := c.grpc.PayOrder(ctx, converter.PayOrderParamsToPB(params))
	if err != nil {
		return "", err
	}

	return paid.TransactionUuid, nil
}
