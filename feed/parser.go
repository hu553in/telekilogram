package feed

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"telekilogram/database"
	"telekilogram/models"
	"telekilogram/summarizer"
)

const telegramSummariesMaxParallelism = 4

type telegramSummarizationCandidate struct {
	postIndex int
	item      channelItem
}

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
	normalizedFeedURL := strings.TrimSpace(feed.URL)
	normalizedFeedTitle := strings.TrimSpace(feed.Title)

	if ok, slug := isTelegramChannelURL(normalizedFeedURL); ok {
		items, channelTitle, err := fetchTelegramChannelPosts(slug)
		if err != nil {
			return nil, fmt.Errorf(
				"failed to fetch Telegram channel items: %w",
				err,
			)
		}

		channelTitle = strings.TrimSpace(channelTitle)

		var updateTitleErr error
		if channelTitle != "" && channelTitle != normalizedFeedTitle {
			if err := fp.db.UpdateFeedTitle(feed.ID, channelTitle); err != nil {
				updateTitleErr = fmt.Errorf(
					"failed to update feed title: %w",
					err,
				)
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
			publishedTime := it.published

			if publishedTime.IsZero() {
				publishedTime = now
			}

			if publishedTime.After(cutoffTime) {
				postURL := strings.TrimSpace(it.URL)
				if postURL == "" {
					slog.Warn("Skipping Telegram post with empty URL",
						slog.String("canonicalFeedURL", canonicalURL),
						slog.String("slug", slug))

					continue
				}

				postIndex := len(newPosts)
				newPosts = append(newPosts, models.Post{
					URL:       postURL,
					FeedID:    feed.ID,
					FeedTitle: feedTitle,
					FeedURL:   canonicalURL,
				})

				candidates = append(candidates, telegramSummarizationCandidate{
					postIndex: postIndex,
					item:      it,
				})
			}
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

	parsed, err := libParser.ParseURL(normalizedFeedURL)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to parse feed by URL %q: %w",
			normalizedFeedURL,
			err,
		)
	}

	parsedTitle := strings.TrimSpace(parsed.Title)

	var updateTitleErr error
	if parsedTitle != "" && parsedTitle != normalizedFeedTitle {
		if err := fp.db.UpdateFeedTitle(feed.ID, parsedTitle); err != nil {
			updateTitleErr = fmt.Errorf("failed to update feed title: %w", err)
		} else {
			normalizedFeedTitle = parsedTitle
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
				slog.Warn("Skipping feed item with empty URL",
					slog.String("feedURL", normalizedFeedURL),
					slog.String("feedTitle", feedTitle),
					slog.String("itemTitle", postTitle))

				continue
			}

			newPosts = append(newPosts, models.Post{
				Title:     postTitle,
				URL:       postURL,
				FeedID:    feed.ID,
				FeedTitle: feedTitle,
				FeedURL:   normalizedFeedURL,
			})
		}
	}

	return newPosts, updateTitleErr
}

func (fp *FeedParser) summarizeTelegramPosts(
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

	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for t := range tasks {
				summaries[t.resultIndex] = fp.summarizeTelegramPost(ctx, t.candidate.item)
			}
		}()
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
			slog.String("url", item.URL),
			slog.Bool("fallback", true),
			slog.String("cacheKey", cacheKey),
			slog.Int("textLen", len(text)))

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
