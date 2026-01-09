package invclient

import (
	"context"

	"github.com/you-humble/rocket-maintenance/order/internal/client/converter"
	"github.com/you-humble/rocket-maintenance/order/internal/model"
	inventorypbv1 "github.com/you-humble/rocket-maintenance/shared/pkg/proto/inventory/v1"
)

type client struct {
	grpc inventorypbv1.InventoryServiceClient
}

func NewClient(grpc inventorypbv1.InventoryServiceClient) *client {
	return &client{grpc: grpc}
}

func (c *client) ListParts(ctx context.Context, filter model.PartsFilter) ([]model.Part, error) {
	parts, err := c.grpc.ListParts(ctx, &inventorypbv1.ListPartsRequest{
		Filter: converter.PartsFilterToPB(filter),
	})
	if err != nil {
		return nil, err
	}

	return converter.PartsListToModel(parts.Parts), nil
}
