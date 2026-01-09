package grpc

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/you-humble/rocket-maintenance/payment/internal/converter"
	"github.com/you-humble/rocket-maintenance/payment/internal/model"
	paymentpbv1 "github.com/you-humble/rocket-maintenance/shared/pkg/proto/payment/v1"
)

type PaymentService interface {
	PayOrder(ctx context.Context, params model.PayOrderParams) (*model.PayOrderResult, error)
}

type handler struct {
	paymentpbv1.UnimplementedPaymentServiceServer
	svc PaymentService
}

func NewPaymentHandler(service PaymentService) *handler {
	return &handler{svc: service}
}

func (h *handler) PayOrder(ctx context.Context, req *paymentpbv1.PayOrderRequest) (*paymentpbv1.PayOrderResponse, error) {
	cmd, err := converter.PayOrderParamsFromPB(req)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	res, err := h.svc.PayOrder(ctx, cmd)
	if err != nil {
		return nil, mapError(err)
	}

	return converter.PayOrderRespToPB(res), nil
}

func mapError(err error) error {
	if err == nil {
		return nil
	}

	switch {
	case errors.Is(err, context.Canceled):
		return status.Error(codes.Canceled, "request canceled")
	default:
		if isValidationError(err) {
			return status.Error(codes.InvalidArgument, err.Error())
		}
		return status.Error(codes.Internal, "internal error")
	}
}

func isValidationError(err error) bool {
	msg := err.Error()
	return msg == "order_uuid is required" ||
		msg == "user_uuid is required" ||
		msg == "payment_method is unknown" ||
		msg == "payment_method unsupported" ||
		msg == "request is nil"
}
