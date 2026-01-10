package repository

import (
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/you-humble/rocket-maintenance/inventory/internal/model"
)

func EntityToModel(e *PartEntity) *model.Part {
	if e == nil {
		return nil
	}

	out := &model.Part{
		ID:            e.ID,
		Name:          e.Name,
		Description:   e.Description,
		PriceCents:    e.PriceCents,
		StockQuantity: e.StockQuantity,
		Category:      e.Category,
		Tags:          e.Tags,
		Metadata:      e.Metadata,
		CreatedAt:     e.CreatedAt,
		UpdatedAt:     e.UpdatedAt,
	}

	if e.Dimensions != nil {
		out.Dimensions = &model.Dimensions{
			Length: e.Dimensions.Length,
			Width:  e.Dimensions.Width,
			Height: e.Dimensions.Height,
			Weight: e.Dimensions.Weight,
		}
	}

	if e.Manufacturer != nil {
		out.Manufacturer = &model.Manufacturer{
			Name:    e.Manufacturer.Name,
			Country: e.Manufacturer.Country,
			Website: e.Manufacturer.Website,
		}
	}

	return out
}

func EntityFromModel(p *model.Part) *PartEntity {
	if p == nil {
		return nil
	}

	out := &PartEntity{
		ID:            p.ID,
		Name:          p.Name,
		Description:   p.Description,
		PriceCents:    p.PriceCents,
		StockQuantity: p.StockQuantity,
		Category:      p.Category,
		Tags:          p.Tags,
		Metadata:      p.Metadata,
		CreatedAt:     p.CreatedAt,
		UpdatedAt:     p.UpdatedAt,
	}

	if p.Dimensions != nil {
		out.Dimensions = &DimensionsEntity{
			Length: p.Dimensions.Length,
			Width:  p.Dimensions.Width,
			Height: p.Dimensions.Height,
			Weight: p.Dimensions.Weight,
		}
	}

	if p.Manufacturer != nil {
		out.Manufacturer = &ManufacturerEntity{
			Name:        p.Manufacturer.Name,
			Country:     p.Manufacturer.Country,
			CountryNorm: normalizeCountry(p.Manufacturer.Country),
			Website:     p.Manufacturer.Website,
		}
	}

	return out
}

func BuildMongoFilter(f model.PartsFilter) bson.M {
	q := bson.M{}

	if len(f.IDs) > 0 {
		q["_id"] = bson.M{"$in": f.IDs}
	}
	if len(f.Names) > 0 {
		q["name"] = bson.M{"$in": f.Names}
	}
	if len(f.Categories) > 0 {
		q["category"] = bson.M{"$in": f.Categories}
	}
	if len(f.Tags) > 0 {
		q["tags"] = bson.M{"$in": f.Tags}
	}
	if len(f.ManufacturerCountries) > 0 {
		norm := make([]string, 0, len(f.ManufacturerCountries))
		for _, c := range f.ManufacturerCountries {
			norm = append(norm, normalizeCountry(c))
		}
		q["manufacturer.country_norm"] = bson.M{"$in": norm}
	}

	return q
}

func normalizeCountry(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}
