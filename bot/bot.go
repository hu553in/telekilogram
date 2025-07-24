package bot

import (
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"telekilogram/database"
	"telekilogram/feed"
	"telekilogram/markdown"
	"telekilogram/model"
)

type Bot struct {
	api          *tgbotapi.BotAPI
	db           *database.Database
	fetcher      *feed.FeedFetcher
	allowedUsers []int64
}

func New(
	token string,
	db *database.Database,
	fetcher *feed.FeedFetcher,
	allowedUsers []int64,
) (*Bot, error) {
	api, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, fmt.Errorf("failed to create bot API: %w", err)
	}

	return &Bot{
		api:          api,
		db:           db,
		fetcher:      fetcher,
		allowedUsers: allowedUsers,
	}, nil
}

func (b *Bot) Start() {
	updateConfig := tgbotapi.NewUpdate(0)
	updateConfig.Timeout = 60

	updates := b.api.GetUpdatesChan(updateConfig)

	for update := range updates {
		switch {
		case update.Message != nil:
			if !b.userAllowed(update.Message.From.ID) {
				return
			}
			if err := b.handleMessage(update.Message); err != nil {
				slog.Error("Failed to handle message",
					slog.Any("update", update),
					slog.Any("err", err))
			}
		case update.CallbackQuery != nil:
			if !b.userAllowed(update.CallbackQuery.From.ID) {
				return
			}
			if err := b.handleCallbackQuery(update.CallbackQuery); err != nil {
				slog.Error("Failed to handle callback query",
					slog.Any("update", update),
					slog.Any("err", err))
			}
		}
	}
}

func (b *Bot) SendNewPosts(chatID int64, posts []model.Post) error {
	if len(posts) == 0 {
		return nil
	}

	messages := feed.FormatPostsAsMessages(posts)

	for _, message := range messages {
		if err := b.sendMessageWithKeyboard(
			chatID,
			message,
			returnKeyboard,
		); err != nil {
			return fmt.Errorf("failed to send message with keyboard: %w", err)
		}
	}

	return nil
}

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

func (b *Bot) handleCallbackQuery(callback *tgbotapi.CallbackQuery) error {
	switch callback.Data {
	case "menu":
		if _, err := b.api.Request(tgbotapi.NewCallback(
			callback.ID,
			"",
		)); err != nil {
			return fmt.Errorf("failed to send request: %w", err)
		}

		return b.withSpinner(callback.Message.Chat.ID, func() error {
			return b.handleMenuCommand(callback.Message.Chat.ID)
		})
	case "menu_list":
		if _, err := b.api.Request(tgbotapi.NewCallback(
			callback.ID,
			"",
		)); err != nil {
			return fmt.Errorf("failed to send request: %w", err)
		}

		return b.withSpinner(callback.Message.Chat.ID, func() error {
			return b.handleListCommand(callback.Message.Chat.ID, callback.From.ID)
		})
	case "menu_digest":
		if _, err := b.api.Request(tgbotapi.NewCallback(
			callback.ID,
			"",
		)); err != nil {
			return fmt.Errorf("failed to send request: %w", err)
		}

		return b.withSpinner(callback.Message.Chat.ID, func() error {
			return b.handleDigestCommand(callback.Message.Chat.ID, callback.From.ID)
		})
	case "menu_settings":
		if _, err := b.api.Request(tgbotapi.NewCallback(
			callback.ID,
			"",
		)); err != nil {
			return fmt.Errorf("failed to send request: %w", err)
		}

		return b.withSpinner(callback.Message.Chat.ID, func() error {
			return b.handleSettingsCommand(callback.Message.Chat.ID, callback.From.ID)
		})
	}

	if feedIDStr, ok := strings.CutPrefix(callback.Data, "unfollow_"); ok {
		return b.withSpinner(callback.Message.Chat.ID, func() error {
			return b.handleUnfollowQuery(feedIDStr, callback)
		})
	}
	if hourUTCStr, ok := strings.CutPrefix(
		callback.Data,
		"settings_auto_digest_hour_utc_",
	); ok {
		return b.withSpinner(callback.Message.Chat.ID, func() error {
			return b.handleSettingsAutoDigestHourUTCQuery(hourUTCStr, callback)
		})
	}

	return nil
}

func (b *Bot) handleUnfollowQuery(
	feedIDStr string,
	callback *tgbotapi.CallbackQuery,
) error {
	feedID, err := strconv.ParseInt(feedIDStr, 10, 64)
	if err != nil {
		return fmt.Errorf("failed to parse feedID: %w", err)
	}

	if err := b.db.RemoveFeed(feedID); err != nil {
		errs := []error{fmt.Errorf("failed to remove feed: %w", err)}

		if _, sendErr := b.api.Request(tgbotapi.NewCallback(
			callback.ID,
			"❌ Failed to remove feed.",
		)); sendErr != nil {
			errs = append(errs, fmt.Errorf("failed to send request: %w", sendErr))
		}

		return errors.Join(errs...)
	}

	if _, err := b.api.Request(tgbotapi.NewCallback(
		callback.ID,
		"✅ Feed is removed.",
	)); err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}

	return b.handleListCommand(callback.Message.Chat.ID, callback.From.ID)
}

func (b *Bot) handleSettingsAutoDigestHourUTCQuery(
	hourUTCStr string,
	callback *tgbotapi.CallbackQuery,
) error {
	hourUTC, err := strconv.ParseInt(hourUTCStr, 10, 64)
	if err != nil {
		errs := []error{fmt.Errorf("failed to parse hourUTC: %w", err)}

		if _, sendErr := b.api.Request(tgbotapi.NewCallback(
			callback.ID,
			"❌ Failed to update settings.",
		)); sendErr != nil {
			errs = append(errs, fmt.Errorf("failed to send request: %w", sendErr))
		}

		return errors.Join(errs...)
	}

	if err := b.db.UpsertUserSettings(&model.UserSettings{
		UserID:            callback.From.ID,
		AutoDigestHourUTC: hourUTC,
	}); err != nil {
		errs := []error{fmt.Errorf("failed to upsert user settings: %w", err)}

		if _, sendErr := b.api.Request(tgbotapi.NewCallback(
			callback.ID,
			"❌ Failed to update settings.",
		)); sendErr != nil {
			errs = append(errs, fmt.Errorf("failed to send request: %w", sendErr))
		}

		return errors.Join(errs...)
	}

	if _, err := b.api.Request(tgbotapi.NewCallback(
		callback.ID,
		"✅ Settings are updated.",
	)); err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}

	return b.handleSettingsCommand(callback.Message.Chat.ID, callback.From.ID)
}

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
