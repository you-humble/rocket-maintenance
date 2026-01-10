package grpc

import (
	"context"
	"errors"
	"log"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/you-humble/rocket-maintenance/inventory/internal/converter"
	"github.com/you-humble/rocket-maintenance/inventory/internal/model"
	inventorypbv1 "github.com/you-humble/rocket-maintenance/shared/pkg/proto/inventory/v1"
)

type InventoryService interface {
	Part(ctx context.Context, partID string) (*model.Part, error)
	ListParts(ctx context.Context, filter model.PartsFilter) ([]*model.Part, error)
}

type handler struct {
	inventorypbv1.UnimplementedInventoryServiceServer
	svc InventoryService
}

func NewInventoryHandler(service InventoryService) *handler {
	return &handler{svc: service}
}

func (h *handler) GetPart(
	ctx context.Context,
	req *inventorypbv1.GetPartRequest,
) (*inventorypbv1.GetPartResponse, error) {
	p, err := h.svc.Part(ctx, req.GetUuid())
	if err != nil {
		return nil, mapError(err)
	}
	return &inventorypbv1.GetPartResponse{Part: converter.PartFromModel(p)}, nil
}

func (h *handler) ListParts(
	ctx context.Context,
	req *inventorypbv1.ListPartsRequest,
) (*inventorypbv1.ListPartsResponse, error) {
	filter := converter.PartsFilterToModel(req.GetFilter())

	parts, err := h.svc.ListParts(ctx, filter)
	if err != nil {
		return nil, mapError(err)
	}

	out := make([]*inventorypbv1.Part, 0, len(parts))
	for i := range parts {
		out = append(out, converter.PartFromModel(parts[i]))
	}
	return &inventorypbv1.ListPartsResponse{Parts: out}, nil
}

func mapError(err error) error {
	log.Println(err)
	switch {
	case errors.Is(err, model.ErrInvalidArgument):
		return status.Error(codes.InvalidArgument, "invalid argument")
	case errors.Is(err, model.ErrPartNotFound):
		return status.Error(codes.NotFound, "part not found")
	default:
		return status.Error(codes.Internal, "internal error")
	}
}
