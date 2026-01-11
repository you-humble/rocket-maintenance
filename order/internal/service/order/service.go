package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/you-humble/rocket-maintenance/order/internal/model"
	"github.com/you-humble/rocket-maintenance/platform/logger"
)

type OrderRepository interface {
	Create(ctx context.Context, ord *model.Order) (uuid.UUID, error)
	OrderByID(ctx context.Context, id uuid.UUID) (*model.Order, error)
	Update(ctx context.Context, upd *model.Order) error
}

type InventoryClient interface {
	ListParts(ctx context.Context, filter model.PartsFilter) ([]model.Part, error)
}

type PaymentClient interface {
	PayOrder(ctx context.Context, params model.PayOrderParams) (string, error)
}

type service struct {
	repo           OrderRepository
	inventory      InventoryClient
	payment        PaymentClient
	readDBTimeout  time.Duration
	writeDBTimeout time.Duration
}

func NewOrderService(
	repository OrderRepository,
	inventory InventoryClient,
	payment PaymentClient,
	readDBTimeout time.Duration,
	writeDBTimeout time.Duration,
) *service {
	return &service{
		repo:           repository,
		inventory:      inventory,
		payment:        payment,
		readDBTimeout:  readDBTimeout,
		writeDBTimeout: writeDBTimeout,
	}
}

func (svc *service) Create(
	ctx context.Context,
	params model.CreateOrderParams,
) (*model.CreateOrderResult, error) {
	const op string = "order.service.Create"
	log := logger.With(
		logger.String("user_id", params.UserID.String()),
		logger.Int("number_part_ids", len(params.PartIDs)),
	)

	if params.UserID == uuid.Nil || len(params.PartIDs) == 0 {
		log.Error(ctx, "wrong params")
		return nil, fmt.Errorf("%s: %w", op, model.ErrValidation)
	}

	partIDs := make([]string, len(params.PartIDs))
	for i := range partIDs {
		partIDs[i] = params.PartIDs[i].String()
	}

	parts, err := svc.inventory.ListParts(ctx, model.PartsFilter{
		IDs: partIDs,
	})
	if err != nil {
		log.Error(ctx, "list parts", logger.ErrorF(err))
		return nil, fmt.Errorf("%s: %w", op, model.ErrBadGateway)
	}

	if len(parts) != len(params.PartIDs) {
		log.Error(ctx, "len list parts", logger.Int("number_received_parts", len(parts)))
		return nil, fmt.Errorf("%s: %w", op, model.ErrPartNotFound)
	}

	var totalPrice int64
	endedParts := make([]string, 0, len(params.PartIDs))
	for _, p := range parts {
		if p.StockQuantity <= 0 {
			log.Warn(ctx, "ended parts",
				logger.String("part_id", p.ID),
				logger.Int("stock_quantity", int(p.StockQuantity)),
			)
			endedParts = append(endedParts, p.ID)
			continue
		}

		totalPrice += p.PriceCents
	}

	if len(endedParts) > 0 {
		log.Warn(ctx, "len ended parts",
			logger.Int("number_ended_parts", len(endedParts)),
		)
		return nil, fmt.Errorf("%s: %w %v", op, model.ErrPartsOutOfStock, endedParts)
	}

	ctx, cancel := context.WithTimeout(ctx, svc.writeDBTimeout)
	defer cancel()

	ordID, err := svc.repo.Create(ctx, &model.Order{
		UserID:     params.UserID,
		PartIDs:    params.PartIDs,
		TotalPrice: totalPrice,
		Status:     model.StatusPendingPayment,
	})
	if err != nil {
		log.Error(ctx, "repository create order", logger.ErrorF(err))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &model.CreateOrderResult{ID: ordID, TotalPrice: totalPrice}, nil
}

func (svc *service) Pay(
	ctx context.Context,
	params model.PayOrderParams,
) (*model.PayOrderResult, error) {
	const op string = "order.service.Pay"
	log := logger.With(
		logger.String("order_id", params.ID.String()),
		logger.String("user_id", params.UserID.String()),
		logger.String("payment_method", string(params.PaymentMethod)),
	)

	rdbCtx, rdbCancel := context.WithTimeout(ctx, svc.readDBTimeout)
	defer rdbCancel()

	ord, err := svc.repo.OrderByID(rdbCtx, params.ID)
	if err != nil {
		log.Error(ctx, "repository order by id", logger.ErrorF(err))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	log = logger.With(
		logger.String("order_status", string(ord.Status)),
	)
	switch ord.Status {
	case model.StatusPendingPayment:
	case model.StatusPaid, model.StatusCancelled:
		log.Error(ctx, "order conflict", logger.String("status", string(ord.Status)))
		return nil, fmt.Errorf("%s: %w", op, model.ErrOrderConflict)
	default:
		log.Error(ctx, "unknown order status", logger.String("status", string(ord.Status)))
		return nil, fmt.Errorf("%s: %w", op, model.ErrUnknownStatus)
	}

	params.UserID = ord.UserID
	log = logger.With(logger.String("user_id", ord.UserID.String()))

	transactionIDStr, err := svc.payment.PayOrder(ctx, params)
	if err != nil {
		log.Error(ctx, "payment pay order", logger.ErrorF(err))
		return nil, fmt.Errorf("%s: %w", op, model.ErrBadGateway)
	}

	transactionID, err := uuid.Parse(transactionIDStr)
	if err != nil {
		log.Error(ctx, "parse transaction id",
			logger.String("transaction_id", transactionIDStr),
			logger.ErrorF(err),
		)
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	ord.TransactionID = &transactionID
	ord.PaymentMethod = &params.PaymentMethod
	ord.Status = model.StatusPaid

	wdbCtx, wdbCancel := context.WithTimeout(ctx, svc.writeDBTimeout)
	defer wdbCancel()

	if err := svc.repo.Update(wdbCtx, ord); err != nil {
		log.Error(ctx, "repository update order", logger.ErrorF(err))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &model.PayOrderResult{TransactionID: transactionID}, nil
}

func (svc *service) OrderByID(ctx context.Context, ordID uuid.UUID) (*model.Order, error) {
	const op string = "order.service.OrderByID"
	log := logger.With(
		logger.String("order_id", ordID.String()),
	)

	ctx, cancel := context.WithTimeout(ctx, svc.readDBTimeout)
	defer cancel()

	ord, err := svc.repo.OrderByID(ctx, ordID)
	if err != nil {
		log.Error(ctx, "repository order by id", logger.ErrorF(err))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return ord, nil
}

func (svc *service) Cancel(ctx context.Context, ordID uuid.UUID) error {
	const op string = "order.service.Cancel"
	log := logger.With(
		logger.String("order_id", ordID.String()),
	)

	rdbCtx, rdbCancel := context.WithTimeout(ctx, svc.readDBTimeout)
	defer rdbCancel()

	ord, err := svc.repo.OrderByID(rdbCtx, ordID)
	if err != nil {
		log.Error(ctx, "repository order by id", logger.ErrorF(err))
		return fmt.Errorf("%s: %w", op, err)
	}

	log = logger.With(logger.String("order_status", string(ord.Status)))

	switch ord.Status {
	case model.StatusPendingPayment:
		ord.Status = model.StatusCancelled

		wdbCtx, wdbCancel := context.WithTimeout(ctx, svc.writeDBTimeout)
		defer wdbCancel()

		if err := svc.repo.Update(wdbCtx, ord); err != nil {
			log.Error(ctx, "repository update order", logger.ErrorF(err))
			return fmt.Errorf("%s: %w", op, err)
		}
	case model.StatusPaid:
		log.Error(ctx, "order conflict: already paid")
		return fmt.Errorf("%s: %w", op, model.ErrOrderConflict)
	default:
		log.Error(ctx, "wrong order status", logger.String("status", string(ord.Status)))
		return fmt.Errorf("%s: %w", op, model.ErrUnknownStatus)
	}
	return nil
}
