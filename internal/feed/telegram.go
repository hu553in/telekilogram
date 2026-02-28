package feed

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

const (
	userAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) " +
		"AppleWebKit/537.36 (KHTML, like Gecko) Chrome/127.0.0.0 Safari/537.36"

	minPartsForTelegramChannelSlugStartingWithS = 2
	minPartsForTelegramChannelAtSignSlug        = 3

	telegramHost = "t.me"
)

var (
	telegramSlugRe       = regexp.MustCompile(`^\w{5,32}$`)
	telegramAtSignSlugRe = regexp.MustCompile(`(\s|^)@(\w{5,32})(\s|$)`)
)

type channelItem struct {
	URL       string
	text      string
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

func TelegramChannelCanonicalURL(slug string) string {
	slug = strings.TrimSpace(slug)
	if slug == "" {
		return ""
	}

	return fmt.Sprintf("https://%s/s/%s", telegramHost, slug)
}

func isTelegramChannelURL(raw string) (bool, string) {
	raw = strings.TrimSpace(raw)
	u, err := url.Parse(raw)
	if err != nil {
		return false, ""
	}

	if u.Host != telegramHost {
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
		if len(parts) < minPartsForTelegramChannelSlugStartingWithS {
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

func (f *Fetcher) fetchTelegramChannelTitle(
	ctx context.Context,
	slug string,
) (string, error) {
	canonicalURL := TelegramChannelCanonicalURL(slug)
	if canonicalURL == "" {
		return "", errors.New("slug is empty")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, canonicalURL, nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("User-Agent", userAgent)

	resp, err := f.telegramClient.Do(req) //nolint:gosec // Telegram URL
	if err != nil {
		return "", fmt.Errorf("do request: %w", err)
	}
	defer func() {
		if err = resp.Body.Close(); err != nil {
			f.log.ErrorContext(ctx, "Failed to close response body",
				"error", err,
				"canonicalURL", canonicalURL,
				"operation", "fetchTelegramChannelTitle",
				"slug", slug)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("do request: unexpected status: %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return "", fmt.Errorf("create document from reader: %w", err)
	}

	if content, ok := doc.Find("meta[property='og:title']").Attr("content"); ok {
		return strings.TrimSpace(content), nil
	}

	return strings.TrimSpace(doc.Find(".tgme_channel_info_header_title > span").Text()), nil
}

func (p *Parser) fetchTelegramChannelPosts(
	ctx context.Context,
	slug string,
) ([]channelItem, string, error) {
	canonicalURL := TelegramChannelCanonicalURL(slug)
	if canonicalURL == "" {
		return nil, "", errors.New("slug is empty")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, canonicalURL, nil)
	if err != nil {
		return nil, "", fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("User-Agent", userAgent)

	resp, err := p.telegramClient.Do(req) //nolint:gosec // Telegram URL
	if err != nil {
		return nil, "", fmt.Errorf("do request: %w", err)
	}
	defer func() {
		if err = resp.Body.Close(); err != nil {
			p.log.ErrorContext(ctx, "Failed to close response body",
				"error", err,
				"canonicalURL", canonicalURL,
				"operation", "fetchTelegramChannelPosts",
				"slug", slug)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("do request: unexpected status: %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("create document from reader: %w", err)
	}

	var items []channelItem
	var errs []error

	doc.Find("a.tgme_widget_message_date").Each(func(_ int, s *goquery.Selection) {
		item, processErr := processFoundDocItem(s)
		if processErr != nil {
			errs = append(errs, fmt.Errorf("process found doc item: %w", processErr))
			return
		}

		items = append(items, item)
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

func processFoundDocItem(s *goquery.Selection) (channelItem, error) {
	href, ok := s.Attr("href")
	if !ok || href == "" {
		return channelItem{}, errors.New("href empty")
	}

	href = TelegramMessageCanonicalURL(href)

	var textBuilder strings.Builder
	message := s.ParentsFiltered(".tgme_widget_message").First()
	message.Find(".tgme_widget_message_text, .tgme_widget_message_caption").Each(
		func(_ int, inner *goquery.Selection) {
			inner.Find("br").Each(func(_ int, br *goquery.Selection) {
				br.ReplaceWithHtml("\n")
			})
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
		parsed, timeParseErr := time.Parse(time.RFC3339, datetime)
		if timeParseErr != nil {
			return channelItem{}, fmt.Errorf("parse datetime: %w", timeParseErr)
		}
		t = parsed
	}

	if t.IsZero() {
		t = time.Now().UTC()
	}

	return channelItem{URL: href, text: text, published: t}, nil
}
