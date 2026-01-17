package config

import "github.com/IBM/sarama"

type Kafka interface {
	Brokers() []string
	OrderPaidTopic() string
	OrderPaidConsumerGroupID() string
	OrderAssembledTopic() string
	OrderAssembledConsumerGroupID() string
	OrderPaidConsumerConfig() *sarama.Config
	OrderAssembledConsumerConfig() *sarama.Config
}

type Telegram interface {
	BotToken() string
}

type Logger interface {
	Level() string
	AsJSON() bool
}
