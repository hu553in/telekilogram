package feed

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"telekilogram/internal/markdown"
	"telekilogram/internal/models"

	"github.com/mmcdole/gofeed"
	"mvdan.cc/xurls/v2"
)

const (
	minPartsForTelegramChannelAtSignSlug = 3
	telegramHost                         = "t.me"
	telegramMessageMaxLength             = 4096
)

var (
	//nolint:gochecknoglobals // TODO: Parser must be created not as global variable.
	libParser = gofeed.NewParser()

	telegramSlugRe       = regexp.MustCompile(`^\w{5,32}$`)
	telegramAtSignSlugRe = regexp.MustCompile(`(\s|^)@(\w{5,32})(\s|$)`)
)

type feedGroupKey struct {
	FeedID    int64
	FeedTitle string
	FeedURL   string
}

func FindValidFeeds(ctx context.Context, text string, log *slog.Logger) ([]models.Feed, error) {
	text = strings.TrimSpace(text)

	var slugs []string
	for _, m := range telegramAtSignSlugRe.FindAllStringSubmatch(text, -1) {
		if len(m) < minPartsForTelegramChannelAtSignSlug {
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

	urls := httpsURLRe.FindAllString(text, -1)

	feeds := make([]models.Feed, 0, len(urls)+len(slugs))
	seen := make(map[string]struct{}, len(urls)+len(slugs))
	var errs []error

	for _, u := range urls {
		trimmed := strings.TrimSpace(u)

		feed, validateFeedErr := validateFeed(ctx, trimmed, log)
		if validateFeedErr != nil {
			errs = append(errs, fmt.Errorf("failed to validate feed: %w", validateFeedErr))

			continue
		}
		if feed == nil {
			errs = append(errs, errors.New("failed to validate feed"))

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

		feed, validateFeedErr := validateFeed(ctx, canonicalURL, log)
		if validateFeedErr != nil {
			errs = append(errs, fmt.Errorf("failed to validate feed: %w", validateFeedErr))

			continue
		}
		if feed == nil {
			errs = append(errs, errors.New("failed to validate feed"))

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

func FormatPostsAsMessages(ctx context.Context, posts []models.Post, log *slog.Logger) []string {
	var messages []string
	var currentMessage strings.Builder

	currentMessage.WriteString("📰 *New posts*\n\n")
	headerLength := currentMessage.Len()

	feedGroups := make(map[feedGroupKey][]models.Post)

	for _, post := range posts {
		normalized, ok := normalizePost(ctx, post, log)
		if !ok {
			continue
		}

		key := feedGroupKey{
			FeedID:    normalized.FeedID,
			FeedTitle: normalized.FeedTitle,
			FeedURL:   normalized.FeedURL,
		}
		feedGroups[key] = append(feedGroups[key], normalized)
	}

	feedGroupKeySeq := maps.Keys(feedGroups)
	feedGroupKeys := slices.SortedFunc(
		feedGroupKeySeq,
		func(a, b feedGroupKey) int { return cmp.Compare(a.FeedID, b.FeedID) },
	)

	for _, key := range feedGroupKeys {
		feedPosts := feedGroups[key]

		feedHeader := fmt.Sprintf("📌 *[%s](%s)*\n\n", markdown.EscapeV2(key.FeedTitle), key.FeedURL)

		firstBulletPoint := fmt.Sprintf("– [%s](%s)\n\n", markdown.EscapeV2(feedPosts[0].Title), feedPosts[0].URL)

		if currentMessage.Len()+
			len(feedHeader)+
			len(firstBulletPoint) > telegramMessageMaxLength {
			messages = append(messages, currentMessage.String())
			currentMessage.Reset()
			currentMessage.WriteString("📰 *New posts \\(continue\\)*\n\n")
		}

		currentMessage.WriteString(feedHeader)

		for _, post := range feedPosts {
			bulletPoint := fmt.Sprintf("– [%s](%s)\n\n", markdown.EscapeV2(post.Title), post.URL)

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

	return fmt.Sprintf("https://%s/s/%s", telegramHost, slug)
}

func normalizePost(ctx context.Context, post models.Post, log *slog.Logger) (models.Post, bool) {
	normalized := post

	normalized.Title = strings.TrimSpace(post.Title)
	normalized.URL = strings.TrimSpace(post.URL)
	normalized.FeedTitle = strings.TrimSpace(post.FeedTitle)
	normalized.FeedURL = strings.TrimSpace(post.FeedURL)

	switch {
	case normalized.FeedURL == "" && normalized.URL != "":
		normalized.FeedURL = normalized.URL
	case normalized.URL == "" && normalized.FeedURL != "":
		normalized.URL = normalized.FeedURL
	case normalized.URL == "" && normalized.FeedURL == "":
		log.WarnContext(ctx, "Skipping post with empty URLs",
			"feedID", post.FeedID,
			"title", normalized.Title)

		return models.Post{}, false
	}

	if normalized.FeedTitle == "" {
		log.WarnContext(ctx, "Empty feed title",
			"feedID", post.FeedID,
			"feedURL", normalized.FeedURL,
			"postURL", normalized.URL)

		normalized.FeedTitle = normalized.FeedURL
		if normalized.FeedTitle == "" {
			normalized.FeedTitle = normalized.URL
		}
	}

	return normalized, true
}

func validateFeed(ctx context.Context, feedURL string, log *slog.Logger) (*models.Feed, error) {
	feedURL = strings.TrimSpace(feedURL)
	if feedURL == "" {
		return nil, errors.New("feed URL is empty")
	}

	if _, err := url.Parse(feedURL); err != nil {
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}

	if ok, slug := isTelegramChannelURL(feedURL); ok {
		title, err := fetchTelegramChannelTitle(ctx, slug, log)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch Telegram channel title: %w", err)
		}

		canonicalURL := TelegramChannelCanonicalURL(slug)
		if canonicalURL == "" {
			return nil, fmt.Errorf("failed to build canonical URL for slug %q", slug)
		}

		title = strings.TrimSpace(title)
		if title == "" {
			log.WarnContext(ctx, "Empty Telegram channel title",
				"canonicalURL", canonicalURL,
				"slug", slug)

			title = canonicalURL
		}

		return &models.Feed{
			URL:   canonicalURL,
			Title: title,
		}, nil
	}

	parsed, err := libParser.ParseURLWithContext(feedURL, ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to parse feed by URL %q: %w", feedURL, err)
	}

	title := strings.TrimSpace(parsed.Title)
	if title == "" {
		log.WarnContext(ctx, "Empty feed title",
			"feedURL", feedURL,
			"fallbackTitle", feedURL)

		title = feedURL
	}

	return &models.Feed{URL: feedURL, Title: title}, nil
}
