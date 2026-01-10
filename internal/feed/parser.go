package feed

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"telekilogram/internal/database"
	"telekilogram/internal/models"
	"telekilogram/internal/summarizer"
	"time"

	"github.com/mmcdole/gofeed"
)

const (
	telegramSummariesMaxParallelism = 4
	parseFeedGracePeriod            = 10 * time.Minute
	fallbackTelegramSummaryMaxChars = 200
)

type telegramSummarizationCandidate struct {
	postIndex int
	item      channelItem
}

type Parser struct {
	db           *database.Database
	summarizer   summarizer.Summarizer
	summaryCache *telegramSummaryCache
	log          *slog.Logger
}

func NewParser(
	db *database.Database,
	s summarizer.Summarizer,
	log *slog.Logger,
) *Parser {
	return &Parser{
		db:           db,
		summarizer:   s,
		summaryCache: newTelegramSummaryCache(telegramSummaryCacheMaxEntries),
		log:          log,
	}
}

func (fp *Parser) ParseFeed(
	ctx context.Context,
	feed *models.UserFeed,
) ([]models.Post, error) {
	normalizedFeedURL := strings.TrimSpace(feed.URL)
	normalizedFeedTitle := strings.TrimSpace(feed.Title)

	if ok, slug := isTelegramChannelURL(normalizedFeedURL); ok {
		return fp.parseTelegramChannelFeed(ctx, feed, slug, normalizedFeedTitle)
	}

	parsed, err := libParser.ParseURLWithContext(normalizedFeedURL, ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to parse feed by URL %q: %w", normalizedFeedURL, err)
	}

	parsedTitle := strings.TrimSpace(parsed.Title)

	var updateTitleErr error
	if parsedTitle != "" && parsedTitle != normalizedFeedTitle {
		if err = fp.db.UpdateFeedTitle(ctx, feed.ID, parsedTitle); err != nil {
			updateTitleErr = fmt.Errorf("failed to update feed title: %w", err)
		} else {
			normalizedFeedTitle = parsedTitle
		}
	}

	var newPosts []models.Post
	now := time.Now().Round(time.Hour)
	cutoffTime := now.Add(-24*time.Hour - parseFeedGracePeriod)

	for _, item := range parsed.Items {
		post, ok := fp.parseFeedItem(
			ctx,
			now,
			cutoffTime,
			normalizedFeedTitle,
			normalizedFeedURL,
			parsedTitle,
			feed.ID,
			item,
		)
		if !ok {
			continue
		}

		newPosts = append(newPosts, post)
	}

	return newPosts, updateTitleErr
}

func (fp *Parser) parseFeedItem(
	ctx context.Context,
	now time.Time,
	cutoffTime time.Time,
	normalizedFeedTitle string,
	normalizedFeedURL string,
	parsedTitle string,
	feedID int64,
	item *gofeed.Item,
) (models.Post, bool) {
	publishedTime := now

	if item.PublishedParsed != nil {
		publishedTime = *item.PublishedParsed
	} else if item.UpdatedParsed != nil {
		publishedTime = *item.UpdatedParsed
	}

	if publishedTime.After(cutoffTime) {
		postTitle := strings.TrimSpace(item.Title)
		postURL := strings.TrimSpace(item.Link)
		feedTitle := normalizedFeedTitle
		if feedTitle == "" {
			feedTitle = parsedTitle
		}
		if feedTitle == "" {
			feedTitle = normalizedFeedURL
		}

		if postURL == "" {
			fp.log.WarnContext(ctx, "Skipping feed item with empty URL",
				"feedURL", normalizedFeedURL,
				"feedTitle", feedTitle,
				"itemTitle", postTitle)

			return models.Post{}, false
		}

		return models.Post{
			Title:     postTitle,
			URL:       postURL,
			FeedID:    feedID,
			FeedTitle: feedTitle,
			FeedURL:   normalizedFeedURL,
		}, true
	}

	return models.Post{}, false
}

func (fp *Parser) parseTelegramChannelFeed(
	ctx context.Context,
	feed *models.UserFeed,
	slug string,
	normalizedFeedTitle string,
) ([]models.Post, error) {
	items, channelTitle, err := fetchTelegramChannelPosts(ctx, slug, fp.log)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Telegram channel items: %w", err)
	}

	channelTitle = strings.TrimSpace(channelTitle)

	var updateTitleErr error
	if channelTitle != "" && channelTitle != normalizedFeedTitle {
		if err = fp.db.UpdateFeedTitle(ctx, feed.ID, channelTitle); err != nil {
			updateTitleErr = fmt.Errorf("failed to update feed title: %w", err)
		} else {
			normalizedFeedTitle = channelTitle
		}
	}

	var (
		newPosts     []models.Post
		now          = time.Now().Round(time.Hour)
		cutoffTime   = now.Add(-24*time.Hour - parseFeedGracePeriod)
		candidates   []telegramSummarizationCandidate
		canonicalURL = TelegramChannelCanonicalURL(slug)
	)
	if canonicalURL == "" {
		return nil, fmt.Errorf("failed to build canonical URL for slug %q", slug)
	}

	feedTitle := channelTitle
	if feedTitle == "" {
		feedTitle = normalizedFeedTitle
	}
	if feedTitle == "" {
		feedTitle = canonicalURL
	}
	feedTitle = strings.TrimSpace(feedTitle)

	for _, it := range items {
		post, candidate, ok := fp.parseTelegramChannelPost(
			ctx,
			now,
			cutoffTime,
			canonicalURL,
			slug,
			it,
			len(newPosts),
			feed.ID,
			feedTitle,
		)
		if !ok {
			continue
		}

		newPosts = append(newPosts, post)
		candidates = append(candidates, candidate)
	}

	if len(candidates) > 0 {
		summaries := fp.summarizeTelegramPosts(ctx, candidates)
		for i := range candidates {
			candidate := candidates[i]
			if candidate.postIndex >= 0 && candidate.postIndex < len(newPosts) {
				newPosts[candidate.postIndex].Title = strings.TrimSpace(summaries[i])
			}
		}
	}

	return newPosts, updateTitleErr
}

func (fp *Parser) parseTelegramChannelPost(
	ctx context.Context,
	now time.Time,
	cutoffTime time.Time,
	canonicalURL string,
	slug string,
	item channelItem,
	processedPostCount int,
	feedID int64,
	feedTitle string,
) (models.Post, telegramSummarizationCandidate, bool) {
	publishedTime := item.published

	if publishedTime.IsZero() {
		publishedTime = now
	}

	if publishedTime.After(cutoffTime) {
		postURL := strings.TrimSpace(item.URL)
		if postURL == "" {
			fp.log.WarnContext(ctx, "Skipping Telegram post with empty URL",
				"canonicalFeedURL", canonicalURL,
				"slug", slug)

			return models.Post{}, telegramSummarizationCandidate{}, false
		}

		return models.Post{
			URL:       postURL,
			FeedID:    feedID,
			FeedTitle: feedTitle,
			FeedURL:   canonicalURL,
		}, telegramSummarizationCandidate{postIndex: processedPostCount, item: item}, true
	}

	return models.Post{}, telegramSummarizationCandidate{}, false
}

func (fp *Parser) summarizeTelegramPosts(
	ctx context.Context,
	candidates []telegramSummarizationCandidate,
) []string {
	summaries := make([]string, len(candidates))
	if len(candidates) == 0 {
		return summaries
	}

	workerCount := telegramSummariesMaxParallelism
	if workerCount <= 0 {
		workerCount = 1
	}
	if workerCount > len(candidates) {
		workerCount = len(candidates)
	}

	type task struct {
		resultIndex int
		candidate   telegramSummarizationCandidate
	}

	tasks := make(chan task)
	var wg sync.WaitGroup

	for range workerCount {
		wg.Go(func() {
			for t := range tasks {
				summaries[t.resultIndex] = fp.summarizeTelegramPost(ctx, t.candidate.item)
			}
		})
	}

	for i := range candidates {
		tasks <- task{
			resultIndex: i,
			candidate:   candidates[i],
		}
	}

	close(tasks)
	wg.Wait()

	return summaries
}

func (fp *Parser) summarizeTelegramPost(
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
		fp.log.ErrorContext(ctx, "Failed to summarize Telegram channel post",
			"error", err,
			"url", item.URL,
			"fallback", true,
			"cacheKey", cacheKey,
			"textLen", len(text))

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
