# Telekilogram

Feed assistant Telegram bot written in Go.

## Functionality

- Follow feeds by sending URLs to bot
- Get feed list with `/list`
- Unfollow feeds directly from list
- Receive auto-digest (now-24h) automatically each 00:00 UTC
- Receive digest (now-24h) with `/digest`

## Roadmap

- [ ] Fill `README.md`
- [ ] Add context to errors (`fmt.Errorf`)
- [ ] Replace 00:00 UTC with setting per user
- [ ] Ensure that there's no blank windows between periods
- [ ] Deploy using Docker instead of `systemd` service (optional)
- [ ] Migrate to https://github.com/go-telegram/bot (optional)
