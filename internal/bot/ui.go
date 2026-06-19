package bot

import (
	"context"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

const sendSpinnerInterval = 3 * time.Second

func (b *Bot) sendTyping(ctx context.Context, chatID int64) {
	_, err := b.rateLimiter.SendChatAction(ctx, &bot.SendChatActionParams{
		ChatID: chatID,
		Action: models.ChatActionTyping,
	})
	if err != nil {
		b.log.ErrorContext(ctx, "Failed to send chat action",
			"error", err,
			"chatID", chatID)
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
