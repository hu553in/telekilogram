package scheduler

import (
	"errors"
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
		return err
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
		slog.Error("Failed to fetch all feeds", slog.Any("error", err))
		return
	}

	var errs []error
	for userID, posts := range userPosts {
		err := s.bot.SendNewPosts(userID, posts)
		if err != nil {
			errs = append(errs, err)
		}
	}

	err = errors.Join(errs...)
	if err != nil {
		slog.Error("Failed to send user posts", slog.Any("error", err))
	}
}
