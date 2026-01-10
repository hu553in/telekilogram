# Telekilogram

[![CI](https://github.com/hu553in/telekilogram/actions/workflows/ci.yml/badge.svg)](https://github.com/hu553in/telekilogram/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/hu553in/telekilogram)](https://goreportcard.com/report/github.com/hu553in/telekilogram)
[![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/hu553in/telekilogram)](https://github.com/hu553in/telekilogram/blob/main/go.mod)

- [License](./LICENSE)
- [How to contribute](./CONTRIBUTING.md)
- [Code of conduct](./CODE_OF_CONDUCT.md)

Feed assistant Telegram bot written in Go.

## Functionality

- Follow RSS / Atom / JSON feeds and public Telegram channels by sending URLs, channel `@username` slugs,
  or forwarding messages from channels to bot
- Get feed list with `/list`
- Unfollow feeds directly from list
- Receive 24h auto-digest daily automatically (default - 00:00 UTC)
- Receive 24h digest with `/digest`
- Summarize Telegram channel posts with OpenAI (falls back to local truncation when `OPENAI_API_KEY` is not provided)
  and cache each summary for 24h to avoid reprocessing the same post across users
- Invalidate cached AI summaries for edited Telegram channel posts
- Message format:
  - RSS / Atom / JSON feeds: grouped digest with post titles and links
  - Telegram channels: grouped digest with AI summary (or trimmed text) linking to the original post
- Configure user settings with `/settings`

## Environment variables

| Name             | Required | Default     | Description                                                                                                 |
| ---------------- | -------- | ----------- | ----------------------------------------------------------------------------------------------------------- |
| `TOKEN`          | Yes      | –           | -                                                                                                           |
| `DB_PATH`        | No       | `db.sqlite` | Filesystem location of the SQLite database. Creates the file on first run if it does not exist.             |
| `ALLOWED_USERS`  | No       | –           | The comma-separated list of Telegram user IDs allowed to interact with the bot.                             |
| `OPENAI_API_KEY` | No       | –           | Enables OpenAI-backed summaries for Telegram channel posts (local truncation will be used if not provided). |

Example:

```
TOKEN="example"
DB_PATH="db.sqlite"
ALLOWED_USERS="1,2"
OPENAI_API_KEY="example"
```

## Future roadmap

- [ ] Fully protect Telegram max message length everywhere
- [ ] Check if it's needed to introduce more detailed errors for users
- [ ] Add tests (at least for critical functionality)
- [ ] Create mini app (optional)
- [ ] Add paid subscription with free tier (optional)
- [ ] Migrate to https://github.com/go-telegram/bot (optional)
