package feed

import (
	"errors"
	"runtime"
	"sync"

	"telekilogram/database"
	"telekilogram/model"
)

var maxConcurrency = runtime.NumCPU() * 10

type FeedFetcher struct {
	db     *database.Database
	parser *FeedParser
}

type UserPosts struct {
	userID int64
	posts  []model.Post
}

func NewFeedFetcher(db *database.Database) *FeedFetcher {
	return &FeedFetcher{
		db:     db,
		parser: NewFeedParser(db),
	}
}

func (ff *FeedFetcher) FetchAllFeeds() (map[int64][]model.Post, error) {
	return ff.FetchFeeds(nil)
}

func (ff *FeedFetcher) FetchFeeds(userID *int64) (map[int64][]model.Post, error) {
	var feeds []model.UserFeed
	var err error

	if userID == nil {
		feeds, err = ff.db.GetAllFeeds()
	} else {
		feeds, err = ff.db.GetUserFeeds(*userID)
	}
	if err != nil {
		return nil, err
	}

	var writeWg sync.WaitGroup

	concurrency := min(maxConcurrency, len(feeds))
	semCh := make(chan struct{}, concurrency)

	userPostCh := make(chan UserPosts, concurrency)
	errCh := make(chan error, concurrency)

	for _, f := range feeds {
		writeWg.Add(1)
		semCh <- struct{}{}

		go func(
			f *model.UserFeed,
			writeWg *sync.WaitGroup,
			semCh chan struct{},
			userPostCh chan UserPosts,
			errCh chan error,
			ff *FeedFetcher,
		) {
			defer writeWg.Done()

			posts, err := ff.parser.ParseFeed(f)
			userPostCh <- UserPosts{userID: f.UserID, posts: posts}
			errCh <- err

			<-semCh
		}(&f, &writeWg, semCh, userPostCh, errCh, ff)
	}

	go func(
		writeWg *sync.WaitGroup,
		semCh chan struct{},
		userPostCh chan UserPosts,
		errCh chan error,
	) {
		writeWg.Wait()
		close(semCh)

		close(userPostCh)
		close(errCh)
	}(&writeWg, semCh, userPostCh, errCh)

	userPostsMap := make(map[int64][]model.Post)
	var errs []error

	for userPosts := range userPostCh {
		userPostsMap[userPosts.userID] = append(
			userPostsMap[userPosts.userID],
			userPosts.posts...,
		)
	}
	for err := range errCh {
		errs = append(errs, err)
	}

	return userPostsMap, errors.Join(errs...)
}
