package envconfig

import (
	"github.com/IBM/sarama"
	"github.com/caarlos0/env/v11"
)

type kafkaEnv struct {
	Brokers                       []string `env:"KAFKA_BROKERS,required"`
	OrderPaidTopicName            string   `env:"ORDER_PAID_TOPIC_NAME,required"`
	OrderPaidConsumerGroupID      string   `env:"ORDER_PAID_CONSUMER_GROUP_ID,required"`
	OrderAssembledTopicName       string   `env:"ORDER_ASSEMBLED_TOPIC_NAME,required"`
	OrderAssembledConsumerGroupID string   `env:"ORDER_ASSEMBLED_CONSUMER_GROUP_ID,required"`
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

func (cfg *kafka) Brokers() []string                { return cfg.raw.Brokers }
func (cfg *kafka) OrderPaidTopic() string           { return cfg.raw.OrderPaidTopicName }
func (cfg *kafka) OrderPaidConsumerGroupID() string { return cfg.raw.OrderPaidConsumerGroupID }
func (cfg *kafka) OrderAssembledTopic() string      { return cfg.raw.OrderAssembledTopicName }
func (cfg *kafka) OrderAssembledConsumerGroupID() string {
	return cfg.raw.OrderAssembledConsumerGroupID
}

func (cfg *kafka) OrderPaidConsumerConfig() *sarama.Config {
	config := sarama.NewConfig()
	config.Version = sarama.V4_0_0_0
	config.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{sarama.NewBalanceStrategyRoundRobin()}
	config.Consumer.Offsets.Initial = sarama.OffsetOldest

	return config
}

func (cfg *kafka) OrderAssembledConsumerConfig() *sarama.Config {
	config := sarama.NewConfig()
	config.Version = sarama.V4_0_0_0
	config.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{sarama.NewBalanceStrategyRoundRobin()}
	config.Consumer.Offsets.Initial = sarama.OffsetOldest

	return config
}
