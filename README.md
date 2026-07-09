# Telekilogram

[![CI](https://github.com/hu553in/telekilogram/actions/workflows/ci.yml/badge.svg)](https://github.com/hu553in/telekilogram/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/hu553in/telekilogram)](https://goreportcard.com/report/github.com/hu553in/telekilogram)
[![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/hu553in/telekilogram)](https://github.com/hu553in/telekilogram/blob/main/go.mod)

A Telegram bot for feed-based updates, written in Go.

Telekilogram aggregates content from RSS, Atom, and JSON feeds, as well as public Telegram channels.
It delivers daily digests and can optionally summarize Telegram posts using OpenAI. It is built for
unattended operation with predictable behavior.

## What it does

- Follows RSS, Atom, JSON feeds, and public Telegram channels
- Accepts feed URLs, channel `@username` values, and forwarded channel messages
- Sends an automatic daily digest and supports manual `/digest`
- Lists and removes subscriptions from Telegram
- Optionally summarizes Telegram posts through OpenAI
- Falls back to local text truncation when `OPENAI_API_KEY` is unset
- Stores feeds, settings, and digest state in SQLite

## Requirements

- Go 1.26+
- Telegram bot token
- Writable SQLite database path
- Optional: Docker for the published image
- Optional: OpenAI API key for Telegram post summaries

## Setup

Local checkout:

```bash
make install-deps
cp .env.example .env
```

Docker image:

```bash
docker pull ghcr.io/hu553in/telekilogram
```

## Configuration

| Name                      | Required | Default        | Description                                                        |
| ------------------------- | -------- | -------------- | ------------------------------------------------------------------ |
| `TOKEN`                   | Yes      | -              | Telegram bot token                                                 |
| `DB_PATH`                 | No       | `db.sqlite`    | SQLite database path                                               |
| `ALLOWED_USERS`           | No       | -              | Comma-separated Telegram user IDs                                  |
| `OPENAI_API_KEY`          | No       | -              | Enables OpenAI summaries (falls back to local truncation if unset) |
| `OPENAI_AI_MODEL`         | No       | `gpt-5.6-luna` | OpenAI model                                                       |
| `OPENAI_SERVICE_TIER`     | No       | `flex`         | OpenAI Responses API service tier                                  |
| `OPENAI_REASONING_EFFORT` | No       | `low`          | OpenAI reasoning effort                                            |

See `.env.example` for all available options including rate limits, scheduler timeouts, feed parsing
parameters, and OpenAI tuning flags.

`OPENAI_SYSTEM_PROMPT`, `TELEGRAM_USER_AGENT`, and `BOT_ISSUE_URL` have long defaults; keep them in
`.env.example` instead of duplicating them here.

## Usage

Local:

```bash
make build
dist/telekilogram
```

Docker:

```bash
docker run --rm --env-file .env -v telekilogram_data:/data ghcr.io/hu553in/telekilogram
```

Telegram UI:

- send a feed URL, `t.me` link, `@channel`, or forwarded public channel message to add a source
- `/list` or `Feed list` - show subscriptions
- unfollow feeds from the list
- receive an automatic 24-hour digest every day (default: 00:00 UTC)
- `/digest` or `24h digest` - send a 24-hour digest now
- Telegram channel posts get concise summaries when OpenAI is configured
- `/settings` or `Settings` - configure user-specific settings

## Runtime behavior

- `DB_PATH` controls the SQLite database path; in Docker, the image runs from `/data`
- `ALLOWED_USERS` is optional; when empty, the bot is public
- OpenAI summaries are disabled when `OPENAI_API_KEY` is unset
- Telegram summaries use a 24-hour cache and invalidate when a Telegram post is edited
- RSS, Atom, and JSON feed digests include post titles and links
- Telegram digests include summaries or trimmed text with links to the original posts

## Development

```bash
make install-deps
make check
```

Focused checks:

```bash
make fmt
make lint
make check-deps
make test
make verify-test-coverage
```

Build and generated SQL:

```bash
make build
make sqlc
```
