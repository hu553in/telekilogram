package bot

import (
	"fmt"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const (
	hoursPerDay                                     = 24
	settingsAutoDigestHourUTCKeyboardRowSize        = 5
	settingsAutoDigestHourUTCKeyboardCallbackPrefix = "settings_auto_digest_hour_utc_"
)

func (b *Bot) sendMessageWithKeyboard(
	chatID int64,
	text string,
	keyboard [][]tgbotapi.InlineKeyboardButton,
) error {
	normalizedText := strings.ToValidUTF8(text, "?")
	if normalizedText != text {
		b.log.Warn("Message text had invalid UTF-8 and was normalized",
			"chatID", chatID,
			"originalLen", len(text),
			"normalizedLen", len(normalizedText))
	}

	message := tgbotapi.NewMessage(chatID, normalizedText)

	// See https://core.telegram.org/bots/api#markdownv2-style.
	message.ParseMode = tgbotapi.ModeMarkdownV2

	message.DisableWebPagePreview = true
	message.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(keyboard...)

	_, err := b.rateLimiter.Send(message)
	return err
}

func getReturnKeyboard() [][]tgbotapi.InlineKeyboardButton {
	return [][]tgbotapi.InlineKeyboardButton{
		{tgbotapi.NewInlineKeyboardButtonData("⬅️ Return to menu", "menu")},
	}
}

func getMenuKeyboard() [][]tgbotapi.InlineKeyboardButton {
	return [][]tgbotapi.InlineKeyboardButton{
		{
			tgbotapi.NewInlineKeyboardButtonData("📄 Feed list", "menu_list"),
			tgbotapi.NewInlineKeyboardButtonData("👈 24h digest", "menu_digest"),
		},
		{
			tgbotapi.NewInlineKeyboardButtonData("⚙️ Settings", "menu_settings"),
		},
	}
}

func getSettingsAutoDigestHourUTCKeyboard() [][]tgbotapi.InlineKeyboardButton {
	var keyboard [][]tgbotapi.InlineKeyboardButton

	for i := 0; i < hoursPerDay; i += settingsAutoDigestHourUTCKeyboardRowSize {
		var row []tgbotapi.InlineKeyboardButton

		for j := i; j < i+settingsAutoDigestHourUTCKeyboardRowSize && j < hoursPerDay; j++ {
			hour := fmt.Sprintf("%02d", j)
			row = append(
				row,
				tgbotapi.NewInlineKeyboardButtonData(hour, settingsAutoDigestHourUTCKeyboardCallbackPrefix+hour),
			)
		}

		keyboard = append(keyboard, row)
	}

	return keyboard
}
