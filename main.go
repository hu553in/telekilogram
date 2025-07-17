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
)

func main() {
	err := godotenv.Load()
	if err != nil {
		slog.Error("Failed to load .env file", slog.Any("error", err))
		return
	}

	token := os.Getenv("TOKEN")
	if token == "" {
		slog.Error("TOKEN is required")
		return
	}

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "./db"
	}

	db, err := database.New(dbPath)
	if err != nil {
		slog.Error("Failed to initialize db", slog.Any("error", err))
		return
	}
	defer func() {
		if err := db.Close(); err != nil {
			slog.Error("Failed to close db", slog.Any("error", err))
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

	fetcher := feed.NewFeedFetcher(db)
	bot, err := bot.New(token, db, fetcher, allowedUsers)
	if err != nil {
		slog.Error("Failed to initialize bot", slog.Any("error", err))
		return
	}
	slog.Info("Bot is initialized")

	scheduler := scheduler.New(bot, fetcher)

	if err = scheduler.Start(); err != nil {
		slog.Error("Failed to start scheduler", slog.Any("error", err))
		return
	}
	defer scheduler.Stop()
	slog.Info("Scheduler is started")

	slog.Info("Starting bot...")
	go func() {
		bot.Start()
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
}
