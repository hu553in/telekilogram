package feed

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	"mvdan.cc/xurls/v2"

	"telekilogram/common"
	"telekilogram/model"
)

type feedGroupKey struct {
	FeedTitle string
	FeedURL   string
}

func FindValidFeeds(text string) ([]model.Feed, error) {
	re, err := xurls.StrictMatchingScheme("https://")
	if err != nil {
		return nil, err
	}

	urls := re.FindAllString(text, -1)
	feeds := make([]model.Feed, 0, len(urls))
	var errs []error

	for _, u := range urls {
		feed, err := validateFeed(u)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		feeds = append(feeds, *feed)
	}

	return feeds, errors.Join(errs...)
}

func FormatPostsAsMessages(posts []model.Post) []string {
	var messages []string
	var currentMessage strings.Builder

	currentMessage.WriteString("📰 *New posts*\n\n")
	headerLength := currentMessage.Len()

	feedGroups := make(map[feedGroupKey][]model.Post)

	for _, post := range posts {
		feedTitle := post.FeedTitle
		if feedTitle == "" {
			feedTitle = post.FeedURL
		}

		key := feedGroupKey{
			FeedTitle: feedTitle,
			FeedURL:   post.FeedURL,
		}
		feedGroups[key] = append(feedGroups[key], post)
	}

	for key, feedPosts := range feedGroups {
		feedHeader := fmt.Sprintf(
			"📌 *[%s](%s)*\n\n",
			common.EscapeMarkdown(key.FeedTitle),
			key.FeedURL,
		)

		if currentMessage.Len()+len(feedHeader) > telegramMessageMaxLength {
			messages = append(messages, currentMessage.String())
			currentMessage.Reset()
			currentMessage.WriteString("📰 *New posts \\(continue\\)*\n\n")
		}

		currentMessage.WriteString(feedHeader)

		for _, post := range feedPosts {
			bulletPoint := fmt.Sprintf(
				"– [%s](%s)\n",
				common.EscapeMarkdown(post.Title),
				post.URL,
			)

			if currentMessage.Len()+len(bulletPoint) > telegramMessageMaxLength {
				messages = append(messages, currentMessage.String())
				currentMessage.Reset()
				currentMessage.WriteString("📰 *New posts \\(continue\\)*\n\n")
				currentMessage.WriteString(feedHeader)
			}

			currentMessage.WriteString(bulletPoint)
		}

		currentMessage.WriteString("\n")
	}

	if currentMessage.Len() > headerLength {
		messages = append(messages, currentMessage.String())
	}

	return messages
}

func validateFeed(feedURL string) (*model.Feed, error) {
	_, err := url.Parse(feedURL)
	if err != nil {
		return nil, err
	}

	parsed, err := libParser.ParseURL(feedURL)
	if err != nil {
		return nil, err
	}

	title := parsed.Title
	if title == "" {
		title = feedURL
	}

	return &model.Feed{
		URL:   feedURL,
		Title: title,
	}, nil
}
