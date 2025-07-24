package bot

import (
	"fmt"
	"strings"
	"telekilogram/feed"
	"telekilogram/markdown"
	"time"

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

func (b *Bot) handleListCommand(chatID int64, userID int64) error {
	feeds, err := b.db.GetUserFeeds(userID)
	if err != nil {
		return fmt.Errorf("failed to get user feeds: %w", err)
	}

	if len(feeds) == 0 {
		return b.sendMessageWithKeyboard(
			chatID,
			"✖️ Feed list is empty or there is a bug\\.",
			returnKeyboard,
		)
	}

	var message strings.Builder
	message.WriteString(fmt.Sprintf("🔍 *Found %d feeds:*\n\n", len(feeds)))

	keyboard := make(
		[][]tgbotapi.InlineKeyboardButton,
		0,
		len(feeds)+len(returnKeyboard),
	)

	for i, f := range feeds {
		message.WriteString(fmt.Sprintf(
			"%d\\. [%s](%s)\n",
			i+1,
			markdown.EscapeV2(f.Title),
			f.URL,
		))

		button := tgbotapi.NewInlineKeyboardButtonData(
			fmt.Sprintf("Unfollow %d", i+1),
			fmt.Sprintf("unfollow_%d", f.ID),
		)
		keyboard = append(keyboard, []tgbotapi.InlineKeyboardButton{button})
	}

	keyboard = append(keyboard, returnKeyboard...)

	if err := b.sendMessageWithKeyboard(
		chatID,
		message.String(),
		keyboard,
	); err != nil {
		return fmt.Errorf("failed to send message with keyboard: %w", err)
	}

	return nil
}

func (b *Bot) handleMenuCommand(chatID int64) error {
	return b.sendMessageWithKeyboard(chatID, "❔ *Choose an option:*", menuKeyboard)
}

func (b *Bot) handleDigestCommand(chatID int64, userID int64) error {
	userPosts, err := b.fetcher.FetchUserFeeds(userID)
	if err != nil {
		return fmt.Errorf("failed to fetch user feeds: %w", err)
	}

	if len(userPosts) == 0 {
		if err := b.sendMessageWithKeyboard(
			chatID,
			"✖️ Feed list is empty or there is a bug\\.",
			returnKeyboard,
		); err != nil {
			return fmt.Errorf("failed to send message with keyboard: %w", err)
		}
	}

	for _, posts := range userPosts {
		if err := b.SendNewPosts(chatID, posts); err != nil {
			return fmt.Errorf("failed to send new posts: %w", err)
		}
	}

	return nil
}

func (b *Bot) handleSettingsCommand(chatID int64, userID int64) error {
	settings, err := b.db.GetUserSettingsWithDefault(userID)
	if err != nil {
		return fmt.Errorf("failed to get user settings with default: %w", err)
	}

	currentUTC := time.Now().UTC().Format("15:04")

	hourUTC := settings.AutoDigestHourUTC
	hourUTCStr := fmt.Sprintf("%d:00", hourUTC)
	if hourUTC < 10 {
		hourUTCStr = fmt.Sprintf("0%s", hourUTCStr)
	}

	if err := b.sendMessageWithKeyboard(
		chatID,
		fmt.Sprintf(settingsText, currentUTC, hourUTCStr),
		settingsAutoDigestHourUTCKeyboard,
	); err != nil {
		return fmt.Errorf("failed to send message with keyboard: %w", err)
	}

	return nil
}

func (b *Bot) handleRandomText(
	text string,
	userID int64,
	message *tgbotapi.Message,
) error {
	feeds, err := feed.FindValidFeeds(text)
	if err != nil {
		return fmt.Errorf("failed to find valid feeds: %w", err)
	}

	if len(feeds) == 0 {
		return b.sendMessageWithKeyboard(
			message.Chat.ID,
			"✖️ Valid feed URLs are not found\\. Ignoring\\.",
			returnKeyboard,
		)
	}

	for _, feed := range feeds {
		if err := b.db.AddFeed(userID, feed.URL, feed.Title); err != nil {
			return fmt.Errorf("failed to add feed: %w", err)
		}
	}

	if err := b.sendMessageWithKeyboard(
		message.Chat.ID,
		"✅ Saved\\.",
		returnKeyboard,
	); err != nil {
		return fmt.Errorf("failed to send message with keyboard: %w", err)
	}

	return nil
}
