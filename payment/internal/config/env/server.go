package envconfig

import (
	"fmt"

	"github.com/caarlos0/env/v11"
)

type grpcServerEnv struct {
	Host string `env:"GRPC_HOST,required"`
	Port int    `env:"GRPC_PORT,required"`
}

type grpcServer struct {
	raw grpcServerEnv
}

func NewGRPCerverConfig() (*grpcServer, error) {
	var raw grpcServerEnv
	if err := env.Parse(&raw); err != nil {
		return nil, err
	}
	return &grpcServer{raw: raw}, nil
}

func (cfg *grpcServer) Host() string { return cfg.raw.Host }
func (cfg *grpcServer) Port() int    { return cfg.raw.Port }
func (cfg *grpcServer) Address() string {
	return fmt.Sprintf("%s:%d", cfg.Host(), cfg.Port())
}
