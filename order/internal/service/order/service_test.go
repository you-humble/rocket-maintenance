package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/you-humble/rocket-maintenance/order/internal/model"
	"github.com/you-humble/rocket-maintenance/order/internal/service/mocks"
)

func TestServiceCreate(t *testing.T) {
	t.Parallel()

	type deps struct {
		repository *mocks.MockOrderRepository
		inventory  *mocks.MockInventoryClient
		payment    *mocks.MockPaymentClient
	}

	newSvc := func(d deps) *service {
		return NewOrderService(d.repository, d.inventory, d.payment)
	}

	userID := uuid.New()
	partID1 := uuid.New()
	partID2 := uuid.New()
	orderID := uuid.New()
	price1 := gofakeit.Price(10, 999)
	price2 := gofakeit.Price(10, 999)

	type testCase struct {
		name   string
		params model.CreateOrderParams
		setup  func(d deps)
		assert func(t *testing.T, res *model.CreateOrderResult, err error, d deps)
	}

	tests := []testCase{
		{
			name: "validation error: empty user id",
			params: model.CreateOrderParams{
				UserID:  uuid.Nil,
				PartIDs: []uuid.UUID{partID1},
			},
			setup: func(d deps) {
				// No calls expected.
			},
			assert: func(t *testing.T, res *model.CreateOrderResult, err error, d deps) {
				require.Error(t, err)
				assert.ErrorIs(t, err, model.ErrValidation)
				assert.Nil(t, res)

				d.repository.AssertExpectations(t)
				d.inventory.AssertExpectations(t)
				d.payment.AssertExpectations(t)
			},
		},
		{
			name: "validation error: empty parts list",
			params: model.CreateOrderParams{
				UserID:  userID,
				PartIDs: nil,
			},
			setup: func(d deps) {
				// No calls expected.
			},
			assert: func(t *testing.T, res *model.CreateOrderResult, err error, d deps) {
				require.Error(t, err)
				assert.ErrorIs(t, err, model.ErrValidation)
				assert.Nil(t, res)

				d.repository.AssertExpectations(t)
				d.inventory.AssertExpectations(t)
				d.payment.AssertExpectations(t)
			},
		},
		{
			name: "inventory bad gateway: ListParts returns error",
			params: model.CreateOrderParams{
				UserID:  userID,
				PartIDs: []uuid.UUID{partID1, partID2},
			},
			setup: func(d deps) {
				d.inventory.
					On("ListParts", mock.Anything, mock.MatchedBy(func(f model.PartsFilter) bool {
						// Ensure IDs are passed as strings.
						return len(f.IDs) == 2
					})).
					Return(nil, errors.New("inventory is down")).
					Once()
			},
			assert: func(t *testing.T, res *model.CreateOrderResult, err error, d deps) {
				require.Error(t, err)
				assert.ErrorIs(t, err, model.ErrBadGateway)
				assert.Nil(t, res)

				d.repository.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
				d.inventory.AssertExpectations(t)
			},
		},
		{
			name: "part not found: inventory returned fewer parts than requested",
			params: model.CreateOrderParams{
				UserID:  userID,
				PartIDs: []uuid.UUID{partID1, partID2},
			},
			setup: func(d deps) {
				d.inventory.
					On("ListParts", mock.Anything, mock.Anything).
					Return([]model.Part{
						{ID: partID1.String(), Price: price1, StockQuantity: 1},
					}, nil).
					Once()
			},
			assert: func(t *testing.T, res *model.CreateOrderResult, err error, d deps) {
				require.Error(t, err)
				assert.ErrorIs(t, err, model.ErrPartNotFound)
				assert.Nil(t, res)

				d.repository.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
				d.inventory.AssertExpectations(t)
			},
		},
		{
			name: "parts out of stock: at least one part has StockQuantity <= 0",
			params: model.CreateOrderParams{
				UserID:  userID,
				PartIDs: []uuid.UUID{partID1, partID2},
			},
			setup: func(d deps) {
				d.inventory.
					On("ListParts", mock.Anything, mock.Anything).
					Return([]model.Part{
						{ID: partID1.String(), Price: price1, StockQuantity: 1},
						{ID: partID2.String(), Price: price2, StockQuantity: 0},
					}, nil).
					Once()
			},
			assert: func(t *testing.T, res *model.CreateOrderResult, err error, d deps) {
				require.Error(t, err)
				assert.ErrorIs(t, err, model.ErrPartsOutOfStock)
				assert.Nil(t, res)

				d.repository.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
				d.inventory.AssertExpectations(t)
			},
		},
		{
			name: "repository error: Create returns error",
			params: model.CreateOrderParams{
				UserID:  userID,
				PartIDs: []uuid.UUID{partID1, partID2},
			},
			setup: func(d deps) {
				d.inventory.
					On("ListParts", mock.Anything, mock.Anything).
					Return([]model.Part{
						{ID: partID1.String(), Price: price1, StockQuantity: 1},
						{ID: partID2.String(), Price: price2, StockQuantity: 2},
					}, nil).
					Once()

				d.repository.
					On("Create", mock.Anything, mock.MatchedBy(func(o *model.Order) bool {
						return o.UserID == userID &&
							len(o.PartIDs) == 2 &&
							o.TotalPrice == price1+price2 &&
							o.Status == model.StatusPendingPayment
					})).
					Return(uuid.Nil, errors.New("db write failed")).
					Once()
			},
			assert: func(t *testing.T, res *model.CreateOrderResult, err error, d deps) {
				require.Error(t, err)
				assert.NotErrorIs(t, err, model.ErrBadGateway)
				assert.Nil(t, res)

				d.repository.AssertExpectations(t)
				d.inventory.AssertExpectations(t)
			},
		},
		{
			name: "success: creates order with total price and pending status",
			params: model.CreateOrderParams{
				UserID:  userID,
				PartIDs: []uuid.UUID{partID1, partID2},
			},
			setup: func(d deps) {
				d.inventory.
					On("ListParts", mock.Anything, mock.Anything).
					Return([]model.Part{
						{ID: partID1.String(), Price: price1, StockQuantity: 1},
						{ID: partID2.String(), Price: price2, StockQuantity: 2},
					}, nil).
					Once()

				d.repository.
					On("Create", mock.Anything, mock.MatchedBy(func(o *model.Order) bool {
						return o.UserID == userID &&
							assert.ElementsMatch(t,
								[]uuid.UUID{partID1, partID2},
								o.PartIDs,
							) &&
							o.TotalPrice == price1+price2 &&
							o.Status == model.StatusPendingPayment
					})).
					Return(orderID, nil).
					Once()
			},
			assert: func(t *testing.T, res *model.CreateOrderResult, err error, d deps) {
				require.NoError(t, err)
				require.NotNil(t, res)
				assert.Equal(t, orderID, res.ID)
				assert.Equal(t, float64(price1+price2), res.TotalPrice)

				d.repository.AssertExpectations(t)
				d.inventory.AssertExpectations(t)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			d := deps{
				repository: mocks.NewMockOrderRepository(t),
				inventory:  mocks.NewMockInventoryClient(t),
				payment:    mocks.NewMockPaymentClient(t),
			}
			if tt.setup != nil {
				tt.setup(d)
			}

			svc := newSvc(d)

			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			res, err := svc.Create(ctx, tt.params)
			tt.assert(t, res, err, d)
		})
	}
}

func TestServicePay(t *testing.T) {
	t.Parallel()

	type deps struct {
		repository *mocks.MockOrderRepository
		inventory  *mocks.MockInventoryClient
		payment    *mocks.MockPaymentClient
	}

	newSvc := func(d deps) *service {
		return NewOrderService(d.repository, d.inventory, d.payment)
	}

	userID := uuid.New()
	ordID := uuid.New()
	txID := uuid.New()

	type testCase struct {
		name   string
		params model.PayOrderParams
		setup  func(d deps)
		assert func(t *testing.T, res *model.PayOrderResult, err error, d deps)
	}

	tests := []testCase{
		{
			name: "repository error: OrderByID fails",
			params: model.PayOrderParams{
				ID:            ordID,
				PaymentMethod: model.PaymentMethodCard,
			},
			setup: func(d deps) {
				d.repository.
					On("OrderByID", mock.Anything, ordID).
					Return((*model.Order)(nil), errors.New("db read failed")).
					Once()
			},
			assert: func(t *testing.T, res *model.PayOrderResult, err error, d deps) {
				require.Error(t, err)
				assert.Nil(t, res)

				d.payment.AssertNotCalled(t, "PayOrder", mock.Anything, mock.Anything)
				d.repository.AssertExpectations(t)
			},
		},
		{
			name: "conflict: already paid",
			params: model.PayOrderParams{
				ID:            ordID,
				PaymentMethod: model.PaymentMethodCard,
			},
			setup: func(d deps) {
				d.repository.
					On("OrderByID", mock.Anything, ordID).
					Return(&model.Order{
						ID:     ordID,
						UserID: userID,
						Status: model.StatusPaid,
					}, nil).
					Once()
			},
			assert: func(t *testing.T, res *model.PayOrderResult, err error, d deps) {
				require.Error(t, err)
				assert.ErrorIs(t, err, model.ErrOrderConflict)
				assert.Nil(t, res)

				d.payment.AssertNotCalled(t, "PayOrder", mock.Anything, mock.Anything)
				d.repository.AssertExpectations(t)
			},
		},
		{
			name: "conflict: cancelled",
			params: model.PayOrderParams{
				ID:            ordID,
				PaymentMethod: model.PaymentMethodCard,
			},
			setup: func(d deps) {
				d.repository.
					On("OrderByID", mock.Anything, ordID).
					Return(&model.Order{
						ID:     ordID,
						UserID: userID,
						Status: model.StatusCancelled,
					}, nil).
					Once()
			},
			assert: func(t *testing.T, res *model.PayOrderResult, err error, d deps) {
				require.Error(t, err)
				assert.ErrorIs(t, err, model.ErrOrderConflict)
				assert.Nil(t, res)

				d.payment.AssertNotCalled(t, "PayOrder", mock.Anything, mock.Anything)
				d.repository.AssertExpectations(t)
			},
		},
		{
			name: "unknown status",
			params: model.PayOrderParams{
				ID:            ordID,
				PaymentMethod: model.PaymentMethodCard,
			},
			setup: func(d deps) {
				d.repository.
					On("OrderByID", mock.Anything, ordID).
					Return(&model.Order{
						ID:     ordID,
						UserID: userID,
						Status: model.OrderStatus("weird_status"),
					}, nil).
					Once()
			},
			assert: func(t *testing.T, res *model.PayOrderResult, err error, d deps) {
				require.Error(t, err)
				assert.ErrorIs(t, err, model.ErrUnknownStatus)
				assert.Nil(t, res)

				d.payment.AssertNotCalled(t, "PayOrder", mock.Anything, mock.Anything)
				d.repository.AssertExpectations(t)
			},
		},
		{
			name: "payment bad gateway: PayOrder returns error",
			params: model.PayOrderParams{
				ID:            ordID,
				PaymentMethod: model.PaymentMethodCard,
			},
			setup: func(d deps) {
				d.repository.
					On("OrderByID", mock.Anything, ordID).
					Return(&model.Order{
						ID:     ordID,
						UserID: userID,
						Status: model.StatusPendingPayment,
					}, nil).
					Once()

				d.payment.
					On("PayOrder", mock.Anything, mock.MatchedBy(func(p model.PayOrderParams) bool {
						// Service must set UserID from the loaded order.
						return p.ID == ordID && p.UserID == userID && p.PaymentMethod == model.PaymentMethodCard
					})).
					Return("", errors.New("payment provider timeout")).
					Once()
			},
			assert: func(t *testing.T, res *model.PayOrderResult, err error, d deps) {
				require.Error(t, err)
				assert.ErrorIs(t, err, model.ErrBadGateway)
				assert.Nil(t, res)

				d.repository.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
				d.repository.AssertExpectations(t)
				d.payment.AssertExpectations(t)
			},
		},
		{
			name: "payment returns invalid transaction id",
			params: model.PayOrderParams{
				ID:            ordID,
				PaymentMethod: model.PaymentMethodCard,
			},
			setup: func(d deps) {
				d.repository.
					On("OrderByID", mock.Anything, ordID).
					Return(&model.Order{
						ID:     ordID,
						UserID: userID,
						Status: model.StatusPendingPayment,
					}, nil).
					Once()

				d.payment.
					On("PayOrder", mock.Anything, mock.Anything).
					Return("not-a-uuid", nil).
					Once()
			},
			assert: func(t *testing.T, res *model.PayOrderResult, err error, d deps) {
				require.Error(t, err)
				assert.Nil(t, res)

				d.repository.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
				d.repository.AssertExpectations(t)
				d.payment.AssertExpectations(t)
			},
		},
		{
			name: "repository error: Update fails after successful payment",
			params: model.PayOrderParams{
				ID:            ordID,
				PaymentMethod: model.PaymentMethodCard,
			},
			setup: func(d deps) {
				d.repository.
					On("OrderByID", mock.Anything, ordID).
					Return(&model.Order{
						ID:     ordID,
						UserID: userID,
						Status: model.StatusPendingPayment,
					}, nil).
					Once()

				d.payment.
					On("PayOrder", mock.Anything, mock.Anything).
					Return(txID.String(), nil).
					Once()

				d.repository.
					On("Update", mock.Anything, mock.MatchedBy(func(o *model.Order) bool {
						// Verify the service mutates order state correctly before persisting.
						return o.ID == ordID &&
							o.Status == model.StatusPaid &&
							o.TransactionID != nil && *o.TransactionID == txID &&
							o.PaymentMethod != nil && *o.PaymentMethod == model.PaymentMethodCard
					})).
					Return(errors.New("db update failed")).
					Once()
			},
			assert: func(t *testing.T, res *model.PayOrderResult, err error, d deps) {
				require.Error(t, err)
				assert.Nil(t, res)

				d.repository.AssertExpectations(t)
				d.payment.AssertExpectations(t)
			},
		},
		{
			name: "success: pending -> paid with transaction id",
			params: model.PayOrderParams{
				ID:            ordID,
				PaymentMethod: model.PaymentMethodCard,
			},
			setup: func(d deps) {
				d.repository.
					On("OrderByID", mock.Anything, ordID).
					Return(&model.Order{
						ID:     ordID,
						UserID: userID,
						Status: model.StatusPendingPayment,
					}, nil).
					Once()

				d.payment.
					On("PayOrder", mock.Anything, mock.MatchedBy(func(p model.PayOrderParams) bool {
						return p.ID == ordID && p.UserID == userID
					})).
					Return(txID.String(), nil).
					Once()

				d.repository.
					On("Update", mock.Anything, mock.AnythingOfType("*model.Order")).
					Return(nil).
					Once()
			},
			assert: func(t *testing.T, res *model.PayOrderResult, err error, d deps) {
				require.NoError(t, err)
				require.NotNil(t, res)
				assert.Equal(t, txID, res.TransactionID)

				d.repository.AssertExpectations(t)
				d.payment.AssertExpectations(t)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			d := deps{
				repository: mocks.NewMockOrderRepository(t),
				inventory:  mocks.NewMockInventoryClient(t),
				payment:    mocks.NewMockPaymentClient(t),
			}
			if tt.setup != nil {
				tt.setup(d)
			}

			svc := newSvc(d)

			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			res, err := svc.Pay(ctx, tt.params)
			tt.assert(t, res, err, d)
		})
	}
}

func TestService_OrderByID(t *testing.T) {
	t.Parallel()

	type deps struct {
		repository *mocks.MockOrderRepository
		inventory  *mocks.MockInventoryClient
		payment    *mocks.MockPaymentClient
	}

	newSVC := func(d deps) *service {
		return NewOrderService(d.repository, d.inventory, d.payment)
	}

	type testCase struct {
		name   string
		ordID  uuid.UUID
		setup  func(d deps)
		assert func(t *testing.T, got *model.Order, err error, d deps)
	}

	gofakeit.Seed(0)

	tests := []testCase{
		{
			name:  "success: returns order from repository",
			ordID: uuid.New(),
			setup: func(d deps) {
				expected := &model.Order{
					ID:     uuid.New(),
					UserID: uuid.New(),
					PartIDs: []uuid.UUID{
						uuid.New(),
						uuid.New(),
					},
					TotalPrice: gofakeit.Price(10, 999),
					Status:     model.StatusPendingPayment,
				}

				d.repository.
					On("OrderByID", mock.Anything, mock.AnythingOfType("uuid.UUID")).
					Return(expected, nil).
					Once()
			},
			assert: func(t *testing.T, got *model.Order, err error, d deps) {
				require.NoError(t, err)
				require.NotNil(t, got)

				// We assert on key fields to keep the test robust to minor model changes.
				assert.NotEqual(t, uuid.Nil, got.ID)
				assert.NotEqual(t, uuid.Nil, got.UserID)
				assert.NotEmpty(t, got.PartIDs)
				assert.NotZero(t, got.TotalPrice)
				assert.NotEmpty(t, got.Status)

				d.repository.AssertExpectations(t)
			},
		},
		{
			name:  "error: repository returns error",
			ordID: uuid.New(),
			setup: func(d deps) {
				d.repository.
					On("OrderByID", mock.Anything, mock.AnythingOfType("uuid.UUID")).
					Return((*model.Order)(nil), gofakeit.Error()).
					Once()
			},
			assert: func(t *testing.T, got *model.Order, err error, d deps) {
				require.Error(t, err)
				assert.Nil(t, got)

				d.repository.AssertExpectations(t)
			},
		},
	}

	for _, tt := range tests {
		tt := tt // capture
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			d := deps{
				repository: mocks.NewMockOrderRepository(t),
				inventory:  mocks.NewMockInventoryClient(t),
				payment:    mocks.NewMockPaymentClient(t),
			}

			if tt.setup != nil {
				tt.setup(d)
			}

			svc := newSVC(d)

			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			got, err := svc.OrderByID(ctx, tt.ordID)
			tt.assert(t, got, err, d)
		})
	}
}

func TestServiceCancel(t *testing.T) {
	t.Parallel()

	type deps struct {
		repository *mocks.MockOrderRepository
		inventory  *mocks.MockInventoryClient
		payment    *mocks.MockPaymentClient
	}

	newSVC := func(d deps) *service {
		return NewOrderService(d.repository, d.inventory, d.payment)
	}

	type testCase struct {
		name   string
		ordID  uuid.UUID
		setup  func(d deps)
		assert func(t *testing.T, err error, d deps)
	}

	userID := uuid.New()
	ordID := uuid.New()

	tests := []testCase{
		{
			name:  "repository error: OrderByID fails",
			ordID: ordID,
			setup: func(d deps) {
				d.repository.
					On("OrderByID", mock.Anything, ordID).
					Return((*model.Order)(nil), errors.New("db read failed")).
					Once()
			},
			assert: func(t *testing.T, err error, d deps) {
				require.Error(t, err)
				d.repository.AssertExpectations(t)
				d.repository.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
			},
		},
		{
			name:  "conflict: cannot cancel paid order",
			ordID: ordID,
			setup: func(d deps) {
				d.repository.
					On("OrderByID", mock.Anything, ordID).
					Return(&model.Order{
						ID:     ordID,
						UserID: userID,
						Status: model.StatusPaid,
					}, nil).
					Once()
			},
			assert: func(t *testing.T, err error, d deps) {
				require.Error(t, err)
				assert.ErrorIs(t, err, model.ErrOrderConflict)
				d.repository.AssertExpectations(t)
				d.repository.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
			},
		},
		{
			name:  "unknown status",
			ordID: ordID,
			setup: func(d deps) {
				d.repository.
					On("OrderByID", mock.Anything, ordID).
					Return(&model.Order{
						ID:     ordID,
						UserID: userID,
						Status: model.OrderStatus("mystery"),
					}, nil).
					Once()
			},
			assert: func(t *testing.T, err error, d deps) {
				require.Error(t, err)
				assert.ErrorIs(t, err, model.ErrUnknownStatus)
				d.repository.AssertExpectations(t)
				d.repository.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
			},
		},
		{
			name:  "repository error: Update fails",
			ordID: ordID,
			setup: func(d deps) {
				d.repository.
					On("OrderByID", mock.Anything, ordID).
					Return(&model.Order{
						ID:     ordID,
						UserID: userID,
						Status: model.StatusPendingPayment,
					}, nil).
					Once()

				d.repository.
					On("Update", mock.Anything, mock.MatchedBy(func(o *model.Order) bool {
						return o.ID == ordID && o.Status == model.StatusCancelled
					})).
					Return(errors.New("db update failed")).
					Once()
			},
			assert: func(t *testing.T, err error, d deps) {
				require.Error(t, err)
				d.repository.AssertExpectations(t)
			},
		},
		{
			name:  "success: pending -> cancelled",
			ordID: ordID,
			setup: func(d deps) {
				d.repository.
					On("OrderByID", mock.Anything, ordID).
					Return(&model.Order{
						ID:     ordID,
						UserID: userID,
						Status: model.StatusPendingPayment,
					}, nil).
					Once()

				d.repository.
					On("Update", mock.Anything, mock.MatchedBy(func(o *model.Order) bool {
						return o.ID == ordID && o.Status == model.StatusCancelled
					})).
					Return(nil).
					Once()
			},
			assert: func(t *testing.T, err error, d deps) {
				require.NoError(t, err)
				d.repository.AssertExpectations(t)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			d := deps{
				repository: mocks.NewMockOrderRepository(t),
				inventory:  mocks.NewMockInventoryClient(t),
				payment:    mocks.NewMockPaymentClient(t),
			}
			if tt.setup != nil {
				tt.setup(d)
			}

			svc := newSVC(d)

			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			err := svc.Cancel(ctx, tt.ordID)
			tt.assert(t, err, d)
		})
	}
}
