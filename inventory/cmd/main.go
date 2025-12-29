package main

import (
	"context"
	"errors"
	"log"
	"net"
	"os"
	"os/signal"
	"slices"
	"strings"
	"sync"
	"syscall"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	inventorypbv1 "github.com/you-humble/rocket-maintenance/shared/pkg/proto/inventory/v1"
)

const GRPCAddr = "127.0.0.1:50051"

var (
	ErrPartNotFound = errors.New("part not found")
	ErrCloneFailed  = errors.New("failed to clone part")
)

// ============ Part Storage ============
type partStorage struct {
	mu   sync.RWMutex
	data map[string]*inventorypbv1.Part
}

func NewPartStorage() *partStorage {
	return &partStorage{data: make(map[string]*inventorypbv1.Part)}
}

func (s *partStorage) PartByID(_ context.Context, id string) (*inventorypbv1.Part, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	p, ok := s.data[id]
	if !ok {
		return nil, ErrPartNotFound
	}

	cp, ok := proto.Clone(p).(*inventorypbv1.Part)
	if !ok {
		return nil, ErrCloneFailed
	}

	return cp, nil
}

func (s *partStorage) ListParts(_ context.Context) ([]*inventorypbv1.Part, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*inventorypbv1.Part, 0, len(s.data))
	for _, p := range s.data {
		cp, ok := proto.Clone(p).(*inventorypbv1.Part)
		if !ok {
			return nil, ErrCloneFailed
		}
		result = append(result, cp)
	}

	return result, nil
}

func (s *partStorage) AddParts(_ context.Context, parts []*inventorypbv1.Part) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, part := range parts {
		if part.GetUuid() == "" {
			return errors.New("uuid must be non-empty")
		}
		cp, ok := proto.Clone(part).(*inventorypbv1.Part)
		if !ok {
			return ErrCloneFailed
		}
		s.data[cp.GetUuid()] = cp
	}

	return nil
}

// ============ Part Service ============

type InventoryService struct {
	inventorypbv1.UnimplementedInventoryServiceServer
	storage *partStorage
}

func NewInventoryService(storage *partStorage) *InventoryService {
	return &InventoryService{
		storage: storage,
	}
}

func (s *InventoryService) GetPart(
	ctx context.Context,
	req *inventorypbv1.GetPartRequest,
) (*inventorypbv1.GetPartResponse, error) {
	const op string = "InventoryService.GetPart"

	if strings.TrimSpace(req.GetUuid()) == "" {
		return nil, status.Error(codes.InvalidArgument, "uuid must be non-empty")
	}

	part, err := s.storage.PartByID(ctx, req.GetUuid())
	if err != nil {
		if errors.Is(err, ErrPartNotFound) {
			return nil, status.Error(codes.NotFound, "part not found")
		}
		log.Printf("%s: %+v", op, err)
		return nil, status.Error(codes.Internal, "failed to get part")
	}

	return &inventorypbv1.GetPartResponse{
		Part: part,
	}, nil
}

func (s *InventoryService) ListParts(
	ctx context.Context,
	req *inventorypbv1.ListPartsRequest,
) (*inventorypbv1.ListPartsResponse, error) {
	const op string = "InventoryService.ListParts"

	allParts, err := s.storage.ListParts(ctx)
	if err != nil {
		log.Printf("%s: %+v", op, err)
		return nil, status.Error(codes.Internal, "failed to list parts")
	}

	filter := req.GetFilter()
	if isEmptyFilter(filter) {
		return &inventorypbv1.ListPartsResponse{
			Parts: allParts,
		}, nil
	}

	result := make([]*inventorypbv1.Part, 0, len(allParts))
	for _, part := range allParts {
		if matchPart(part, filter) {
			result = append(result, part)
		}
	}

	return &inventorypbv1.ListPartsResponse{
		Parts: result,
	}, nil
}

func isEmptyFilter(f *inventorypbv1.PartsFilter) bool {
	if f == nil {
		return true
	}

	return len(f.GetUuids()) == 0 &&
		len(f.GetNames()) == 0 &&
		len(f.GetCategories()) == 0 &&
		len(f.GetManufacturerCountries()) == 0 &&
		len(f.GetTags()) == 0
}

func matchPart(part *inventorypbv1.Part, f *inventorypbv1.PartsFilter) bool {
	if len(f.GetUuids()) > 0 && !slices.Contains(f.GetUuids(), part.GetUuid()) {
		return false
	}

	if len(f.GetNames()) > 0 && !slices.Contains(f.GetNames(), part.GetName()) {
		return false
	}

	if len(f.GetCategories()) > 0 && !slices.Contains(f.GetCategories(), part.GetCategory()) {
		return false
	}

	if len(f.GetManufacturerCountries()) > 0 {
		country := ""
		if m := part.GetManufacturer(); m != nil {
			country = m.GetCountry()
		}

		if !slices.ContainsFunc(f.GetManufacturerCountries(), func(c string) bool {
			return strings.EqualFold(c, country)
		}) {
			return false
		}
	}

	if len(f.GetTags()) > 0 && !hasIntersection(part.GetTags(), f.GetTags()) {
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

// ============ gRPC Server Run ============

func Run(ctx context.Context, gRPCAddr string, invSvc inventorypbv1.InventoryServiceServer) error {
	const op string = "inventory"

	lis, err := net.Listen("tcp", gRPCAddr)
	if err != nil {
		log.Printf("%s: failed to listen: %v\n", op, err)
		return err
	}
	defer func() {
		if cerr := lis.Close(); cerr != nil {
			log.Printf("%s: failed to close listener: %v\n", op, cerr)
		}
	}()

	s := grpc.NewServer()
	inventorypbv1.RegisterInventoryServiceServer(s, invSvc)

	reflection.Register(s)

	errCh := make(chan error, 1)
	go func() {
		log.Printf("🚀 gRPC server listening on %s\n", gRPCAddr)
		if err := s.Serve(lis); err != nil {
			errCh <- err
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-ctx.Done():
		log.Println("🛑 context cancelled, stopping gRPC server")
	case sig := <-quit:
		log.Printf("🛑 received signal %s, stopping gRPC server", sig)
	case err := <-errCh:
		log.Printf("❌ gRPC server error: %v", err)
	}

	log.Println("🛑 Shutting down gRPC server...")
	s.GracefulStop()
	log.Println("✅ Server stopped")
	return nil
}

func main() {
	ctx := context.Background()

	storage := NewPartStorage()

	if err := seedStorage(ctx, storage); err != nil {
		log.Fatalf("failed to seed storage: %v", err)
	}

	svc := NewInventoryService(storage)

	if err := Run(ctx, GRPCAddr, svc); err != nil {
		log.Fatalf("❌😵‍💫 inventory server stopped with error: %v", err)
	}
}

// =====================================================

func seedStorage(ctx context.Context, storage *partStorage) error {
	now := timestamppb.Now()

	parts := []*inventorypbv1.Part{
		{
			Uuid:          uuid.NewString(),
			Name:          "HyperDrive Engine Mk1",
			Description:   "Основной гипердрайв для малых космических кораблей.",
			Price:         125000.50,
			StockQuantity: 10,
			Category:      inventorypbv1.Category_CATEGORY_ENGINE,
			Dimensions: &inventorypbv1.Dimensions{
				Length: 250.0,
				Width:  180.0,
				Height: 140.0,
				Weight: 3200.0,
			},
			Manufacturer: &inventorypbv1.Manufacturer{
				Name:    "Andromeda Drives Inc.",
				Country: "USA",
				Website: "https://andromeda-drives.example.com",
			},
			Tags: []string{"engine", "hyperdrive", "mk1", "small-ship"},
			Metadata: map[string]*inventorypbv1.Value{
				"max_thrust_kn":  {Value: &inventorypbv1.Value_DoubleValue{DoubleValue: 850.0}},
				"warranty_years": {Value: &inventorypbv1.Value_Int64Value{Int64Value: 5}},
				"military_grade": {Value: &inventorypbv1.Value_BoolValue{BoolValue: true}},
				"fuel_type":      {Value: &inventorypbv1.Value_StringValue{StringValue: "quantum-plasma"}},
			},
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			Uuid:          uuid.NewString(),
			Name:          "Quantum Fuel Cell QF-200",
			Description:   "Топливная ячейка для гипердрайвов серии QF.",
			Price:         7800.0,
			StockQuantity: 120,
			Category:      inventorypbv1.Category_CATEGORY_FUEL,
			Dimensions: &inventorypbv1.Dimensions{
				Length: 80.0,
				Width:  40.0,
				Height: 35.0,
				Weight: 45.0,
			},
			Manufacturer: &inventorypbv1.Manufacturer{
				Name:    "Sirius Energy Systems",
				Country: "Germany",
				Website: "https://sirius-energy.example.com",
			},
			Tags: []string{"fuel", "quantum", "cell", "qf-series"},
			Metadata: map[string]*inventorypbv1.Value{
				"capacity_kwh":      {Value: &inventorypbv1.Value_DoubleValue{DoubleValue: 250.0}},
				"compatible_engine": {Value: &inventorypbv1.Value_StringValue{StringValue: "HyperDrive Engine Mk1"}},
				"hazard_class":      {Value: &inventorypbv1.Value_Int64Value{Int64Value: 3}},
			},
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			Uuid:          uuid.NewString(),
			Name:          "Panoramic Porthole PX-360",
			Description:   "Панорамный иллюминатор с круговым обзором 360°.",
			Price:         15200.0,
			StockQuantity: 35,
			Category:      inventorypbv1.Category_CATEGORY_PORTHOLE,
			Dimensions: &inventorypbv1.Dimensions{
				Length: 120.0,
				Width:  120.0,
				Height: 12.0,
				Weight: 65.0,
			},
			Manufacturer: &inventorypbv1.Manufacturer{
				Name:    "Orion Optics",
				Country: "Japan",
				Website: "https://orion-optics.example.com",
			},
			Tags: []string{"porthole", "glass", "panoramic", "px-360"},
			Metadata: map[string]*inventorypbv1.Value{
				"glass_type":           {Value: &inventorypbv1.Value_StringValue{StringValue: "triplex-titanium"}},
				"max_pressure_bar":     {Value: &inventorypbv1.Value_DoubleValue{DoubleValue: 120.0}},
				"radiation_protection": {Value: &inventorypbv1.Value_BoolValue{BoolValue: true}},
			},
			CreatedAt: now,
			UpdatedAt: now,
		},
	}

	return storage.AddParts(ctx, parts)
}
