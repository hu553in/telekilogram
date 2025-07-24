package feed

import (
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"strings"

	"mvdan.cc/xurls/v2"

	"telekilogram/markdown"
	"telekilogram/models"
)

type feedGroupKey struct {
	FeedTitle string
	FeedURL   string
}

func FindValidFeeds(text string) ([]models.Feed, error) {
	re, err := xurls.StrictMatchingScheme("https://")
	if err != nil {
		return nil, fmt.Errorf("failed to create regexp: %w", err)
	}

	urls := re.FindAllString(text, -1)
	feeds := make([]models.Feed, 0, len(urls))
	var errs []error

	for _, u := range urls {
		feed, err := validateFeed(u)
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to validate feed: %w", err))
		}

		feeds = append(feeds, *feed)
	}

	return feeds, errors.Join(errs...)
}

func FormatPostsAsMessages(posts []models.Post) []string {
	var messages []string
	var currentMessage strings.Builder

	currentMessage.WriteString("📰 *New posts*\n\n")
	headerLength := currentMessage.Len()

	feedGroups := make(map[feedGroupKey][]models.Post)

	for _, post := range posts {
		feedTitle := post.FeedTitle
		if feedTitle == "" {
			slog.Warn("Empty feed title",
				slog.Any("post", post))

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
			markdown.EscapeV2(key.FeedTitle),
			key.FeedURL,
		)

		firstBulletPoint := fmt.Sprintf(
			"– [%s](%s)\n",
			markdown.EscapeV2(feedPosts[0].Title),
			feedPosts[0].URL,
		)

		if currentMessage.Len()+
			len(feedHeader)+
			len(firstBulletPoint) > telegramMessageMaxLength {
			messages = append(messages, currentMessage.String())
			currentMessage.Reset()
			currentMessage.WriteString("📰 *New posts \\(continue\\)*\n\n")
		}

		currentMessage.WriteString(feedHeader)

		for _, post := range feedPosts {
			bulletPoint := fmt.Sprintf(
				"– [%s](%s)\n",
				markdown.EscapeV2(post.Title),
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

func validateFeed(feedURL string) (*models.Feed, error) {
	if _, err := url.Parse(feedURL); err != nil {
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}

	parsed, err := libParser.ParseURL(feedURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse feed by URL %q: %w", feedURL, err)
	}

	title := parsed.Title
	if title == "" {
		slog.Warn("Empty feed title",
			slog.Any("feedURL", feedURL))

		title = feedURL
	}

	return &models.Feed{
		URL:   feedURL,
		Title: title,
	}, nil
}
