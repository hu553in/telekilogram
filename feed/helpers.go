package feed

import (
	"cmp"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"net/url"
	"slices"
	"strings"

	"mvdan.cc/xurls/v2"

	"telekilogram/markdown"
	"telekilogram/models"
)

type feedGroupKey struct {
	FeedID    int64
	FeedTitle string
	FeedURL   string
}

func FindValidFeeds(text string) ([]models.Feed, error) {
	text = strings.TrimSpace(text)

	var slugs []string
	for _, m := range telegramAtSignSlugRe.FindAllStringSubmatch(text, -1) {
		if len(m) < 3 {
			continue
		}

		slug := strings.TrimSpace(m[2])
		if !telegramSlugRe.MatchString(slug) {
			continue
		}

		slugs = append(slugs, slug)
	}

	httpsURLRe, err := xurls.StrictMatchingScheme("https://")
	if err != nil {
		return nil, fmt.Errorf("failed to create regexp: %w", err)
	}

	URLs := httpsURLRe.FindAllString(text, -1)

	feeds := make([]models.Feed, 0, len(URLs)+len(slugs))
	seen := make(map[string]struct{}, len(URLs)+len(slugs))
	var errs []error

	for _, u := range URLs {
		u := strings.TrimSpace(u)

		feed, err := validateFeed(u)
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to validate feed: %w", err))

			continue
		}
		if feed == nil {
			errs = append(errs, fmt.Errorf("failed to validate feed"))

			continue
		}

		if _, ok := seen[feed.URL]; ok {
			continue
		}

		feeds = append(feeds, *feed)
		seen[feed.URL] = struct{}{}
	}

	for _, slug := range slugs {
		canonicalURL := TelegramChannelCanonicalURL(slug)

		feed, err := validateFeed(canonicalURL)
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to validate feed: %w", err))

			continue
		}
		if feed == nil {
			errs = append(errs, fmt.Errorf("failed to validate feed"))

			continue
		}

		if _, ok := seen[feed.URL]; ok {
			continue
		}

		feeds = append(feeds, *feed)
		seen[feed.URL] = struct{}{}
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
		feedTitle := strings.TrimSpace(post.FeedTitle)
		feedURL := strings.TrimSpace(post.FeedURL)
		postTitle := strings.TrimSpace(post.Title)
		postURL := strings.TrimSpace(post.URL)

		if feedURL == "" && postURL != "" {
			feedURL = postURL
		} else if postURL == "" && feedURL != "" {
			postURL = feedURL
		} else if postURL == "" && feedURL == "" {
			slog.Warn("Skipping post with empty URLs",
				slog.Int64("feedID", post.FeedID),
				slog.String("title", postTitle))

			continue
		}

		if feedTitle == "" {
			slog.Warn("Empty feed title",
				slog.Int64("feedID", post.FeedID),
				slog.String("feedURL", feedURL),
				slog.String("postURL", postURL))

			feedTitle = feedURL
			if feedTitle == "" {
				feedTitle = postURL
			}
		}

		normalized := post
		normalized.Title = postTitle
		normalized.URL = postURL
		normalized.FeedTitle = feedTitle
		normalized.FeedURL = feedURL

		key := feedGroupKey{
			FeedID:    post.FeedID,
			FeedTitle: feedTitle,
			FeedURL:   feedURL,
		}
		feedGroups[key] = append(feedGroups[key], normalized)
	}

	feedGroupKeySeq := maps.Keys(feedGroups)
	feedGroupKeys := slices.SortedFunc(
		feedGroupKeySeq,
		func(a, b feedGroupKey) int {
			return cmp.Compare(a.FeedID, b.FeedID)
		},
	)

	for _, key := range feedGroupKeys {
		feedPosts := feedGroups[key]

		feedHeader := fmt.Sprintf(
			"📌 *[%s](%s)*\n\n",
			markdown.EscapeV2(key.FeedTitle),
			key.FeedURL,
		)

		firstBulletPoint := fmt.Sprintf(
			"– [%s](%s)\n\n",
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
				"– [%s](%s)\n\n",
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
	}

	if currentMessage.Len() > headerLength {
		messages = append(messages, currentMessage.String())
	}

	return messages
}

func TelegramChannelCanonicalURL(slug string) string {
	slug = strings.TrimSpace(slug)
	if slug == "" {
		return ""
	}

	return fmt.Sprintf("https://%s/s/%s", TelegramHost, slug)
}

func validateFeed(feedURL string) (*models.Feed, error) {
	feedURL = strings.TrimSpace(feedURL)
	if feedURL == "" {
		return nil, fmt.Errorf("feed URL is empty")
	}

	if _, err := url.Parse(feedURL); err != nil {
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}

	if ok, slug := isTelegramChannelURL(feedURL); ok {
		title, err := fetchTelegramChannelTitle(slug)
		if err != nil {
			return nil, fmt.Errorf(
				"failed to fetch Telegram channel title: %w",
				err,
			)
		}

		canonicalURL := TelegramChannelCanonicalURL(slug)
		if canonicalURL == "" {
			return nil, fmt.Errorf("failed to build canonical URL for slug %q", slug)
		}

		title = strings.TrimSpace(title)
		if title == "" {
			slog.Warn("Empty Telegram channel title",
				slog.Any("canonicalURL", canonicalURL),
				slog.String("slug", slug))

			title = canonicalURL
		}

		return &models.Feed{
			URL:   canonicalURL,
			Title: title,
		}, nil
	}

	parsed, err := libParser.ParseURL(feedURL)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to parse feed by URL %q: %w",
			feedURL,
			err,
		)
	}

	title := strings.TrimSpace(parsed.Title)
	if title == "" {
		slog.Warn("Empty feed title",
			slog.String("feedURL", feedURL),
			slog.String("fallbackTitle", feedURL))

		title = feedURL
	}

	return &models.Feed{URL: feedURL, Title: title}, nil
}
