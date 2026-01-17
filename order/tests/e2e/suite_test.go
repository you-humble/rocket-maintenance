//go:build integration

package e2e

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/IBM/sarama"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	tc "github.com/testcontainers/testcontainers-go"
	kafkaTc "github.com/testcontainers/testcontainers-go/modules/kafka"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"google.golang.org/protobuf/proto"

	"github.com/you-humble/rocket-maintenance/order/internal/app"
	"github.com/you-humble/rocket-maintenance/order/internal/converter"
	"github.com/you-humble/rocket-maintenance/order/internal/model"
	repository "github.com/you-humble/rocket-maintenance/order/internal/repository/order"
	ordconsumer "github.com/you-humble/rocket-maintenance/order/internal/service/consumer/order"
	service "github.com/you-humble/rocket-maintenance/order/internal/service/order"
	ordproducer "github.com/you-humble/rocket-maintenance/order/internal/service/producer/order"
	"github.com/you-humble/rocket-maintenance/platform/db/migrator"
	"github.com/you-humble/rocket-maintenance/platform/kafka"
	"github.com/you-humble/rocket-maintenance/platform/kafka/consumer"
	"github.com/you-humble/rocket-maintenance/platform/kafka/middleware"
	"github.com/you-humble/rocket-maintenance/platform/kafka/producer"
	"github.com/you-humble/rocket-maintenance/platform/logger"
	assemblypbv1 "github.com/you-humble/rocket-maintenance/shared/pkg/proto/assembly/v1"
)

// ```bash
// # Очистка кеша тестов
// go clean -testcache

// # Очистка Docker ресурсов
// docker system prune -f
// ```

const (
	pgImage = "postgres:17.0-alpine3.20"

	pgUser       = "order-service-user"
	pgPass       = "12CXZ43_U_w"
	pgDB         = "order-db"
	migrationDir = "../../migrations"

	kafkaImage = "confluentinc/cp-kafka:7.6.1"

	topicPaid       = "order.paid"
	topicAssembled  = "order.assembled"
	consumerGroupID = "order-group-order-assembled"
)

var (
	ctx context.Context

	pgC   *postgres.PostgresContainer
	pool  *pgxpool.Pool
	dbURL string

	kafkaC       tc.Container
	kafkaBrokers []string

	repo        service.OrderRepository
	ordSvc      app.OrderService
	ordConsumer app.OrderConsumer
)

func TestIntegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Order Repository Integration Suite")
}

var _ = BeforeSuite(func() {
	ctx = context.Background()

	By("starting postgres container")
	var err error
	logger.SetNopLogger()
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
	err = migrator.Up()
	Expect(err).NotTo(HaveOccurred())
	defer migrator.Close()

	By("starting kafka container (cp-kafka)")
	kafkaC, kafkaBrokers, err = runKafka(ctx)
	Expect(err).NotTo(HaveOccurred())

	By("setting env for app config (Kafka brokers/topics/group)")
	Expect(os.Setenv("KAFKA_BROKERS", kafkaBrokers[0])).To(Succeed())
	Expect(os.Setenv("ORDER_ASSEMBLED_CONSUMER_GROUP_ID", "order-service-it")).To(Succeed())
	Expect(os.Setenv("ORDER_PAID_TOPIC_NAME", topicPaid)).To(Succeed())
	Expect(os.Setenv("ORDER_ASSEMBLED_TOPIC_NAME", topicAssembled)).To(Succeed())

	By("creating kafka topics")
	Expect(createTopics(ctx, kafkaBrokers, topicPaid, topicAssembled)).To(Succeed())

	By("creating repository")
	repo = repository.NewOrderRepository(pool)

	orderPaidProducerConfig := sarama.NewConfig()
	orderPaidProducerConfig.Version = sarama.V4_0_0_0
	orderPaidProducerConfig.Producer.Return.Successes = true

	p, err := sarama.NewSyncProducer(kafkaBrokers, orderPaidProducerConfig)
	Expect(err).NotTo(HaveOccurred())

	opProducer := producer.NewProducer(p, topicPaid, logger.L())
	conv := converter.NewKafkaCoverter()

	producer := ordproducer.NewOrderProducer(opProducer, conv)

	paymentClient := newStubPaymentClient()
	ordSvc = service.NewOrderService(repo, nil, paymentClient, producer, 2*time.Second, 2*time.Second)

	orderAssembledConsumerConfig := sarama.NewConfig()
	orderAssembledConsumerConfig.Version = sarama.V4_0_0_0
	orderAssembledConsumerConfig.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{sarama.NewBalanceStrategyRoundRobin()}
	orderAssembledConsumerConfig.Consumer.Offsets.Initial = sarama.OffsetOldest

	consumerGr, err := sarama.NewConsumerGroup(
		kafkaBrokers,
		consumerGroupID,
		orderAssembledConsumerConfig,
	)
	Expect(err).NotTo(HaveOccurred())

	oaConsumer := consumer.NewConsumer(
		consumerGr,
		[]string{
			topicAssembled,
		},
		logger.L(),
		middleware.Recovery(logger.L()),
		middleware.Logging(logger.L()),
	)

	ordConsumer = ordconsumer.NewOrderConsumer(oaConsumer, conv, ordSvc)
	By("starting order assembled consumer in background")
	consumerErrCh := make(chan error)
	go func() {
		consumerErrCh <- ordConsumer.RunShipAssembledConsume(ctx)
	}()
	Consistently(consumerErrCh, 2*time.Second).ShouldNot(Receive())
})

var _ = AfterSuite(func() {
	if pool != nil {
		pool.Close()
	}
	if pgC != nil {
		_ = pgC.Terminate(ctx)
	}
	mustTerminate(ctx, kafkaC)
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

		It("completes order after assembled event is consumed", func() {
			userID := uuid.New()
			partID := uuid.New()

			By("creating order")
			id, err := repo.Create(ctx, &model.Order{
				UserID:     userID,
				PartIDs:    []uuid.UUID{partID},
				TotalPrice: 12345,
				Status:     model.StatusPendingPayment,
			})
			Expect(err).NotTo(HaveOccurred())

			By("preparing assembled payload using the same converter as app")
			btSec := 100 * time.Millisecond
			pb := &assemblypbv1.AssembledShipRecord{
				EventUuid:    uuid.NewString(),
				OrderUuid:    id.String(),
				UserUuid:     userID.String(),
				BuildTimeSec: int64(btSec),
			}

			assembledPayload, err := proto.Marshal(pb)
			Expect(err).NotTo(HaveOccurred())

			By("simulating assembler service: consumes paid and produces assembled")
			errCh := make(chan error, 1)
			go func() {
				errCh <- simulateAssemblerOnce(
					ctx,
					kafkaBrokers,
					id,
					assembledPayload,
				)
			}()

			By("marking order as PAID and sending paid event")
			res, err := ordSvc.Pay(ctx, model.PayOrderParams{
				ID:            id,
				UserID:        userID,
				PaymentMethod: model.PaymentMethodCard,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(res).NotTo(BeNil())
			Expect(res.TransactionID).NotTo(Equal(uuid.Nil))

			By("waiting until order becomes COMPLETED in DB")
			Eventually(func(g Gomega) {
				got, err := repo.OrderByID(ctx, id)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(got.Status).To(Equal(model.StatusCompleted))
			}).WithTimeout(15 * time.Second).WithPolling(200 * time.Millisecond).Should(Succeed())
		})
	})
})

func runKafka(ctx context.Context) (tc.Container, []string, error) {
	c, err := kafkaTc.Run(ctx,
		kafkaImage,
		kafkaTc.WithClusterID("Mk3OEYBSD34fcwNTJENDM2Qk"),
	)
	if err != nil {
		return nil, []string{}, err
	}

	bootstrap, err := c.Brokers(ctx)
	if err != nil {
		_ = c.Terminate(ctx)
		return nil, []string{}, err
	}

	return c, bootstrap, nil
}

func mustTerminate(ctx context.Context, c tc.Container) {
	if c != nil {
		_ = c.Terminate(ctx)
	}
}

func createTopics(_ context.Context, brokers []string, topics ...string) error {
	cfg := sarama.NewConfig()
	cfg.Version = sarama.V4_0_0_0
	cfg.Producer.Return.Successes = true
	cfg.Admin.Timeout = 10 * time.Second

	admin, err := sarama.NewClusterAdmin(brokers, cfg)
	if err != nil {
		return err
	}
	defer admin.Close()

	for _, t := range topics {
		err := admin.CreateTopic(t, &sarama.TopicDetail{
			NumPartitions:     1,
			ReplicationFactor: 1,
		}, false)
		if err != nil && !errors.Is(err, sarama.ErrTopicAlreadyExists) {
			return err
		}
	}
	return nil
}

func simulateAssemblerOnce(
	ctx context.Context,
	brokers []string,
	orderID uuid.UUID,
	assembledPayload []byte,
) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	cfg := sarama.NewConfig()
	cfg.Version = sarama.V4_0_0_0
	cfg.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{sarama.NewBalanceStrategyRoundRobin()}
	cfg.Consumer.Offsets.Initial = sarama.OffsetOldest
	cfg.Consumer.Return.Errors = true
	cfg.Producer.Return.Successes = true
	cfg.Producer.RequiredAcks = sarama.WaitForAll

	consumerGr, err := sarama.NewConsumerGroup(
		brokers,
		consumerGroupID,
		cfg,
	)
	if err != nil {
		return err
	}
	defer consumerGr.Close()

	c := consumer.NewConsumer(
		consumerGr,
		[]string{
			topicPaid,
		},
		logger.L(),
	)

	prod, err := sarama.NewSyncProducer(brokers, cfg)
	if err != nil {
		return err
	}
	defer prod.Close()

	errCh := make(chan error, 1)
	go func() {
		errCh <- c.Consume(ctx, func(ctx context.Context, msg kafka.Message) error {
			if msg.Value == nil {
				return errors.New("msg is nil")
			}
			time.Sleep(100 * time.Millisecond)
			_, _, err := prod.SendMessage(&sarama.ProducerMessage{
				Topic: topicAssembled,
				Key:   sarama.ByteEncoder(orderID[:]),
				Value: sarama.ByteEncoder(assembledPayload),
			})
			if err != nil {
				return err
			}

			cancel()
			return nil
		})
	}()

	select {
	case err := <-errCh:
		if err != nil &&
			!errors.Is(err, context.Canceled) &&
			!errors.Is(err, context.DeadlineExceeded) {
			return err
		}
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

type stubPaymentClient struct{}

func newStubPaymentClient() *stubPaymentClient { return &stubPaymentClient{} }

func (c *stubPaymentClient) PayOrder(ctx context.Context, params model.PayOrderParams) (string, error) {
	return uuid.NewString(), nil
}
