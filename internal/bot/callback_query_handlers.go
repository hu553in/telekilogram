package bot

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"telekilogram/internal/domain"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (b *Bot) handleCallbackQuery(ctx context.Context, callback *tgbotapi.CallbackQuery) error {
	return b.withSpinner(ctx, callback.Message.Chat.ID, func() error {
		data := strings.TrimSpace(callback.Data)

		switch data {
		case "menu":
			return b.withEmptyCallbackAnswer(callback, func() error {
				return b.handleMenuCommand(callback.Message.Chat.ID)
			})
		case "menu_list":
			return b.withEmptyCallbackAnswer(callback, func() error {
				return b.handleListCommand(ctx, callback.Message.Chat.ID, callback.From.ID)
			})
		case "menu_digest":
			return b.withEmptyCallbackAnswer(callback, func() error {
				return b.handleDigestCommand(ctx, callback.Message.Chat.ID, callback.From.ID)
			})
		case "menu_settings":
			return b.withEmptyCallbackAnswer(callback, func() error {
				return b.handleSettingsCommand(ctx, callback.Message.Chat.ID, callback.From.ID)
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
	callback *tgbotapi.CallbackQuery,
) error {
	hourUTCStr = strings.TrimSpace(hourUTCStr)

	hourUTC, err := strconv.ParseInt(hourUTCStr, 10, 64)
	if err != nil {
		return b.errorCallbackAnswer(callback, fmt.Errorf("parse hourUTC: %w", err))
	}

	if err = b.db.UpsertUserSettings(ctx, &domain.UserSettings{
		UserID:            callback.From.ID,
		AutoDigestHourUTC: hourUTC,
	}); err != nil {
		return b.errorCallbackAnswer(callback, fmt.Errorf("upsert user settings: %w", err))
	}

	if _, err = b.rateLimiter.Request(tgbotapi.NewCallback(callback.ID, "✅ Settings are updated.")); err != nil {
		return fmt.Errorf("send request: %w", err)
	}

	return b.handleSettingsCommand(ctx, callback.Message.Chat.ID, callback.From.ID)
}

func (b *Bot) withEmptyCallbackAnswer(
	callback *tgbotapi.CallbackQuery,
	fn func() error,
) error {
	var errs []error

	if _, err := b.rateLimiter.Request(tgbotapi.NewCallback(callback.ID, "")); err != nil {
		errs = append(errs, b.errorCallbackAnswer(callback, fmt.Errorf("send request: %w", err)))
	}

	err := fn()
	if err != nil {
		errs = append(errs, fmt.Errorf("call fn: %w", err))
	}

	return errors.Join(errs...)
}

func (b *Bot) errorCallbackAnswer(
	callback *tgbotapi.CallbackQuery,
	err error,
) error {
	if _, sendErr := b.rateLimiter.Request(tgbotapi.NewCallback(callback.ID, "❌ Failed.")); sendErr != nil {
		return errors.Join(err, fmt.Errorf("send request: %w", sendErr))
	}
	return err
}
