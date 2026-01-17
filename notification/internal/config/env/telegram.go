package envconfig

import "github.com/caarlos0/env/v11"

type telegramEnv struct {
	BotToken string `env:"TELEGRAM_BOT_TOKEN,required"`
}

type telegram struct {
	raw telegramEnv
}

func NewTelegramConfig() (*telegram, error) {
	var raw telegramEnv
	if err := env.Parse(&raw); err != nil {
		return nil, err
	}
	return &telegram{raw: raw}, nil
}

func (cfg *telegram) BotToken() string { return cfg.raw.BotToken }
