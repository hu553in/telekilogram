package feed

import (
	"time"

	"github.com/mmcdole/gofeed"

	model "telekilogram/model"
)

type FeedParser struct {
	parser *gofeed.Parser
}

func NewFeedParser() *FeedParser {
	return &FeedParser{
		parser: gofeed.NewParser(),
	}
}

func (fp *FeedParser) ParseFeed(feed model.Feed) ([]model.Post, error) {
	parsedFeed, err := fp.parser.ParseURL(feed.URL)
	if err != nil {
		return nil, err
	}

	var newPosts []model.Post
	now := time.Now().Round(time.Hour)
	cutoffTime := now.AddDate(0, 0, -1)

	for _, item := range parsedFeed.Items {
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
				FeedTitle: parsedFeed.Title,
			}
			newPosts = append(newPosts, post)
		}
	}

	return newPosts, nil
}
