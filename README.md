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
- Message format:
  - RSS / Atom / JSON feeds: grouped digest with post titles and links
  - Telegram channels: each new post sent as a separate URL message
    with link preview enabled
- Configure user settings with `/settings`

## Development

1. Install [Just](https://just.systems/)
1. Install Go (you can find required version in `go.mod`)
1. Install [golangci-lint](https://golangci-lint.run/)
1. Run `cp .env.example .env`
1. Fill `.env`
1. Run `just`

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
- [ ] Check if logs can be enriched with some useful contextual info
- [ ] Understand if it is needed to implement graceful shutdown, etc.
- [ ] Trim whitespaces in any significant places
- [ ] Fully protect Telegram max message length in feed list and digest
- [ ] Add tests (at least for critical functionality)
- [ ] Deploy using Docker instead of `systemd` service (optional)
- [ ] Migrate to https://github.com/go-telegram/bot (optional)
