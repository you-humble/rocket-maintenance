package model

import (
	"time"

	"github.com/google/uuid"
)

type AssembledShip struct {
	EventID   uuid.UUID
	OrderID   uuid.UUID
	UserID    uuid.UUID
	BuildTime time.Duration
}
