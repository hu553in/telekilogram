package bot

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"telekilogram/markdown"
)

func (b *Bot) handleStartCommand(
	text string,
	chatID int64,
	userID int64,
) error {
	text = strings.TrimSpace(text)

	if feedIDStr, ok := strings.CutPrefix(text, "/start unfollow_"); ok {
		return b.handleUnfollowDeepLink(strings.TrimSpace(feedIDStr), chatID, userID)
	}

	return b.sendMessageWithKeyboard(chatID, welcomeText, menuKeyboard)
}

func (b *Bot) handleUnfollowDeepLink(
	feedIDStr string,
	chatID int64,
	userID int64,
) error {
	feedIDStr = strings.TrimSpace(feedIDStr)

	feedID, err := strconv.ParseInt(feedIDStr, 10, 64)
	if err != nil {
		errs := []error{
			fmt.Errorf("failed to parse feedID: %w", err),
		}

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

	if err := b.db.RemoveFeed(feedID); err != nil {
		errs := []error{
			fmt.Errorf("failed to remove feed: %w", err),
		}

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

	if err := b.sendMessageWithKeyboard(
		chatID,
		"✅ Feed is removed\\.",
		returnKeyboard,
	); err != nil {
		return fmt.Errorf("failed to send message with keyboard: %w", err)
	}

	return b.handleListCommand(chatID, userID)
}

func (b *Bot) handleListCommand(chatID int64, userID int64) error {
	feeds, err := b.db.GetUserFeeds(userID)

	if len(feeds) == 0 {
		var errs []error
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to get user feeds: %w", err))
		}

		sendErr := b.sendMessageWithKeyboard(
			chatID,
			"✖️ Feed list is empty or there is a bug\\.",
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
		errs = append(errs, fmt.Errorf("failed to get user feeds: %w", err))
	}

	var message strings.Builder
	message.WriteString(fmt.Sprintf("🔍 *Found %d feeds:*\n\n", len(feeds)))

	botInfo, botInfoErr := b.api.GetMe()
	if botInfoErr != nil {
		errs = append(errs, fmt.Errorf("failed to get bot info: %w", botInfoErr))
	}

	for i, f := range feeds {
		url := strings.TrimSpace(f.URL)
		if url == "" {
			continue
		}

		title := strings.TrimSpace(f.Title)
		if title == "" {
			title = url
		}

		if botInfoErr == nil {
			message.WriteString(fmt.Sprintf(
				"%d\\. [%s](%s) \\[[unfollow](https://t\\.me/%s?start=unfollow_%d)\\]\n",
				i+1,
				markdown.EscapeV2(title),
				url,
				botInfo.UserName,
				f.ID,
			))
		} else {
			message.WriteString(fmt.Sprintf(
				"%d\\. [%s](%s)\n",
				i+1,
				markdown.EscapeV2(title),
				url,
			))
		}
	}

	if err := b.sendMessageWithKeyboard(
		chatID,
		message.String(),
		returnKeyboard,
	); err != nil {
		errs = append(
			errs,
			fmt.Errorf("failed to send message with keyboard: %w", err),
		)
	}

	return errors.Join(errs...)
}

func (b *Bot) handleMenuCommand(chatID int64) error {
	return b.sendMessageWithKeyboard(chatID, "❔ *Choose an option:*", menuKeyboard)
}

func (b *Bot) handleDigestCommand(chatID int64, userID int64) error {
	userPosts, err := b.fetcher.FetchUserFeeds(userID)

	if len(userPosts) == 0 {
		var errs []error
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to fetch user feeds: %w", err))
		}

		sendErr := b.sendMessageWithKeyboard(
			chatID,
			"✖️ Feed list is empty or there is a bug\\.",
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
		errs = append(errs, fmt.Errorf("failed to fetch user feeds: %w", err))
	}

	for _, posts := range userPosts {
		if err := b.SendNewPosts(chatID, posts); err != nil {
			errs = append(errs, fmt.Errorf("failed to send new posts: %w", err))
		}
	}

	return errors.Join(errs...)
}

func (b *Bot) handleSettingsCommand(chatID int64, userID int64) error {
	settings, err := b.db.GetUserSettingsWithDefault(userID)
	if err != nil {
		errs := []error{
			fmt.Errorf("failed to get user settings with default: %w", err),
		}

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
