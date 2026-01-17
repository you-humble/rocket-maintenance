package config

import (
	"time"

	"github.com/IBM/sarama"
)

type Client interface {
	Host() string
	Port() int
	Address() string
}

type Server interface {
	Client
	ReadTimeout() time.Duration
	ShutdownTimeout() time.Duration
	BDEReadTimeout() time.Duration
	DBWriteTimeout() time.Duration
}

type Logger interface {
	Level() string
	AsJSON() bool
}

type Database interface {
	MigrationDirectory() string
	DSN() string
}

type Kafka interface {
	Brokers() []string
	OrderPaidTopic() string
	OrderAssembledTopic() string
	ConsumerGroupID() string
	OrderAssembledConsumerConfig() *sarama.Config
	OrderPaidProducerConfig() *sarama.Config
}
