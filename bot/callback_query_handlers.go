package bot

import (
	"fmt"
	"strconv"
	"strings"

	"telekilogram/models"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (b *Bot) handleCallbackQuery(callback *tgbotapi.CallbackQuery) error {
	return b.withSpinner(callback.Message.Chat.ID, func() error {
		data := strings.TrimSpace(callback.Data)

		switch data {
		case "menu":
			return b.withEmptyCallbackAnswer(callback, func() error {
				return b.handleMenuCommand(callback.Message.Chat.ID)
			})
		case "menu_list":
			return b.withEmptyCallbackAnswer(callback, func() error {
				return b.handleListCommand(callback.Message.Chat.ID, callback.From.ID)
			})
		case "menu_digest":
			return b.withEmptyCallbackAnswer(callback, func() error {
				return b.handleDigestCommand(callback.Message.Chat.ID, callback.From.ID)
			})
		case "menu_settings":
			return b.withEmptyCallbackAnswer(callback, func() error {
				return b.handleSettingsCommand(callback.Message.Chat.ID, callback.From.ID)
			})
		}

		if hourUTCStr, ok := strings.CutPrefix(
			data,
			"settings_auto_digest_hour_utc_",
		); ok {
			return b.handleSettingsAutoDigestHourUTCQuery(hourUTCStr, callback)
		}

		return nil
	})
}

func (b *Bot) handleSettingsAutoDigestHourUTCQuery(
	hourUTCStr string,
	callback *tgbotapi.CallbackQuery,
) error {
	hourUTCStr = strings.TrimSpace(hourUTCStr)

	hourUTC, err := strconv.ParseInt(hourUTCStr, 10, 64)
	if err != nil {
		return b.errorCallbackAnswer(
			callback,
			fmt.Errorf("failed to parse hourUTC: %w", err),
		)
	}

	if err := b.db.UpsertUserSettings(&models.UserSettings{
		UserID:            callback.From.ID,
		AutoDigestHourUTC: hourUTC,
	}); err != nil {
		return b.errorCallbackAnswer(
			callback,
			fmt.Errorf("failed to upsert user settings: %w", err),
		)
	}

	if _, err := b.rateLimiter.Request(tgbotapi.NewCallback(
		callback.ID,
		"✅ Settings are updated.",
	)); err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}

	return b.handleSettingsCommand(callback.Message.Chat.ID, callback.From.ID)
}
