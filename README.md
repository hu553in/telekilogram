# Telekilogram

[![CI](https://github.com/hu553in/telekilogram/actions/workflows/ci.yml/badge.svg)](https://github.com/hu553in/telekilogram/actions/workflows/ci.yml)
[![go-test-coverage](https://raw.githubusercontent.com/hu553in/telekilogram/badges/.badges/main/coverage.svg)](https://github.com/hu553in/telekilogram/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/hu553in/telekilogram)](https://goreportcard.com/report/github.com/hu553in/telekilogram)
[![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/hu553in/telekilogram)](https://github.com/hu553in/telekilogram/blob/main/go.mod)

- [License](./LICENSE)
- [How to contribute](./CONTRIBUTING.md)
- [Code of conduct](./CODE_OF_CONDUCT.md)

A feed assistant Telegram bot written in Go.

Telekilogram aggregates content from RSS/Atom/JSON feeds and public Telegram channels, delivers daily digests,
and optionally summarizes Telegram posts using OpenAI. It is designed to be reliable, predictable, and suitable
for unattended operation.

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
  - summaries are cached for 24 hours to avoid reprocessing the same post across users
  - cached summaries are invalidated if a Telegram post is edited
- Message formatting:
  - RSS / Atom / JSON feeds: grouped digest with post titles and links
  - Telegram channels: grouped digest with AI-generated summaries (or trimmed text) linking to the original posts
- Configure user-specific settings via `/settings`.

---

## Environment variables

| Name             | Required | Default     | Description                                                                                            |
| ---------------- | -------- | ----------- | ------------------------------------------------------------------------------------------------------ |
| `TOKEN`          | Yes      | –           | Telegram bot token.                                                                                    |
| `DB_PATH`        | No       | `db.sqlite` | Filesystem path to the SQLite database. The file is created automatically on first run.                |
| `ALLOWED_USERS`  | No       | –           | Comma-separated list of Telegram user IDs allowed to interact with the bot.                            |
| `OPENAI_API_KEY` | No       | –           | Enables OpenAI-based summaries for Telegram channel posts. If unset, local truncation is used instead. |

## Example configuration

```
TOKEN="example"
DB_PATH="db.sqlite"
ALLOWED_USERS="1,2"
OPENAI_API_KEY="example"
```

---

## Future roadmap

- [ ] Fully enforce Telegram maximum message length limits
- [ ] Evaluate the need for more detailed user-facing error messages
- [ ] Add tests for critical functionality
- [ ] Create a mini app (optional)
- [ ] Introduce paid subscriptions with a free tier (optional)
- [ ] Migrate to https://github.com/go-telegram/bot (optional)
