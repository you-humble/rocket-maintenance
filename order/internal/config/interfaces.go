package config

import "time"

type Client interface {
	Host() string
	Port() int
	Address() string
}

type Server interface {
	Client
	ReadTimeout() time.Duration
	ShutdownTimeout() time.Duration
	BDEReadTimeout() time.Duration
	DBWriteTimeout() time.Duration
}

type Logger interface {
	Level() string
	AsJSON() bool
}

type Database interface {
	MigrationDirectory() string
	DSN() string
}
