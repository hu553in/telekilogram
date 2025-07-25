# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code)
when working with code in this repository.

## Development Commands

This project uses Just (`justfile`) for task automation:

- `just all` - Run check and start the application
- `just check` - Run full check: install deps, lint, format, test, and build
- `just install-deps` - Download Go module dependencies
- `just lint` - Run `golangci-lint`
- `just fmt` - Format code with `golangci-lint`
- `just test` - Run tests with coverage output to `./build/coverage.out`
- `just build` - Build binary to `./build/app`
- `just run` - Execute the built application

## Architecture Overview

This is a Telegram bot written in Go that helps users manage feeds.
The architecture follows a clear separation of concerns:

### Core Components

- **main.go**: Entry point that initializes all components,
  handles environment variables (`TOKEN`, `DB_PATH`, `ALLOWED_USERS`),
  and sets up graceful shutdown
- **bot/**: Telegram bot interface handling user commands with deep link support
  (`/start`, `/menu`, `/list`, `/digest`, `/filter`, `/settings`),
  callback queries with helper functions for error handling, and message processing
- **database/**: SQLite database layer with embedded migrations,
  managing feed storage, user associations and settings
- **feed/**: Feed processing system with fetcher, parser, and URL validation utilities
- **scheduler/**: Cron-based scheduler that automatically sends digests daily
  (default - 00:00 UTC)
- **models/**: Data structures for `Feed`, `UserFeed`, `Post` and `UserSettings` entities
- **markdown/**: Markdown utilities

### Key Patterns

- Uses structured logging with `log/slog` throughout with contextual information
- Database migrations are embedded in the binary using `//go:embed`
- Error handling uses `fmt.Errorf` for error context wrapping
  and `errors.Join()` for collecting multiple errors from concurrent operations
- User authorization is handled via `ALLOWED_USERS` environment variable
  (comma-separated `int64` list)
- Feed URLs are automatically detected from user messages and validated
- Bot responses use inline keyboards for navigation with improved separation
  between command handlers, callback query handlers, and helper functions
- Feed list displays unfollow links using deep links

### Environment Configuration

Required:

- `TOKEN`: Telegram bot token

Optional:

- `DB_PATH`: SQLite database path (defaults to `./db`)
- `ALLOWED_USERS`: Comma-separated list of allowed Telegram user IDs (empty = allow all)

### Database Schema

- `feeds` table: stores `user_id`, `url`, `title` associations with auto-generated IDs
- `user_settings` table: stores `user_id`, `auto_digest_hour_utc` associations
- Migrations in `database/migrations/` are automatically applied on startup

### Feed Processing

- Supports feeds via `github.com/mmcdole/gofeed`
- Filters posts to last 24 hours for digest functionality
- Formats posts as Telegram messages with MarkdownV2 escaping
- Handles feed parsing errors gracefully
