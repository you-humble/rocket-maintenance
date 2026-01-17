package app

import (
	"context"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"golang.org/x/sync/errgroup"

	"github.com/you-humble/rocket-maintenance/notification/internal/config"
	"github.com/you-humble/rocket-maintenance/platform/closer"
	"github.com/you-humble/rocket-maintenance/platform/logger"
)

type app struct {
	di *di
}

func New(ctx context.Context) (*app, error) {
	a := &app{}

	if err := a.init(ctx); err != nil {
		return nil, err
	}

	return a, nil
}

func (a *app) Run(ctx context.Context) error { return a.run(ctx) }

func (a *app) init(ctx context.Context) error {
	inits := []func(context.Context) error{
		a.initConfig,
		a.initLogger,
		a.initCloser,
		a.initDI,
		a.initTelegramBot,
	}

	for _, initFn := range inits {
		if err := initFn(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (a *app) initConfig(_ context.Context) error {
	return config.Load()
}

func (a *app) initLogger(_ context.Context) error {
	return logger.Init(
		config.C().Logger.Level(),
		config.C().Logger.AsJSON(),
	)
}

func (a *app) initCloser(_ context.Context) error {
	closer.SetLogger(logger.L())
	return nil
}

func (a *app) initDI(_ context.Context) error {
	a.di = NewDI()
	return nil
}

func (a *app) initTelegramBot(ctx context.Context) error {
	const startMsg = `
	👋 **Привет! Я бот уведомлений AstraDock.**
	
	Я присылаю важные события по твоим заказам:
	🚀 сборка корабля завершена  
	💳 заказ успешно оплачен  
	
	Чтобы начать, просто оформи заказ в сервисе — а дальше я буду держать тебя в курсе.  
	Если уведомления приходят не туда — проверь, что ты вошёл под нужным аккаунтом.
	`

	telegramBot := a.di.TelegramBot(ctx)
	tgSvc := a.di.TelegramService(ctx)

	telegramBot.RegisterHandler(
		bot.HandlerTypeMessageText,
		"/start",
		bot.MatchTypeExact,
		func(ctx context.Context, b *bot.Bot, update *models.Update) {
			logger.Info(ctx, "New user",
				logger.String("username", update.Message.From.Username),
				logger.Int64("chat_id", update.Message.Chat.ID),
			)

			_, err := b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID:    update.Message.Chat.ID,
				Text:      startMsg,
				ParseMode: models.ParseModeMarkdownV1,
			})
			if err != nil {
				logger.Error(ctx, "Failed to send activation message", logger.ErrorF(err))
			}

			tgSvc.AddChatID(ctx, update.Message.Chat.ID)
		})

	go func() {
		logger.Info(ctx, "🤖 Telegram bot started...")
		telegramBot.Start(ctx)
	}()

	return nil
}

func (a *app) run(ctx context.Context) error {
	defer gracefulShutdown()

	eg, egCtx := errgroup.WithContext(ctx)

	eg.Go(func() error {
		logger.Info(egCtx, "🚀 order.paid consumer running")
		if err := a.di.OrderPaidConsumer(egCtx).RunOrderPaidConsume(egCtx); err != nil {
			return err
		}
		return nil
	})

	eg.Go(func() error {
		logger.Info(egCtx, "🚀 order.assembled consumer running")
		if err := a.di.OrderAssembledConsumer(egCtx).RunOrderAssembledConsume(egCtx); err != nil {
			return err
		}
		return nil
	})

	if err := eg.Wait(); err != nil {
		return err
	}
	return nil
}

//nolint:contextcheck
func gracefulShutdown() {
	ctx, cancel := context.WithTimeout(
		context.Background(), // do not inherit cancellation from ctx
		10*time.Second,
	)
	defer cancel()

	err := closer.CloseAll(ctx)
	if err != nil {
		logger.Error(ctx, "❌ Error during server shutdown", logger.ErrorF(err))
		logger.Error(ctx, "❌😵‍💫 Server stopped")
		return
	}
	logger.Info(ctx, "✅ Server stopped")
}
