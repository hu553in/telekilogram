package feed

import (
	"time"

	"github.com/mmcdole/gofeed"

	"telekilogram/database"
	"telekilogram/model"
)

var parser = gofeed.NewParser()

type FeedParser struct {
	db *database.Database
}

func NewFeedParser(db *database.Database) *FeedParser {
	return &FeedParser{
		db: db,
	}
}

func (fp *FeedParser) ParseFeed(feed model.UserFeed) ([]model.Post, error) {
	parsed, err := parser.ParseURL(feed.URL)
	if err != nil {
		return nil, err
	}

	if parsed.Title != feed.Title {
		err = fp.db.UpdateFeedTitle(feed.ID, parsed.Title)
		if err != nil {
			return nil, err
		}
	}

	var newPosts []model.Post
	now := time.Now().Round(time.Hour)
	cutoffTime := now.AddDate(0, 0, -1)

	for _, item := range parsed.Items {
		publishedTime := now

		if item.PublishedParsed != nil {
			publishedTime = *item.PublishedParsed
		} else if item.UpdatedParsed != nil {
			publishedTime = *item.UpdatedParsed
		}

		if publishedTime.After(cutoffTime) {
			post := model.Post{
				Title:     item.Title,
				URL:       item.Link,
				FeedTitle: parsed.Title,
				FeedURL:   feed.URL,
			}
			newPosts = append(newPosts, post)
		}
	}

	return newPosts, nil
}
