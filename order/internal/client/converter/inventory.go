package converter

import (
	"time"

	"github.com/you-humble/rocket-maintenance/order/internal/model"
	inventorypbv1 "github.com/you-humble/rocket-maintenance/shared/pkg/proto/inventory/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func PartsListToModel(parts []*inventorypbv1.Part) []model.Part {
	res := make([]model.Part, len(parts))
	for i := range parts {
		res[i] = partToModel(parts[i])
	}

	return res
}

func partToModel(p *inventorypbv1.Part) model.Part {
	if p == nil {
		return model.Part{}
	}

	return model.Part{
		ID:            p.Uuid,
		Name:          p.Name,
		Description:   p.Description,
		Price:         p.Price,
		StockQuantity: p.StockQuantity,
		Category:      model.Category(p.Category),
		Dimensions:    dimensionsToModel(p.Dimensions),
		Manufacturer:  manufacturerToModel(p.Manufacturer),
		Tags:          append([]string(nil), p.Tags...),
		Metadata:      metadataToModel(p.Metadata),
		CreatedAt:     tsToTimePtr(p.CreatedAt),
		UpdatedAt:     tsToTimePtr(p.UpdatedAt),
	}
}

func dimensionsToModel(d *inventorypbv1.Dimensions) *model.Dimensions {
	if d == nil {
		return nil
	}
	return &model.Dimensions{
		Length: d.Length,
		Width:  d.Width,
		Height: d.Height,
		Weight: d.Weight,
	}
}

func manufacturerToModel(m *inventorypbv1.Manufacturer) *model.Manufacturer {
	if m == nil {
		return nil
	}
	return &model.Manufacturer{
		Name:    m.Name,
		Country: m.Country,
		Website: m.Website,
	}
}

func metadataToModel(src map[string]*inventorypbv1.Value) map[string]any {
	if src == nil {
		return nil
	}

	dst := make(map[string]any, len(src))
	for k, v := range src {
		if v == nil {
			continue
		}
		switch vv := v.Value.(type) {
		case *inventorypbv1.Value_StringValue:
			dst[k] = vv.StringValue
		case *inventorypbv1.Value_Int64Value:
			dst[k] = vv.Int64Value
		case *inventorypbv1.Value_DoubleValue:
			dst[k] = vv.DoubleValue
		case *inventorypbv1.Value_BoolValue:
			dst[k] = vv.BoolValue
		default:
		}
	}
	return dst
}

func tsToTimePtr(ts *timestamppb.Timestamp) *time.Time {
	if ts == nil {
		return nil
	}
	t := ts.AsTime()
	return &t
}

func PartsFilterToPB(filter model.PartsFilter) *inventorypbv1.PartsFilter {
	cats := make([]inventorypbv1.Category, len(filter.Categories))
	for i, c := range filter.Categories {
		cats[i] = inventorypbv1.Category(c)
	}

	return &inventorypbv1.PartsFilter{
		Uuids:                 filter.IDs,
		Names:                 filter.Names,
		Categories:            cats,
		ManufacturerCountries: filter.ManufacturerCountries,
		Tags:                  filter.Tags,
	}
}
