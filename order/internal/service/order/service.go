package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"

	"github.com/you-humble/rocket-maintenance/order/internal/model"
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

	if params.UserID == uuid.Nil || len(params.PartIDs) == 0 {
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
		return nil, fmt.Errorf("%s: %w", op, model.ErrBadGateway)
	}

	if len(parts) != len(params.PartIDs) {
		return nil, fmt.Errorf("%s: %w", op, model.ErrPartNotFound)
	}

	var totalPrice int64
	endedParts := make([]string, 0, len(params.PartIDs))
	for _, p := range parts {
		if p.StockQuantity <= 0 {
			log.Printf("%s: %#v", op, p)
			endedParts = append(endedParts, p.ID)
			continue
		}

		totalPrice += p.PriceCents
	}

	if len(endedParts) > 0 {
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
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &model.CreateOrderResult{ID: ordID, TotalPrice: totalPrice}, nil
}

func (svc *service) Pay(
	ctx context.Context,
	params model.PayOrderParams,
) (*model.PayOrderResult, error) {
	const op string = "order.service.Pay"

	rdbCtx, rdbCancel := context.WithTimeout(ctx, svc.readDBTimeout)
	defer rdbCancel()

	ord, err := svc.repo.OrderByID(rdbCtx, params.ID)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	switch ord.Status {
	case model.StatusPendingPayment:
	case model.StatusPaid, model.StatusCancelled:
		return nil, fmt.Errorf("%s: %w", op, model.ErrOrderConflict)
	default:
		return nil, fmt.Errorf("%s: %w", op, model.ErrUnknownStatus)
	}

	params.UserID = ord.UserID
	transactionIDStr, err := svc.payment.PayOrder(ctx, params)
	if err != nil {
		log.Printf("op %s: bad gateway: %v", op, err)
		return nil, fmt.Errorf("%s: %w", op, model.ErrBadGateway)
	}

	transactionID, err := uuid.Parse(transactionIDStr)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	ord.TransactionID = &transactionID
	ord.PaymentMethod = &params.PaymentMethod
	ord.Status = model.StatusPaid

	wdbCtx, wdbCancel := context.WithTimeout(ctx, svc.writeDBTimeout)
	defer wdbCancel()

	if err := svc.repo.Update(wdbCtx, ord); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &model.PayOrderResult{TransactionID: transactionID}, nil
}

func (svc *service) OrderByID(ctx context.Context, ordID uuid.UUID) (*model.Order, error) {
	ctx, cancel := context.WithTimeout(ctx, svc.readDBTimeout)
	defer cancel()

	return svc.repo.OrderByID(ctx, ordID)
}

func (svc *service) Cancel(ctx context.Context, ordID uuid.UUID) error {
	const op string = "order.service.Cancel"

	rdbCtx, rdbCancel := context.WithTimeout(ctx, svc.readDBTimeout)
	defer rdbCancel()

	ord, err := svc.repo.OrderByID(rdbCtx, ordID)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	switch ord.Status {
	case model.StatusPendingPayment:
		ord.Status = model.StatusCancelled

		wdbCtx, wdbCancel := context.WithTimeout(ctx, svc.writeDBTimeout)
		defer wdbCancel()

		if err := svc.repo.Update(wdbCtx, ord); err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}
	case model.StatusPaid:
		return fmt.Errorf("%s: %w", op, model.ErrOrderConflict)
	default:
		return fmt.Errorf("%s: %w", op, model.ErrUnknownStatus)
	}
	return nil
}
