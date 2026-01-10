package service

import (
	"context"
	"errors"
	"strings"

	"github.com/you-humble/rocket-maintenance/inventory/internal/model"
)

type PartRepository interface {
	PartByID(ctx context.Context, id string) (*model.Part, error)
	List(ctx context.Context, filter model.PartsFilter) ([]*model.Part, error)
}

type service struct {
	repo PartRepository
}

func NewInventoryService(repo PartRepository) *service {
	return &service{repo: repo}
}

func (s *service) Part(ctx context.Context, partID string) (*model.Part, error) {
	partID = strings.TrimSpace(partID)
	if partID == "" {
		return nil, errors.Join(model.ErrInvalidArgument, errors.New("uuid must be non-empty"))
	}

	p, err := s.repo.PartByID(ctx, partID)
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (s *service) ListParts(ctx context.Context, filter model.PartsFilter) ([]*model.Part, error) {
	out, err := s.repo.List(ctx, filter)
	if err != nil {
		return nil, err
	}

	return out, nil
}
