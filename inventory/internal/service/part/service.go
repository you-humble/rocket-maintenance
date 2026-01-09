package service

import (
	"context"
	"errors"
	"slices"
	"strings"

	"github.com/you-humble/rocket-maintenance/inventory/internal/model"
)

type PartRepository interface {
	PartByID(ctx context.Context, id string) (*model.Part, error)
	List(ctx context.Context) ([]*model.Part, error)
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
	all, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	if filter.Empty() {
		return all, nil
	}

	out := make([]*model.Part, 0, len(all))
	for i := range all {
		if matchPart(all[i], filter) {
			out = append(out, all[i])
		}
	}
	return out, nil
}

func matchPart(part *model.Part, f model.PartsFilter) bool {
	if len(f.IDs) > 0 && !slices.Contains(f.IDs, part.ID) {
		return false
	}

	if len(f.Names) > 0 && !slices.Contains(f.Names, part.Name) {
		return false
	}

	if len(f.Categories) > 0 && !slices.Contains(f.Categories, part.Category) {
		return false
	}

	if len(f.ManufacturerCountries) > 0 {
		country := ""
		if m := part.Manufacturer; m != nil {
			country = m.Country
		}

		if !slices.ContainsFunc(f.ManufacturerCountries, func(c string) bool {
			return strings.EqualFold(c, country)
		}) {
			return false
		}
	}

	if len(f.Tags) > 0 && !hasIntersection(part.Tags, f.Tags) {
		return false
	}

	return true
}

func hasIntersection(a, b []string) bool {
	if len(a) == 0 || len(b) == 0 {
		return false
	}
	for _, x := range a {
		if slices.Contains(b, x) {
			return true
		}
	}
	return false
}
