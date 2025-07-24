package bot

import (
	"fmt"
	"slices"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (b *Bot) sendMessageWithKeyboard(
	chatID int64,
	text string,
	keyboard [][]tgbotapi.InlineKeyboardButton,
) error {
	msg := tgbotapi.NewMessage(chatID, text)

	// https://core.telegram.org/bots/api#markdownv2-style
	msg.ParseMode = "MarkdownV2"

	msg.DisableWebPagePreview = true
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(keyboard...)

	if _, err := b.api.Send(msg); err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	return nil
}

func (b *Bot) sendChatAction(chatID int64, action string) error {
	config := tgbotapi.NewChatAction(chatID, action)
	if _, err := b.api.Request(config); err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}

	return nil
}

func (b *Bot) withSpinner(chatID int64, callback func() error) error {
	if err := b.sendChatAction(chatID, tgbotapi.ChatTyping); err != nil {
		return fmt.Errorf("failed to send chat action: %w", err)
	}

	return callback()
}

func (b *Bot) userAllowed(userID int64) bool {
	return len(b.allowedUsers) == 0 || slices.Contains(b.allowedUsers, userID)
}
