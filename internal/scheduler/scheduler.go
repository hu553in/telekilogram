package scheduler

import (
	"context"
	"log/slog"
	"telekilogram/internal/bot"
	"telekilogram/internal/domain"
	"telekilogram/internal/feed"
	"time"

	"github.com/robfig/cron/v3"
)

const (
	HourlyDigestSpec      = "0 * * * *"
	Timezone              = "UTC"
	TimezoneOffsetSeconds = 0
	checkHourFeedsTimeout = 15 * time.Minute
)

type Scheduler struct {
	ctx     context.Context
	cron    *cron.Cron
	bot     *bot.Bot
	fetcher *feed.Fetcher
	log     *slog.Logger
}

func New(ctx context.Context, bot *bot.Bot, fetcher *feed.Fetcher, log *slog.Logger) *Scheduler {
	c := cron.New(cron.WithLocation(time.FixedZone(Timezone, TimezoneOffsetSeconds)))

	return &Scheduler{
		ctx:     ctx,
		cron:    c,
		bot:     bot,
		fetcher: fetcher,
		log:     log,
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
	ctx, cancel := context.WithTimeout(s.ctx, checkHourFeedsTimeout)
	defer cancel()

	select {
	case <-ctx.Done():
		s.log.InfoContext(ctx, "Scheduler context is done",
			"error", ctx.Err())
		return
	default:
	}

	hourUTC := int64(time.Now().UTC().Hour())

	userPosts, err := s.fetcher.FetchHourFeeds(ctx, hourUTC)
	if err != nil {
		s.log.ErrorContext(ctx, "Failed to fetch hour feeds",
			"error", err,
			"hourUTC", hourUTC,
			"usersWithPosts", len(userPosts))
	}

	if ctx.Err() != nil {
		s.log.InfoContext(ctx, "Scheduler context is done",
			"error", ctx.Err())
		return
	}

	for userID, posts := range userPosts {
		if err = s.bot.SendNewPosts(ctx, userID, posts); err != nil {
			s.log.ErrorContext(ctx, "Failed to send user posts",
				"error", err,
				"hourUTC", hourUTC,
				"userID", userID,
				"postCount", len(posts),
				"feedIDs", feedIDs(posts))
		}
	}
}

func feedIDs(posts []domain.Post) []int64 {
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
