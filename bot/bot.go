package bot

import (
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"telekilogram/common"
	"telekilogram/database"
	"telekilogram/feed"
	"telekilogram/model"
)

const welcomeText = `🤖 *Welcome to Telekilogram\!*

I'm your feed assistant\. I can help you:

– Follow feeds by sending me URLs
– Get feed list with /list
– Unfollow feeds directly from list
– Receive auto\-digest \(now\-24h\) automatically each 00:00 UTC
– Receive digest \(now\-24h\) with /digest`

const filterText = `Telekilogram does not support filtering\.\.\.

But you can use awesome [siftrss](https://siftrss.com/) instead\! ✨
It's totally great\. Bot author is also using it\.`

var (
	menuKeyboard = [][]tgbotapi.InlineKeyboardButton{
		{
			tgbotapi.NewInlineKeyboardButtonData("📄 Feed list", "menu_list"),
			tgbotapi.NewInlineKeyboardButtonData("👈 Digest (now-24h)", "menu_digest"),
		},
	}
	returnKeyboard = [][]tgbotapi.InlineKeyboardButton{
		{tgbotapi.NewInlineKeyboardButtonData("⬅️ Return to menu", "menu")},
	}
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
	updateConfig.Timeout = 60

	updates := b.api.GetUpdatesChan(updateConfig)

	for update := range updates {
		switch {
		case update.Message != nil:
			if !b.userAllowed(update.Message.From.ID) {
				return
			}
			if err := b.handleMessage(update.Message); err != nil {
				slog.Error("Failed to handle message", slog.Any("error", err))
			}
		case update.CallbackQuery != nil:
			if !b.userAllowed(update.CallbackQuery.From.ID) {
				return
			}
			if err := b.handleCallbackQuery(update.CallbackQuery); err != nil {
				slog.Error("Failed to handle callback query", slog.Any("error", err))
			}
		}
	}
}

func (b *Bot) SendNewPosts(chatID int64, posts []model.Post) error {
	if len(posts) == 0 {
		return nil
	}

	messages := feed.FormatPostsAsMessages(posts)
	var errs []error

	for _, message := range messages {
		err := b.sendMessageWithKeyboard(
			chatID,
			message,
			returnKeyboard,
		)
		if err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
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
	default:
		return b.withSpinner(message.Chat.ID, func() error {
			return b.handleRandomText(text, userID, message)
		})
	}
}

func (b *Bot) handleListCommand(chatID int64, userID int64) error {
	feeds, err := b.db.GetUserFeeds(userID)
	if err != nil {
		return err
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

	keyboard := make([][]tgbotapi.InlineKeyboardButton, 0, len(feeds)+len(returnKeyboard))
	var errs []error

	for i, f := range feeds {
		message.WriteString(fmt.Sprintf(
			"%d\\. [%s](%s)\n",
			i+1,
			common.EscapeMarkdown(f.Title),
			f.URL,
		))

		button := tgbotapi.NewInlineKeyboardButtonData(
			fmt.Sprintf("Unfollow %d", i+1),
			fmt.Sprintf("unfollow_%d", f.ID),
		)
		keyboard = append(keyboard, []tgbotapi.InlineKeyboardButton{button})
	}

	keyboard = append(keyboard, returnKeyboard...)

	err = b.sendMessageWithKeyboard(chatID, message.String(), keyboard)
	if err != nil {
		errs = append(errs, err)
	}

	return errors.Join(errs...)
}

func (b *Bot) handleMenuCommand(chatID int64) error {
	return b.sendMessageWithKeyboard(chatID, "❔ Choose an option:", menuKeyboard)
}

func (b *Bot) handleDigestCommand(chatID int64, userID int64) error {
	var errs []error

	userPosts, err := b.fetcher.FetchFeeds(&userID)
	if err != nil {
		errs = append(errs, err)
	}

	if len(userPosts) == 0 {
		err := b.sendMessageWithKeyboard(
			chatID,
			"✖️ Feed list is empty or there is a bug\\.",
			returnKeyboard,
		)
		if err != nil {
			errs = append(errs, err)
		}
	}

	for _, posts := range userPosts {
		err := b.SendNewPosts(chatID, posts)
		if err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func (b *Bot) handleRandomText(text string, userID int64, message *tgbotapi.Message) error {
	feeds, err := feed.FindValidFeeds(text)
	if len(feeds) == 0 {
		return errors.Join(err, b.sendMessageWithKeyboard(
			message.Chat.ID,
			"✖️ Valid feed URLs are not found\\. Ignoring\\.",
			returnKeyboard,
		))
	}

	var errs []error
	if err != nil {
		errs = append(errs, err)
	}

	savedCount := 0
	for _, feed := range feeds {
		err := b.db.AddFeed(userID, feed.URL, feed.Title)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		savedCount++
	}

	if savedCount > 0 {
		if savedCount == len(feeds) {
			err := b.sendMessageWithKeyboard(
				message.Chat.ID,
				"✅ Saved\\.",
				returnKeyboard,
			)
			if err != nil {
				errs = append(errs, err)
			}
		} else {
			err := b.sendMessageWithKeyboard(
				message.Chat.ID,
				"❌ Partially saved with errors\\.",
				returnKeyboard,
			)
			if err != nil {
				errs = append(errs, err)
			}
		}
	} else {
		err := b.sendMessageWithKeyboard(
			message.Chat.ID,
			"❌ Failed to save anything\\.",
			returnKeyboard,
		)
		if err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

func (b *Bot) handleCallbackQuery(callback *tgbotapi.CallbackQuery) error {
	if feedIDStr, ok := strings.CutPrefix(callback.Data, "unfollow_"); ok {
		return b.withSpinner(callback.Message.Chat.ID, func() error {
			feedID, err := strconv.ParseInt(feedIDStr, 10, 64)
			if err != nil {
				return err
			}

			if err = b.db.RemoveFeed(feedID); err != nil {
				_, sendErr := b.api.Request(tgbotapi.NewCallback(
					callback.ID,
					"❌ Failed to remove feed\\.",
				))
				return errors.Join(err, sendErr)
			}

			_, err = b.api.Request(tgbotapi.NewCallback(callback.ID, "✅ Feed is removed\\."))
			if err != nil {
				return err
			}

			return b.handleListCommand(callback.Message.Chat.ID, callback.From.ID)
		})
	} else if callback.Data == "menu" {
		_, err := b.api.Request(tgbotapi.NewCallback(callback.ID, ""))
		if err != nil {
			return err
		}

		return b.withSpinner(callback.Message.Chat.ID, func() error {
			return b.handleMenuCommand(callback.Message.Chat.ID)
		})
	} else if callback.Data == "menu_list" {
		_, err := b.api.Request(tgbotapi.NewCallback(callback.ID, ""))
		if err != nil {
			return err
		}

		return b.withSpinner(callback.Message.Chat.ID, func() error {
			return b.handleListCommand(callback.Message.Chat.ID, callback.From.ID)
		})
	} else if callback.Data == "menu_digest" {
		_, err := b.api.Request(tgbotapi.NewCallback(callback.ID, ""))
		if err != nil {
			return err
		}

		return b.withSpinner(callback.Message.Chat.ID, func() error {
			return b.handleDigestCommand(callback.Message.Chat.ID, callback.From.ID)
		})
	}

	return nil
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

	_, err := b.api.Send(msg)
	return err
}

func (b *Bot) sendChatAction(chatID int64, action string) error {
	config := tgbotapi.NewChatAction(chatID, action)
	_, err := b.api.Request(config)
	return err
}

func (b *Bot) withSpinner(chatID int64, callback func() error) error {
	if err := b.sendChatAction(chatID, tgbotapi.ChatTyping); err != nil {
		return err
	}
	return callback()
}

func (b *Bot) userAllowed(userID int64) bool {
	return len(b.allowedUsers) == 0 || slices.Contains(b.allowedUsers, userID)
}
