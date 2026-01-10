package envconfig

import (
	"fmt"

	"github.com/caarlos0/env/v11"
)

type mongoEnv struct {
	Host            string `env:"MONGO_HOST,required"`
	Port            int    `env:"MONGO_PORT,required"`
	User            string `env:"MONGO_INITDB_ROOT_USERNAME,required"`
	Password        string `env:"MONGO_INITDB_ROOT_PASSWORD,required"`
	DBName          string `env:"MONGO_DATABASE,required"`
	AuthDB          string `env:"MONGO_AUTH_DB,required"`
	PartsCollection string `env:"MONGO_PARTS_COLLECTION,required"`
}

type mongo struct {
	raw mongoEnv
}

func NewMongoConfig() (*mongo, error) {
	var raw mongoEnv
	if err := env.Parse(&raw); err != nil {
		return nil, err
	}
	return &mongo{raw: raw}, nil
}

func (cfg *mongo) DatabaseName() string {
	return cfg.raw.DBName
}

func (cfg *mongo) PartsCollection() string {
	return cfg.raw.PartsCollection
}

func (cfg *mongo) DSN() string {
	return fmt.Sprintf(
		"mongodb://%s:%s@%s:%d/%s?authSource=%s",
		cfg.raw.User,
		cfg.raw.Password,
		cfg.raw.Host,
		cfg.raw.Port,
		cfg.raw.DBName,
		cfg.raw.AuthDB,
	)
}
