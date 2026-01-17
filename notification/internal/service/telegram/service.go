package service

import (
	"context"
	"sync"

	converter "github.com/you-humble/rocket-maintenance/notification/internal/converter/telegram"
	"github.com/you-humble/rocket-maintenance/notification/internal/model"
)

type MessageSender interface {
	SendMessage(ctx context.Context, chatID int64, text string) error
}

type service struct {
	client  MessageSender
	mu      sync.RWMutex
	storage map[int64]struct{}
}

func NewTgService(client MessageSender) *service {
	return &service{client: client, storage: map[int64]struct{}{}}
}

func (svc *service) NotifyShipAssembled(ctx context.Context, event model.AssembledShip) error {
	msg, err := converter.BuildShipAssembled(event)
	if err != nil {
		return err
	}

	svc.mu.RLock()
	defer svc.mu.RUnlock()
	for chatID := range svc.storage {
		if err := svc.client.SendMessage(ctx, chatID, msg); err != nil {
			return err
		}
	}

	return nil
}

func (svc *service) NotifyPaidOrder(ctx context.Context, event model.PaidOrder) error {
	msg, err := converter.BuildPaidOrder(event)
	if err != nil {
		return err
	}

	svc.mu.RLock()
	defer svc.mu.RUnlock()
	for chatID := range svc.storage {
		if err := svc.client.SendMessage(ctx, chatID, msg); err != nil {
			return err
		}
	}

	return nil
}

func (svc *service) AddChatID(ctx context.Context, chatID int64) {
	svc.mu.Lock()
	defer svc.mu.Unlock()
	svc.storage[chatID] = struct{}{}
}
