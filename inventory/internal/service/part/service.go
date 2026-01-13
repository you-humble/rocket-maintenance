package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/you-humble/rocket-maintenance/inventory/internal/model"
	"github.com/you-humble/rocket-maintenance/platform/logger"
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
	const op = "inventory.service.Part"
	log := logger.With(
		logger.String("part_id", partID),
	)

	partID = strings.TrimSpace(partID)
	if partID == "" {
		log.Error(ctx, "validation: empty part id")
		return nil, errors.Join(model.ErrInvalidArgument, errors.New("uuid must be non-empty"))
	}

	ctx, cancel := context.WithTimeout(ctx, s.readDBTimeout)
	defer cancel()

	p, err := s.repo.PartByID(ctx, partID)
	if err != nil {
		log.Error(ctx, "repository part by id", logger.ErrorF(err))
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return p, nil
}

func (s *service) ListParts(ctx context.Context, filter model.PartsFilter) ([]*model.Part, error) {
	const op = "inventory.service.ListParts"
	log := logger.With(
		logger.Int("ids_count", len(filter.IDs)),
	)

	ctx, cancel := context.WithTimeout(ctx, s.readDBTimeout)
	defer cancel()

	out, err := s.repo.List(ctx, filter)
	if err != nil {
		log.Error(ctx, "repository list parts", logger.ErrorF(err))
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return out, nil
}
