package envconfig

import (
	"fmt"

	"github.com/caarlos0/env/v11"
)

// ======= Inventory =======

type inventoryEnv struct {
	GRPCHost string `env:"INVENTORY_GRPC_HOST,required"`
	GRPCPort int    `env:"INVENTORY_GRPC_PORT,required"`
}

type inventory struct {
	raw inventoryEnv
}

func NewInventoryConfig() (*inventory, error) {
	var raw inventoryEnv
	if err := env.Parse(&raw); err != nil {
		return nil, err
	}

	return &inventory{raw: raw}, nil
}

func (cfg *inventory) Host() string { return cfg.raw.GRPCHost }
func (cfg *inventory) Port() int    { return cfg.raw.GRPCPort }
func (cfg *inventory) Address() string {
	return fmt.Sprintf("%s:%d", cfg.Host(), cfg.Port())
}

// ======= Payment =======

type paymentEnv struct {
	GRPCHost string `env:"PAYMENT_GRPC_HOST,required"`
	GRPCPort int    `env:"PAYMENT_GRPC_PORT,required"`
}

type payment struct {
	raw paymentEnv
}

func NewPaymentConfig() (*payment, error) {
	var raw paymentEnv
	if err := env.Parse(&raw); err != nil {
		return nil, err
	}
	return &payment{raw: raw}, nil
}

func (cfg *payment) Host() string { return cfg.raw.GRPCHost }
func (cfg *payment) Port() int    { return cfg.raw.GRPCPort }
func (cfg *payment) Address() string {
	return fmt.Sprintf("%s:%d", cfg.Host(), cfg.Port())
}
