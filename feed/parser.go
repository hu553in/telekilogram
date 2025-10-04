package feed

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"telekilogram/database"
	"telekilogram/models"
	"telekilogram/summarizer"
)

type FeedParser struct {
	db           *database.Database
	summarizer   summarizer.Summarizer
	summaryCache *telegramSummaryCache
}

func NewFeedParser(
	db *database.Database,
	s summarizer.Summarizer,
) *FeedParser {
	return &FeedParser{
		db:           db,
		summarizer:   s,
		summaryCache: newTelegramSummaryCache(telegramSummaryCacheMaxEntries),
	}
}

func (fp *FeedParser) ParseFeed(
	ctx context.Context,
	feed *models.UserFeed,
) ([]models.Post, error) {
	if ok, slug := isTelegramChannelURL(feed.URL); ok {
		items, channelTitle, err := fetchTelegramChannelPosts(slug)
		if err != nil {
			return nil, fmt.Errorf(
				"failed to fetch Telegram channel items: %w",
				err,
			)
		}

		var updateTitleErr error
		if channelTitle != "" && channelTitle != feed.Title {
			if err := fp.db.UpdateFeedTitle(feed.ID, channelTitle); err != nil {
				updateTitleErr = fmt.Errorf(
					"failed to update feed title: %w",
					err,
				)
			}
		}

		var newPosts []models.Post
		now := time.Now().Round(time.Hour)
		cutoffTime := now.Add(-24*time.Hour - parseFeedGracePeriod)

		canonicalURL := TelegramChannelCanonicalURL(slug)
		feedTitle := channelTitle
		if feedTitle == "" {
			feedTitle = feed.Title
		}
		if feedTitle == "" {
			feedTitle = canonicalURL
		}

		for _, it := range items {
			publishedTime := it.published

			if publishedTime.IsZero() {
				publishedTime = now
			}

			if publishedTime.After(cutoffTime) {
				newPosts = append(newPosts, models.Post{
					Title:     fp.summarizeTelegramPost(ctx, it),
					URL:       it.URL,
					FeedID:    feed.ID,
					FeedTitle: feedTitle,
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

func (fp *FeedParser) summarizeTelegramPost(
	ctx context.Context,
	item channelItem,
) string {
	text := strings.TrimSpace(item.Text)
	if text == "" {
		return item.URL
	}

	now := time.Now().UTC()
	cacheKey := telegramSummaryCacheKey(item.URL, text)

	if cacheKey != "" && fp.summaryCache != nil {
		if summary, ok := fp.summaryCache.get(cacheKey, now); ok {
			return summary
		}
	}

	if fp.summarizer == nil {
		return fallbackTelegramSummary(text, item.URL)
	}

	summary, err := fp.summarizer.Summarize(ctx, summarizer.Input{
		Text:      text,
		SourceURL: item.URL,
	})
	if err != nil {
		slog.Error("Failed to summarize Telegram channel post",
			slog.Any("err", err),
			slog.String("url", item.URL))

		return fallbackTelegramSummary(text, item.URL)
	}

	summary = strings.TrimSpace(summary)
	if summary == "" {
		return fallbackTelegramSummary(text, item.URL)
	}

	published := item.published
	if published.IsZero() {
		published = now
	}

	expiresAt := published.Add(24*time.Hour + parseFeedGracePeriod)
	if expiresAt.After(now) && cacheKey != "" && fp.summaryCache != nil {
		fp.summaryCache.set(cacheKey, summary, expiresAt, now)
	}

	return summary
}

func telegramSummaryCacheKey(rawURL string, text string) string {
	canonicalURL := TelegramMessageCanonicalURL(rawURL)
	if canonicalURL == "" {
		return ""
	}

	normalizedText := strings.TrimSpace(text)
	if normalizedText == "" {
		return ""
	}

	hash := sha256.Sum256([]byte(normalizedText))

	return canonicalURL + "|" + hex.EncodeToString(hash[:])
}

func fallbackTelegramSummary(text string, itemURL string) string {
	normalized := strings.Join(strings.Fields(text), " ")
	if normalized == "" {
		return itemURL
	}

	runes := []rune(normalized)
	if len(runes) <= fallbackTelegramSummaryMaxChars {
		return normalized
	}

	trimmed := strings.TrimSpace(string(runes[:fallbackTelegramSummaryMaxChars]))
	if trimmed == "" {
		return normalized
	}

	return trimmed + "..."
}
