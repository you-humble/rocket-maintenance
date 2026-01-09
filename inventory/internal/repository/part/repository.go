package repository

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"

	"github.com/you-humble/rocket-maintenance/inventory/internal/model"
)

type repository struct {
	mu   sync.RWMutex
	data map[string]*model.Part
}

func NewPartRepository(ctx context.Context) (*repository, error) {
	repo := &repository{data: make(map[string]*model.Part)}

	if err := bootstrap(ctx, repo); err != nil {
		return nil, err
	}

	return repo, nil
}

func (s *repository) PartByID(_ context.Context, id string) (*model.Part, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	p, ok := s.data[id]
	if !ok {
		return nil, model.ErrPartNotFound
	}

	return clonePart(p), nil
}

func (r *repository) List(_ context.Context) ([]*model.Part, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]*model.Part, 0, len(r.data))
	for _, p := range r.data {
		out = append(out, clonePart(p))
	}
	return out, nil
}

func (r *repository) add(_ context.Context, parts []*model.Part) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, p := range parts {
		if p.ID == "" {
			return model.ErrInvalidArgument
		}
		r.data[p.ID] = clonePart(p)
	}
	return nil
}

func clonePart(p *model.Part) *model.Part {
	cp := *p

	if p.Manufacturer != nil {
		m := *p.Manufacturer
		cp.Manufacturer = &m
	}
	return &cp
}

func bootstrap(ctx context.Context, repository *repository) error {
	now := time.Now()

	parts := []*model.Part{
		{
			ID:            uuid.NewString(),
			Name:          "HyperDrive Engine Mk1",
			Description:   "Основной гипердрайв для малых космических кораблей.",
			Price:         125000.50,
			StockQuantity: 10,
			Category:      model.CategoryEngine,
			Dimensions: &model.Dimensions{
				Length: 250.0,
				Width:  180.0,
				Height: 140.0,
				Weight: 3200.0,
			},
			Manufacturer: &model.Manufacturer{
				Name:    "Andromeda Drives Inc.",
				Country: "USA",
				Website: "https://andromeda-drives.example.com",
			},
			Tags: []string{"engine", "hyperdrive", "mk1", "small-ship"},
			Metadata: map[string]any{
				"max_thrust_kn":  850.0,
				"warranty_years": 5,
				"military_grade": true,
				"fuel_type":      "quantum-plasma",
			},
			CreatedAt: lo.ToPtr(now),
			UpdatedAt: lo.ToPtr(now),
		},
		{
			ID:            uuid.NewString(),
			Name:          "Quantum Fuel Cell QF-200",
			Description:   "Топливная ячейка для гипердрайвов серии QF.",
			Price:         7800.0,
			StockQuantity: 120,
			Category:      model.CategoryFuel,
			Dimensions: &model.Dimensions{
				Length: 80.0,
				Width:  40.0,
				Height: 35.0,
				Weight: 45.0,
			},
			Manufacturer: &model.Manufacturer{
				Name:    "Sirius Energy Systems",
				Country: "Germany",
				Website: "https://sirius-energy.example.com",
			},
			Tags: []string{"fuel", "quantum", "cell", "qf-series"},
			Metadata: map[string]any{
				"capacity_kwh":      250.0,
				"compatible_engine": "HyperDrive Engine Mk1",
				"hazard_class":      3,
			},
			CreatedAt: lo.ToPtr(now),
			UpdatedAt: lo.ToPtr(now),
		},
		{
			ID:            uuid.NewString(),
			Name:          "Panoramic Porthole PX-360",
			Description:   "Панорамный иллюминатор с круговым обзором 360°.",
			Price:         15200.0,
			StockQuantity: 35,
			Category:      model.CategoryPorthole,
			Dimensions: &model.Dimensions{
				Length: 120.0,
				Width:  120.0,
				Height: 12.0,
				Weight: 65.0,
			},
			Manufacturer: &model.Manufacturer{
				Name:    "Orion Optics",
				Country: "Japan",
				Website: "https://orion-optics.example.com",
			},
			Tags: []string{"porthole", "glass", "panoramic", "px-360"},
			Metadata: map[string]any{
				"glass_type":           "triplex-titanium",
				"max_pressure_bar":     120.0,
				"radiation_protection": true,
			},
			CreatedAt: lo.ToPtr(now),
			UpdatedAt: lo.ToPtr(now),
		},
	}

	return repository.add(ctx, parts)
}
