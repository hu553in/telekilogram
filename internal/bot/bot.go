package bot

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"telekilogram/internal/database"
	"telekilogram/internal/feed"
	"telekilogram/internal/models"
	"telekilogram/internal/ratelimiter"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const (
	maxBackoffSeconds         = 60
	initialBackoffSeconds     = 3
	backoffGrowthFactor       = 2
	resetOffsetBackoffSeconds = 30
	updateProcessingTimeout   = 60 * time.Second

	BotUpdateTimeout = 60

	hoursPerDay                                     = 24
	settingsAutoDigestHourUTCKeyboardRowSize        = 5
	settingsAutoDigestHourUTCKeyboardCallbackPrefix = "settings_auto_digest_hour_utc_"
)

type Bot struct {
	api                               *tgbotapi.BotAPI
	rateLimiter                       *ratelimiter.RateLimiter
	db                                *database.Database
	fetcher                           *feed.Fetcher
	allowedUsers                      []int64
	returnKeyboard                    [][]tgbotapi.InlineKeyboardButton
	settingsAutoDigestHourUTCKeyboard [][]tgbotapi.InlineKeyboardButton
	menuKeyboard                      [][]tgbotapi.InlineKeyboardButton
	log                               *slog.Logger
}

func New(
	token string,
	db *database.Database,
	fetcher *feed.Fetcher,
	allowedUsers []int64,
	log *slog.Logger,
) (*Bot, error) {
	token = strings.TrimSpace(token)

	api, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, err
	}

	rateLimiter := ratelimiter.New(api, log)

	return &Bot{
		api:                               api,
		rateLimiter:                       rateLimiter,
		db:                                db,
		fetcher:                           fetcher,
		allowedUsers:                      allowedUsers,
		returnKeyboard:                    getReturnKeyboard(),
		settingsAutoDigestHourUTCKeyboard: getSettingsAutoDigestHourUTCKeyboard(),
		menuKeyboard:                      getMenuKeyboard(),
		log:                               log,
	}, nil
}

func (b *Bot) Start(ctx context.Context) {
	updateConfig := tgbotapi.NewUpdate(0)
	updateConfig.Timeout = BotUpdateTimeout

	backoffSeconds := initialBackoffSeconds

	for {
		select {
		case <-ctx.Done():
			b.log.InfoContext(ctx, "Bot context is done",
				"error", ctx.Err())
			return
		default:
		}

		updates := b.api.GetUpdatesChan(updateConfig)
		updatesClosed := false

		for !updatesClosed {
			select {
			case <-ctx.Done():
				b.log.InfoContext(ctx, "Bot context is done",
					"error", ctx.Err())
				return

			case update, ok := <-updates:
				if !ok {
					updatesClosed = true
					continue
				}
				updateConfig.Offset = update.UpdateID + 1

				b.handleUpdate(ctx, &update)
			}
		}

		if ctx.Err() != nil {
			return
		}

		b.log.WarnContext(ctx, "Update channel is closed, reconnecting...",
			"offset", updateConfig.Offset,
			"backoffSeconds", backoffSeconds)

		time.Sleep(time.Duration(backoffSeconds) * time.Second)

		backoffSeconds = updateBackoffSeconds(backoffSeconds)

		if backoffSeconds >= resetOffsetBackoffSeconds {
			updateConfig.Offset = 0
		}
	}
}

func (b *Bot) handleUpdate(ctx context.Context, update *tgbotapi.Update) {
	updateCtx, cancel := context.WithTimeout(ctx, updateProcessingTimeout)
	defer cancel()

	switch {
	case update.Message != nil:
		chatID, chatType := chatContext(update.Message.Chat)

		userID := update.Message.From.ID
		if !b.userAllowed(update.Message.From.ID) {
			b.log.DebugContext(updateCtx, "User is not allowed",
				"userID", userID,
				"chatID", chatID,
				"username", update.Message.From.UserName,
				"chatType", chatType)

			return
		}

		if err := b.handleMessage(updateCtx, update.Message); err != nil {
			b.log.ErrorContext(updateCtx, "Failed to handle message",
				"error", err,
				"chatID", chatID,
				"userID", userID,
				"chatType", chatType,
				"messageID", update.Message.MessageID)
		}

	case update.CallbackQuery != nil:
		chatID := callbackChatID(update.CallbackQuery)

		if !b.userAllowed(update.CallbackQuery.From.ID) {
			b.log.DebugContext(updateCtx, "User is not allowed",
				"userID", update.CallbackQuery.From.ID,
				"chatID", chatID,
				"username", update.CallbackQuery.From.UserName,
				"data", update.CallbackQuery.Data)

			return
		}

		if err := b.handleCallbackQuery(updateCtx, update.CallbackQuery); err != nil {
			b.log.ErrorContext(updateCtx, "Failed to handle callback query",
				"error", err,
				"chatID", chatID,
				"userID", update.CallbackQuery.From.ID,
				"data", update.CallbackQuery.Data,
				"messageID", callbackMessageID(update.CallbackQuery))
		}
	}
}

func chatContext(chat *tgbotapi.Chat) (int64, string) {
	if chat == nil {
		return 0, ""
	}

	return chat.ID, chat.Type
}

func callbackChatID(cb *tgbotapi.CallbackQuery) int64 {
	if cb != nil && cb.Message != nil && cb.Message.Chat != nil {
		return cb.Message.Chat.ID
	}

	return 0
}

func callbackMessageID(cb *tgbotapi.CallbackQuery) int {
	if cb != nil && cb.Message != nil {
		return cb.Message.MessageID
	}

	return 0
}

func (b *Bot) SendNewPosts(ctx context.Context, chatID int64, posts []models.Post) error {
	if len(posts) == 0 {
		return nil
	}

	var errs []error

	messages := feed.FormatPostsAsMessages(ctx, posts, b.log)

	for _, message := range messages {
		if err := b.sendMessageWithKeyboard(chatID, message, b.returnKeyboard); err != nil {
			errs = append(errs, fmt.Errorf("failed to send message with keyboard: %w", err))
		}
	}

	return errors.Join(errs...)
}

func (b *Bot) Stop() {
	if b.rateLimiter != nil {
		b.rateLimiter.Stop()
	}
}

func updateBackoffSeconds(backoffSeconds int) int {
	if backoffSeconds < maxBackoffSeconds {
		backoffSeconds *= backoffGrowthFactor
		if backoffSeconds > maxBackoffSeconds {
			backoffSeconds = maxBackoffSeconds
		}
	}
	return backoffSeconds
}

func getReturnKeyboard() [][]tgbotapi.InlineKeyboardButton {
	return [][]tgbotapi.InlineKeyboardButton{
		{tgbotapi.NewInlineKeyboardButtonData("⬅️ Return to menu", "menu")},
	}
}

func getMenuKeyboard() [][]tgbotapi.InlineKeyboardButton {
	return [][]tgbotapi.InlineKeyboardButton{
		{
			tgbotapi.NewInlineKeyboardButtonData("📄 Feed list", "menu_list"),
			tgbotapi.NewInlineKeyboardButtonData("👈 24h digest", "menu_digest"),
		},
		{
			tgbotapi.NewInlineKeyboardButtonData("⚙️ Settings", "menu_settings"),
		},
	}
}

func getSettingsAutoDigestHourUTCKeyboard() [][]tgbotapi.InlineKeyboardButton {
	var keyboard [][]tgbotapi.InlineKeyboardButton

	for i := 0; i < hoursPerDay; i += settingsAutoDigestHourUTCKeyboardRowSize {
		var row []tgbotapi.InlineKeyboardButton

		for j := i; j < i+settingsAutoDigestHourUTCKeyboardRowSize && j < hoursPerDay; j++ {
			hour := fmt.Sprintf("%02d", j)
			row = append(
				row,
				tgbotapi.NewInlineKeyboardButtonData(hour, settingsAutoDigestHourUTCKeyboardCallbackPrefix+hour),
			)
		}

		keyboard = append(keyboard, row)
	}

	return keyboard
}
