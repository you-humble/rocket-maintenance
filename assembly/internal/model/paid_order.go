package model

import "github.com/google/uuid"

type PaidOrder struct {
	EventID       uuid.UUID
	OrderID       uuid.UUID
	UserID        uuid.UUID
	PaymentMethod string
	TransactionID uuid.UUID
}
