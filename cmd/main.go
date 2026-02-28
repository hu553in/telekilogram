package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"telekilogram/internal/bot"
	"telekilogram/internal/config"
	"telekilogram/internal/database"
	"telekilogram/internal/feed"
	"telekilogram/internal/scheduler"
	"telekilogram/internal/summarizer"
	"time"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(log)

	start := time.Now()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := config.LoadConfig()

	db, err := database.New(ctx, cfg.DBPath, log)
	if err != nil {
		log.ErrorContext(ctx, "Failed to initialize db",
			"error", err,
			"dbPath", cfg.DBPath)

		return
	}
	log.InfoContext(ctx, "DB is initialized",
		"dbPath", cfg.DBPath)

	summarizer := initOpenAISummarizer(ctx, cfg.OpenAIAPIKey, log)
	fetcher := feed.NewFetcher(db, summarizer, log)

	botInst, err := bot.New(cfg.Token, db, fetcher, cfg.AllowedUsers, log)
	if err != nil {
		log.ErrorContext(ctx, "Failed to initialize bot",
			"error", err,
			"allowedUsersCount", len(cfg.AllowedUsers))

		return
	}
	log.InfoContext(ctx, "Bot is initialized",
		"allowedUsersCount", len(cfg.AllowedUsers))

	sched := scheduler.New(ctx, botInst, fetcher, log)

	if err = sched.Start(); err != nil {
		log.ErrorContext(ctx, "Failed to start scheduler",
			"error", err,
			"spec", scheduler.HourlyDigestSpec,
			"timezone", time.FixedZone(scheduler.Timezone, scheduler.TimezoneOffsetSeconds).String())

		return
	}
	defer sched.Stop()
	log.InfoContext(ctx, "Scheduler is started",
		"spec", scheduler.HourlyDigestSpec,
		"timezone", time.FixedZone(scheduler.Timezone, scheduler.TimezoneOffsetSeconds).String())

	go func() {
		botInst.Start(ctx)
	}()
	log.InfoContext(ctx, "Bot is started",
		"updateTimeoutSeconds", bot.BotUpdateTimeout)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	sig := <-c
	log.InfoContext(ctx, "Shutdown signal is received",
		"signal", sig.String())
	cancel()

	log.InfoContext(ctx, "Exiting...",
		"signal", sig.String(),
		"uptimeSeconds", time.Since(start).Seconds())

	botInst.Stop()
	log.InfoContext(ctx, "Bot is stopped",
		"uptimeSeconds", time.Since(start).Seconds())
}

func initOpenAISummarizer(ctx context.Context, apiKey string, log *slog.Logger) summarizer.Summarizer {
	if apiKey == "" {
		log.WarnContext(ctx, "OPENAI_API_KEY is missing so fallback will be used",
			"envVar", "OPENAI_API_KEY")

		return nil
	}

	s, err := summarizer.NewOpenAISummarizer(apiKey)
	if err != nil {
		log.ErrorContext(ctx, "Failed to create OpenAI summarizer so fallback will be used",
			"error", err,
			"envVar", "OPENAI_API_KEY")

		return nil
	}

	log.InfoContext(ctx, "OpenAI summarizer is initialized",
		"provider", "openai")

	return s
}
