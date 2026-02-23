package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"telekilogram/internal/bot"
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

	token := strings.TrimSpace(os.Getenv("TOKEN"))
	if token == "" {
		log.ErrorContext(ctx, "TOKEN is required",
			"envVar", "TOKEN")

		return
	}

	dbPath, db, err := initDatabase(ctx, log)
	if err != nil {
		log.ErrorContext(ctx, "Failed to initialize db",
			"error", err,
			"dbPath", dbPath)

		return
	}
	defer func() {
		if err = db.Close(); err != nil {
			log.ErrorContext(ctx, "Failed to close db",
				"error", err,
				"dbPath", dbPath)
		}
	}()
	log.InfoContext(ctx, "DB is initialized",
		"dbPath", dbPath)

	allowedUsersTrimmed := strings.TrimSpace(os.Getenv("ALLOWED_USERS"))
	allowedUsersStr := strings.Split(allowedUsersTrimmed, ",")
	allowedUsers, ok := processAllowedUsersStr(allowedUsersStr)
	if !ok {
		log.ErrorContext(ctx, "ALLOWED_USERS must be empty or comma-separated int64 list",
			"ALLOWED_USERS", allowedUsersTrimmed)

		return
	}

	summarizer := initOpenAISummarizer(ctx, log)
	fetcher := feed.NewFetcher(db, summarizer, log)

	botInst, err := bot.New(token, db, fetcher, allowedUsers, log)
	if err != nil {
		log.ErrorContext(ctx, "Failed to initialize bot",
			"error", err,
			"allowedUsersCount", len(allowedUsers))

		return
	}
	log.InfoContext(ctx, "Bot is initialized",
		"allowedUsersCount", len(allowedUsers))

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

func processAllowedUsersStr(allowedUsersStr []string) ([]int64, bool) {
	var allowedUsers []int64

	for _, userIDStr := range allowedUsersStr {
		userIDStr = strings.TrimSpace(userIDStr)
		if userIDStr == "" {
			continue
		}

		userID, parseIntErr := strconv.ParseInt(userIDStr, 10, 64)
		if parseIntErr != nil {
			return nil, false
		}
		allowedUsers = append(allowedUsers, userID)
	}

	return allowedUsers, true
}

func initDatabase(ctx context.Context, log *slog.Logger) (string, *database.Database, error) {
	dbPath := strings.TrimSpace(os.Getenv("DB_PATH"))
	if dbPath == "" {
		dbPath = "db.sqlite"
		log.InfoContext(ctx, "Using default DB path",
			"dbPath", dbPath)
	}

	db, err := database.New(ctx, dbPath, log)
	return dbPath, db, err
}

func initOpenAISummarizer(ctx context.Context, log *slog.Logger) summarizer.Summarizer {
	apiKey := strings.TrimSpace(os.Getenv("OPENAI_API_KEY"))
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
