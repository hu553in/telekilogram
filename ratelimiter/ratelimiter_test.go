package ratelimiter

import (
	"testing"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func TestGetDelay(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		chatID   int64
		lastSent time.Time
		wantZero bool
	}{
		{
			"Private chat - no delay needed",
			123456789,
			now.Add(-2 * time.Second),
			true,
		},
		{
			"Private chat - delay needed",
			123456789,
			now.Add(-500 * time.Millisecond),
			false,
		},
		{
			"Group chat - no delay needed",
			-123456789,
			now.Add(-4 * time.Second),
			true,
		},
		{
			"Group chat - delay needed",
			-123456789,
			now.Add(-1 * time.Second),
			false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := getDelay(test.chatID, test.lastSent)

			if test.wantZero && got > 0 {
				t.Errorf("Expected zero delay, got %v", got)
			}

			if !test.wantZero && got <= 0 {
				t.Errorf("Expected positive delay, got %v", got)
			}
		})
	}
}

func TestGetChatID(t *testing.T) {
	tests := []struct {
		name    string
		message tgbotapi.Chattable
		want    int64
	}{
		{
			"MessageConfig",
			tgbotapi.NewMessage(12345, "test"),
			12345,
		},
		{
			"ChatActionConfig",
			tgbotapi.NewChatAction(67890, tgbotapi.ChatTyping),
			67890,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := getChatID(test.message)

			if got != test.want {
				t.Errorf("Expected %v chatID, got %v", test.want, got)
			}
		})
	}
}

func TestGetRate(t *testing.T) {
	tests := []struct {
		name   string
		chatID int64
		want   time.Duration
	}{
		{
			"PrivateChatRate",
			1,
			privateChatRate,
		},
		{
			"GroupChatRate",
			-1,
			groupChatRate,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := getRate(test.chatID)

			if got != test.want {
				t.Errorf("Expected %v rate, got %v", test.want, got)
			}
		})
	}
}
