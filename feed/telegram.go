package feed

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"log/slog"

	"github.com/PuerkitoBio/goquery"
)

type channelItem struct {
	URL       string
	published time.Time
}

func isTelegramChannelURL(raw string) (bool, string) {
	u, err := url.Parse(raw)
	if err != nil {
		return false, ""
	}

	if u.Host != TelegramHost {
		return false, ""
	}

	path := strings.Trim(u.Path, "/")
	if path == "" {
		return false, ""
	}

	parts := strings.Split(path, "/")
	if len(parts) == 0 {
		return false, ""
	}

	var slug string

	switch parts[0] {
	case "s":
		if len(parts) < 2 {
			return false, ""
		}
		slug = parts[1]
	default:
		slug = parts[0]
	}

	if !telegramSlugRe.MatchString(slug) {
		return false, ""
	}

	return true, slug
}

func fetchTelegramChannelTitle(slug string) (string, error) {
	canonicalURL := TelegramChannelCanonicalURL(slug)

	req, err := http.NewRequest(http.MethodGet, canonicalURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", userAgent)

	resp, err := telegramClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to do request: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			slog.Error("Failed to close response body",
				slog.Any("err", err),
				slog.String("canonicalURL", canonicalURL))
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf(
			"failed to do request: unexpected status: %d",
			resp.StatusCode,
		)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to create document from reader: %w", err)
	}

	if content, ok := doc.Find("meta[property='og:title']").Attr("content"); ok {
		return strings.TrimSpace(content), nil
	}

	return strings.TrimSpace(
		doc.Find(".tgme_channel_info_header_title > span").Text(),
	), nil
}

func fetchTelegramChannelItems(slug string) ([]channelItem, string, error) {
	canonicalURL := TelegramChannelCanonicalURL(slug)

	req, err := http.NewRequest(http.MethodGet, canonicalURL, nil)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", userAgent)

	resp, err := telegramClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("failed to do request: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			slog.Error("Failed to close response body",
				slog.Any("err", err),
				slog.String("canonicalURL", canonicalURL))
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf(
			"failed to do request: unexpected status: %d",
			resp.StatusCode,
		)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create document from reader: %w", err)
	}

	var items []channelItem
	var errs []error

	doc.Find("a.tgme_widget_message_date").Each(func(_ int, s *goquery.Selection) {
		href, ok := s.Attr("href")
		if !ok || href == "" {
			return
		}

		var t time.Time
		datetime := strings.TrimSpace(s.Find("time").AttrOr("datetime", ""))

		if datetime != "" {
			if parsed, err := time.Parse(time.RFC3339, datetime); err != nil {
				errs = append(errs, fmt.Errorf("failed to parse datetime: %w", err))
			} else {
				t = parsed
			}
		}

		if t.IsZero() {
			t = time.Now().UTC()
		}

		items = append(items, channelItem{URL: href, published: t})
	})

	var title string

	if content, ok := doc.Find("meta[property='og:title']").Attr("content"); ok {
		title = strings.TrimSpace(content)
	}

	if title == "" {
		title = strings.TrimSpace(doc.Find(".tgme_channel_info_header_title").Text())
	}

	return items, title, errors.Join(errs...)
}
