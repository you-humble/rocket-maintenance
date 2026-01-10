package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"

	"github.com/you-humble/rocket-maintenance/inventory/internal/model"
)

type BatchCreator interface {
	CreateBatch(ctx context.Context, parts []*model.Part) error
}

func PartsBootstrap(ctx context.Context, c BatchCreator) error {
	now := time.Now()

	parts := []*model.Part{
		{
			ID:            uuid.NewString(),
			Name:          "HyperDrive Engine Mk1",
			Description:   "Основной гипердрайв для малых космических кораблей.",
			PriceCents:    12500050,
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
			PriceCents:    780000,
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
			PriceCents:    1520000,
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

	return c.CreateBatch(ctx, parts)
}
