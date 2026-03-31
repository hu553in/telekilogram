package ratelimiter

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"telekilogram/internal/config"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type request struct {
	chatID   int64
	ctx      context.Context
	label    string
	run      func(context.Context) response
	response chan response
}

type response struct {
	message *models.Message
	ok      bool
	err     error
}

type RateLimiter struct {
	api      *bot.Bot
	queue    chan request
	lastSent map[int64]time.Time
	mu       sync.Mutex
	ctx      context.Context
	cancel   context.CancelFunc
	cfg      config.RateLimiterConfig
	log      *slog.Logger
}

func New(api *bot.Bot, cfg config.RateLimiterConfig, log *slog.Logger) *RateLimiter {
	ctx, cancel := context.WithCancel(context.Background())

	rl := &RateLimiter{
		api:      api,
		queue:    make(chan request, cfg.QueueSize),
		lastSent: make(map[int64]time.Time),
		ctx:      ctx,
		cancel:   cancel,
		cfg:      cfg,
		log:      log,
	}

	go rl.processQueue()

	return rl
}

func (rl *RateLimiter) SendMessage(
	ctx context.Context,
	params *bot.SendMessageParams,
) (*models.Message, error) {
	if params == nil {
		return nil, errors.New("send message params are nil")
	}

	chatID, err := chatIDFromAny(params.ChatID)
	if err != nil {
		return nil, err
	}

	resp, err := rl.enqueue(ctx, request{
		chatID: chatID,
		ctx:    ctx,
		label:  "sendMessage",
		run: func(ctx context.Context) response {
			message, sendErr := rl.api.SendMessage(ctx, params)
			return response{
				message: message,
				err:     sendErr,
			}
		},
	})
	if err != nil {
		return nil, err
	}

	return resp.message, nil
}

func (rl *RateLimiter) AnswerCallbackQuery(
	ctx context.Context,
	params *bot.AnswerCallbackQueryParams,
) (bool, error) {
	if params == nil {
		return false, errors.New("answer callback query params are nil")
	}

	return rl.api.AnswerCallbackQuery(ctx, params)
}

func (rl *RateLimiter) SendChatAction(
	ctx context.Context,
	params *bot.SendChatActionParams,
) (bool, error) {
	if params == nil {
		return false, errors.New("send chat action params are nil")
	}

	chatID, err := chatIDFromAny(params.ChatID)
	if err != nil {
		return false, err
	}

	resp, err := rl.enqueue(ctx, request{
		chatID: chatID,
		ctx:    ctx,
		label:  "sendChatAction",
		run: func(ctx context.Context) response {
			ok, sendErr := rl.api.SendChatAction(ctx, params)
			return response{
				ok:  ok,
				err: sendErr,
			}
		},
	})
	if err != nil {
		return false, err
	}

	return resp.ok, nil
}

func (rl *RateLimiter) Stop() {
	rl.cancel()
}

func (rl *RateLimiter) enqueue(ctx context.Context, req request) (response, error) {
	req.response = make(chan response, 1)

	select {
	case rl.queue <- req:
	case <-ctx.Done():
		return response{}, ctx.Err()
	case <-rl.ctx.Done():
		return response{}, rl.ctx.Err()
	}

	select {
	case resp := <-req.response:
		return resp, resp.err
	case <-ctx.Done():
		return response{}, ctx.Err()
	case <-rl.ctx.Done():
		return response{}, rl.ctx.Err()
	}
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
	if err := requestContextError(req.ctx); err != nil {
		req.response <- response{err: err}
		return
	}

	if req.chatID != 0 {
		rl.waitForTurn(req)
		if err := requestContextError(req.ctx); err != nil {
			req.response <- response{err: err}
			return
		}
		if rl.ctx.Err() != nil {
			req.response <- response{err: rl.ctx.Err()}
			return
		}
	}

	resp := req.run(req.ctx)

	if req.chatID != 0 && requestContextError(req.ctx) == nil {
		rl.mu.Lock()
		rl.lastSent[req.chatID] = time.Now()
		rl.mu.Unlock()
	}

	req.response <- resp
}

func (rl *RateLimiter) waitForTurn(req request) {
	rl.mu.Lock()
	lastSent, exists := rl.lastSent[req.chatID]
	rl.mu.Unlock()

	if !exists {
		return
	}

	delay := rl.getDelay(req.chatID, lastSent)
	if delay <= 0 {
		return
	}

	rl.log.DebugContext(rl.ctx, "Rate limiting message",
		"chatID", req.chatID,
		"delay", delay,
		"operation", req.label,
		"queueLen", len(rl.queue))

	select {
	case <-time.After(delay):
	case <-req.ctx.Done():
	case <-rl.ctx.Done():
	}
}

func requestContextError(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
	return ctx.Err()
}

func chatIDFromAny(raw any) (int64, error) {
	switch v := raw.(type) {
	case int64:
		return v, nil
	case int:
		return int64(v), nil
	case string:
		return 0, nil
	default:
		return 0, fmt.Errorf("unsupported chat id type %T", raw)
	}
}

func (rl *RateLimiter) getDelay(
	chatID int64,
	lastSent time.Time,
) time.Duration {
	elapsed := time.Since(lastSent)
	rate := rl.getRate(chatID)

	return max(rate-elapsed, 0)
}

func (rl *RateLimiter) getRate(chatID int64) time.Duration {
	if chatID < 0 {
		return rl.cfg.GroupChatRate
	}
	return rl.cfg.PrivateChatRate
}
