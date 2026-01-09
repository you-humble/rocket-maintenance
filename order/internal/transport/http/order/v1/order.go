package http

import (
	"context"
	"errors"
	"net/http"

	"github.com/google/uuid"

	"github.com/you-humble/rocket-maintenance/order/internal/converter"
	"github.com/you-humble/rocket-maintenance/order/internal/model"
	orderv1 "github.com/you-humble/rocket-maintenance/shared/pkg/openapi/order/v1"
)

type OrderService interface {
	Create(
		ctx context.Context,
		params model.CreateOrderParams,
	) (*model.CreateOrderResult, error)
	Pay(
		ctx context.Context,
		params model.PayOrderParams,
	) (*model.PayOrderResult, error)
	OrderByID(ctx context.Context, ordID uuid.UUID) (*model.Order, error)
	Cancel(ctx context.Context, ordID uuid.UUID) error
}

type handler struct {
	svc OrderService
}

func NewOrderHandler(service OrderService) *handler {
	return &handler{svc: service}
}

func (h *handler) CreateOrder(ctx context.Context, req *orderv1.CreateOrderRequest) (orderv1.CreateOrderRes, error) {
	res, err := h.svc.Create(ctx, converter.CreateOrderRequestToParams(req))
	if err != nil {
		return mapErrorToCreateOrderRes(err), nil
	}

	return converter.CreateOrderResultToResponse(res), nil
}

func (h *handler) PayOrder(ctx context.Context, req *orderv1.PayOrderRequest, params orderv1.PayOrderParams) (orderv1.PayOrderRes, error) {
	ordID, err := uuid.Parse(params.OrderUUID.String())
	if err != nil {
		return &orderv1.BadRequestError{ // 400
			Code:    orderv1.NewOptInt32(int32(http.StatusBadRequest)),
			Message: orderv1.NewOptString("invalid order_uuid"),
		}, nil
	}

	res, err := h.svc.Pay(ctx, converter.PayOrderRequestToParams(ordID, req))
	if err != nil {
		return mapErrorToPayOrderRes(err), nil
	}

	return converter.PayOrderResultToResponse(res), nil
}

func (h *handler) GetOrderByUUID(ctx context.Context, params orderv1.GetOrderByUUIDParams) (orderv1.GetOrderByUUIDRes, error) {
	ordID, err := uuid.Parse(params.OrderUUID.String())
	if err != nil {
		return &orderv1.BadRequestError{ // 400
			Code:    orderv1.NewOptInt32(int32(http.StatusBadRequest)),
			Message: orderv1.NewOptString("invalid order_uuid"),
		}, nil
	}

	ord, err := h.svc.OrderByID(ctx, ordID)
	if err != nil {
		return mapErrorToGetOrderRes(err), nil
	}

	return converter.OrderToOAPI(ord), nil
}

func (h *handler) CancelOrder(ctx context.Context, params orderv1.CancelOrderParams) (orderv1.CancelOrderRes, error) {
	ordID, err := uuid.Parse(params.OrderUUID.String())
	if err != nil {
		return &orderv1.BadRequestError{ // 400
			Code:    orderv1.NewOptInt32(int32(http.StatusBadRequest)),
			Message: orderv1.NewOptString("invalid order_uuid"),
		}, nil
	}

	if err := h.svc.Cancel(ctx, ordID); err != nil {
		return mapErrorToCancelOrderRes(err), nil
	}

	return &orderv1.CancelOrderNoContent{}, nil
}

//nolint:dupl
func mapErrorToCreateOrderRes(err error) orderv1.CreateOrderRes {
	switch {
	case errors.Is(err, model.ErrValidation):
		return &orderv1.ValidationError{ // 400
			Code:    orderv1.NewOptInt32(int32(http.StatusBadRequest)),
			Message: orderv1.NewOptString(err.Error()),
		}
	case errors.Is(err, model.ErrPartNotFound):
		return &orderv1.NotFoundError{ // 404
			Code:    orderv1.NewOptInt32(int32(http.StatusNotFound)),
			Message: orderv1.NewOptString(err.Error()),
		}
	case errors.Is(err, model.ErrPartsOutOfStock):
		return &orderv1.ValidationError{ // 422
			Code:    orderv1.NewOptInt32(int32(http.StatusUnprocessableEntity)),
			Message: orderv1.NewOptString(err.Error()),
		}
	case errors.Is(err, model.ErrBadGateway):
		return &orderv1.BadGatewayError{ // 502
			Code:    orderv1.NewOptInt32(int32(http.StatusBadGateway)),
			Message: orderv1.NewOptString(err.Error()),
		}
	case errors.Is(err, model.ErrServiceUnavailable):
		return &orderv1.ServiceUnavailableError{ // 503
			Code:    orderv1.NewOptInt32(int32(http.StatusServiceUnavailable)),
			Message: orderv1.NewOptString(err.Error()),
		}
	default:
		return &orderv1.InternalServerError{ // 500
			Code:    orderv1.NewOptInt32(int32(http.StatusInternalServerError)),
			Message: orderv1.NewOptString(err.Error()),
		}
	}
}

//nolint:dupl
func mapErrorToPayOrderRes(err error) orderv1.PayOrderRes {
	switch {
	case errors.Is(err, model.ErrValidation):
		return &orderv1.ValidationError{ // 400
			Code:    orderv1.NewOptInt32(int32(http.StatusBadRequest)),
			Message: orderv1.NewOptString(err.Error()),
		}
	case errors.Is(err, model.ErrOrderNotFound):
		return &orderv1.NotFoundError{ // 404
			Code:    orderv1.NewOptInt32(int32(http.StatusNotFound)),
			Message: orderv1.NewOptString(err.Error()),
		}
	case errors.Is(err, model.ErrOrderConflict):
		return &orderv1.ConflictError{ // 409
			Code:    orderv1.NewOptInt32(int32(http.StatusConflict)),
			Message: orderv1.NewOptString(err.Error()),
		}
	case errors.Is(err, model.ErrBadGateway):
		return &orderv1.BadGatewayError{ // 502
			Code:    orderv1.NewOptInt32(int32(http.StatusBadGateway)),
			Message: orderv1.NewOptString(err.Error()),
		}
	case errors.Is(err, model.ErrServiceUnavailable):
		return &orderv1.ServiceUnavailableError{ // 503
			Code:    orderv1.NewOptInt32(int32(http.StatusServiceUnavailable)),
			Message: orderv1.NewOptString(err.Error()),
		}
	default:
		return &orderv1.InternalServerError{ // 500
			Code:    orderv1.NewOptInt32(int32(http.StatusInternalServerError)),
			Message: orderv1.NewOptString(err.Error()),
		}
	}
}

func mapErrorToGetOrderRes(err error) orderv1.GetOrderByUUIDRes {
	switch {
	case errors.Is(err, model.ErrOrderNotFound):
		return &orderv1.NotFoundError{ // 404
			Code:    orderv1.NewOptInt32(int32(http.StatusNotFound)),
			Message: orderv1.NewOptString(err.Error()),
		}
	default:
		return &orderv1.InternalServerError{ // 500
			Code:    orderv1.NewOptInt32(int32(http.StatusInternalServerError)),
			Message: orderv1.NewOptString(err.Error()),
		}
	}
}

func mapErrorToCancelOrderRes(err error) orderv1.CancelOrderRes {
	switch {
	case errors.Is(err, model.ErrOrderNotFound):
		return &orderv1.NotFoundError{ // 404
			Code:    orderv1.NewOptInt32(int32(http.StatusNotFound)),
			Message: orderv1.NewOptString(err.Error()),
		}
	case errors.Is(err, model.ErrOrderConflict):
		return &orderv1.ConflictError{ // 409
			Code:    orderv1.NewOptInt32(int32(http.StatusConflict)),
			Message: orderv1.NewOptString(err.Error()),
		}
	default:
		return &orderv1.InternalServerError{ // 500
			Code:    orderv1.NewOptInt32(int32(http.StatusInternalServerError)),
			Message: orderv1.NewOptString(err.Error()),
		}
	}
}
