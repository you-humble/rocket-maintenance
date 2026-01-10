package config

import "time"

type Server interface {
	Host() string
	Port() int
	Address() string
	BDEReadTimeout() time.Duration
}

type Logger interface {
	Level() string
	AsJSON() bool
}

type Database interface {
	DatabaseName() string
	PartsCollection() string
	DSN() string
}
