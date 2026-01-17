package app

import (
	"context"
	"time"

	"github.com/you-humble/rocket-maintenance/assembly/internal/config"
	"github.com/you-humble/rocket-maintenance/platform/closer"
	"github.com/you-humble/rocket-maintenance/platform/logger"
)

type app struct {
	di *di
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

func (a *app) run(ctx context.Context) error {
	defer gracefulShutdown()

	errCh := make(chan error)

	go func() {
		logger.Info(ctx, "🚀 assembly server running")
		if err := a.di.AssemblyService(ctx).RunOrderPaidConsume(ctx); err != nil {
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
func gracefulShutdown() {
	ctx, cancel := context.WithTimeout(
		context.Background(), // do not inherit cancellation from ctx
		10*time.Second,
	)
	defer cancel()

	err := closer.CloseAll(ctx)
	if err != nil {
		logger.Error(ctx, "❌ Error during server shutdown", logger.ErrorF(err))
		logger.Error(ctx, "❌😵‍💫 Server stopped")
		return
	}
	logger.Info(ctx, "✅ Server stopped")
}
