package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/you-humble/rocket-maintenance/inventory/internal/model"
)

type PartRepository interface {
	PartByID(ctx context.Context, id string) (*model.Part, error)
	List(ctx context.Context, filter model.PartsFilter) ([]*model.Part, error)
}

type service struct {
	repo          PartRepository
	readDBTimeout time.Duration
}

func NewInventoryService(
	repo PartRepository,
	readDBTimeout time.Duration,
) *service {
	return &service{repo: repo, readDBTimeout: readDBTimeout}
}

func (s *service) Part(ctx context.Context, partID string) (*model.Part, error) {
	partID = strings.TrimSpace(partID)
	if partID == "" {
		return nil, errors.Join(model.ErrInvalidArgument, errors.New("uuid must be non-empty"))
	}

	ctx, cancel := context.WithTimeout(ctx, s.readDBTimeout)
	defer cancel()

	p, err := s.repo.PartByID(ctx, partID)
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (s *service) ListParts(ctx context.Context, filter model.PartsFilter) ([]*model.Part, error) {
	ctx, cancel := context.WithTimeout(ctx, s.readDBTimeout)
	defer cancel()

	out, err := s.repo.List(ctx, filter)
	if err != nil {
		return nil, err
	}

	return out, nil
}
