package ratelimiter

import (
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func getChatID(message tgbotapi.Chattable) int64 {
	switch m := message.(type) {
	case tgbotapi.MessageConfig:
		return m.ChatID
	case tgbotapi.EditMessageTextConfig:
		return m.ChatID
	case tgbotapi.DeleteMessageConfig:
		return m.ChatID
	case tgbotapi.ChatActionConfig:
		return m.ChatID
	default:
		return 0
	}
}

func getDelay(
	chatID int64,
	lastSent time.Time,
) time.Duration {
	elapsed := time.Since(lastSent)
	rate := getRate(chatID)

	return max(rate-elapsed, 0)
}

func getRate(chatID int64) time.Duration {
	if chatID < 0 {
		return groupChatRate
	} else {
		return privateChatRate
	}
}
