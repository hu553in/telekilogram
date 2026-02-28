package ratelimiter

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const (
	privateChatRate = time.Second
	groupChatRate   = 3 * time.Second
	queueSize       = 1000
)

type request struct {
	message  tgbotapi.Chattable
	response chan response
}

type response struct {
	message tgbotapi.Message
	err     error
}

type RateLimiter struct {
	api      *tgbotapi.BotAPI
	queue    chan request
	lastSent map[int64]time.Time
	mu       sync.Mutex
	ctx      context.Context
	cancel   context.CancelFunc
	log      *slog.Logger
}

func New(api *tgbotapi.BotAPI, log *slog.Logger) *RateLimiter {
	ctx, cancel := context.WithCancel(context.Background())

	rl := &RateLimiter{
		api:      api,
		queue:    make(chan request, queueSize),
		lastSent: make(map[int64]time.Time),
		ctx:      ctx,
		cancel:   cancel,
		log:      log,
	}

	go rl.processQueue()

	return rl
}

func (rl *RateLimiter) Send(
	message tgbotapi.Chattable,
) (tgbotapi.Message, error) {
	req := request{
		message:  message,
		response: make(chan response, 1),
	}

	select {
	case rl.queue <- req:
		resp := <-req.response

		return resp.message, resp.err
	case <-rl.ctx.Done():
		return tgbotapi.Message{}, rl.ctx.Err()
	}
}

func (rl *RateLimiter) Request(
	c tgbotapi.Chattable,
) (*tgbotapi.APIResponse, error) {
	return rl.api.Request(c)
}

func (rl *RateLimiter) Stop() {
	rl.cancel()
}

func (rl *RateLimiter) processQueue() {
	for {
		select {
		case req := <-rl.queue:
			rl.handleRequest(req)
		case <-rl.ctx.Done():
			close(rl.queue)

			for req := range rl.queue {
				req.response <- response{
					err: rl.ctx.Err(),
				}
			}

			return
		}
	}
}

func (rl *RateLimiter) handleRequest(req request) {
	chatID := getChatID(req.message)

	rl.mu.Lock()
	lastSent, exists := rl.lastSent[chatID]
	rl.mu.Unlock()

	if exists {
		delay := getDelay(chatID, lastSent)

		if delay > 0 {
			messageType := fmt.Sprintf("%T", req.message)
			rl.log.DebugContext(rl.ctx, "Rate limiting message",
				"chatID", chatID,
				"delay", delay,
				"chattableType", messageType,
				"queueLen", len(rl.queue))

			select {
			case <-time.After(delay):
			case <-rl.ctx.Done():
				req.response <- response{
					err: rl.ctx.Err(),
				}

				return
			}
		}
	}

	message, err := rl.api.Send(req.message)

	rl.mu.Lock()
	rl.lastSent[chatID] = time.Now()
	rl.mu.Unlock()

	req.response <- response{
		message: message,
		err:     err,
	}
}

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
	}
	return privateChatRate
}
