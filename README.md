# Telekilogram

[![CI](https://github.com/hu553in/telekilogram/actions/workflows/ci.yml/badge.svg)](https://github.com/hu553in/telekilogram/actions/workflows/ci.yml)

Feed assistant Telegram bot written in Go.

## Functionality

- Follow RSS / Atom / JSON feeds and public Telegram channels by sending URLs,
  channel `@username` slugs, or forwarding messages from channels to bot
- Get feed list with `/list`
- Unfollow feeds directly from list
- Receive 24h auto-digest daily automatically (default - 00:00 UTC)
- Receive 24h digest with `/digest`
- Summarize Telegram channel posts with OpenAI (falls back to local truncation
  when `OPENAI_API_KEY` is not provided) and cache each summary for 24h to avoid
  reprocessing the same post across users
- Invalidate cached AI summaries for edited Telegram channel posts
- Message format:
  - RSS / Atom / JSON feeds: grouped digest with post titles and links
  - Telegram channels: grouped digest with AI summary (or trimmed text)
    linking to the original post
- Configure user settings with `/settings`

## Development

1. Install [Just](https://just.systems/)
1. Install Go (you can find required version in `go.mod`)
1. Install [golangci-lint](https://golangci-lint.run/)
1. Run `cp .env.example .env`
1. Fill `.env` (`TOKEN` required; `OPENAI_API_KEY` optional for summaries)
1. Run `just`

## Environment variables

| Name             | Required | Default | Description                                                                                                                                                                                                 |
| ---------------- | -------- | ------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `TOKEN`          | Yes      | –       | Telegram bot token obtained from BotFather.                                                                                                                                                                 |
| `DB_PATH`        | No       | `./db`  | Filesystem location of the SQLite database. Creates the file on first run if it does not exist.                                                                                                             |
| `ALLOWED_USERS`  | No       | –       | Comma-separated list of Telegram user IDs allowed to interact with the bot. Leading/trailing whitespace is ignored. Each entry must be a valid 64-bit integer; startup fails if any value cannot be parsed. |
| `OPENAI_API_KEY` | No       | –       | Enables OpenAI-backed summaries for Telegram channel posts. Leave unset to fall back to local truncation.                                                                                                   |

**Note:** Environment values are trimmed for leading/trailing whitespace before use,
so pasting secrets with extra spaces will not break startup.

## Roadmap

- [x] Fill `README.md`
- [x] Optimize work with slices
- [x] Optimize performance of business functions (they are really slow)
- [x] Ensure that there's no blank windows between periods
- [x] Add possibility to set inclusion and/or exclusion filters for posts
  - decided to use awesome [siftrss](https://siftrss.com/) instead ✨
- [x] Replace 00:00 UTC with setting per user
- [x] Add context to errors (`fmt.Errorf`)
- [x] Respond with at least error to any request from user
- [x] Respond with partial data if something is loaded correctly
- [x] Check if adding some debug logs can be useful
  - decided to add when needed
- [x] Rethink unfollow keyboard (now it's row for each feed)
- [x] Properly structure new code related to public Telegram channels
  - decided that it's not needed
- [x] Support adding public Telegram channels from forwarded messages
- [x] Add AI summaries of Telegram channel posts
- [x] Cache AI summaries of Telegram channel posts
- [x] Parallelise AI summarization of Telegram channel posts
- [x] Check if logs can be enriched with some useful contextual info
- [x] Trim whitespaces in any significant places
- [ ] Add graceful shutdown
- [ ] Fully protect Telegram max message length everywhere
- [ ] Check if it's needed to introduce more detailed errors for users
- [ ] Add tests (at least for critical functionality)
- [ ] Create mini app (optional)
- [ ] Add paid subscription (with free tier) (optional)
- [ ] Deploy using Docker instead of `systemd` service (optional)
- [ ] Migrate to https://github.com/go-telegram/bot (optional)
