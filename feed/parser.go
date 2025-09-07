package feed

import (
	"fmt"
	"time"

	"telekilogram/database"
	"telekilogram/models"
)

type FeedParser struct {
	db *database.Database
}

func NewFeedParser(db *database.Database) *FeedParser {
	return &FeedParser{
		db: db,
	}
}

func (fp *FeedParser) ParseFeed(feed *models.UserFeed) ([]models.Post, error) {
	if ok, slug := isTelegramChannelURL(feed.URL); ok {
		items, title, err := fetchTelegramChannelItems(slug)
		if err != nil {
			return nil, fmt.Errorf(
				"failed to fetch Telegram channel items: %w",
				err,
			)
		}

		var updateTitleErr error
		if title != "" && title != feed.Title {
			if err := fp.db.UpdateFeedTitle(feed.ID, title); err != nil {
				updateTitleErr = fmt.Errorf(
					"failed to update feed title: %w",
					err,
				)
			}
		}

		var newPosts []models.Post
		now := time.Now().Round(time.Hour)
		cutoffTime := now.Add(-24*time.Hour - parseFeedGracePeriod)

		canonicalURL := telegramChannelCanonicalURL(slug)

		for _, it := range items {
			publishedTime := it.published

			if publishedTime.IsZero() {
				publishedTime = now
			}

			if publishedTime.After(cutoffTime) {
				newPosts = append(newPosts, models.Post{
					Title:     it.URL,
					URL:       it.URL,
					FeedID:    feed.ID,
					FeedTitle: title,
					FeedURL:   canonicalURL,
				})
			}
		}

		return newPosts, updateTitleErr
	}

	parsed, err := libParser.ParseURL(feed.URL)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to parse feed by URL %q: %w",
			feed.URL,
			err,
		)
	}

	var updateTitleErr error
	if parsed.Title != feed.Title {
		if err := fp.db.UpdateFeedTitle(feed.ID, parsed.Title); err != nil {
			updateTitleErr = fmt.Errorf("failed to update feed title: %w", err)
		}
	}

	var newPosts []models.Post
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
			newPosts = append(newPosts, models.Post{
				Title:     item.Title,
				URL:       item.Link,
				FeedID:    feed.ID,
				FeedTitle: parsed.Title,
				FeedURL:   feed.URL,
			})
		}
	}

	return newPosts, updateTitleErr
}
