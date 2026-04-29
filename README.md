# Telekilogram

[![CI](https://github.com/hu553in/telekilogram/actions/workflows/ci.yml/badge.svg)](https://github.com/hu553in/telekilogram/actions/workflows/ci.yml)
[![go-test-coverage](https://raw.githubusercontent.com/hu553in/telekilogram/badges/.badges/main/coverage.svg)](https://github.com/hu553in/telekilogram/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/hu553in/telekilogram)](https://goreportcard.com/report/github.com/hu553in/telekilogram)
[![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/hu553in/telekilogram)](https://github.com/hu553in/telekilogram/blob/main/go.mod)

- [License](./LICENSE)
- [Contributing](./CONTRIBUTING.md)
- [Code of conduct](./CODE_OF_CONDUCT.md)

A Telegram bot for feed-based updates, written in Go.

Telekilogram aggregates content from RSS, Atom, and JSON feeds, as well as public Telegram channels. It delivers
daily digests and can optionally summarize Telegram posts using OpenAI. It is designed to be reliable, predictable,
and suitable for unattended operation.

## Features

- Follow RSS, Atom, and JSON feeds, as well as public Telegram channels:
  - send a feed URL
  - send a channel `@username`
  - forward a message from a channel to the bot
- View the current feed list with `/list`.
- Unfollow feeds directly from the list.
- Receive an automatic 24-hour digest every day (default: 00:00 UTC).
- Request a 24-hour digest manually with `/digest`.
- Summarize Telegram channel posts using OpenAI:
  - automatically falls back to local text truncation if `OPENAI_API_KEY` is not set
  - caches summaries for 24 hours to avoid reprocessing the same post across users
  - invalidates cached summaries if a Telegram post is edited
- Message formatting:
  - RSS, Atom, and JSON feeds: grouped digests with post titles and links
  - Telegram channels: grouped digests with AI-generated summaries (or trimmed text) linking to the original posts
- Configure user-specific settings via `/settings`.

## Environment variables

| Name                      | Required | Default        | Description                                                        |
| ------------------------- | -------- | -------------- | ------------------------------------------------------------------ |
| `TOKEN`                   | Yes      | –              | Telegram bot token                                                 |
| `DB_PATH`                 | No       | `db.sqlite`    | SQLite database path                                               |
| `ALLOWED_USERS`           | No       | –              | Comma-separated Telegram user IDs                                  |
| `OPENAI_API_KEY`          | No       | –              | Enables OpenAI summaries (falls back to local truncation if unset) |
| `OPENAI_AI_MODEL`         | No       | `gpt-5.4-nano` | OpenAI model                                                       |
| `OPENAI_SERVICE_TIER`     | No       | `flex`         | OpenAI Responses API service tier                                  |
| `OPENAI_REASONING_EFFORT` | No       | `low`          | OpenAI reasoning effort                                            |

See `.env.example` for all available options including rate limits, scheduler timeouts,
feed parsing parameters, and OpenAI tuning flags.

See the source code or `.env.example` for full default values of `OPENAI_SYSTEM_PROMPT`,
`TELEGRAM_USER_AGENT`, and `BOT_ISSUE_URL`.

## Example configuration

```bash
TOKEN="example"
DB_PATH="db.sqlite"
ALLOWED_USERS="1,2"
OPENAI_API_KEY="example"
OPENAI_AI_MODEL="gpt-5.4-nano"
OPENAI_SERVICE_TIER="flex"
OPENAI_REASONING_EFFORT="low"
SCHEDULER_CHECK_HOUR_FEEDS_TIMEOUT="15m"
RATE_LIMITER_PRIVATE_CHAT_RATE="1s"
RATE_LIMITER_GROUP_CHAT_RATE="3s"
RATE_LIMITER_QUEUE_SIZE="1000"
FEED_TELEGRAM_SUMMARY_CACHE_MAX_ENTRIES="1024"
FEED_TELEGRAM_SUMMARIES_MAX_PARALLELISM="4"
FEED_PARSE_FEED_GRACE_PERIOD="10m"
FEED_FALLBACK_TELEGRAM_SUMMARY_MAX_CHARS="200"
FEED_FETCH_FEEDS_MAX_CONCURRENCY_GROWTH_FACTOR="10"
TELEGRAM_CLIENT_TIMEOUT="20s"
BOT_UPDATE_PROCESSING_TIMEOUT="60s"
BOT_ISSUE_URL="https://github.com/hu553in/telekilogram/issues/new"
```
