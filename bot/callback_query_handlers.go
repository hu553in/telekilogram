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
		switch callback.Data {
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

		if feedIDStr, ok := strings.CutPrefix(callback.Data, "unfollow_"); ok {
			return b.handleUnfollowQuery(feedIDStr, callback)
		}

		if hourUTCStr, ok := strings.CutPrefix(
			callback.Data,
			"settings_auto_digest_hour_utc_",
		); ok {
			return b.handleSettingsAutoDigestHourUTCQuery(hourUTCStr, callback)
		}

		return nil
	})
}

func (b *Bot) handleUnfollowQuery(
	feedIDStr string,
	callback *tgbotapi.CallbackQuery,
) error {
	feedID, err := strconv.ParseInt(feedIDStr, 10, 64)
	if err != nil {
		return b.errorCallbackAnswer(
			callback,
			fmt.Errorf("failed to parse feedID: %w", err),
		)
	}

	if err := b.db.RemoveFeed(feedID); err != nil {
		return b.errorCallbackAnswer(
			callback,
			fmt.Errorf("failed to remove feed: %w", err),
		)
	}

	if _, err := b.api.Request(tgbotapi.NewCallback(
		callback.ID,
		"✅ Feed is removed.",
	)); err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}

	return b.handleListCommand(callback.Message.Chat.ID, callback.From.ID)
}

func (b *Bot) handleSettingsAutoDigestHourUTCQuery(
	hourUTCStr string,
	callback *tgbotapi.CallbackQuery,
) error {
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

	if _, err := b.api.Request(tgbotapi.NewCallback(
		callback.ID,
		"✅ Settings are updated.",
	)); err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}

	return b.handleSettingsCommand(callback.Message.Chat.ID, callback.From.ID)
}
