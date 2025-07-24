package bot

import (
	"errors"
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

	_, err := b.api.Send(msg)

	return err
}

func (b *Bot) sendChatAction(chatID int64, action string) error {
	config := tgbotapi.NewChatAction(chatID, action)
	_, err := b.api.Request(config)

	return err
}

func (b *Bot) withSpinner(chatID int64, function func() error) error {
	var errs []error

	if err := b.sendChatAction(chatID, tgbotapi.ChatTyping); err != nil {
		errs = append(errs, fmt.Errorf("failed to send chat action: %w", err))
	}

	err := function()
	if err != nil {
		errs = append(errs, fmt.Errorf("failed to call function: %w", err))
	}

	return errors.Join(errs...)
}

func (b *Bot) userAllowed(userID int64) bool {
	return len(b.allowedUsers) == 0 || slices.Contains(b.allowedUsers, userID)
}

func (b *Bot) withEmptyCallbackAnswer(
	callback *tgbotapi.CallbackQuery,
	function func() error,
) error {
	var errs []error

	if _, err := b.api.Request(tgbotapi.NewCallback(
		callback.ID,
		"",
	)); err != nil {
		errs = append(
			errs,
			b.errorCallbackAnswer(
				callback,
				fmt.Errorf("failed to send request: %w", err),
			),
		)
	}

	err := b.withSpinner(callback.Message.Chat.ID, func() error {
		return function()
	})
	if err != nil {
		errs = append(errs, fmt.Errorf("failed to call function: %w", err))
	}

	return errors.Join(errs...)
}

func (b *Bot) errorCallbackAnswer(
	callback *tgbotapi.CallbackQuery,
	err error,
) error {
	if _, sendErr := b.api.Request(tgbotapi.NewCallback(
		callback.ID,
		"❌ Failed.",
	)); sendErr != nil {
		return errors.Join(err, fmt.Errorf("failed to send request: %w", sendErr))
	}

	return err
}
