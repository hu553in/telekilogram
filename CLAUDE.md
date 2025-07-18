# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

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
  handles environment variables (`TOKEN`, `DB_PATH`, `ALLOWED_USERS`), and sets up graceful shutdown
- **bot/**: Telegram bot interface handling user commands (`/start`, `/menu`, `/list`, `/digest`),
  callback queries, and message processing
- **database/**: SQLite database layer with embedded migrations, managing feed storage and user associations
- **feed/**: Feed processing system with fetcher, parser, and URL validation utilities
- **scheduler/**: Cron-based scheduler that automatically sends digests at 00:00 UTC daily
- **model/**: Data structures for Feed and Post entities

### Key Patterns

- Uses structured logging with `log/slog` throughout
- Database migrations are embedded in the binary using `//go:embed`
- Error handling uses `errors.Join()` for collecting multiple errors
- User authorization is handled via `ALLOWED_USERS` environment variable (comma-separated `int64` list)
- Feed URLs are automatically detected from user messages and validated
- Bot responses use inline keyboards for navigation

### Environment Configuration

Required:

- `TOKEN`: Telegram bot token

Optional:

- `DB_PATH`: SQLite database path (defaults to `./db`)
- `ALLOWED_USERS`: Comma-separated list of allowed Telegram user IDs (empty = allow all)

### Database Schema

- `feeds` table: stores `user_id`, `url` associations with auto-generated IDs
- Migrations in `database/migrations/` are automatically applied on startup

### Feed Processing

- Supports feeds via `github.com/mmcdole/gofeed`
- Filters posts to last 24 hours for digest functionality
- Formats posts as Telegram messages with proper escaping
- Handles feed parsing errors gracefully
