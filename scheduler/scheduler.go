package scheduler

import (
	"log/slog"
	"time"

	"github.com/robfig/cron/v3"

	"telekilogram/bot"
	"telekilogram/feed"
	"telekilogram/models"
)

const HourlyDigestSpec = "0 * * * *"

var CronLocation = time.UTC

type Scheduler struct {
	cron    *cron.Cron
	bot     *bot.Bot
	fetcher *feed.FeedFetcher
}

func New(bot *bot.Bot, fetcher *feed.FeedFetcher) *Scheduler {
	c := cron.New(cron.WithLocation(CronLocation))

	return &Scheduler{
		cron:    c,
		bot:     bot,
		fetcher: fetcher,
	}
}

func (s *Scheduler) Start() error {
	if _, err := s.cron.AddFunc(HourlyDigestSpec, s.checkHourFeeds); err != nil {
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
		slog.Error("Failed to fetch hour feeds",
			slog.Any("err", err),
			slog.Int64("hourUTC", hourUTC),
			slog.Int("usersWithPosts", len(userPosts)))
	}

	for userID, posts := range userPosts {
		if err := s.bot.SendNewPosts(userID, posts); err != nil {
			slog.Error("Failed to send user posts",
				slog.Any("err", err),
				slog.Int64("hourUTC", hourUTC),
				slog.Int64("userID", userID),
				slog.Int("postCount", len(posts)),
				slog.Any("feedIDs", feedIDs(posts)))
		}
	}
}

func feedIDs(posts []models.Post) []int64 {
	seen := make(map[int64]struct{})
	var ids []int64

	for _, post := range posts {
		if _, ok := seen[post.FeedID]; ok {
			continue
		}

		seen[post.FeedID] = struct{}{}
		ids = append(ids, post.FeedID)
	}

	return ids
}
