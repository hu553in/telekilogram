# Telekilogram

[![CI](https://github.com/hu553in/telekilogram/actions/workflows/ci.yml/badge.svg)](https://github.com/hu553in/telekilogram/actions/workflows/ci.yml)
[![go-test-coverage](https://raw.githubusercontent.com/hu553in/telekilogram/badges/.badges/main/coverage.svg)](https://github.com/hu553in/telekilogram/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/hu553in/telekilogram)](https://goreportcard.com/report/github.com/hu553in/telekilogram)
[![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/hu553in/telekilogram)](https://github.com/hu553in/telekilogram/blob/main/go.mod)

- [License](./LICENSE)
- [How to contribute](./CONTRIBUTING.md)
- [Code of conduct](./CODE_OF_CONDUCT.md)

A Telegram bot for feed-based updates, written in Go.

Telekilogram aggregates content from RSS, Atom, and JSON feeds, as well as public Telegram channels. It delivers
daily digests and can optionally summarize Telegram posts using OpenAI. It is designed to be reliable, predictable,
and suitable for unattended operation.

---

## Functionality

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

---

## Environment variables

| Name                                             | Required | Default               | Description                                                                                            |
| ------------------------------------------------ | -------- | --------------------- | ------------------------------------------------------------------------------------------------------ |
| `TOKEN`                                          | Yes      | –                     | Telegram bot token.                                                                                    |
| `DB_PATH`                                        | No       | `db.sqlite`           | Filesystem path to the SQLite database. The file is created automatically on first run.                |
| `ALLOWED_USERS`                                  | No       | –                     | Comma-separated list of Telegram user IDs allowed to interact with the bot.                            |
| `OPENAI_API_KEY`                                 | No       | –                     | Enables OpenAI-based summaries for Telegram channel posts. If unset, local truncation is used instead. |
| `OPENAI_BASE_MAX_OUTPUT_TOKENS`                  | No       | `512`                 | Initial `max_output_tokens` for OpenAI summarization requests.                                         |
| `OPENAI_LIMIT_MAX_OUTPUT_TOKENS`                 | No       | `2048`                | Maximum `max_output_tokens` reached after retry growth.                                                |
| `OPENAI_MAX_OUTPUT_TOKENS_GROWTH_FACTOR`         | No       | `2`                   | Multiplier applied when retrying a truncated OpenAI response.                                          |
| `OPENAI_SYSTEM_PROMPT`                           | No       | built-in prompt       | System prompt used for Telegram post summarization.                                                    |
| `OPENAI_AI_MODEL`                                | No       | `gpt-5.4-nano`        | OpenAI model used for summarization.                                                                   |
| `OPENAI_SERVICE_TIER`                            | No       | `flex`                | OpenAI Responses API service tier.                                                                     |
| `OPENAI_REASONING_EFFORT`                        | No       | `low`                 | OpenAI reasoning effort for summarization requests.                                                    |
| `SCHEDULER_CHECK_HOUR_FEEDS_TIMEOUT`             | No       | `15m`                 | Timeout for a scheduled hourly digest pass.                                                            |
| `RATE_LIMITER_PRIVATE_CHAT_RATE`                 | No       | `1s`                  | Minimum delay between sends to the same private chat.                                                  |
| `RATE_LIMITER_GROUP_CHAT_RATE`                   | No       | `3s`                  | Minimum delay between sends to the same group chat.                                                    |
| `RATE_LIMITER_QUEUE_SIZE`                        | No       | `1000`                | Buffered queue size for outgoing Telegram operations.                                                  |
| `FEED_TELEGRAM_SUMMARY_CACHE_MAX_ENTRIES`        | No       | `1024`                | Maximum size of the in-memory Telegram summary cache.                                                  |
| `FEED_TELEGRAM_SUMMARIES_MAX_PARALLELISM`        | No       | `4`                   | Maximum number of parallel summarizations during Telegram feed parsing.                                |
| `FEED_PARSE_FEED_GRACE_PERIOD`                   | No       | `10m`                 | Grace period added to the 24-hour post window.                                                         |
| `FEED_FALLBACK_TELEGRAM_SUMMARY_MAX_CHARS`       | No       | `200`                 | Character limit for local fallback Telegram summaries.                                                 |
| `FEED_FETCH_FEEDS_MAX_CONCURRENCY_GROWTH_FACTOR` | No       | `10`                  | Feed fetch concurrency multiplier relative to CPU count.                                               |
| `TELEGRAM_USER_AGENT`                            | No       | Chrome-like UA string | `User-Agent` header used for public Telegram page fetches.                                             |
| `TELEGRAM_CLIENT_TIMEOUT`                        | No       | `20s`                 | HTTP timeout for Telegram page fetches.                                                                |
| `BOT_UPDATE_PROCESSING_TIMEOUT`                  | No       | `60s`                 | Timeout for processing a single Telegram update.                                                       |

## Example configuration

```
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
```
