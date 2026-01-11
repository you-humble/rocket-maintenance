package app

import (
	"context"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"

	"github.com/you-humble/rocket-maintenance/order/internal/config"
	"github.com/you-humble/rocket-maintenance/platform/closer"
	"github.com/you-humble/rocket-maintenance/platform/logger"
	orderv1 "github.com/you-humble/rocket-maintenance/shared/pkg/openapi/order/v1"
)

type app struct {
	di     *di
	server *http.Server
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
		a.initTables,
		a.initServer,
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

func (a *app) initTables(ctx context.Context) error {
	if err := a.di.Migrator(ctx).Up(); err != nil {
		logger.Error(ctx, "failed to apply migrations", logger.ErrorF(err))
		return err
	}
	return nil
}

func (a *app) initServer(ctx context.Context) error {
	cfg := config.C()

	orderServer, err := orderv1.NewServer(a.di.OrderHandler(ctx))
	if err != nil {
		logger.Error(ctx, "failed to create a new server", logger.ErrorF(err))
		return err
	}

	r := a.di.Router(ctx)
	r.Use(
		middleware.Recoverer,
		middleware.Logger,
	)
	r.Mount("/", orderServer)

	a.server = &http.Server{
		Addr:              cfg.Server.Address(),
		Handler:           r,
		ReadHeaderTimeout: cfg.Server.ReadTimeout(),
	}
	return nil
}

func (a *app) run(ctx context.Context) error {
	defer gracefulShutdown()

	errCh := make(chan error)

	go func() {
		defer close(errCh)

		logger.Info(ctx,
			"🚀 inventory server listening",
			logger.String("address", config.C().Server.Address()),
		)
		err := a.server.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			select {
			case <-ctx.Done():
			case errCh <- err:
			}
		}
	}()

	select {
	case <-ctx.Done():
		logger.Error(ctx, "server context", logger.ErrorF(ctx.Err()))
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
		config.C().Server.ShutdownTimeout(),
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
