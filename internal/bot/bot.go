package bot

import (
	"context"
	"log/slog"
	"slices"
	"strings"
	"telekilogram/internal/database"
	"telekilogram/internal/feed"
	"telekilogram/internal/ratelimiter"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

const updateProcessingTimeout = 60 * time.Second

type Bot struct {
	api         *bot.Bot
	rateLimiter *ratelimiter.RateLimiter
	db          *database.Database
	fetcher     *feed.Fetcher

	allowedUsers []int64

	returnKeyboard                    [][]models.InlineKeyboardButton
	settingsAutoDigestHourUTCKeyboard [][]models.InlineKeyboardButton
	menuKeyboard                      [][]models.InlineKeyboardButton

	log *slog.Logger
}

func New(
	token string,
	db *database.Database,
	fetcher *feed.Fetcher,
	allowedUsers []int64,
	log *slog.Logger,
) (*Bot, error) {
	token = strings.TrimSpace(token)
	allowedUpdates := bot.AllowedUpdates{
		models.AllowedUpdateMessage,
		models.AllowedUpdateCallbackQuery,
	}

	b := &Bot{
		db:      db,
		fetcher: fetcher,

		allowedUsers: allowedUsers,

		returnKeyboard:                    getReturnKeyboard(),
		settingsAutoDigestHourUTCKeyboard: getSettingsAutoDigestHourUTCKeyboard(),
		menuKeyboard:                      getMenuKeyboard(),

		log: log,
	}

	api, err := bot.New(
		token,
		bot.WithAllowedUpdates(allowedUpdates),
		bot.WithErrorsHandler(func(err error) {
			log.Error("Telegram runtime error", "error", err)
		}),
		bot.WithNotAsyncHandlers(),
		bot.WithDefaultHandler(func(ctx context.Context, _ *bot.Bot, update *models.Update) {
			b.handleUpdate(ctx, update)
		}),
	)
	if err != nil {
		return nil, err
	}

	b.api = api
	b.rateLimiter = ratelimiter.New(api, log)

	return b, nil
}

func (b *Bot) Start(ctx context.Context) {
	b.api.Start(ctx)
}

func (b *Bot) Stop() {
	if b.rateLimiter != nil {
		b.rateLimiter.Stop()
	}
}

func (b *Bot) handleUpdate(ctx context.Context, update *models.Update) {
	updateCtx, cancel := context.WithTimeout(ctx, updateProcessingTimeout)
	defer cancel()

	switch {
	case update.Message != nil:
		chatID, chatType := chatContext(&update.Message.Chat)

		if update.Message.From == nil {
			b.log.ErrorContext(updateCtx, "Message update has no sender",
				"updateID", update.ID,
				"messageID", update.Message.ID,
				"chatID", chatID,
				"chatType", chatType)
			return
		}

		userID := update.Message.From.ID
		if !b.userAllowed(update.Message.From.ID) {
			b.log.DebugContext(updateCtx, "User is not allowed",
				"userID", userID,
				"chatID", chatID,
				"username", username(update.Message.From),
				"chatType", chatType)
			return
		}

		if err := b.handleMessage(updateCtx, update.Message); err != nil {
			b.log.ErrorContext(updateCtx, "Failed to handle message",
				"error", err,
				"chatID", chatID,
				"userID", userID,
				"chatType", chatType,
				"messageID", update.Message.ID)
		}

	case update.CallbackQuery != nil:
		chatID := callbackChatID(update.CallbackQuery)
		message := callbackMessage(update.CallbackQuery)
		if message == nil {
			err := b.answerCallbackError(updateCtx, update.CallbackQuery, "❌ Failed.")

			args := []any{
				"callbackQueryID", update.CallbackQuery.ID,
				"userID", update.CallbackQuery.From.ID,
				"chatID", chatID,
				"data", update.CallbackQuery.Data,
			}
			if err != nil {
				args = append(args, "answerError", err)
			}

			b.log.WarnContext(updateCtx, "Callback query has no accessible message", args...)
			return
		}

		if !b.userAllowed(update.CallbackQuery.From.ID) {
			b.log.DebugContext(updateCtx, "User is not allowed",
				"callbackQueryID", update.CallbackQuery.ID,
				"userID", update.CallbackQuery.From.ID,
				"chatID", chatID,
				"username", update.CallbackQuery.From.Username,
				"data", update.CallbackQuery.Data)
			return
		}

		if err := b.handleCallbackQuery(updateCtx, update.CallbackQuery); err != nil {
			b.log.ErrorContext(updateCtx, "Failed to handle callback query",
				"error", err,
				"callbackQueryID", update.CallbackQuery.ID,
				"chatID", chatID,
				"userID", update.CallbackQuery.From.ID,
				"data", update.CallbackQuery.Data,
				"messageID", callbackMessageID(update.CallbackQuery))
		}
	}
}

func (b *Bot) userAllowed(userID int64) bool {
	return len(b.allowedUsers) == 0 || slices.Contains(b.allowedUsers, userID)
}

func chatContext(chat *models.Chat) (int64, string) {
	if chat == nil {
		return 0, ""
	}
	return chat.ID, string(chat.Type)
}

func username(user *models.User) string {
	if user == nil {
		return ""
	}
	return user.Username
}

func callbackChatID(cb *models.CallbackQuery) int64 {
	if message := callbackMessage(cb); message != nil {
		return message.Chat.ID
	}
	return 0
}

func callbackMessageID(cb *models.CallbackQuery) int {
	if message := callbackMessage(cb); message != nil {
		return message.ID
	}
	return 0
}

func callbackMessage(cb *models.CallbackQuery) *models.Message {
	if cb == nil {
		return nil
	}
	return cb.Message.Message
}

func (b *Bot) answerCallbackError(ctx context.Context, callback *models.CallbackQuery, text string) error {
	if callback == nil {
		return nil
	}

	_, err := b.rateLimiter.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
		CallbackQueryID: callback.ID,
		Text:            text,
	})
	return err
}
