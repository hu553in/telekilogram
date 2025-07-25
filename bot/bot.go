package bot

import (
	"errors"
	"log/slog"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"telekilogram/database"
	"telekilogram/feed"
	"telekilogram/models"
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
		return nil, err
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
	updateConfig.Timeout = updateTimeout

	updates := b.api.GetUpdatesChan(updateConfig)

	for update := range updates {
		switch {
		case update.Message != nil:
			if !b.userAllowed(update.Message.From.ID) {
				slog.Debug("User is not allowed",
					slog.Int64("userID", update.Message.From.ID))

				return
			}
			if err := b.handleMessage(update.Message); err != nil {
				slog.Error("Failed to handle message",
					slog.Any("err", err),
					slog.Any("update", update))
			}
		case update.CallbackQuery != nil:
			if !b.userAllowed(update.CallbackQuery.From.ID) {
				slog.Debug("User is not allowed",
					slog.Int64("userID", update.CallbackQuery.From.ID))

				return
			}
			if err := b.handleCallbackQuery(update.CallbackQuery); err != nil {
				slog.Error("Failed to handle callback query",
					slog.Any("err", err),
					slog.Any("update", update))
			}
		}
	}
}

func (b *Bot) SendNewPosts(chatID int64, posts []models.Post) error {
	if len(posts) == 0 {
		return nil
	}

	messages := feed.FormatPostsAsMessages(posts)
	var errs []error

	for _, message := range messages {
		if err := b.sendMessageWithKeyboard(
			chatID,
			message,
			returnKeyboard,
		); err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}
