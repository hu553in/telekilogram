# Repository Guidelines

## Project Structure & Modules
- `main.go`: Entry point; loads `.env`, initializes bot, DB, scheduler.
- `bot/`: Telegram handlers, keyboards, and helpers; uses rate limiter.
- `database/`: SQLite access with embedded migrations (`database/migrations/`).
- `feed/`: RSS/Atom/JSON parsing plus public Telegram channel support:
  scrapes `t.me/<channel>` summary pages with `goquery`, filters last 24h,
  formats feed digests, and emits channel posts as URLs.
- `scheduler/`: Cron job that triggers hourly digests (UTC).
- `ratelimiter/`: Queued sending with chat-aware delays.
- `models/`, `markdown/`: Shared types and MarkdownV2 escaping.
- `scripts/`: Deploy helpers used by CI.
- Build artifacts go to `./build/`.

## Build, Test, Run
- `just all`: Lint, test, build, then run.
- `just check`: Install deps, format, lint, test, build.
- `just test`: Run `go test ./...` with coverage → `build/coverage.out`.
- `just build`: Compile binary → `build/app`.
- `just run`: Execute built binary.
Example first run: `cp .env.example .env && just all`.

## Coding Style
- Language: Go. Follow `golangci-lint` rules (see `.golangci.yaml`).
- Line length: 80 (enforced by `lll`/`golines`).
- Imports: `goimports` with local prefix `telekilogram`; alias `tgbotapi`.
- Indentation: Go defaults (tabs). Keep functions small and cohesive.
- Logging: Use `log/slog` with structured fields.
- Errors: Wrap with `fmt.Errorf` and use `errors.Join` when aggregating.
- Messaging: For grouped digests use MarkdownV2 and keyboards.
  For Telegram channel posts send plain URL via `sendURLWithPreview` (previews on).

## Testing Guidelines
- Framework: standard `testing`. Place tests next to code as `*_test.go`.
- Naming: `TestXxx` for unit tests; prefer table-driven tests.
- Run locally: `just test`. Aim to cover critical paths (rate limiter, feed
  formatting, DB queries with tmp DB). Add unit tests for Telegram URL detection
  and canonicalization when practical.

## Commits & PRs
- Commits: Conventional Commits enforced via pre-commit hook
  (e.g., `feat(bot): add settings keyboard`, `fix(feed): escape titles`).
- Before pushing: run `just check` and ensure CI passes.
- PRs: include clear description, linked issue, reproduction/verification
  steps, and screenshots of Telegram output if UI behavior changes.

## Security & Configuration
- Secrets: never commit `.env`. Required: `TOKEN`. Optional: `DB_PATH`,
  `ALLOWED_USERS` (comma-separated int64s).
- CI deploy uses SSH secrets and `scripts/deploy.sh`; verify env values are set.
- Avoid logging sensitive values; prefer IDs over tokens/URLs in logs.
