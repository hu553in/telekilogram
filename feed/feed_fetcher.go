package feed

import (
	"errors"

	"telekilogram/database"
	"telekilogram/model"
)

type FeedFetcher struct {
	db     *database.Database
	parser *FeedParser
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

	userPosts := make(map[int64][]model.Post)
	errs := make([]error, 0, len(feeds))

	for _, f := range feeds {
		newPosts, err := fr.parser.ParseFeed(f)
		errs = append(errs, err)
		if len(newPosts) > 0 {
			userPosts[f.UserID] = append(userPosts[f.UserID], newPosts...)
		}
	}

	return userPosts, errors.Join(errs...)
}
