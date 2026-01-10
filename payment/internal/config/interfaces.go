package config

type Server interface {
	Host() string
	Port() int
	Address() string
}

type Logger interface {
	Level() string
	AsJSON() bool
}
