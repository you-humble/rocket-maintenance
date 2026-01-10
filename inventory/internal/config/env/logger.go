package envconfig

import "github.com/caarlos0/env/v11"

type loggerEnv struct {
	Level  string `env:"LOGGER_LEVEL,required"`
	AsJSON bool   `env:"LOGGER_AS_JSON,required"`
}

type logger struct {
	raw loggerEnv
}

func NewLoggerConfig() (*logger, error) {
	var raw loggerEnv
	if err := env.Parse(&raw); err != nil {
		return nil, err
	}
	return &logger{raw: raw}, nil
}

func (cfg *logger) Level() string { return cfg.raw.Level }
func (cfg *logger) AsJSON() bool  { return cfg.raw.AsJSON }
