package bot

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"telekilogram/internal/feed"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const filterText = `Telekilogram does not support filtering\.\.\.

But you can use awesome [siftrss](https://siftrss.com/) instead\! ✨
It's totally great\. Bot author is also using it\.`

func (b *Bot) handleMessage(ctx context.Context, message *tgbotapi.Message) error {
	return b.withSpinner(ctx, message.Chat.ID, func() error {
		if message.ForwardFromChat != nil && // If message is forwarded...
			message.ForwardFromChat.Type == "channel" && // ...from channel...
			message.ForwardFromChat.UserName != "" { // ...with public user name.
			return b.handleForwardedChannel(ctx, message.ForwardFromChat, message.Chat.ID, message.From.ID)
		}

		text := strings.TrimSpace(message.Text)

		switch {
		case strings.HasPrefix(text, "/start"):
			return b.handleStartCommand(ctx, text, message.Chat.ID, message.From.ID)
		case strings.HasPrefix(text, "/menu"):
			return b.handleMenuCommand(message.Chat.ID)
		case strings.HasPrefix(text, "/list"):
			return b.handleListCommand(ctx, message.Chat.ID, message.From.ID)
		case strings.HasPrefix(text, "/digest"):
			return b.handleDigestCommand(ctx, message.Chat.ID, message.From.ID)
		case strings.HasPrefix(text, "/filter"):
			return b.sendMessageWithKeyboard(message.Chat.ID, filterText, b.menuKeyboard)
		case strings.HasPrefix(text, "/settings"):
			return b.handleSettingsCommand(ctx, message.Chat.ID, message.From.ID)
		default:
			return b.handleRandomText(ctx, text, message.From.ID, message)
		}
	})
}

func (b *Bot) handleRandomText(
	ctx context.Context,
	text string,
	userID int64,
	message *tgbotapi.Message,
) error {
	text = strings.TrimSpace(text)

	feeds, err := b.fetcher.FindValidFeeds(ctx, text)

	if len(feeds) == 0 {
		var errs []error
		if err != nil {
			errs = append(errs, fmt.Errorf("find valid feeds: %w", err))
		}

		sendErr := b.sendMessageWithKeyboard(
			message.Chat.ID,
			"✖️ Valid feed URLs are not found or there is a bug\\.",
			b.returnKeyboard,
		)
		if sendErr != nil {
			errs = append(errs, fmt.Errorf("send message with keyboard: %w", sendErr))
		}

		return errors.Join(errs...)
	}

	var errs []error
	if err != nil {
		errs = append(errs, fmt.Errorf("find valid feeds: %w", err))
	}

	added := 0
	for _, feed := range feeds {
		if err = b.db.AddFeed(ctx, userID, feed.URL, feed.Title); err != nil {
			errs = append(errs, fmt.Errorf("add feed: %w", err))
		} else {
			added++
		}
	}

	if added == 0 {
		if err = b.sendMessageWithKeyboard(message.Chat.ID, "❌ Failed\\.", b.returnKeyboard); err != nil {
			errs = append(errs, fmt.Errorf("send message with keyboard: %w", err))

			return errors.Join(errs...)
		}
	}

	if len(errs) > 0 {
		if err = b.sendMessageWithKeyboard(
			message.Chat.ID,
			fmt.Sprintf("⚠️ Partial success \\(%d added\\)\\.", added),
			b.returnKeyboard,
		); err != nil {
			errs = append(errs, fmt.Errorf("send message with keyboard: %w", err))
			return errors.Join(errs...)
		}
	}

	err = b.sendMessageWithKeyboard(message.Chat.ID, "✅ Success\\.", b.returnKeyboard)
	if err != nil {
		return fmt.Errorf("send message with keyboard: %w", err)
	}

	return nil
}

func (b *Bot) handleForwardedChannel(
	ctx context.Context,
	chat *tgbotapi.Chat,
	chatID int64,
	userID int64,
) error {
	slug := strings.TrimSpace(chat.UserName)

	canonicalURL := feed.TelegramChannelCanonicalURL(slug)
	if canonicalURL == "" {
		b.log.WarnContext(ctx, "Empty canonical URL for forwarded channel",
			"slug", slug,
			"chatID", chatID,
			"userID", userID)

		return b.sendMessageWithKeyboard(chatID, "❌ Failed\\.", b.returnKeyboard)
	}

	title := strings.TrimSpace(chat.Title)
	if title == "" {
		b.log.WarnContext(ctx, "Empty Telegram channel title",
			"canonicalURL", canonicalURL,
			"slug", slug)

		title = canonicalURL
	}

	if err := b.db.AddFeed(ctx, userID, canonicalURL, title); err != nil {
		errs := []error{fmt.Errorf("add feed: %w", err)}

		sendErr := b.sendMessageWithKeyboard(chatID, "❌ Failed\\.", b.returnKeyboard)
		if sendErr != nil {
			errs = append(errs, fmt.Errorf("send message with keyboard: %w", sendErr))
		}

		return errors.Join(errs...)
	}

	return b.sendMessageWithKeyboard(chatID, "✅ Success\\.", b.returnKeyboard)
}
