package converter

import (
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/you-humble/rocket-maintenance/inventory/internal/model"
	inventorypbv1 "github.com/you-humble/rocket-maintenance/shared/pkg/proto/inventory/v1"
)

func PartFromModel(p *model.Part) *inventorypbv1.Part {
	out := &inventorypbv1.Part{
		Uuid:          p.ID,
		Name:          p.Name,
		Description:   p.Description,
		PriceCents:    p.PriceCents,
		StockQuantity: p.StockQuantity,
		Category:      categoriyFromModel(p.Category),
		Dimensions:    dimensionsFromModel(p.Dimensions),
		Manufacturer:  manufacturerFromModel(p.Manufacturer),
		Tags:          append([]string(nil), p.Tags...),
		Metadata:      metadataFromModel(p.Metadata),
		CreatedAt:     timestamppb.New(*p.CreatedAt),
		UpdatedAt:     timestamppb.New(*p.UpdatedAt),
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

func dimensionsFromModel(d *model.Dimensions) *inventorypbv1.Dimensions {
	if d == nil {
		return nil
	}
	return &inventorypbv1.Dimensions{
		Length: d.Length,
		Width:  d.Width,
		Height: d.Height,
		Weight: d.Weight,
	}
}

func manufacturerFromModel(m *model.Manufacturer) *inventorypbv1.Manufacturer {
	if m == nil {
		return nil
	}
	return &inventorypbv1.Manufacturer{
		Name:    m.Name,
		Country: m.Country,
		Website: m.Website,
	}
}

func metadataFromModel(src map[string]any) map[string]*inventorypbv1.Value {
	if src == nil {
		return nil
	}

	dst := make(map[string]*inventorypbv1.Value, len(src))
	for k, v := range src {
		if v == nil {
			continue
		}
		switch vv := v.(type) {
		case string:
			dst[k] = &inventorypbv1.Value{Value: &inventorypbv1.Value_StringValue{StringValue: vv}}
		case int64:
			dst[k] = &inventorypbv1.Value{Value: &inventorypbv1.Value_Int64Value{Int64Value: vv}}
		case float64:
			dst[k] = &inventorypbv1.Value{Value: &inventorypbv1.Value_DoubleValue{DoubleValue: vv}}
		case bool:
			dst[k] = &inventorypbv1.Value{Value: &inventorypbv1.Value_BoolValue{BoolValue: vv}}
		default:
		}
	}
	return dst
}
