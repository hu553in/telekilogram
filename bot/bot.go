package bot

import (
	"errors"
	"fmt"
	"log/slog"
	"net/url"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"telekilogram/database"
	"telekilogram/feed"
	"telekilogram/models"
	"telekilogram/ratelimiter"
)

type Bot struct {
	api          *tgbotapi.BotAPI
	rateLimiter  *ratelimiter.RateLimiter
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

	rateLimiter := ratelimiter.New(api)

	return &Bot{
		api:          api,
		rateLimiter:  rateLimiter,
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

	var tgPosts []models.Post
	var otherPosts []models.Post

	for _, p := range posts {
		if u, err := url.Parse(p.FeedURL); err == nil && u.Host == feed.TelegramHost {
			tgPosts = append(tgPosts, p)
		} else {
			otherPosts = append(otherPosts, p)
		}
	}

	var errs []error

	if len(otherPosts) > 0 {
		messages := feed.FormatPostsAsMessages(otherPosts)

		for _, message := range messages {
			if err := b.sendMessageWithKeyboard(
				chatID,
				message,
				returnKeyboard,
			); err != nil {
				errs = append(
					errs,
					fmt.Errorf("failed to send message with keyboard: %w", err),
				)
			}
		}
	}

	for _, p := range tgPosts {
		if err := b.sendURLWithPreview(chatID, p.URL); err != nil {
			errs = append(
				errs,
				fmt.Errorf("failed to send URL with preview: %w", err),
			)
		}
	}

	if len(tgPosts) > 0 {
		if err := b.sendMessageWithKeyboard(
			chatID,
			"❔ *Choose an option:*",
			returnKeyboard,
		); err != nil {
			errs = append(
				errs,
				fmt.Errorf("failed to send message with keyboard: %w", err),
			)
		}
	}

	return errors.Join(errs...)
}

func (b *Bot) Stop() {
	if b.rateLimiter != nil {
		b.rateLimiter.Stop()
	}
}
