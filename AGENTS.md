# Repository Guidelines

## Development Commands
- `just all` - Run checks and start the application.
- `just check` - Install deps, lint, format, test, and build.
- `just install-deps` - Download Go module dependencies.
- `just lint` - Run `golangci-lint`.
- `just fmt` - Format code via `golangci-lint`.
- `just test` - Run tests with coverage output to `./build/coverage.out`.
- `just build` - Build binary to `./build/app`.
- `just run` - Execute the built application.
Example first run: `cp .env.example .env && just all`.

## Project Structure & Modules
- `main.go`: Loads `.env`, initializes the bot, database, scheduler, and
  graceful shutdown while validating required env vars.
- `bot/`: Telegram handlers, keyboards, and helpers with deep links
  (`/start`, `/menu`, `/list`, `/digest`, `/filter`, `/settings`), callback
  error helpers, and forwarded message support for public channels.
- `database/`: SQLite access with embedded migrations in `database/migrations/`.
- `feed/`: RSS / Atom / JSON parsing plus Telegram channel scraping via
  `goquery`, `@username` detection, canonical channel URL resolution, 24h feed
  filtering, summary caching, Markdown digest formatting, and post emission.
- `scheduler/`: Cron job that triggers hourly digests (UTC).
- `ratelimiter/`: Queued sending with chat-aware delays (1s private, 3s group)
  and graceful shutdown.
- `models/`, `markdown/`: Shared types and MarkdownV2 escaping helpers.
- `summarizer/`: Pluggable interface with OpenAI-backed implementation for
  Telegram channel items (optional at runtime).
- `scripts/`: Deploy helpers used by CI.
- Build artifacts go to `./build/`.

## Key Patterns
- Structured logging with `log/slog` and contextual fields.
- Database migrations embedded via `//go:embed`.
- Errors wrapped with `fmt.Errorf`; concurrent errors combined with
  `errors.Join`.
- Authorization via `ALLOWED_USERS` env var (comma-separated `int64`s).
- Inline keyboards for navigation and deep-link unfollow flows.
- Telegram channel items summarized before inclusion in digests.

## Database Schema
- `feeds`: Stores `user_id`, `url`, `title` associations with generated IDs.
- `user_settings`: Stores `user_id`, `auto_digest_hour_utc` preferences.
- Migrations in `database/migrations/` run automatically on startup.

## Feed Processing
- Uses `github.com/mmcdole/gofeed` for feed parsing.
- Filters posts to the last 24 hours for digest generation.
- Summaries prefer OpenAI when configured; otherwise fall back to truncation.
- Markdown output escapes for Telegram MarkdownV2 requirements.
- Edited Telegram posts invalidate the 24h cache before re-summarizing.

## Coding Style
- Language: Go. Follow `golangci-lint` rules (see `.golangci.yaml`).
- Line length: 80 (enforced by `lll`/`golines`).
- Imports: `goimports` with local prefix `telekilogram`; alias `tgbotapi`.
- Indentation: Go defaults (tabs). Tabs are mandatory in `.go` files — see
  `.editorconfig`. Do not replace tabs with spaces.
- Logging: Use `log/slog` with structured fields.
- Errors: Wrap with `fmt.Errorf` and use `errors.Join` when aggregating.
- Messaging: For grouped digests use MarkdownV2 and keyboards. Telegram
  channel posts are summarized (OpenAI when configured, otherwise local
  truncation) and included in digests as Markdown links.

## Testing Guidelines
- Framework: standard `testing`. Place tests next to code as `*_test.go`.
- Naming: `TestXxx` for unit tests; prefer table-driven tests.
- Run locally: `just test`. Aim to cover critical paths (rate limiter, feed
  formatting, DB queries with tmp DB). Add unit tests for Telegram URL
  detection, `@username` detection, and canonicalization when practical.

## Commits & PRs
- Commits: Conventional Commits enforced via pre-commit hook (e.g.,
  `feat(bot): add settings keyboard`, `fix(feed): escape titles`).
- Before pushing: run `just check` and ensure CI passes.
- PRs: include clear description, linked issue, reproduction/verification
  steps, and screenshots of Telegram output if UI behavior changes.

## Security & Configuration
- Secrets: never commit `.env`. Required: `TOKEN`. Optional: `DB_PATH`,
  `ALLOWED_USERS` (comma-separated `int64s`), `OPENAI_API_KEY` (enables
  Telegram summary generation).
- CI deploy uses SSH secrets and `scripts/deploy.sh`; verify env values are set.
- Avoid logging sensitive values; prefer IDs over tokens/URLs in logs.

### Environment variables

| Name             | Required | Default | Description                                                                                                                                                                                                 |
| ---------------- | -------- | ------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `TOKEN`          | Yes      | –       | Telegram bot token obtained from BotFather.                                                                                                                                                                 |
| `DB_PATH`        | No       | `./db`  | Filesystem location of the SQLite database. Creates the file on first run if it does not exist.                                                                                                             |
| `ALLOWED_USERS`  | No       | –       | Comma-separated list of Telegram user IDs allowed to interact with the bot. Leading/trailing whitespace is ignored. Each entry must be a valid 64-bit integer; startup fails if any value cannot be parsed. |
| `OPENAI_API_KEY` | No       | –       | Enables OpenAI-backed summaries for Telegram channel posts. Leave unset to fall back to local truncation.                                                                                                   |

Environment values are trimmed for leading/trailing whitespace before use to
avoid configuration mistakes.
