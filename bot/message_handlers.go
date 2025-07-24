package bot

import (
	"errors"
	"fmt"
	"strings"

	"telekilogram/feed"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (b *Bot) handleMessage(message *tgbotapi.Message) error {
	userID := message.From.ID
	text := message.Text

	switch {
	case strings.HasPrefix(text, "/start"):
		return b.withSpinner(message.Chat.ID, func() error {
			return b.sendMessageWithKeyboard(message.Chat.ID, welcomeText, menuKeyboard)
		})
	case strings.HasPrefix(text, "/menu"):
		return b.withSpinner(message.Chat.ID, func() error {
			return b.handleMenuCommand(message.Chat.ID)
		})
	case strings.HasPrefix(text, "/list"):
		return b.withSpinner(message.Chat.ID, func() error {
			return b.handleListCommand(message.Chat.ID, userID)
		})
	case strings.HasPrefix(text, "/digest"):
		return b.withSpinner(message.Chat.ID, func() error {
			return b.handleDigestCommand(message.Chat.ID, userID)
		})
	case strings.HasPrefix(text, "/filter"):
		return b.withSpinner(message.Chat.ID, func() error {
			return b.sendMessageWithKeyboard(message.Chat.ID, filterText, menuKeyboard)
		})
	case strings.HasPrefix(text, "/settings"):
		return b.withSpinner(message.Chat.ID, func() error {
			return b.handleSettingsCommand(message.Chat.ID, userID)
		})
	default:
		return b.withSpinner(message.Chat.ID, func() error {
			return b.handleRandomText(text, userID, message)
		})
	}
}

func (b *Bot) handleRandomText(
	text string,
	userID int64,
	message *tgbotapi.Message,
) error {
	feeds, err := feed.FindValidFeeds(text)

	if len(feeds) == 0 {
		var errs []error
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to find valid feeds: %w", err))
		}

		sendErr := b.sendMessageWithKeyboard(
			message.Chat.ID,
			"✖️ Valid feed URLs are not found or there is a bug\\.",
			returnKeyboard,
		)
		if sendErr != nil {
			errs = append(
				errs,
				fmt.Errorf("failed to send message with keyboard: %w", sendErr),
			)
		}

		return errors.Join(errs...)
	}

	var errs []error
	if err != nil {
		errs = append(errs, fmt.Errorf("failed to find valid feeds: %w", err))
	}

	added := 0
	for _, feed := range feeds {
		if err := b.db.AddFeed(userID, feed.URL, feed.Title); err != nil {
			errs = append(errs, fmt.Errorf("failed to add feed: %w", err))

			continue
		}

		added += 1
	}

	if added == 0 {
		if err := b.sendMessageWithKeyboard(
			message.Chat.ID,
			"❌ Failed\\.",
			returnKeyboard,
		); err != nil {
			errs = append(
				errs,
				fmt.Errorf("failed to send message with keyboard: %w", err),
			)

			return errors.Join(errs...)
		}
	}

	if len(errs) > 0 {
		if err := b.sendMessageWithKeyboard(
			message.Chat.ID,
			fmt.Sprintf("⚠️ Partial success \\(%d added\\)\\.", added),
			returnKeyboard,
		); err != nil {
			errs = append(
				errs,
				fmt.Errorf("failed to send message with keyboard: %w", err),
			)

			return errors.Join(errs...)
		}
	}

	if err := b.sendMessageWithKeyboard(
		message.Chat.ID,
		"✅ Success\\.",
		returnKeyboard,
	); err != nil {
		return fmt.Errorf("failed to send message with keyboard: %w", err)
	}

	return nil
}
