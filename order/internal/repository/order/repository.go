package repository

import (
	"context"
	"sync"

	"github.com/google/uuid"
	"github.com/you-humble/rocket-maintenance/order/internal/model"
)

type repository struct {
	mu   sync.RWMutex
	data map[uuid.UUID]*model.Order
}

func NewOrderRepository() *repository {
	return &repository{
		data: make(map[uuid.UUID]*model.Order),
	}
}

func (s *repository) Create(_ context.Context, ord *model.Order) (uuid.UUID, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ord.ID = uuid.New()
	s.data[ord.ID] = ord

	return ord.ID, nil
}

func (s *repository) OrderByID(_ context.Context, id uuid.UUID) (*model.Order, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ord, ok := s.data[id]
	if !ok {
		return nil, model.ErrOrderNotFound
	}

	cpOrd := *ord
	return &cpOrd, nil
}

func (s *repository) Update(_ context.Context, upd *model.Order) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	ord, ok := s.data[upd.ID]
	if !ok {
		return model.ErrOrderNotFound
	}

	if len(upd.PartIDs) > 0 {
		ord.PartIDs = upd.PartIDs
	}

	if upd.TotalPrice != 0 {
		ord.TotalPrice = upd.TotalPrice
	}

	if upd.TransactionID != nil {
		ord.TransactionID = upd.TransactionID
	}

	if upd.PaymentMethod != nil {
		ord.PaymentMethod = upd.PaymentMethod
	}

	if upd.Status != "" {
		ord.Status = upd.Status
	}

	s.data[ord.ID] = ord
	return nil
}
