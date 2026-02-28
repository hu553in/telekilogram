package bot

import (
	"context"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const sendSpinnerInterval = 3 * time.Second

func (b *Bot) sendTyping(ctx context.Context, chatID int64) {
	config := tgbotapi.NewChatAction(chatID, tgbotapi.ChatTyping)
	_, err := b.rateLimiter.Request(config)
	if err != nil {
		b.log.ErrorContext(ctx, "Failed to send chat action",
			"error", err)
	}
}

func (b *Bot) withSpinner(ctx context.Context, chatID int64, fn func() error) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	go func() {
		b.sendTyping(ctx, chatID)

		t := time.NewTicker(sendSpinnerInterval)
		defer t.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				b.sendTyping(ctx, chatID)
			}
		}
	}()

	return fn()
}
