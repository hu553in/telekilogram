package main

import (
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"telekilogram/bot"
	"telekilogram/database"
	"telekilogram/feed"
	"telekilogram/scheduler"
	"telekilogram/summarizer"
)

const envFile = ".env"

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	start := time.Now()

	if err := godotenv.Load(); err != nil {
		slog.Error("Failed to load .env file",
			slog.Any("err", err),
			slog.String("path", envFile))

		return
	}
	slog.Info(".env file is loaded",
		slog.String("path", envFile))

	token := strings.TrimSpace(os.Getenv("TOKEN"))
	if token == "" {
		slog.Error("TOKEN is required",
			slog.String("envVar", "TOKEN"))

		return
	}

	dbPath := strings.TrimSpace(os.Getenv("DB_PATH"))
	if dbPath == "" {
		dbPath = "./db"
		slog.Info("Using default DB path",
			slog.String("dbPath", dbPath))
	}

	db, err := database.New(dbPath)
	if err != nil {
		slog.Error("Failed to initialize db",
			slog.Any("err", err),
			slog.String("dbPath", dbPath))

		return
	}
	defer func() {
		if err := db.Close(); err != nil {
			slog.Error("Failed to close db",
				slog.Any("err", err),
				slog.String("dbPath", dbPath))
		}
	}()
	slog.Info("DB is initialized",
		slog.String("dbPath", dbPath))

	allowedUsersTrimmed := strings.TrimSpace(os.Getenv("ALLOWED_USERS"))
	allowedUsersStr := strings.Split(allowedUsersTrimmed, ",")
	var allowedUsers []int64

	for _, userIDStr := range allowedUsersStr {
		userIDStr = strings.TrimSpace(userIDStr)
		if userIDStr == "" {
			continue
		}

		userID, err := strconv.ParseInt(userIDStr, 10, 64)
		if err != nil {
			slog.Error(
				"ALLOWED_USERS must be empty or comma-separated int64 list",
				slog.String("ALLOWED_USERS", allowedUsersTrimmed),
				slog.String("value", userIDStr),
			)

			return
		}
		allowedUsers = append(allowedUsers, userID)
	}

	summarizer := initOpenAISummarizer()
	fetcher := feed.NewFeedFetcher(db, summarizer)

	botInst, err := bot.New(token, db, fetcher, allowedUsers)
	if err != nil {
		slog.Error("Failed to initialize bot",
			slog.Any("err", err),
			slog.Int("allowedUsersCount", len(allowedUsers)))

		return
	}
	slog.Info("Bot is initialized",
		slog.Int("allowedUsersCount", len(allowedUsers)))

	sched := scheduler.New(botInst, fetcher)

	if err := sched.Start(); err != nil {
		slog.Error("Failed to start scheduler",
			slog.Any("err", err),
			slog.String("spec", scheduler.HourlyDigestSpec),
			slog.String("timezone", scheduler.CronLocation.String()))

		return
	}
	defer sched.Stop()
	slog.Info("Scheduler is started",
		slog.String("spec", scheduler.HourlyDigestSpec),
		slog.String("timezone", scheduler.CronLocation.String()))

	go func(bot *bot.Bot) {
		bot.Start()
	}(botInst)
	slog.Info("Bot is started",
		slog.Int("updateTimeoutSeconds", bot.UpdateTimeout))

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	sig := <-c

	slog.Info("Exiting...",
		slog.String("signal", sig.String()),
		slog.Float64("uptimeSeconds", time.Since(start).Seconds()))

	botInst.Stop()
	slog.Info("Bot is stopped",
		slog.Float64("uptimeSeconds", time.Since(start).Seconds()))
}

func initOpenAISummarizer() summarizer.Summarizer {
	apiKey := strings.TrimSpace(os.Getenv("OPENAI_API_KEY"))
	if apiKey == "" {
		slog.Warn("OPENAI_API_KEY is missing so fallback will be used",
			slog.String("envVar", "OPENAI_API_KEY"))

		return nil
	}

	s, err := summarizer.NewOpenAISummarizer(
		summarizer.OpenAIConfig{APIKey: apiKey},
	)
	if err != nil {
		slog.Error("Failed to create OpenAI summarizer so fallback will be used",
			slog.Any("err", err),
			slog.String("envVar", "OPENAI_API_KEY"))

		return nil
	}

	slog.Info("OpenAI summarizer is initialized",
		slog.String("provider", "openai"))

	return s
}
