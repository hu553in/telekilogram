package feed

import (
	"fmt"
	"time"

	"telekilogram/database"
	"telekilogram/model"
)

type FeedParser struct {
	db *database.Database
}

func NewFeedParser(db *database.Database) *FeedParser {
	return &FeedParser{
		db: db,
	}
}

func (fp *FeedParser) ParseFeed(feed *model.UserFeed) ([]model.Post, error) {
	parsed, err := libParser.ParseURL(feed.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse feed by URL: %w", err)
	}

	if parsed.Title != feed.Title {
		if err := fp.db.UpdateFeedTitle(feed.ID, parsed.Title); err != nil {
			return nil, fmt.Errorf("failed to update feed title: %w", err)
		}
	}

	var newPosts []model.Post
	now := time.Now().Round(time.Hour)
	cutoffTime := now.Add(-24*time.Hour - parseFeedGracePeriod)

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
