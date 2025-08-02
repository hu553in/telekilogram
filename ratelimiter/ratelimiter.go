package ratelimiter

import (
	"context"
	"log/slog"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
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
}

func New(api *tgbotapi.BotAPI) *RateLimiter {
	ctx, cancel := context.WithCancel(context.Background())

	rl := &RateLimiter{
		api:      api,
		queue:    make(chan request, queueSize),
		lastSent: make(map[int64]time.Time),
		ctx:      ctx,
		cancel:   cancel,
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
			slog.Debug("Rate limiting message",
				slog.Int64("chatID", chatID),
				slog.Duration("delay", delay))

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
