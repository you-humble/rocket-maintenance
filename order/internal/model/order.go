package model

import "github.com/google/uuid"

type (
	PaymentMethod string
	OrderStatus   string
)

const (
	PaymentMethodUnknown       PaymentMethod = "PAYMENT_METHOD_UNKNOWN"
	PaymentMethodCard          PaymentMethod = "PAYMENT_METHOD_CARD"
	PaymentMethodSBP           PaymentMethod = "PAYMENT_METHOD_SBP"
	PaymentMethodCreditCard    PaymentMethod = "PAYMENT_METHOD_CREDIT_CARD"
	PaymentMethodInvestorMoney PaymentMethod = "PAYMENT_METHOD_INVESTOR_MONEY"
)

const (
	StatusPendingPayment OrderStatus = "PENDING_PAYMENT"
	StatusPaid           OrderStatus = "PAID"
	StatusCancelled      OrderStatus = "CANCELLED"
)

type Order struct {
	// Unique identifier of the order.
	ID uuid.UUID
	// UUID of the user who created the order.
	UserID uuid.UUID
	// List of UUIDs of spacecraft parts included in the order.
	PartIDs []uuid.UUID
	// Total price calculated based on selected spacecraft parts.
	TotalPrice float64
	// UUID of the payment transaction (present if the order is paid).
	TransactionID *uuid.UUID
	// Payment method used to pay for the order (present if the order is paid).
	PaymentMethod *PaymentMethod
	Status        OrderStatus
}

type CreateOrderParams struct {
	UserID  uuid.UUID
	PartIDs []uuid.UUID
}

type CreateOrderResult struct {
	ID         uuid.UUID
	TotalPrice float64
}

type PayOrderParams struct {
	ID            uuid.UUID
	UserID        uuid.UUID
	PaymentMethod PaymentMethod
}

type PayOrderResult struct {
	TransactionID uuid.UUID
}
