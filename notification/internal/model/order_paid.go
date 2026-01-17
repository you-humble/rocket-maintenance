package model

import "github.com/google/uuid"

type PaidOrder struct {
	EventID       uuid.UUID
	OrderID       uuid.UUID
	UserID        uuid.UUID
	PaymentMethod string
	TransactionID uuid.UUID
}

type PaidOrderNotification struct {
	OrderID       string
	UserID        string
	PaymentMethod string
	TransactionID string
}
