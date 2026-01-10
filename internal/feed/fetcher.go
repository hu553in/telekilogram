package feed

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"runtime"
	"sync"
	"telekilogram/internal/models"

	"telekilogram/internal/database"
	"telekilogram/internal/summarizer"
)

const fetchFeedsMaxConcurrencyGrowthFactor = 10

type Fetcher struct {
	db     *database.Database
	parser *Parser
}

func NewFetcher(
	db *database.Database,
	s summarizer.Summarizer,
	log *slog.Logger,
) *Fetcher {
	return &Fetcher{
		db:     db,
		parser: NewParser(db, s, log),
	}
}

func (ff *Fetcher) FetchHourFeeds(
	ctx context.Context,
	hourUTC int64,
) (map[int64][]models.Post, error) {
	feeds, err := ff.db.GetHourFeeds(ctx, hourUTC)
	if err != nil {
		return nil, fmt.Errorf("failed to get hour feeds: %w", err)
	}

	return ff.fetchFeeds(ctx, feeds)
}

func (ff *Fetcher) FetchUserFeeds(
	ctx context.Context,
	userID int64,
) (map[int64][]models.Post, error) {
	feeds, err := ff.db.GetUserFeeds(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user feeds: %w", err)
	}

	return ff.fetchFeeds(ctx, feeds)
}

func (ff *Fetcher) fetchFeeds(
	ctx context.Context,
	feeds []models.UserFeed,
) (map[int64][]models.Post, error) {
	var writeWg sync.WaitGroup

	concurrency := min(runtime.NumCPU()*fetchFeedsMaxConcurrencyGrowthFactor, len(feeds))
	semCh := make(chan struct{}, concurrency)

	userPostCh := make(chan models.UserPosts, concurrency)
	errCh := make(chan error, concurrency)

	for _, f := range feeds {
		writeWg.Add(1)
		semCh <- struct{}{}

		go func(copiedFeed models.UserFeed) {
			defer writeWg.Done()

			posts, err := ff.parser.ParseFeed(ctx, &copiedFeed)
			if err != nil {
				errCh <- fmt.Errorf("failed to parse feed: %w", err)
			}

			if len(posts) != 0 {
				userPostCh <- models.UserPosts{UserID: copiedFeed.UserID, Posts: posts}
			}

			<-semCh
		}(f)
	}

	go func() {
		writeWg.Wait()
		close(semCh)

		close(userPostCh)
		close(errCh)
	}()

	userPostsMap := make(map[int64][]models.Post)
	for userPosts := range userPostCh {
		userPostsMap[userPosts.UserID] = append(userPostsMap[userPosts.UserID], userPosts.Posts...)
	}

	var errs []error
	for err := range errCh {
		errs = append(errs, err)
	}

	return userPostsMap, errors.Join(errs...)
}
