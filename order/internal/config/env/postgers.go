package envconfig

import (
	"fmt"

	"github.com/caarlos0/env/v11"
)

type postgresEnv struct {
	Host          string `env:"POSTGRES_HOST,required"`
	Port          int    `env:"POSTGRES_PORT,required"`
	User          string `env:"POSTGRES_USER,required"`
	Password      string `env:"POSTGRES_PASSWORD,required"`
	DBName        string `env:"POSTGRES_DB,required"`
	SSLMode       string `env:"POSTGRES_SSL_MODE,required"`
	MigrationsDir string `env:"MIGRATION_DIRECTORY,required"`
}

type postgres struct {
	raw postgresEnv
}

func NewPostgresConfig() (*postgres, error) {
	var raw postgresEnv
	if err := env.Parse(&raw); err != nil {
		return nil, err
	}
	return &postgres{raw: raw}, nil
}

func (cfg *postgres) MigrationDirectory() string {
	return cfg.raw.MigrationsDir
}

func (cfg *postgres) DSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.raw.User,
		cfg.raw.Password,
		cfg.raw.Host,
		cfg.raw.Port,
		cfg.raw.DBName,
		cfg.raw.SSLMode,
	)
}
