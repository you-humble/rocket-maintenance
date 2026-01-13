package app

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	service "github.com/you-humble/rocket-maintenance/payment/internal/service/payment"
	"github.com/you-humble/rocket-maintenance/payment/internal/transport/grpc/interceptors"
	tgrpc "github.com/you-humble/rocket-maintenance/payment/internal/transport/grpc/payment/v1"
	"github.com/you-humble/rocket-maintenance/platform/grpc/health"
	paymentpbv1 "github.com/you-humble/rocket-maintenance/shared/pkg/proto/payment/v1"
)

type di struct {
	service tgrpc.PaymentService
	handler paymentpbv1.PaymentServiceServer

	server *grpc.Server
}

func NewDI() *di { return &di{} }

func (d *di) PaymentService(ctx context.Context) tgrpc.PaymentService {
	if d.service == nil {
		d.service = service.NewPaymentService()
	}

	return d.service
}

func (d *di) PaymentHandler(ctx context.Context) paymentpbv1.PaymentServiceServer {
	if d.handler == nil {
		d.handler = tgrpc.NewPaymentHandler(d.PaymentService(ctx))
	}

	return d.handler
}

func (d *di) Server(ctx context.Context) *grpc.Server {
	if d.server == nil {
		d.server = grpc.NewServer(
			grpc.ChainUnaryInterceptor(
				interceptors.UnaryLogging(),
				interceptors.RejectNilRequest(),
			),
		)
		paymentpbv1.RegisterPaymentServiceServer(d.server, d.PaymentHandler(ctx))

		reflection.Register(d.server)

		health.RegisterService(d.server)
	}

	return d.server
}
