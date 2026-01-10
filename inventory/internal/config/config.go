package config

import (
	"errors"
	"fmt"
	"os"

	"github.com/joho/godotenv"

	envconfig "github.com/you-humble/rocket-maintenance/inventory/internal/config/env"
)

var cfg *config

type config struct {
	Server Server
	Logger Logger
	Mongo  Database
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

	loggerCfg, err := envconfig.NewLoggerConfig()
	if err != nil {
		return fmt.Errorf("%s Logger: %w", op, err)
	}

	mongoCfg, err := envconfig.NewMongoConfig()
	if err != nil {
		return fmt.Errorf("%s Mongo: %w", op, err)
	}

	cfg = &config{
		Server: serverCfg,
		Logger: loggerCfg,
		Mongo:  mongoCfg,
	}

	return nil
}

func C() *config { return cfg }

func shouldLoadDotenv() bool {
	return os.Getenv("APP_ENV") == "local"
}
