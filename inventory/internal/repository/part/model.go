package repository

import (
	"time"

	"github.com/you-humble/rocket-maintenance/inventory/internal/model"
)

type PartEntity struct {
	ID            string              `bson:"_id"`
	Name          string              `bson:"name"`
	Description   string              `bson:"description,omitempty"`
	PriceCents    int64               `bson:"price_cents"`
	StockQuantity int64               `bson:"stock_quantity"`
	Category      model.Category      `bson:"category"`
	Dimensions    *DimensionsEntity   `bson:"dimensions,omitempty"`
	Manufacturer  *ManufacturerEntity `bson:"manufacturer,omitempty"`
	Tags          []string            `bson:"tags,omitempty"`
	Metadata      map[string]any      `bson:"metadata,omitempty"`
	CreatedAt     *time.Time          `bson:"created_at,omitempty"`
	UpdatedAt     *time.Time          `bson:"updated_at,omitempty"`
}

type ManufacturerEntity struct {
	Name        string `bson:"name"`
	Country     string `bson:"country"`
	CountryNorm string `bson:"country_norm"`
	Website     string `bson:"website,omitempty"`
}

type DimensionsEntity struct {
	Length float64 `bson:"length"`
	Width  float64 `bson:"width"`
	Height float64 `bson:"height"`
	Weight float64 `bson:"weight"`
}
