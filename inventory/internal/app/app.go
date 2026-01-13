package app

import (
	"context"
	"errors"
	"net"

	"google.golang.org/grpc"

	"github.com/you-humble/rocket-maintenance/inventory/internal/config"
	repository "github.com/you-humble/rocket-maintenance/inventory/internal/repository/part"
	"github.com/you-humble/rocket-maintenance/platform/closer"
	"github.com/you-humble/rocket-maintenance/platform/logger"
)

type app struct {
	di       *di
	listener net.Listener
	server   *grpc.Server
}

func New(ctx context.Context) (*app, error) {
	a := &app{}

	if err := a.init(ctx); err != nil {
		return nil, err
	}

	return a, nil
}

func (a *app) Run(ctx context.Context) error { return a.run(ctx) }

func (a *app) init(ctx context.Context) error {
	inits := []func(context.Context) error{
		a.initConfig,
		a.initLogger,
		a.initCloser,
		a.initDI,
		a.initListener,
		a.initServer,
		a.initPartsData,
	}

	for _, initFn := range inits {
		if err := initFn(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (a *app) initConfig(_ context.Context) error {
	return config.Load()
}

func (a *app) initLogger(_ context.Context) error {
	return logger.Init(
		config.C().Logger.Level(),
		config.C().Logger.AsJSON(),
	)
}

func (a *app) initCloser(_ context.Context) error {
	closer.SetLogger(logger.L())
	return nil
}

func (a *app) initDI(_ context.Context) error {
	a.di = NewDI()
	return nil
}

func (a *app) initPartsData(ctx context.Context) error {
	err := repository.PartsBootstrap(ctx, a.di.PartsRepository(ctx))
	if err != nil {
		logger.Error(ctx, "failed to bootstrap", logger.ErrorF(err))
		return err
	}
	return nil
}

func (a *app) initListener(ctx context.Context) error {
	lis, err := net.Listen("tcp", config.C().Server.Address())
	if err != nil {
		logger.Error(ctx, "failed to listen", logger.ErrorF(err))
		return err
	}
	closer.AddNamed("TCP listener",
		func(ctx context.Context) error {
			lerr := lis.Close()
			if lerr != nil && !errors.Is(lerr, net.ErrClosed) {
				return lerr
			}
			return nil
		})

	a.listener = lis
	return nil
}

func (a *app) initServer(ctx context.Context) error {
	a.server = a.di.Server(ctx)
	return nil
}

func (a *app) run(ctx context.Context) error {
	defer gracefulShutdown(ctx, a.server)

	errCh := make(chan error)

	go func() {
		defer close(errCh)

		logger.Info(ctx,
			"🚀 order server listening",
			logger.String("address", config.C().Server.Address()),
		)
		err := a.server.Serve(a.listener)
		if err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			select {
			case <-ctx.Done():
			case errCh <- err:
			}
		}
	}()

	select {
	case <-ctx.Done():
		logger.Error(ctx, "🛑 server context cancelled", logger.ErrorF(ctx.Err()))
		return ctx.Err()
	case err, ok := <-errCh:
		if !ok {
			return nil
		}
		return err
	}
}

//nolint:contextcheck
func gracefulShutdown(ctx context.Context, s *grpc.Server) {
	logger.Info(ctx, "🛑 Shutting down gRPC server...")
	s.GracefulStop()
	logger.Info(ctx, "✅ Server stopped")
}
