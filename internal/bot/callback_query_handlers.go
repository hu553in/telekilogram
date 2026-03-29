package bot

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"telekilogram/internal/domain"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func (b *Bot) handleCallbackQuery(ctx context.Context, callback *models.CallbackQuery) error {
	message := callbackMessage(callback)
	if message == nil {
		return errors.New("callback query has no accessible message")
	}

	return b.withSpinner(ctx, message.Chat.ID, func() error {
		data := strings.TrimSpace(callback.Data)

		switch data {
		case "menu":
			return b.withEmptyCallbackAnswer(ctx, callback, func() error {
				return b.handleMenuCommand(ctx, message.Chat.ID)
			})
		case "menu_list":
			return b.withEmptyCallbackAnswer(ctx, callback, func() error {
				return b.handleListCommand(ctx, message.Chat.ID, callback.From.ID)
			})
		case "menu_digest":
			return b.withEmptyCallbackAnswer(ctx, callback, func() error {
				return b.handleDigestCommand(ctx, message.Chat.ID, callback.From.ID)
			})
		case "menu_settings":
			return b.withEmptyCallbackAnswer(ctx, callback, func() error {
				return b.handleSettingsCommand(ctx, message.Chat.ID, callback.From.ID)
			})
		}

		if hourUTCStr, ok := strings.CutPrefix(data, "settings_auto_digest_hour_utc_"); ok {
			return b.handleSettingsAutoDigestHourUTCQuery(ctx, hourUTCStr, callback)
		}

		return nil
	})
}

func (b *Bot) handleSettingsAutoDigestHourUTCQuery(
	ctx context.Context,
	hourUTCStr string,
	callback *models.CallbackQuery,
) error {
	message := callbackMessage(callback)
	if message == nil {
		return errors.New("callback query has no accessible message")
	}

	hourUTCStr = strings.TrimSpace(hourUTCStr)

	hourUTC, err := strconv.ParseInt(hourUTCStr, 10, 64)
	if err != nil {
		return b.errorCallbackAnswer(ctx, callback, fmt.Errorf("parse hourUTC: %w", err))
	}

	if err = b.db.UpsertUserSettings(ctx, &domain.UserSettings{
		UserID:            callback.From.ID,
		AutoDigestHourUTC: hourUTC,
	}); err != nil {
		return b.errorCallbackAnswer(ctx, callback, fmt.Errorf("upsert user settings: %w", err))
	}

	if _, err = b.rateLimiter.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
		CallbackQueryID: callback.ID,
		Text:            "✅ Settings are updated.",
	}); err != nil {
		return fmt.Errorf("answer callback query: %w", err)
	}

	return b.handleSettingsCommand(ctx, message.Chat.ID, callback.From.ID)
}

func (b *Bot) withEmptyCallbackAnswer(
	ctx context.Context,
	callback *models.CallbackQuery,
	fn func() error,
) error {
	var errs []error

	if _, err := b.rateLimiter.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
		CallbackQueryID: callback.ID,
	}); err != nil {
		errs = append(errs, b.errorCallbackAnswer(ctx, callback, fmt.Errorf("answer callback query: %w", err)))
	}

	err := fn()
	if err != nil {
		errs = append(errs, fmt.Errorf("call fn: %w", err))
	}

	return errors.Join(errs...)
}

func (b *Bot) errorCallbackAnswer(
	ctx context.Context,
	callback *models.CallbackQuery,
	err error,
) error {
	if _, sendErr := b.rateLimiter.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
		CallbackQueryID: callback.ID,
		Text:            "❌ Failed.",
	}); sendErr != nil {
		return errors.Join(err, fmt.Errorf("answer callback query: %w", sendErr))
	}
	return err
}
