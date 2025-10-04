package main

import (
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/joho/godotenv"

	"telekilogram/bot"
	"telekilogram/database"
	"telekilogram/feed"
	"telekilogram/scheduler"
	"telekilogram/summarizer"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	if err := godotenv.Load(); err != nil {
		slog.Error("Failed to load .env file",
			slog.Any("err", err))

		return
	}
	slog.Info(".env file is loaded")

	token := os.Getenv("TOKEN")
	if token == "" {
		slog.Error("TOKEN is required")

		return
	}

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		slog.Info("Using default DB path")
		dbPath = "./db"
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
	slog.Info("DB is initialized")

	allowedUsersRaw := os.Getenv("ALLOWED_USERS")
	allowedUsersStr := strings.Split(allowedUsersRaw, ",")
	var allowedUsers []int64

	for _, userIDStr := range allowedUsersStr {
		userID, err := strconv.ParseInt(userIDStr, 10, 64)
		if err != nil {
			slog.Error(
				"ALLOWED_USERS must be empty or comma-separated int64 list",
				slog.String("ALLOWED_USERS", allowedUsersRaw),
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
			slog.Any("err", err))

		return
	}
	slog.Info("Bot is initialized")

	scheduler := scheduler.New(botInst, fetcher)

	if err := scheduler.Start(); err != nil {
		slog.Error("Failed to start scheduler",
			slog.Any("err", err))

		return
	}
	defer scheduler.Stop()
	slog.Info("Scheduler is started")

	go func(bot *bot.Bot) {
		bot.Start()
	}(botInst)
	slog.Info("Bot is started")

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
	slog.Info("Exiting...")

	botInst.Stop()
	slog.Info("Bot is stopped")
}

func initOpenAISummarizer() summarizer.Summarizer {
	apiKey := strings.TrimSpace(os.Getenv("OPENAI_API_KEY"))
	if apiKey == "" {
		slog.Warn("OPENAI_API_KEY is missing so fallback will be used")

		return nil
	}

	s, err := summarizer.NewOpenAISummarizer(
		summarizer.OpenAIConfig{APIKey: apiKey},
	)
	if err != nil {
		slog.Error("Failed to create OpenAI summarizer so fallback will be used",
			slog.Any("err", err))

		return nil
	}

	slog.Info("OpenAI summarizer is initialized")

	return s
}
