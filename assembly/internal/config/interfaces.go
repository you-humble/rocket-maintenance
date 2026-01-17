package config

import "github.com/IBM/sarama"

type Kafka interface {
	Brokers() []string
	OrderPaidTopic() string
	OrderAssembledTopic() string
	ConsumerGroupID() string
	OrderPaidConsumerConfig() *sarama.Config
	OrderAssembledProducerConfig() *sarama.Config
}

type Logger interface {
	Level() string
	AsJSON() bool
}
