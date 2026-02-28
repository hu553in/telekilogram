package feed

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"runtime"
	"strings"
	"sync"
	"telekilogram/internal/domain"
	"time"

	"telekilogram/internal/database"
	"telekilogram/internal/summarizer"

	"github.com/mmcdole/gofeed"
	"mvdan.cc/xurls/v2"
)

const (
	telegramClientTimeout                = 20 * time.Second
	fetchFeedsMaxConcurrencyGrowthFactor = 10
)

type Fetcher struct {
	db             *database.Database
	parser         *Parser
	libParser      *gofeed.Parser
	telegramClient *http.Client
	log            *slog.Logger
}

func NewFetcher(
	db *database.Database,
	s summarizer.Summarizer,
	log *slog.Logger,
) *Fetcher {
	libParser := gofeed.NewParser()
	telegramClient := &http.Client{Timeout: telegramClientTimeout}

	return &Fetcher{
		db:             db,
		parser:         NewParser(db, s, libParser, telegramClient, log),
		libParser:      libParser,
		telegramClient: telegramClient,
		log:            log,
	}
}

func (f *Fetcher) FindValidFeeds(
	ctx context.Context,
	text string,
) ([]domain.Feed, error) {
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
		return nil, fmt.Errorf("create regexp: %w", err)
	}

	urls := httpsURLRe.FindAllString(text, -1)

	feeds := make([]domain.Feed, 0, len(urls)+len(slugs))
	seen := make(map[string]struct{}, len(urls)+len(slugs))
	var errs []error

	for _, u := range urls {
		trimmed := strings.TrimSpace(u)

		feed, validateFeedErr := f.validateFeed(ctx, trimmed)
		if validateFeedErr != nil {
			errs = append(errs, fmt.Errorf("validate feed: %w", validateFeedErr))
			continue
		}
		if feed == nil {
			errs = append(errs, errors.New("validate feed"))
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

		feed, validateFeedErr := f.validateFeed(ctx, canonicalURL)
		if validateFeedErr != nil {
			errs = append(errs, fmt.Errorf("validate feed: %w", validateFeedErr))
			continue
		}
		if feed == nil {
			errs = append(errs, errors.New("validate feed"))
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

func (f *Fetcher) FetchHourFeeds(
	ctx context.Context,
	hourUTC int64,
) (map[int64][]domain.Post, error) {
	feeds, err := f.db.GetHourFeeds(ctx, hourUTC)
	if err != nil {
		return nil, fmt.Errorf("get hour feeds: %w", err)
	}

	return f.fetchFeeds(ctx, feeds)
}

func (f *Fetcher) FetchUserFeeds(
	ctx context.Context,
	userID int64,
) (map[int64][]domain.Post, error) {
	feeds, err := f.db.GetUserFeeds(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get user feeds: %w", err)
	}

	return f.fetchFeeds(ctx, feeds)
}

func (f *Fetcher) validateFeed(
	ctx context.Context,
	feedURL string,
) (*domain.Feed, error) {
	feedURL = strings.TrimSpace(feedURL)
	if feedURL == "" {
		return nil, errors.New("feed URL is empty")
	}

	if _, err := url.Parse(feedURL); err != nil {
		return nil, fmt.Errorf("parse URL: %w", err)
	}

	if ok, slug := isTelegramChannelURL(feedURL); ok {
		title, err := f.fetchTelegramChannelTitle(ctx, slug)
		if err != nil {
			return nil, fmt.Errorf("fetch Telegram channel title: %w", err)
		}

		canonicalURL := TelegramChannelCanonicalURL(slug)
		if canonicalURL == "" {
			return nil, fmt.Errorf("build canonical URL (slug = %s)", slug)
		}

		title = strings.TrimSpace(title)
		if title == "" {
			f.log.WarnContext(ctx, "Empty Telegram channel title",
				"canonicalURL", canonicalURL,
				"slug", slug)

			title = canonicalURL
		}

		return &domain.Feed{
			URL:   canonicalURL,
			Title: title,
		}, nil
	}

	parsed, err := f.libParser.ParseURLWithContext(feedURL, ctx)
	if err != nil {
		return nil, fmt.Errorf("parse feed (URL = %s): %w", feedURL, err)
	}

	title := strings.TrimSpace(parsed.Title)
	if title == "" {
		f.log.WarnContext(ctx, "Empty feed title",
			"feedURL", feedURL,
			"fallbackTitle", feedURL)

		title = feedURL
	}

	return &domain.Feed{URL: feedURL, Title: title}, nil
}

func (f *Fetcher) fetchFeeds(
	ctx context.Context,
	feeds []domain.UserFeed,
) (map[int64][]domain.Post, error) {
	var writeWg sync.WaitGroup

	concurrency := min(runtime.NumCPU()*fetchFeedsMaxConcurrencyGrowthFactor, len(feeds))
	semCh := make(chan struct{}, concurrency)

	userPostCh := make(chan domain.UserPosts, concurrency)
	errCh := make(chan error, concurrency)

	for _, feed := range feeds {
		writeWg.Add(1)
		semCh <- struct{}{}

		go func(copiedFeed domain.UserFeed) {
			defer writeWg.Done()

			posts, err := f.parser.ParseFeed(ctx, &copiedFeed)
			if err != nil {
				errCh <- fmt.Errorf("parse feed: %w", err)
			}

			if len(posts) != 0 {
				userPostCh <- domain.UserPosts{UserID: copiedFeed.UserID, Posts: posts}
			}

			<-semCh
		}(feed)
	}

	go func() {
		writeWg.Wait()
		close(semCh)
		close(userPostCh)
		close(errCh)
	}()

	userPostsMap := make(map[int64][]domain.Post)
	for userPosts := range userPostCh {
		userPostsMap[userPosts.UserID] = append(userPostsMap[userPosts.UserID], userPosts.Posts...)
	}

	var errs []error
	for err := range errCh {
		errs = append(errs, err)
	}

	return userPostsMap, errors.Join(errs...)
}
