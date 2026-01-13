//go:build integration

package e2e

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/docker/go-connections/nat"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	tc "github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	tcnetwork "github.com/you-humble/rocket-maintenance/platform/testcontainers/network"
	"github.com/you-humble/rocket-maintenance/platform/testcontainers/path"
	inventorypbv1 "github.com/you-humble/rocket-maintenance/shared/pkg/proto/inventory/v1"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ```bash
// # Очистка кеша тестов
// go clean -testcache

// # Очистка Docker ресурсов
// docker system prune -f
// ```

const (
	projectName = "inventory_e2e"
	mongoImage  = "mongo:8.2.3"

	mongoUser = "inventory_admin"
	mongoPass = "inv123ghU_w"
	mongoAuth = "admin"

	mongoDB         = "inventory-db"
	mongoCollection = "parts"

	grpcPort = "50051"
)

var (
	ctx context.Context

	net        *tcnetwork.Network
	mongoC     tc.Container
	inventoryC tc.Container

	mongoClient *mongo.Client
	partsColl   *mongo.Collection

	grpcConn *grpc.ClientConn
	grpcAddr string

	invClient    inventorypbv1.InventoryServiceClient
	healthClient grpc_health_v1.HealthClient
)

type Category int32

const (
	CategoryUnknown Category = iota
	CategoryEngine
	CategoryFuel
	CategoryPorthole
	CategoryWing
)

type mongoPart struct {
	ID            string             `bson:"_id"`
	Name          string             `bson:"name"`
	Description   string             `bson:"description,omitempty"`
	PriceCents    int64              `bson:"price_cents"`
	StockQuantity int64              `bson:"stock_quantity"`
	Category      Category           `bson:"category"`
	Dimensions    *mongoDimensions   `bson:"dimensions,omitempty"`
	Manufacturer  *mongoManufacturer `bson:"manufacturer,omitempty"`
	Tags          []string           `bson:"tags,omitempty"`
	Metadata      map[string]any     `bson:"metadata,omitempty"`
	CreatedAt     *time.Time         `bson:"created_at,omitempty"`
	UpdatedAt     *time.Time         `bson:"updated_at,omitempty"`
}

type mongoManufacturer struct {
	Name        string `bson:"name"`
	Country     string `bson:"country"`
	CountryNorm string `bson:"country_norm"`
	Website     string `bson:"website,omitempty"`
}

type mongoDimensions struct {
	Length float64 `bson:"length"`
	Width  float64 `bson:"width"`
	Height float64 `bson:"height"`
	Weight float64 `bson:"weight"`
}

func NewFakeMongoPart(category inventorypbv1.Category, country string) mongoPart {
	now := time.Now().UTC()

	return mongoPart{
		ID:            gofakeit.UUID(),
		Name:          gofakeit.ProductName(),
		Description:   gofakeit.Sentence(10),
		PriceCents:    int64(gofakeit.Number(100, 500000)),
		StockQuantity: int64(gofakeit.Number(0, 5000)),
		Category:      Category(category),

		Dimensions: &mongoDimensions{
			Length: gofakeit.Float64Range(1, 500),
			Width:  gofakeit.Float64Range(1, 500),
			Height: gofakeit.Float64Range(1, 500),
			Weight: gofakeit.Float64Range(0.1, 2000),
		},

		Manufacturer: &mongoManufacturer{
			Name:        gofakeit.Company(),
			Country:     country,
			CountryNorm: strings.ToLower(country),
			Website:     gofakeit.URL(),
		},

		Tags: []string{gofakeit.Word(), gofakeit.Word()},
		Metadata: map[string]any{
			"batch":   gofakeit.Int64(),
			"fragile": gofakeit.Bool(),
		},

		CreatedAt: lo.ToPtr(now.Add(-time.Duration(gofakeit.Number(1, 1000)) * time.Hour)),
		UpdatedAt: lo.ToPtr(now),
	}
}

// type TestEnvironment struct {
// 	Network *network.Network
// 	Mongo   *mongo.Container
// 	App     *app.Container
// }

func mustDialGRPC(addr string) *grpc.ClientConn {
	conn, err := grpc.NewClient(
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	Expect(err).NotTo(HaveOccurred())
	return conn
}

func toProtoTimestamp(t time.Time) *timestamppb.Timestamp {
	return timestamppb.New(t)
}

func TestIntegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Inventory E2E Suite")
}

var _ = BeforeSuite(func() {
	ctx = context.Background()
	gofakeit.Seed(0)

	By("creating isolated docker network")
	var err error
	net, err = tcnetwork.NewNetwork(ctx, projectName)
	Expect(err).NotTo(HaveOccurred())

	By("starting mongo container")
	mongoReq := tc.ContainerRequest{
		Image:        mongoImage,
		ExposedPorts: []string{"27017/tcp"},
		Env: map[string]string{
			"MONGO_INITDB_ROOT_USERNAME": mongoUser,
			"MONGO_INITDB_ROOT_PASSWORD": mongoPass,
		},
		Networks: []string{net.Name()},
		NetworkAliases: map[string][]string{
			net.Name(): {"mongo-inventory"},
		},
		WaitingFor: wait.ForListeningPort("27017" + "/tcp").WithStartupTimeout(60 * time.Second),
	}
	mongoC, err = tc.GenericContainer(ctx, tc.GenericContainerRequest{
		ContainerRequest: mongoReq,
		Started:          true,
	})
	Expect(err).NotTo(HaveOccurred())

	By("connecting to mongo via mapped port (from test process)")
	mongoHost, err := mongoC.Host(ctx)
	Expect(err).NotTo(HaveOccurred())

	mongoMapped, err := mongoC.MappedPort(ctx, "27017/tcp")
	Expect(err).NotTo(HaveOccurred())

	mongoURI := fmt.Sprintf(
		"mongodb://%s:%s@%s:%s/?authSource=%s",
		mongoUser, mongoPass, mongoHost, mongoMapped.Port(), mongoAuth,
	)

	mongoClient, err = mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	Expect(err).NotTo(HaveOccurred())
	Expect(mongoClient.Ping(ctx, nil)).To(Succeed())

	partsColl = mongoClient.Database(mongoDB).Collection(mongoCollection)

	By("starting inventory container from Dockerfile via testcontainers build")
	projectRoot := path.GetProjectRoot()

	// ВАЖНО: подставьте путь к Dockerfile ровно как у вас в compose:
	// inventory/cmd/inventory/Dockerfile
	dockerfilePath := filepath.Join("inventory", "cmd", "inventory", "Dockerfile")

	invReq := tc.ContainerRequest{
		FromDockerfile: tc.FromDockerfile{
			Context:    projectRoot,
			Dockerfile: dockerfilePath,
		},
		ExposedPorts: []string{grpcPort + "/tcp"},
		Env: map[string]string{
			"APP_ENV":          "test",
			"GRPC_HOST":        "0.0.0.0",
			"GRPC_PORT":        grpcPort,
			"SHUTDOWN_TIMEOUT": "10s",

			"DB_READ_TIMEOUT":  "5s",
			"DB_WRITE_TIMEOUT": "5s",

			"LOGGER_LEVEL":   "info",
			"LOGGER_AS_JSON": "true",

			"MONGO_HOST":     "mongo-inventory",
			"MONGO_PORT":     "27017",
			"MONGO_DATABASE": mongoDB,

			"MONGO_PARTS_COLLECTION": mongoCollection,

			"MONGO_AUTH_DB":              mongoAuth,
			"MONGO_INITDB_ROOT_USERNAME": mongoUser,
			"MONGO_INITDB_ROOT_PASSWORD": mongoPass,
		},
		Networks:   []string{net.Name()},
		WaitingFor: wait.ForListeningPort(nat.Port(grpcPort + "/tcp")).WithStartupTimeout(90 * time.Second),
	}

	inventoryC, err = tc.GenericContainer(ctx, tc.GenericContainerRequest{
		ContainerRequest: invReq,
		Started:          true,
	})
	Expect(err).NotTo(HaveOccurred())

	By("dialing inventory gRPC from test process")
	invHost, err := inventoryC.Host(ctx)
	Expect(err).NotTo(HaveOccurred())

	invMapped, err := inventoryC.MappedPort(ctx, nat.Port(grpcPort+"/tcp"))
	Expect(err).NotTo(HaveOccurred())

	grpcAddr = fmt.Sprintf("%s:%s", invHost, invMapped.Port())

	grpcConn = mustDialGRPC(grpcAddr)
	invClient = inventorypbv1.NewInventoryServiceClient(grpcConn)
	healthClient = grpc_health_v1.NewHealthClient(grpcConn)

	By("waiting for gRPC health check OK")
	Eventually(func(g Gomega) {
		resp, err := healthClient.Check(ctx, &grpc_health_v1.HealthCheckRequest{})
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(resp.GetStatus()).To(Equal(grpc_health_v1.HealthCheckResponse_SERVING))
	}).WithTimeout(30 * time.Second).WithPolling(500 * time.Millisecond).Should(Succeed())
})

var _ = AfterSuite(func() {
	if grpcConn != nil {
		_ = grpcConn.Close()
	}
	if mongoClient != nil {
		_ = mongoClient.Disconnect(ctx)
	}
	if inventoryC != nil {
		_ = inventoryC.Terminate(ctx)
	}
	if mongoC != nil {
		_ = mongoC.Terminate(ctx)
	}
	if net != nil {
		_ = net.Remove(ctx)
	}
})

var _ = Describe("InventoryService e2e", func() {
	BeforeEach(func() {
		By("cleaning parts collection")
		_, err := partsColl.DeleteMany(ctx, bson.M{})
		Expect(err).NotTo(HaveOccurred())
	})

	Context("GetPart", func() {
		It("returns existing part by uuid", func() {
			p := NewFakeMongoPart(inventorypbv1.Category_CATEGORY_ENGINE, "Germany")

			By("inserting part into mongo")
			_, err := partsColl.InsertOne(ctx, p)
			Expect(err).NotTo(HaveOccurred())

			By("calling gRPC GetPart")
			resp, err := invClient.GetPart(ctx, &inventorypbv1.GetPartRequest{Uuid: p.ID})
			Expect(err).NotTo(HaveOccurred())

			Expect(resp.GetPart().GetUuid()).To(Equal(p.ID))
			Expect(resp.GetPart().GetName()).To(Equal(p.Name))
			Expect(resp.GetPart().GetCategory()).To(Equal(inventorypbv1.Category(p.Category)))
			Expect(resp.GetPart().GetManufacturer().GetCountry()).To(Equal("Germany"))
		})

		It("returns NotFound for missing uuid", func() {
			_, err := invClient.GetPart(ctx, &inventorypbv1.GetPartRequest{Uuid: gofakeit.UUID()})
			Expect(err).To(HaveOccurred())

			st, ok := status.FromError(err)
			Expect(ok).To(BeTrue())
			Expect(st.Code()).To(Equal(codes.NotFound))
		})
	})

	Context("ListParts", func() {
		It("filters by category AND manufacturer_countries", func() {
			target1 := NewFakeMongoPart(inventorypbv1.Category_CATEGORY_ENGINE, "Germany")
			target2 := NewFakeMongoPart(inventorypbv1.Category_CATEGORY_ENGINE, "Germany")

			other1 := NewFakeMongoPart(inventorypbv1.Category_CATEGORY_FUEL, "Germany")
			other2 := NewFakeMongoPart(inventorypbv1.Category_CATEGORY_ENGINE, "France")

			_, err := partsColl.InsertMany(ctx, []any{target1, target2, other1, other2})
			Expect(err).NotTo(HaveOccurred())

			resp, err := invClient.ListParts(ctx, &inventorypbv1.ListPartsRequest{
				Filter: &inventorypbv1.PartsFilter{
					Categories:            []inventorypbv1.Category{inventorypbv1.Category_CATEGORY_ENGINE},
					ManufacturerCountries: []string{"Germany"},
				},
			})
			Expect(err).NotTo(HaveOccurred())

			uuids := map[string]bool{}
			for _, part := range resp.GetParts() {
				uuids[part.GetUuid()] = true
				Expect(part.GetCategory()).To(Equal(inventorypbv1.Category_CATEGORY_ENGINE))
				Expect(part.GetManufacturer().GetCountry()).To(Equal("Germany"))
			}

			Expect(uuids[target1.ID]).To(BeTrue())
			Expect(uuids[target2.ID]).To(BeTrue())
			Expect(uuids[other1.ID]).To(BeFalse())
			Expect(uuids[other2.ID]).To(BeFalse())
		})

		It("filters by uuids OR within the field", func() {
			p1 := NewFakeMongoPart(inventorypbv1.Category_CATEGORY_WING, "USA")
			p2 := NewFakeMongoPart(inventorypbv1.Category_CATEGORY_WING, "USA")
			p3 := NewFakeMongoPart(inventorypbv1.Category_CATEGORY_WING, "USA")

			_, err := partsColl.InsertMany(ctx, []any{p1, p2, p3})
			Expect(err).NotTo(HaveOccurred())

			resp, err := invClient.ListParts(ctx, &inventorypbv1.ListPartsRequest{
				Filter: &inventorypbv1.PartsFilter{
					Uuids: []string{p1.ID, p3.ID},
				},
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(resp.GetParts()).To(HaveLen(2))

			got := map[string]bool{}
			for _, part := range resp.GetParts() {
				got[part.GetUuid()] = true
			}
			Expect(got[p1.ID]).To(BeTrue())
			Expect(got[p2.ID]).To(BeFalse())
			Expect(got[p3.ID]).To(BeTrue())
		})

		It("returns all parts when filter is empty", func() {
			p1 := NewFakeMongoPart(inventorypbv1.Category_CATEGORY_ENGINE, "Japan")
			p2 := NewFakeMongoPart(inventorypbv1.Category_CATEGORY_FUEL, "Canada")

			_, err := partsColl.InsertMany(ctx, []any{p1, p2})
			Expect(err).NotTo(HaveOccurred())

			resp, err := invClient.ListParts(ctx, &inventorypbv1.ListPartsRequest{})
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.GetParts()).To(HaveLen(2))
		})
	})
})
