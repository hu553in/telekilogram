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
		parser: NewFeedParser(),
	}
}

func (fr *FeedFetcher) FetchAllFeeds() (map[int64][]model.Post, error) {
	return fr.FetchFeeds(nil)
}

func (fr *FeedFetcher) FetchFeeds(userID *int64) (map[int64][]model.Post, error) {
	var feeds []model.Feed
	var err error

	if userID == nil {
		feeds, err = fr.db.GetAllFeeds()
	} else {
		feeds, err = fr.db.GetUserFeeds(*userID)
	}
	if err != nil {
		return nil, err
	}

	var writeWg sync.WaitGroup

	concurrency := min(maxConcurrency, len(feeds))
	semCh := make(chan int, concurrency)

	userPostCh := make(chan UserPosts, concurrency)
	errCh := make(chan error, concurrency)

	for _, f := range feeds {
		writeWg.Add(1)
		semCh <- 1

		go func() {
			defer writeWg.Done()

			posts, err := fr.parser.ParseFeed(f)
			userPostCh <- UserPosts{userID: f.UserID, posts: posts}
			errCh <- err

			<-semCh
		}()
	}

	go func() {
		writeWg.Wait()
		close(semCh)

		close(userPostCh)
		close(errCh)
	}()

	userPostsMap := make(map[int64][]model.Post)
	errs := make([]error, 0, len(feeds))

	var readWg sync.WaitGroup
	readWg.Add(2)

	go func() {
		for userPosts := range userPostCh {
			userPostsMap[userPosts.userID] = append(
				userPostsMap[userPosts.userID],
				userPosts.posts...,
			)
		}
		readWg.Done()
	}()
	go func() {
		for err := range errCh {
			errs = append(errs, err)
		}
		readWg.Done()
	}()

	readWg.Wait()
	return userPostsMap, errors.Join(errs...)
}
