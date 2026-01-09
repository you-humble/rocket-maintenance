package converter

import (
	"github.com/you-humble/rocket-maintenance/inventory/internal/model"
	inventorypbv1 "github.com/you-humble/rocket-maintenance/shared/pkg/proto/inventory/v1"
)

func PartFromModel(p *model.Part) *inventorypbv1.Part {
	out := &inventorypbv1.Part{
		Uuid:     p.ID,
		Name:     p.Name,
		Category: categoriyFromModel(p.Category),
		Tags:     append([]string(nil), p.Tags...),
	}
	if p.Manufacturer != nil {
		out.Manufacturer = &inventorypbv1.Manufacturer{
			Name:    p.Manufacturer.Name,
			Country: p.Manufacturer.Country,
		}
	}
	return out
}

func PartsFilterToModel(f *inventorypbv1.PartsFilter) model.PartsFilter {
	if f == nil {
		return model.PartsFilter{}
	}
	return model.PartsFilter{
		IDs:                   append([]string(nil), f.GetUuids()...),
		Names:                 append([]string(nil), f.GetNames()...),
		Categories:            append([]model.Category(nil), categoriesToModel(f.GetCategories())...),
		ManufacturerCountries: append([]string(nil), f.GetManufacturerCountries()...),
		Tags:                  append([]string(nil), f.GetTags()...),
	}
}

func categoriesToModel(categoties []inventorypbv1.Category) []model.Category {
	res := make([]model.Category, len(categoties))

	var val model.Category
	for i := range categoties {
		switch categoties[i] {
		case inventorypbv1.Category_CATEGORY_UNKNOWN:
			val = model.CategoryUnknown
		case inventorypbv1.Category_CATEGORY_ENGINE:
			val = model.CategoryEngine
		case inventorypbv1.Category_CATEGORY_FUEL:
			val = model.CategoryFuel
		case inventorypbv1.Category_CATEGORY_PORTHOLE:
			val = model.CategoryPorthole
		case inventorypbv1.Category_CATEGORY_WING:
			val = model.CategoryWing
		default:
			val = model.CategoryUnknown
		}

		res[i] = val
	}

	return res
}

func categoriyFromModel(c model.Category) inventorypbv1.Category {
	switch c {
	case model.CategoryUnknown:
		return inventorypbv1.Category_CATEGORY_UNKNOWN
	case model.CategoryEngine:
		return inventorypbv1.Category_CATEGORY_ENGINE
	case model.CategoryFuel:
		return inventorypbv1.Category_CATEGORY_FUEL
	case model.CategoryPorthole:
		return inventorypbv1.Category_CATEGORY_PORTHOLE
	case model.CategoryWing:
		return inventorypbv1.Category_CATEGORY_WING
	default:
		return inventorypbv1.Category_CATEGORY_UNKNOWN
	}
}
