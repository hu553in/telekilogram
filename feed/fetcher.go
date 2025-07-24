package feed

import (
	"errors"
	"fmt"
	"sync"

	"telekilogram/database"
	"telekilogram/models"
)

type FeedFetcher struct {
	db     *database.Database
	parser *FeedParser
}

func NewFeedFetcher(db *database.Database) *FeedFetcher {
	return &FeedFetcher{
		db:     db,
		parser: NewFeedParser(db),
	}
}

func (ff *FeedFetcher) FetchHourFeeds(
	hourUTC int64,
) (map[int64][]models.Post, error) {
	feeds, err := ff.db.GetHourFeeds(hourUTC)
	if err != nil {
		return nil, fmt.Errorf("failed to get hour feeds: %w", err)
	}

	return ff.fetchFeeds(feeds)
}

func (ff *FeedFetcher) FetchUserFeeds(
	userID int64,
) (map[int64][]models.Post, error) {
	feeds, err := ff.db.GetUserFeeds(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user feeds: %w", err)
	}

	return ff.fetchFeeds(feeds)
}

func (ff *FeedFetcher) fetchFeeds(
	feeds []models.UserFeed,
) (map[int64][]models.Post, error) {
	var writeWg sync.WaitGroup

	concurrency := min(fetchFeedsMaxConcurrency, len(feeds))
	semCh := make(chan struct{}, concurrency)

	userPostCh := make(chan models.UserPosts, concurrency)
	errCh := make(chan error, concurrency)

	for _, f := range feeds {
		writeWg.Add(1)
		semCh <- struct{}{}

		go func(
			f *models.UserFeed,
			writeWg *sync.WaitGroup,
			semCh chan struct{},
			userPostCh chan models.UserPosts,
			errCh chan error,
			ff *FeedFetcher,
		) {
			defer writeWg.Done()

			posts, err := ff.parser.ParseFeed(f)
			if err != nil {
				errCh <- fmt.Errorf("failed to parse feed: %w", err)
			}

			if len(posts) != 0 {
				userPostCh <- models.UserPosts{UserID: f.UserID, Posts: posts}
			}

			<-semCh
		}(&f, &writeWg, semCh, userPostCh, errCh, ff)
	}

	go func(
		writeWg *sync.WaitGroup,
		semCh chan struct{},
		userPostCh chan models.UserPosts,
		errCh chan error,
	) {
		writeWg.Wait()
		close(semCh)

		close(userPostCh)
		close(errCh)
	}(&writeWg, semCh, userPostCh, errCh)

	userPostsMap := make(map[int64][]models.Post)
	for userPosts := range userPostCh {
		userPostsMap[userPosts.UserID] = append(
			userPostsMap[userPosts.UserID],
			userPosts.Posts...,
		)
	}

	var errs []error
	for err := range errCh {
		errs = append(errs, err)
	}

	return userPostsMap, errors.Join(errs...)
}
