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
	Text      string
	published time.Time
}

func TelegramMessageCanonicalURL(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}

	u, err := url.Parse(trimmed)
	if err != nil {
		return trimmed
	}

	u.RawQuery = ""
	u.Fragment = ""

	return u.String()
}

func isTelegramChannelURL(raw string) (bool, string) {
	raw = strings.TrimSpace(raw)
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

	slug = strings.TrimSpace(slug)

	if !telegramSlugRe.MatchString(slug) {
		return false, ""
	}

	return true, slug
}

func fetchTelegramChannelTitle(slug string) (string, error) {
	canonicalURL := TelegramChannelCanonicalURL(slug)
	if canonicalURL == "" {
		return "", fmt.Errorf("slug is empty")
	}

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
				slog.String("canonicalURL", canonicalURL),
				slog.String("operation", "fetchTelegramChannelTitle"),
				slog.String("slug", slug))
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

func fetchTelegramChannelPosts(slug string) ([]channelItem, string, error) {
	canonicalURL := TelegramChannelCanonicalURL(slug)
	if canonicalURL == "" {
		return nil, "", fmt.Errorf("slug is empty")
	}

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
				slog.String("canonicalURL", canonicalURL),
				slog.String("operation", "fetchTelegramChannelPosts"),
				slog.String("slug", slug))
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

		href = TelegramMessageCanonicalURL(href)

		var textBuilder strings.Builder
		message := s.ParentsFiltered(".tgme_widget_message").First()
		message.Find(".tgme_widget_message_text, .tgme_widget_message_caption").Each(
			func(_ int, inner *goquery.Selection) {
				fragment := strings.TrimSpace(inner.Text())
				if fragment == "" {
					return
				}
				if textBuilder.Len() > 0 {
					textBuilder.WriteString("\n")
				}
				textBuilder.WriteString(fragment)
			},
		)
		text := strings.TrimSpace(textBuilder.String())

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

		items = append(items, channelItem{URL: href, Text: text, published: t})
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
