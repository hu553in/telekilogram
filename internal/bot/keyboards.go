package bot

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

const (
	hoursPerDay                                     = 24
	settingsAutoDigestHourUTCKeyboardRowSize        = 5
	settingsAutoDigestHourUTCKeyboardCallbackPrefix = "settings_auto_digest_hour_utc_"
)

func (b *Bot) sendMessageWithKeyboard(
	ctx context.Context,
	chatID int64,
	text string,
	keyboard [][]models.InlineKeyboardButton,
) error {
	normalizedText := strings.ToValidUTF8(text, "?")
	if normalizedText != text {
		b.log.WarnContext(ctx, "Message text had invalid UTF-8 and was normalized",
			"chatID", chatID,
			"originalLen", len(text),
			"normalizedLen", len(normalizedText))
	}

	for _, chunk := range splitTelegramText(normalizedText) {
		_, err := b.rateLimiter.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   chunk,
			// See https://core.telegram.org/bots/api#markdownv2-style.
			ParseMode: models.ParseModeMarkdown,
			LinkPreviewOptions: &models.LinkPreviewOptions{
				IsDisabled: bot.True(),
			},
			ReplyMarkup: &models.InlineKeyboardMarkup{
				InlineKeyboard: keyboard,
			},
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func getReturnKeyboard() [][]models.InlineKeyboardButton {
	return [][]models.InlineKeyboardButton{
		{{Text: "⬅️ Return to menu", CallbackData: "menu"}},
	}
}

func getMenuKeyboard() [][]models.InlineKeyboardButton {
	return [][]models.InlineKeyboardButton{
		{
			{Text: "📄 Feed list", CallbackData: "menu_list"},
			{Text: "👈 24h digest", CallbackData: "menu_digest"},
		},
		{
			{Text: "⚙️ Settings", CallbackData: "menu_settings"},
		},
	}
}

func getSettingsAutoDigestHourUTCKeyboard() [][]models.InlineKeyboardButton {
	var keyboard [][]models.InlineKeyboardButton

	for i := 0; i < hoursPerDay; i += settingsAutoDigestHourUTCKeyboardRowSize {
		var row []models.InlineKeyboardButton

		for j := i; j < i+settingsAutoDigestHourUTCKeyboardRowSize && j < hoursPerDay; j++ {
			hour := fmt.Sprintf("%02d", j)
			row = append(
				row,
				models.InlineKeyboardButton{
					Text:         hour,
					CallbackData: settingsAutoDigestHourUTCKeyboardCallbackPrefix + hour,
				},
			)
		}

		keyboard = append(keyboard, row)
	}

	return keyboard
}

func splitTelegramText(text string) []string {
	if utf8.RuneCountInString(text) <= telegramMessageMaxLength {
		return []string{text}
	}

	runes := []rune(text)
	chunks := make([]string, 0, (len(runes)+telegramMessageMaxLength-1)/telegramMessageMaxLength)

	for len(runes) > 0 {
		size := min(telegramMessageMaxLength, len(runes))
		chunks = append(chunks, string(runes[:size]))
		runes = runes[size:]
	}

	return chunks
}
