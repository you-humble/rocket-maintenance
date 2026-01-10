package envconfig

import (
	"fmt"
	"time"

	"github.com/caarlos0/env/v11"
)

type httpServerEnv struct {
	Host string `env:"GRPC_HOST,required"`
	Port int    `env:"GRPC_PORT,required"`

	DBReadTimeout time.Duration `env:"DB_READ_TIMEOUT,required"`
}

type httpServer struct {
	raw httpServerEnv
}

func NewHTTPServerConfig() (*httpServer, error) {
	var raw httpServerEnv
	if err := env.Parse(&raw); err != nil {
		return nil, err
	}
	return &httpServer{raw: raw}, nil
}

func (cfg *httpServer) Host() string { return cfg.raw.Host }
func (cfg *httpServer) Port() int    { return cfg.raw.Port }
func (cfg *httpServer) Address() string {
	return fmt.Sprintf("%s:%d", cfg.Host(), cfg.Port())
}

func (cfg *httpServer) BDEReadTimeout() time.Duration {
	return cfg.raw.DBReadTimeout
}
