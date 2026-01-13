//go:build integration

package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	tc "github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/you-humble/rocket-maintenance/order/internal/model"
	repository "github.com/you-humble/rocket-maintenance/order/internal/repository/order"
	service "github.com/you-humble/rocket-maintenance/order/internal/service/order"
	"github.com/you-humble/rocket-maintenance/platform/db/migrator"
)

const (
	pgImage = "postgres:17.0-alpine3.20"

	pgUser       = "order-service-user"
	pgPass       = "12CXZ43_U_w"
	pgDB         = "order-db"
	migrationDir = "../../migrations"
)

var (
	ctx context.Context

	pgC  *postgres.PostgresContainer
	pool *pgxpool.Pool

	repo service.OrderRepository

	dbURL string
)

func TestIntegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Order Repository Integration Suite")
}

var _ = BeforeSuite(func() {
	ctx = context.Background()

	By("starting postgres container")
	var err error

	pgC, err = postgres.Run(ctx,
		pgImage,
		postgres.WithDatabase(pgDB),
		postgres.WithUsername(pgUser),
		postgres.WithPassword(pgPass),
		tc.WithWaitStrategy(
			wait.ForListeningPort("5432/tcp").WithStartupTimeout(60*time.Second),
		),
	)
	Expect(err).NotTo(HaveOccurred())

	By("building postgres connection string")
	dbURL, err = pgC.ConnectionString(ctx, "sslmode=disable")
	Expect(err).NotTo(HaveOccurred())

	By("creating pgx pool")
	pool, err = pgxpool.New(ctx, dbURL)
	Expect(err).NotTo(HaveOccurred())

	Eventually(func(g Gomega) {
		err := pool.Ping(ctx)
		g.Expect(err).NotTo(HaveOccurred())
	}).WithTimeout(10 * time.Second).WithPolling(200 * time.Millisecond).Should(Succeed())

	migrator := migrator.NewMigrator(
		stdlib.OpenDBFromPool(pool),
		migrationDir,
	)

	By("running migrations")
	migrator.Up()
	defer migrator.Close()

	By("creating repository")
	repo = repository.NewOrderRepository(pool)
})

var _ = AfterSuite(func() {
	if pool != nil {
		pool.Close()
	}
	if pgC != nil {
		_ = pgC.Terminate(ctx)
	}
})

var _ = BeforeEach(func() {
	By("cleaning orders table")
	_, err := pool.Exec(ctx, "TRUNCATE TABLE orders RESTART IDENTITY CASCADE")
	Expect(err).NotTo(HaveOccurred())
})

var _ = Describe("Order repository", func() {
	Context("Create + OrderByID", func() {
		It("creates order row in DB and can be fetched", func() {
			userID := uuid.New()
			partID := uuid.New()

			ord := &model.Order{
				UserID:        userID,
				PartIDs:       []uuid.UUID{partID},
				TotalPrice:    12345,
				TransactionID: nil,
				PaymentMethod: nil,
				Status:        model.StatusPendingPayment,
			}

			By("creating order via repository")
			id, err := repo.Create(ctx, ord)
			Expect(err).NotTo(HaveOccurred())
			Expect(id).NotTo(Equal(uuid.Nil))

			By("checking row exists in DB via direct SQL")
			var (
				gotID         uuid.UUID
				gotUserID     uuid.UUID
				gotPartIDs    []uuid.UUID
				gotTotalPrice int64
				gotTxID       *uuid.UUID
				gotPayMethod  *model.PaymentMethod
				gotStatus     model.OrderStatus
			)

			err = pool.QueryRow(ctx,
				`SELECT id, user_id, part_ids, total_price, transaction_id, payment_method, status
				 FROM orders WHERE id = $1`,
				id,
			).Scan(&gotID, &gotUserID, &gotPartIDs, &gotTotalPrice, &gotTxID, &gotPayMethod, &gotStatus)
			Expect(err).NotTo(HaveOccurred())

			Expect(gotID).To(Equal(id))
			Expect(gotUserID).To(Equal(userID))
			Expect(gotPartIDs).To(Equal([]uuid.UUID{partID}))
			Expect(gotTotalPrice).To(Equal(int64(12345)))
			Expect(gotTxID).To(BeNil())
			Expect(gotPayMethod).To(BeNil())
			Expect(gotStatus).To(Equal(model.StatusPendingPayment))

			By("fetching order via repository OrderByID")
			gotOrd, err := repo.OrderByID(ctx, id)
			Expect(err).NotTo(HaveOccurred())
			Expect(gotOrd.ID).To(Equal(id))
			Expect(gotOrd.UserID).To(Equal(userID))
			Expect(gotOrd.PartIDs).To(Equal([]uuid.UUID{partID}))
			Expect(gotOrd.TotalPrice).To(Equal(int64(12345)))
			Expect(gotOrd.TransactionID).To(BeNil())
			Expect(gotOrd.PaymentMethod).To(BeNil())
			Expect(gotOrd.Status).To(Equal(model.StatusPendingPayment))
		})

		It("OrderByID returns ErrOrderNotFound when missing", func() {
			_, err := repo.OrderByID(ctx, uuid.New())
			Expect(err).To(Equal(model.ErrOrderNotFound))
		})
	})

	Context("Update", func() {
		It("updates status to PAID with transaction_id and payment_method", func() {
			userID := uuid.New()
			partID := uuid.New()

			ord := &model.Order{
				UserID:     userID,
				PartIDs:    []uuid.UUID{partID},
				TotalPrice: 500,
				Status:     model.StatusPendingPayment,
			}

			id, err := repo.Create(ctx, ord)
			Expect(err).NotTo(HaveOccurred())

			txID := uuid.New()
			pm := model.PaymentMethodCard

			By("updating via repository")
			err = repo.Update(ctx, &model.Order{
				ID:            id,
				Status:        model.StatusPaid,
				TransactionID: &txID,
				PaymentMethod: &pm,
			})
			Expect(err).NotTo(HaveOccurred())

			By("verifying DB state via direct SQL")
			var gotStatus model.OrderStatus
			var gotTxID uuid.UUID
			var gotPM model.PaymentMethod

			err = pool.QueryRow(ctx,
				`SELECT status, transaction_id, payment_method FROM orders WHERE id=$1`,
				id,
			).Scan(&gotStatus, &gotTxID, &gotPM)
			Expect(err).NotTo(HaveOccurred())

			Expect(gotStatus).To(Equal(model.StatusPaid))
			Expect(gotTxID).To(Equal(txID))
			Expect(gotPM).To(Equal(pm))
		})

		It("returns error when setting PAID without tx/payment fields", func() {
			userID := uuid.New()
			partID := uuid.New()

			id, err := repo.Create(ctx, &model.Order{
				UserID:     userID,
				PartIDs:    []uuid.UUID{partID},
				TotalPrice: 500,
				Status:     model.StatusPendingPayment,
			})
			Expect(err).NotTo(HaveOccurred())

			err = repo.Update(ctx, &model.Order{
				ID:     id,
				Status: model.StatusPaid,
			})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("requires transaction_id and payment_method"))
		})

		It("returns ErrOrderNotFound when updating missing order", func() {
			txID := uuid.New()
			pm := model.PaymentMethodCard

			err := repo.Update(ctx, &model.Order{
				ID:            uuid.New(),
				Status:        model.StatusPaid,
				TransactionID: &txID,
				PaymentMethod: &pm,
			})
			Expect(err).To(Equal(model.ErrOrderNotFound))
		})
	})
})
