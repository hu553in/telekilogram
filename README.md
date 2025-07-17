# Telekilogram

[![CI](https://github.com/hu553in/telekilogram/actions/workflows/ci.yml/badge.svg)](https://github.com/hu553in/telekilogram/actions/workflows/ci.yml)

Feed assistant Telegram bot written in Go.

## Functionality

- Follow feeds by sending URLs to bot
- Get feed list with `/list`
- Unfollow feeds directly from list
- Receive auto-digest (now-24h) automatically each 00:00 UTC
- Receive digest (now-24h) with `/digest`

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
- [ ] Add debug logs
- [ ] Trim whitespaces in any significant places
- [ ] Ensure that there's no blank windows between periods
- [ ] Add context to errors (`fmt.Errorf`)
- [ ] Replace 00:00 UTC with setting per user
- [ ] Add possibility to set inclusion and/or exclusion filters for posts
- [ ] Deploy using Docker instead of `systemd` service (optional)
- [ ] Migrate to https://github.com/go-telegram/bot (optional)
