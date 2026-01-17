package kafka

import (
	"context"
)

type (
	Middleware     func(next MessageHandler) MessageHandler
	MessageHandler func(ctx context.Context, msg Message) error
)

type Consumer interface {
	Consume(ctx context.Context, handler MessageHandler) error
}

type Producer interface {
	Send(ctx context.Context, key, value []byte) error
}
