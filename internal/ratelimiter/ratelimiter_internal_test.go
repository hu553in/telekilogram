package ratelimiter

import (
	"context"
	"errors"
	"telekilogram/internal/config"
	"testing"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func TestGetDelay(t *testing.T) {
	now := time.Now()
	cfg := config.RateLimiterConfig{
		PrivateChatRate: time.Second,
		GroupChatRate:   3 * time.Second,
	}

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
			rl := &RateLimiter{cfg: cfg}
			got := rl.getDelay(test.chatID, test.lastSent)

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
		name   string
		chatID any
		want   int64
	}{
		{
			"Int64 chat ID",
			int64(12345),
			12345,
		},
		{
			"Int chat ID",
			67890,
			67890,
		},
		{
			"String chat ID",
			"@channel",
			0,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := chatIDFromAny(test.chatID)
			if err != nil {
				t.Fatalf("chatIDFromAny() error = %v", err)
			}

			if got != test.want {
				t.Errorf("Expected %v chatID, got %v", test.want, got)
			}
		})
	}
}

func TestChatIDFromAnyUnsupportedType(t *testing.T) {
	_, err := chatIDFromAny(&bot.SendChatActionParams{
		ChatID: 1,
		Action: models.ChatActionTyping,
	})
	if err == nil {
		t.Fatal("expected error for unsupported chat ID type")
	}
}

func TestHandleRequestSkipsCanceledRequest(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	rl := &RateLimiter{
		lastSent: make(map[int64]time.Time),
		ctx:      context.Background(),
	}

	called := false
	req := request{
		chatID:   123,
		ctx:      ctx,
		response: make(chan response, 1),
		run: func(context.Context) response {
			called = true
			return response{}
		},
	}

	rl.handleRequest(req)

	if called {
		t.Fatal("expected canceled request to be skipped")
	}

	resp := <-req.response
	if !errors.Is(resp.err, context.Canceled) {
		t.Fatalf("expected context canceled error, got %v", resp.err)
	}

	if _, ok := rl.lastSent[req.chatID]; ok {
		t.Fatal("expected canceled request to not update lastSent")
	}
}

func TestGetRate(t *testing.T) {
	cfg := config.RateLimiterConfig{
		PrivateChatRate: time.Second,
		GroupChatRate:   3 * time.Second,
	}

	tests := []struct {
		name   string
		chatID int64
		want   time.Duration
	}{
		{
			"PrivateChatRate",
			1,
			cfg.PrivateChatRate,
		},
		{
			"GroupChatRate",
			-1,
			cfg.GroupChatRate,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			rl := &RateLimiter{cfg: cfg}
			got := rl.getRate(test.chatID)

			if got != test.want {
				t.Errorf("Expected %v rate, got %v", test.want, got)
			}
		})
	}
}
