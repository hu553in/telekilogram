package bot

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"telekilogram/internal/markdown"
	"time"
)

const maxHourForAddingLeadingZero = 9

const welcomeText = `🤖 *Welcome to Telekilogram\!*

I'm your feed assistant\. I can help you:

– Follow RSS / Atom / JSON feeds and public Telegram channels by sending URLs,
  channel @username slugs, or forwarding messages from channels to me
– Get feed list with /list
– Unfollow feeds directly from list
– Receive 24h auto\-digest daily automatically \(default \- 00:00 UTC\)
– Receive 24h digest with /digest
– Get concise summaries for Telegram channel posts \(AI when configured\)
– Configure user settings with /settings`

const settingsText = `*⚙️ Settings*

Current UTC time is %s\.

Current auto\-digest hour \(UTC\) setting is %s\.

You can choose different setting below:`

func (b *Bot) handleStartCommand(
	ctx context.Context,
	text string,
	chatID int64,
	userID int64,
) error {
	text = strings.TrimSpace(text)

	if feedIDStr, ok := strings.CutPrefix(text, "/start unfollow_"); ok {
		return b.handleUnfollowDeepLink(ctx, strings.TrimSpace(feedIDStr), chatID, userID)
	}

	return b.sendMessageWithKeyboard(chatID, welcomeText, b.menuKeyboard)
}

func (b *Bot) handleUnfollowDeepLink(
	ctx context.Context,
	feedIDStr string,
	chatID int64,
	userID int64,
) error {
	feedIDStr = strings.TrimSpace(feedIDStr)

	feedID, err := strconv.ParseInt(feedIDStr, 10, 64)
	if err != nil {
		errs := []error{fmt.Errorf("failed to parse feedID: %w", err)}

		sendErr := b.sendMessageWithKeyboard(chatID, "❌ Failed\\.", b.returnKeyboard)
		if sendErr != nil {
			errs = append(errs, fmt.Errorf("failed to send message with keyboard: %w", sendErr))
		}

		return errors.Join(errs...)
	}

	if err = b.db.RemoveFeed(ctx, feedID); err != nil {
		errs := []error{fmt.Errorf("failed to remove feed: %w", err)}

		sendErr := b.sendMessageWithKeyboard(chatID, "❌ Failed\\.", b.returnKeyboard)
		if sendErr != nil {
			errs = append(errs, fmt.Errorf("failed to send message with keyboard: %w", sendErr))
		}

		return errors.Join(errs...)
	}

	if err = b.sendMessageWithKeyboard(chatID, "✅ Feed is removed\\.", b.returnKeyboard); err != nil {
		return fmt.Errorf("failed to send message with keyboard: %w", err)
	}

	return b.handleListCommand(ctx, chatID, userID)
}

func (b *Bot) handleListCommand(ctx context.Context, chatID int64, userID int64) error {
	feeds, err := b.db.GetUserFeeds(ctx, userID)

	if len(feeds) == 0 {
		var errs []error
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to get user feeds: %w", err))
		}

		sendErr := b.sendMessageWithKeyboard(chatID, "✖️ Feed list is empty or there is a bug\\.", b.returnKeyboard)
		if sendErr != nil {
			errs = append(errs, fmt.Errorf("failed to send message with keyboard: %w", sendErr))
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
			message.WriteString(fmt.Sprintf("%d\\. [%s](%s)\n", i+1, markdown.EscapeV2(title), url))
		}
	}

	if err = b.sendMessageWithKeyboard(chatID, message.String(), b.returnKeyboard); err != nil {
		errs = append(errs, fmt.Errorf("failed to send message with keyboard: %w", err))
	}

	return errors.Join(errs...)
}

func (b *Bot) handleMenuCommand(chatID int64) error {
	return b.sendMessageWithKeyboard(chatID, "❔ *Choose an option:*", b.menuKeyboard)
}

func (b *Bot) handleDigestCommand(ctx context.Context, chatID int64, userID int64) error {
	userPosts, err := b.fetcher.FetchUserFeeds(ctx, userID)

	if len(userPosts) == 0 {
		var errs []error
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to fetch user feeds: %w", err))
		}

		sendErr := b.sendMessageWithKeyboard(chatID, "✖️ Feed list is empty or there is a bug\\.", b.returnKeyboard)
		if sendErr != nil {
			errs = append(errs, fmt.Errorf("failed to send message with keyboard: %w", sendErr))
		}

		return errors.Join(errs...)
	}

	var errs []error
	if err != nil {
		errs = append(errs, fmt.Errorf("failed to fetch user feeds: %w", err))
	}

	for _, posts := range userPosts {
		if err = b.SendNewPosts(ctx, chatID, posts); err != nil {
			errs = append(errs, fmt.Errorf("failed to send new posts: %w", err))
		}
	}

	return errors.Join(errs...)
}

func (b *Bot) handleSettingsCommand(ctx context.Context, chatID int64, userID int64) error {
	settings, err := b.db.GetUserSettingsWithDefault(ctx, userID)
	if err != nil {
		errs := []error{fmt.Errorf("failed to get user settings with default: %w", err)}

		sendErr := b.sendMessageWithKeyboard(chatID, "❌ Failed\\.", b.returnKeyboard)
		if sendErr != nil {
			errs = append(errs, fmt.Errorf("failed to send message with keyboard: %w", sendErr))
		}

		return errors.Join(errs...)
	}

	currentUTC := time.Now().UTC().Format("15:04")

	hourUTC := settings.AutoDigestHourUTC
	hourUTCStr := fmt.Sprintf("%d:00", hourUTC)
	if hourUTC <= maxHourForAddingLeadingZero {
		hourUTCStr = fmt.Sprintf("0%s", hourUTCStr)
	}

	if err = b.sendMessageWithKeyboard(
		chatID,
		fmt.Sprintf(settingsText, currentUTC, hourUTCStr),
		b.settingsAutoDigestHourUTCKeyboard,
	); err != nil {
		return fmt.Errorf("failed to send message with keyboard: %w", err)
	}

	return nil
}
