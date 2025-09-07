package bot

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"telekilogram/feed"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (b *Bot) handleMessage(message *tgbotapi.Message) error {
	return b.withSpinner(message.Chat.ID, func() error {
		if message.ForwardFromChat != nil && // if message is forwarded...
			message.ForwardFromChat.Type == "channel" && // ...from channel...
			message.ForwardFromChat.UserName != "" { // ...with public user name

			return b.handleForwardedChannel(
				message.ForwardFromChat,
				message.Chat.ID,
				message.From.ID,
			)
		}

		switch {
		case strings.HasPrefix(message.Text, "/start"):
			return b.handleStartCommand(message.Text, message.Chat.ID, message.From.ID)
		case strings.HasPrefix(message.Text, "/menu"):
			return b.handleMenuCommand(message.Chat.ID)
		case strings.HasPrefix(message.Text, "/list"):
			return b.handleListCommand(message.Chat.ID, message.From.ID)
		case strings.HasPrefix(message.Text, "/digest"):
			return b.handleDigestCommand(message.Chat.ID, message.From.ID)
		case strings.HasPrefix(message.Text, "/filter"):
			return b.sendMessageWithKeyboard(message.Chat.ID, filterText, menuKeyboard)
		case strings.HasPrefix(message.Text, "/settings"):
			return b.handleSettingsCommand(message.Chat.ID, message.From.ID)
		default:
			return b.handleRandomText(message.Text, message.From.ID, message)
		}
	})
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
		} else {
			added += 1
		}
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

func (b *Bot) handleForwardedChannel(
	chat *tgbotapi.Chat,
	chatID int64,
	userID int64,
) error {
	slug := chat.UserName
	canonicalURL := feed.TelegramChannelCanonicalURL(slug)

	title := chat.Title
	if title == "" {
		slog.Warn("Empty Telegram channel title",
			slog.Any("canonicalURL", canonicalURL))

		title = canonicalURL
	}

	if err := b.db.AddFeed(userID, canonicalURL, title); err != nil {
		errs := []error{fmt.Errorf("failed to add feed: %w", err)}

		sendErr := b.sendMessageWithKeyboard(
			chatID,
			"❌ Failed\\.",
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

	return b.sendMessageWithKeyboard(
		chatID,
		"✅ Success\\.",
		returnKeyboard,
	)
}
