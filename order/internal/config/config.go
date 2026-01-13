package config

import (
	"errors"
	"fmt"
	"os"

	"github.com/joho/godotenv"

	envconfig "github.com/you-humble/rocket-maintenance/order/internal/config/env"
)

var cfg *config

type config struct {
	Server    Server
	Inventory Client
	Payment   Client
	Logger    Logger
	Postgres  Database
}

func Load(path ...string) error {
	const op = "config.Load"

	if shouldLoadDotenv() {
		if err := godotenv.Load(path...); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("%s: load .env: %w", op, err)
		}
	}

	serverCfg, err := envconfig.NewHTTPServerConfig()
	if err != nil {
		return fmt.Errorf("%s Server: %w", op, err)
	}

	inventoryCfg, err := envconfig.NewInventoryConfig()
	if err != nil {
		return fmt.Errorf("%s Inventory: %w", op, err)
	}

	paymentCfg, err := envconfig.NewPaymentConfig()
	if err != nil {
		return fmt.Errorf("%s Payment: %w", op, err)
	}

	loggerCfg, err := envconfig.NewLoggerConfig()
	if err != nil {
		return fmt.Errorf("%s Logger: %w", op, err)
	}

	postgresCfg, err := envconfig.NewPostgresConfig()
	if err != nil {
		return fmt.Errorf("%s Postgres: %w", op, err)
	}

	cfg = &config{
		Server:    serverCfg,
		Inventory: inventoryCfg,
		Payment:   paymentCfg,
		Logger:    loggerCfg,
		Postgres:  postgresCfg,
	}

	return nil
}

func C() *config { return cfg }

func shouldLoadDotenv() bool {
	return os.Getenv("APP_ENV") == "local"
}
