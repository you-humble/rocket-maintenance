package envconfig

import (
	"github.com/IBM/sarama"
	"github.com/caarlos0/env/v11"
)

type kafkaEnv struct {
	Brokers                 []string `env:"KAFKA_BROKERS,required"`
	OrderPaidTopicName      string   `env:"ORDER_PAID_TOPIC_NAME,required"`
	OrderAssembledTopicName string   `env:"ORDER_ASSEMBLED_TOPIC_NAME,required"`
	ConsumerGroupID         string   `env:"ORDER_PAID_CONSUMER_GROUP_ID,required"`
}

type kafka struct {
	raw kafkaEnv
}

func NewKafkaConfig() (*kafka, error) {
	var raw kafkaEnv
	if err := env.Parse(&raw); err != nil {
		return nil, err
	}
	return &kafka{raw: raw}, nil
}

func (cfg *kafka) Brokers() []string           { return cfg.raw.Brokers }
func (cfg *kafka) OrderPaidTopic() string      { return cfg.raw.OrderPaidTopicName }
func (cfg *kafka) OrderAssembledTopic() string { return cfg.raw.OrderAssembledTopicName }
func (cfg *kafka) ConsumerGroupID() string     { return cfg.raw.ConsumerGroupID }

func (cfg *kafka) OrderPaidConsumerConfig() *sarama.Config {
	config := sarama.NewConfig()
	config.Version = sarama.V4_0_0_0
	config.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{sarama.NewBalanceStrategyRoundRobin()}
	config.Consumer.Offsets.Initial = sarama.OffsetOldest

	return config
}

// Config возвращает конфигурацию для sarama consumer
func (cfg *kafka) OrderAssembledProducerConfig() *sarama.Config {
	config := sarama.NewConfig()
	config.Version = sarama.V4_0_0_0
	config.Producer.Return.Successes = true

	return config
}
