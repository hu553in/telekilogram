package bot

import (
	"fmt"
	"strings"
	"telekilogram/markdown"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

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
