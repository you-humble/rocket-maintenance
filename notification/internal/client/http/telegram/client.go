package tgclient

import (
	"context"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type client struct {
	bot *bot.Bot
}

func NewClient(bot *bot.Bot) *client {
	return &client{bot: bot}
}

func (c *client) SendMessage(ctx context.Context, chatID int64, text string) error {
	if _, err := c.bot.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    chatID,
		Text:      text,
		ParseMode: models.ParseModeMarkdownV1,
	}); err != nil {
		return err
	}

	return nil
}
