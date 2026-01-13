package config

import (
	"errors"
	"fmt"
	"os"

	"github.com/joho/godotenv"

	envconfig "github.com/you-humble/rocket-maintenance/payment/internal/config/env"
)

var cfg *config

type config struct {
	Server Server
	Logger Logger
}

func Load(path ...string) error {
	const op = "config.Load"

	if shouldLoadDotenv() {
		if err := godotenv.Load(path...); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("%s: load .env: %w", op, err)
		}
	}

	serverCfg, err := envconfig.NewGRPCerverConfig()
	if err != nil {
		return fmt.Errorf("%s Server: %w", op, err)
	}

	loggerCfg, err := envconfig.NewLoggerConfig()
	if err != nil {
		return fmt.Errorf("%s Logger: %w", op, err)
	}

	cfg = &config{
		Server: serverCfg,
		Logger: loggerCfg,
	}

	return nil
}

func C() *config { return cfg }

func shouldLoadDotenv() bool {
	return os.Getenv("APP_ENV") == "local"
}
