package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/you-humble/rocket-maintenance/assembly/internal/converter"
	"github.com/you-humble/rocket-maintenance/assembly/internal/model"
	"github.com/you-humble/rocket-maintenance/platform/kafka"
	"github.com/you-humble/rocket-maintenance/platform/logger"
)

type fakeConsumer struct {
	consumeFn func(ctx context.Context, handler func(context.Context, kafka.Message) error) error
}

func (c fakeConsumer) Consume(ctx context.Context, handler kafka.MessageHandler) error {
	return c.consumeFn(ctx, handler)
}

type fakeProducer struct {
	sendFn func(ctx context.Context, key, value []byte) error

	calls int
	lastK []byte
	lastV []byte
}

func (p *fakeProducer) Send(ctx context.Context, key, value []byte) error {
	p.calls++
	p.lastK = append([]byte(nil), key...)
	p.lastV = append([]byte(nil), value...)
	if p.sendFn == nil {
		return nil
	}
	return p.sendFn(ctx, key, value)
}

type fakeConverter struct {
	paidOrderToModelFn       func([]byte) (model.PaidOrder, error)
	assembledShipToPayloadFn func(model.AssembledShip) ([]byte, error)
}

func (c fakeConverter) PaidOrderToModel(b []byte) (model.PaidOrder, error) {
	return c.paidOrderToModelFn(b)
}

func (c fakeConverter) AssembledShipToPayload(m model.AssembledShip) ([]byte, error) {
	return c.assembledShipToPayloadFn(m)
}

func TestService_Run_Table(t *testing.T) {
	t.Parallel()

	logger.SetNopLogger()
	wantErr := errors.New("consume error")

	tests := []struct {
		name    string
		consume func(ctx context.Context, handler func(context.Context, kafka.Message) error) error
		wantErr error
	}{
		{
			name: "success",
			consume: func(ctx context.Context, handler func(context.Context, kafka.Message) error) error {
				return nil
			},
			wantErr: nil,
		},
		{
			name: "consumer error returned",
			consume: func(ctx context.Context, handler func(context.Context, kafka.Message) error) error {
				return wantErr
			},
			wantErr: wantErr,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			s := NewAssemblyService(
				fakeConsumer{consumeFn: tt.consume},
				&fakeProducer{},
				converter.NewKafkaCoverter(),
			)

			err := s.Run(context.Background())
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected err=%v, got=%v", tt.wantErr, err)
			}
		})
	}
}

func TestService_PaidOrderHandler_Table(t *testing.T) {
	t.Parallel()

	logger.SetNopLogger()
	convDecodeErr := errors.New("decode err")
	convEncodeErr := errors.New("encode err")
	prodErr := errors.New("send err")

	var event model.PaidOrder

	tests := []struct {
		name           string
		ctx            func() (context.Context, context.CancelFunc)
		delay          time.Duration
		timerImmediate bool

		paidOrderToModelErr   error
		assembledToPayloadErr error
		producerErr           error

		wantErrIs     error
		wantSendCalls int
	}{
		{
			name: "success (delay=0) -> send once",
			ctx: func() (context.Context, context.CancelFunc) {
				return context.WithCancel(context.Background())
			},
			delay:          0,
			timerImmediate: true,
			wantErrIs:      nil,
			wantSendCalls:  1,
		},
		{
			name: "context canceled before timer fires -> ctx error, no send",
			ctx: func() (context.Context, context.CancelFunc) {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx, func() {}
			},
			delay:          10 * time.Second,
			timerImmediate: false,
			wantErrIs:      context.Canceled,
			wantSendCalls:  0,
		},
		{
			name: "converter PaidOrderToModel error -> no send",
			ctx: func() (context.Context, context.CancelFunc) {
				return context.WithCancel(context.Background())
			},
			delay:               0,
			timerImmediate:      true,
			paidOrderToModelErr: convDecodeErr,
			wantErrIs:           convDecodeErr,
			wantSendCalls:       0,
		},
		{
			name: "converter AssembledShipToPayload error -> no send",
			ctx: func() (context.Context, context.CancelFunc) {
				return context.WithCancel(context.Background())
			},
			delay:                 0,
			timerImmediate:        true,
			assembledToPayloadErr: convEncodeErr,
			wantErrIs:             convEncodeErr,
			wantSendCalls:         0,
		},
		{
			name: "producer send error -> returned",
			ctx: func() (context.Context, context.CancelFunc) {
				return context.WithCancel(context.Background())
			},
			delay:          0,
			timerImmediate: true,
			producerErr:    prodErr,
			wantErrIs:      prodErr,
			wantSendCalls:  1,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx, cancel := tt.ctx()
			defer cancel()

			prod := &fakeProducer{
				sendFn: func(ctx context.Context, key, value []byte) error {
					return tt.producerErr
				},
			}

			s := NewAssemblyService(fakeConsumer{
				consumeFn: func(ctx context.Context, handler func(context.Context, kafka.Message) error) error {
					return nil
				},
			}, prod, converter.NewKafkaCoverter())

			s.delay = tt.delay
			s.conv = fakeConverter{
				paidOrderToModelFn: func(b []byte) (model.PaidOrder, error) {
					if tt.paidOrderToModelErr != nil {
						return model.PaidOrder{}, tt.paidOrderToModelErr
					}
					return event, nil
				},
				assembledShipToPayloadFn: func(m model.AssembledShip) ([]byte, error) {
					if tt.assembledToPayloadErr != nil {
						return nil, tt.assembledToPayloadErr
					}
					return []byte("payload"), nil
				},
			}

			// Ускоряем/управляем таймером: либо "мгновенный", либо настоящий (для теста ctx cancel).
			if tt.timerImmediate {
				s.newTimer = func(d time.Duration) *time.Timer {
					return time.NewTimer(0)
				}
			} else {
				s.newTimer = time.NewTimer
			}

			msg := kafka.Message{
				Topic:     "order.paid",
				Partition: 1,
				Offset:    10,
				Value:     []byte("paid-order-bytes"),
			}

			err := s.paidOrderHandler(ctx, msg)
			if tt.wantErrIs == nil {
				if err != nil {
					t.Fatalf("expected nil err, got=%v", err)
				}
			} else {
				if !errors.Is(err, tt.wantErrIs) {
					t.Fatalf("expected err is=%v, got=%v", tt.wantErrIs, err)
				}
			}

			if prod.calls != tt.wantSendCalls {
				t.Fatalf("expected producer calls=%d, got=%d", tt.wantSendCalls, prod.calls)
			}
		})
	}
}
