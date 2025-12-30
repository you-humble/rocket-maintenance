package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	orderv1 "github.com/you-humble/rocket-maintenance/shared/pkg/openapi/order/v1"
	inventorypbv1 "github.com/you-humble/rocket-maintenance/shared/pkg/proto/inventory/v1"
	paymentpbv1 "github.com/you-humble/rocket-maintenance/shared/pkg/proto/payment/v1"
)

const (
	httpAddr          = "127.0.0.1:8080"
	inventoryGRPCAddr = "127.0.0.1:50051"
	paymentGRPCAddr   = "127.0.0.1:50052"
	readHeaderTimeout = 5 * time.Second
	writeDBTimeout    = 5 * time.Second
	readDBTimeout     = 3 * time.Second
	shutdownTimeout   = 10 * time.Second
)

func PaymentMethodToPB(m orderv1.PaymentMethod) paymentpbv1.PaymentMethod {
	switch m {
	case orderv1.PaymentMethodPAYMENTMETHODUNKNOWN:
		return paymentpbv1.PaymentMethod_PAYMENT_METHOD_UNKNOWN
	case orderv1.PaymentMethodPAYMENTMETHODCARD:
		return paymentpbv1.PaymentMethod_PAYMENT_METHOD_CARD
	case orderv1.PaymentMethodPAYMENTMETHODSBP:
		return paymentpbv1.PaymentMethod_PAYMENT_METHOD_SBP
	case orderv1.PaymentMethodPAYMENTMETHODCREDITCARD:
		return paymentpbv1.PaymentMethod_PAYMENT_METHOD_CREDIT_CARD
	case orderv1.PaymentMethodPAYMENTMETHODINVESTORMONEY:
		return paymentpbv1.PaymentMethod_PAYMENT_METHOD_INVESTOR_MONEY
	default:
		return paymentpbv1.PaymentMethod_PAYMENT_METHOD_UNKNOWN
	}
}

const (
	StatusPendingPayment = orderv1.OrderStatusPENDINGPAYMENT
	StatusPaid           = orderv1.OrderStatusPAID
	StatusCancelled      = orderv1.OrderStatusCANCELLED
)

var (
	ErrValidation         = errors.New("validation error")    // 400
	ErrOrderNotFound      = errors.New("order not found")     // 404
	ErrOrderConflict      = errors.New("order conflict")      // 409
	ErrRateLimited        = errors.New("rate limited")        // 429
	ErrBadGateway         = errors.New("bad gateway")         // 502
	ErrServiceUnavailable = errors.New("service unavailable") // 503
	ErrUnauthorized       = errors.New("unauthorized user")
	ErrForbidden          = errors.New("forbidden")
	ErrPartsOutOfStock    = errors.New("parts out of stock")
	ErrUnknownStatus      = errors.New("unknown status")
	ErrPartNotFound       = errors.New("part not found")
)

// ============ Order Storage ============

type orderStorage struct {
	mu   sync.RWMutex
	data map[uuid.UUID]*orderv1.Order
}

func NewOrderStorage() *orderStorage {
	return &orderStorage{
		data: make(map[uuid.UUID]*orderv1.Order),
	}
}

func (s *orderStorage) Create(_ context.Context, ord *orderv1.Order) (uuid.UUID, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ord.OrderUUID = uuid.New()
	s.data[ord.OrderUUID] = ord

	return ord.OrderUUID, nil
}

func (s *orderStorage) OrderByID(_ context.Context, id uuid.UUID) (*orderv1.Order, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ord, ok := s.data[id]
	if !ok {
		return nil, ErrOrderNotFound
	}

	cpOrd := *ord
	return &cpOrd, nil
}

func (s *orderStorage) Update(_ context.Context, ord *orderv1.Order) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.data[ord.OrderUUID] = ord
	return nil
}

// ============ Order Service ============

type orderService struct {
	storage   *orderStorage
	inventory inventorypbv1.InventoryServiceClient
	payment   paymentpbv1.PaymentServiceClient
}

func NewOrderService(
	storage *orderStorage,
	inventory inventorypbv1.InventoryServiceClient,
	payment paymentpbv1.PaymentServiceClient,
) *orderService {
	return &orderService{
		storage:   storage,
		inventory: inventory,
		payment:   payment,
	}
}

func (svc *orderService) Create(
	ctx context.Context,
	req *orderv1.CreateOrderRequest,
) (*orderv1.CreateOrderResponse, error) {
	const op string = "orderService.Create"

	if req == nil || req.UserUUID == uuid.Nil || len(req.PartUuids) == 0 {
		return nil, fmt.Errorf("%s: %w", op, ErrValidation)
	}

	partIDs := make([]string, len(req.PartUuids))
	for i := range partIDs {
		partIDs[i] = req.PartUuids[i].String()
	}

	l, err := svc.inventory.ListParts(ctx, &inventorypbv1.ListPartsRequest{
		Filter: &inventorypbv1.PartsFilter{Uuids: partIDs},
	})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, ErrBadGateway)
	}

	if len(l.Parts) != len(req.PartUuids) {
		return nil, fmt.Errorf("%s: %w", op, ErrPartNotFound)
	}

	var totalPrice float64
	endedParts := make([]string, 0, len(req.PartUuids))
	for _, p := range l.Parts {
		if p.GetStockQuantity() <= 0 {
			endedParts = append(endedParts, p.Uuid)
			continue
		}

		totalPrice += p.GetPrice()
	}

	if len(endedParts) > 0 {
		return nil, fmt.Errorf("%s: %w %v", op, ErrPartsOutOfStock, endedParts)
	}

	ctx, cancel := context.WithTimeout(ctx, writeDBTimeout)
	defer cancel()

	ordID, err := svc.storage.Create(ctx, &orderv1.Order{
		UserUUID:   req.UserUUID,
		PartUuids:  req.PartUuids,
		TotalPrice: totalPrice,
		Status:     StatusPendingPayment,
	})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &orderv1.CreateOrderResponse{UUID: ordID, TotalPrice: totalPrice}, nil
}

func (svc *orderService) Pay(
	ctx context.Context,
	ordID uuid.UUID,
	req orderv1.PayOrderRequest,
) (*orderv1.PayOrderResponse, error) {
	const op string = "orderService.Pay"

	rdbCtx, rdbCancel := context.WithTimeout(ctx, readDBTimeout)
	defer rdbCancel()

	ord, err := svc.storage.OrderByID(rdbCtx, ordID)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	switch ord.Status {
	case StatusPendingPayment:
	case StatusPaid, StatusCancelled:
		return nil, fmt.Errorf("%s: %w", op, ErrOrderConflict)
	default:
		return nil, fmt.Errorf("%s: %w", op, ErrUnknownStatus)
	}

	paid, err := svc.payment.PayOrder(ctx, &paymentpbv1.PayOrderRequest{
		OrderUuid:     ord.OrderUUID.String(),
		UserUuid:      ord.UserUUID.String(),
		PaymentMethod: PaymentMethodToPB(req.PaymentMethod),
	})
	if err != nil {
		log.Printf("op %s: bad gateway: %v", op, err)
		return nil, fmt.Errorf("%s: %w", op, ErrBadGateway)
	}

	tID, err := uuid.Parse(paid.TransactionUuid)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	ord.TransactionUUID.SetTo(tID)
	ord.PaymentMethod.SetTo(req.PaymentMethod)
	ord.Status = StatusPaid

	wdbCtx, wdbCancel := context.WithTimeout(ctx, writeDBTimeout)
	defer wdbCancel()

	if err := svc.storage.Update(wdbCtx, ord); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &orderv1.PayOrderResponse{TransactionUUID: tID}, nil
}

func (svc *orderService) OrderByID(ctx context.Context, ordID uuid.UUID) (*orderv1.Order, error) {
	ctx, cancel := context.WithTimeout(ctx, readDBTimeout)
	defer cancel()

	return svc.storage.OrderByID(ctx, ordID)
}

func (svc *orderService) Cancel(ctx context.Context, ordID uuid.UUID) error {
	const op string = "orderService.Cancel"

	rdbCtx, rdbCancel := context.WithTimeout(ctx, readDBTimeout)
	defer rdbCancel()

	ord, err := svc.storage.OrderByID(rdbCtx, ordID)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	switch ord.Status {
	case StatusPendingPayment:
		ord.Status = StatusCancelled

		wdbCtx, wdbCancel := context.WithTimeout(ctx, writeDBTimeout)
		defer wdbCancel()

		if err := svc.storage.Update(wdbCtx, ord); err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}
	case StatusPaid:
		return fmt.Errorf("%s: %w", op, ErrOrderConflict)
	default:
		return fmt.Errorf("%s: %w", op, ErrUnknownStatus)
	}
	return nil
}

// ============ Order Handler ============

type orderHandler struct {
	service *orderService
}

func NewOrderHandler(service *orderService) *orderHandler {
	return &orderHandler{service: service}
}

func (h *orderHandler) CreateOrder(ctx context.Context, req *orderv1.CreateOrderRequest) (orderv1.CreateOrderRes, error) {
	resp, err := h.service.Create(ctx, req)
	if err != nil {
		return mapErrorToCreateOrderRes(err)
	}

	return resp, nil
}

func (h *orderHandler) PayOrder(ctx context.Context, req *orderv1.PayOrderRequest, params orderv1.PayOrderParams) (orderv1.PayOrderRes, error) {
	ordID, err := uuid.Parse(params.OrderUUID.String())
	if err != nil {
		return &orderv1.BadRequestError{ // 400
			Code:    orderv1.NewOptInt32(int32(http.StatusBadRequest)),
			Message: orderv1.NewOptString("invalid order_uuid"),
		}, nil
	}

	resp, err := h.service.Pay(ctx, ordID, *req)
	if err != nil {
		return mapErrorToPayOrderRes(err)
	}

	return resp, nil
}

func (h *orderHandler) GetOrderByUUID(ctx context.Context, params orderv1.GetOrderByUUIDParams) (orderv1.GetOrderByUUIDRes, error) {
	ordID, err := uuid.Parse(params.OrderUUID.String())
	if err != nil {
		return &orderv1.BadRequestError{ // 400
			Code:    orderv1.NewOptInt32(int32(http.StatusBadRequest)),
			Message: orderv1.NewOptString("invalid order_uuid"),
		}, nil
	}

	ord, err := h.service.OrderByID(ctx, ordID)
	if err != nil {
		return mapErrorToGetOrderRes(err)
	}

	return ord, nil
}

func (h *orderHandler) CancelOrder(ctx context.Context, params orderv1.CancelOrderParams) (orderv1.CancelOrderRes, error) {
	ordID, err := uuid.Parse(params.OrderUUID.String())
	if err != nil {
		return &orderv1.BadRequestError{ // 400
			Code:    orderv1.NewOptInt32(int32(http.StatusBadRequest)),
			Message: orderv1.NewOptString("invalid order_uuid"),
		}, nil
	}

	if err := h.service.Cancel(ctx, ordID); err != nil {
		return mapErrorToCancelOrderRes(err)
	}

	return &orderv1.CancelOrderNoContent{}, nil
}

// Error mapping

//nolint:dupl
func mapErrorToCreateOrderRes(err error) (orderv1.CreateOrderRes, error) {
	switch {
	case errors.Is(err, ErrValidation):
		return &orderv1.ValidationError{ // 400
			Code:    orderv1.NewOptInt32(int32(http.StatusBadRequest)),
			Message: orderv1.NewOptString(err.Error()),
		}, nil
	case errors.Is(err, ErrPartNotFound):
		return &orderv1.NotFoundError{ // 404
			Code:    orderv1.NewOptInt32(int32(http.StatusNotFound)),
			Message: orderv1.NewOptString(err.Error()),
		}, nil
	case errors.Is(err, ErrPartsOutOfStock):
		return &orderv1.ValidationError{ // 422
			Code:    orderv1.NewOptInt32(int32(http.StatusUnprocessableEntity)),
			Message: orderv1.NewOptString(err.Error()),
		}, nil
	case errors.Is(err, ErrBadGateway):
		return &orderv1.BadGatewayError{ // 502
			Code:    orderv1.NewOptInt32(int32(http.StatusBadGateway)),
			Message: orderv1.NewOptString(err.Error()),
		}, nil
	case errors.Is(err, ErrServiceUnavailable):
		return &orderv1.ServiceUnavailableError{ // 503
			Code:    orderv1.NewOptInt32(int32(http.StatusServiceUnavailable)),
			Message: orderv1.NewOptString(err.Error()),
		}, nil
	default:
		return &orderv1.InternalServerError{ // 500
			Code:    orderv1.NewOptInt32(int32(http.StatusInternalServerError)),
			Message: orderv1.NewOptString(err.Error()),
		}, nil
	}
}

//nolint:dupl
func mapErrorToPayOrderRes(err error) (orderv1.PayOrderRes, error) {
	switch {
	case errors.Is(err, ErrValidation):
		return &orderv1.ValidationError{ // 400
			Code:    orderv1.NewOptInt32(int32(http.StatusBadRequest)),
			Message: orderv1.NewOptString(err.Error()),
		}, nil
	case errors.Is(err, ErrOrderNotFound):
		return &orderv1.NotFoundError{ // 404
			Code:    orderv1.NewOptInt32(int32(http.StatusNotFound)),
			Message: orderv1.NewOptString(err.Error()),
		}, nil
	case errors.Is(err, ErrOrderConflict):
		return &orderv1.ConflictError{ // 409
			Code:    orderv1.NewOptInt32(int32(http.StatusConflict)),
			Message: orderv1.NewOptString(err.Error()),
		}, nil
	case errors.Is(err, ErrBadGateway):
		return &orderv1.BadGatewayError{ // 502
			Code:    orderv1.NewOptInt32(int32(http.StatusBadGateway)),
			Message: orderv1.NewOptString(err.Error()),
		}, nil
	case errors.Is(err, ErrServiceUnavailable):
		return &orderv1.ServiceUnavailableError{ // 503
			Code:    orderv1.NewOptInt32(int32(http.StatusServiceUnavailable)),
			Message: orderv1.NewOptString(err.Error()),
		}, nil
	default:
		return &orderv1.InternalServerError{ // 500
			Code:    orderv1.NewOptInt32(int32(http.StatusInternalServerError)),
			Message: orderv1.NewOptString(err.Error()),
		}, nil
	}
}

func mapErrorToGetOrderRes(err error) (orderv1.GetOrderByUUIDRes, error) {
	switch {
	case errors.Is(err, ErrOrderNotFound):
		return &orderv1.NotFoundError{ // 404
			Code:    orderv1.NewOptInt32(int32(http.StatusNotFound)),
			Message: orderv1.NewOptString(err.Error()),
		}, nil
	default:
		return &orderv1.InternalServerError{ // 500
			Code:    orderv1.NewOptInt32(int32(http.StatusInternalServerError)),
			Message: orderv1.NewOptString(err.Error()),
		}, nil
	}
}

func mapErrorToCancelOrderRes(err error) (orderv1.CancelOrderRes, error) {
	switch {
	case errors.Is(err, ErrOrderNotFound):
		return &orderv1.NotFoundError{ // 404
			Code:    orderv1.NewOptInt32(int32(http.StatusNotFound)),
			Message: orderv1.NewOptString(err.Error()),
		}, nil
	case errors.Is(err, ErrOrderConflict):
		return &orderv1.ConflictError{ // 409
			Code:    orderv1.NewOptInt32(int32(http.StatusConflict)),
			Message: orderv1.NewOptString(err.Error()),
		}, nil
	default:
		return &orderv1.InternalServerError{ // 500
			Code:    orderv1.NewOptInt32(int32(http.StatusInternalServerError)),
			Message: orderv1.NewOptString(err.Error()),
		}, nil
	}
}

func main() {
	// Inventory
	invConn, err := grpc.NewClient(
		inventoryGRPCAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Printf("failed to connect to inventory service %s: %v\n", inventoryGRPCAddr, err)
		return
	}
	defer func() {
		if cerr := invConn.Close(); cerr != nil {
			log.Printf("failed to close inventory service connect: %v", cerr)
		}
	}()

	inventoryClient := inventorypbv1.NewInventoryServiceClient(invConn)

	// Payment
	payConn, err := grpc.NewClient(
		paymentGRPCAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Printf("failed to connect to payment service %s: %v\n", paymentGRPCAddr, err)
		return
	}
	defer func() {
		if cerr := payConn.Close(); cerr != nil {
			log.Printf("failed to close payment service connect: %v", cerr)
		}
	}()

	paymentClient := paymentpbv1.NewPaymentServiceClient(payConn)

	storage := NewOrderStorage()
	service := NewOrderService(storage, inventoryClient, paymentClient)

	handler := NewOrderHandler(service)

	orderServer, err := orderv1.NewServer(handler)
	if err != nil {
		log.Printf("failed to create a new server: %v\n", err)
		return
	}

	r := chi.NewRouter()

	r.Use(
		middleware.Recoverer,
		middleware.Logger,
	)

	r.Mount("/", orderServer)

	server := &http.Server{
		Addr:              httpAddr,
		Handler:           r,
		ReadHeaderTimeout: readHeaderTimeout,
	}

	go func() {
		log.Printf("🚀 order server listening on %s", httpAddr)
		err := server.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("❌ order server error: %v\n", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("🛑 Server shutdown...")

	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("❌ Error during server shutdown: %v\n", err)
		log.Println("❌😵‍💫 Server stopped")
		return
	}

	log.Println("✅ Server stopped")
}
