package scheduler

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/robfig/cron/v3"

	"telekilogram/bot"
	"telekilogram/feed"
)

type Scheduler struct {
	cron    *cron.Cron
	bot     *bot.Bot
	fetcher *feed.FeedFetcher
}

func New(bot *bot.Bot, fetcher *feed.FeedFetcher) *Scheduler {
	c := cron.New(cron.WithLocation(time.UTC))
	return &Scheduler{
		cron:    c,
		bot:     bot,
		fetcher: fetcher,
	}
}

func (s *Scheduler) Start() error {
	if _, err := s.cron.AddFunc("0 * * * *", s.checkHourFeeds); err != nil {
		return fmt.Errorf("failed to add cron job: %w", err)
	}

	s.cron.Start()
	return nil
}

func (s *Scheduler) Stop() {
	s.cron.Stop()
}

func (s *Scheduler) checkHourFeeds() {
	hourUTC := int64(time.Now().UTC().Hour())

	userPosts, err := s.fetcher.FetchHourFeeds(hourUTC)
	if err != nil {
		slog.Error("Failed to fetch all feeds",
			slog.Int64("hourUTC", hourUTC),
			slog.Any("err", err))
		return
	}

	for userID, posts := range userPosts {
		err := s.bot.SendNewPosts(userID, posts)
		if err != nil {
			slog.Error("Failed to send user posts",
				slog.Int64("hourUTC", hourUTC),
				slog.Int64("userID", userID),
				slog.Any("posts", posts),
				slog.Any("err", err))
		}
	}
}
