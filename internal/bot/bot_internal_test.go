package bot

import (
	"log/slog"
	"strings"
	"telekilogram/internal/config"
	"telekilogram/internal/domain"
	"testing"
	"unicode/utf8"
)

func botWithIssueURL(url string) *Bot {
	return &Bot{cfg: config.BotConfig{IssueURL: url}}
}

func TestWelcomeTextIncludesIssueLink(t *testing.T) {
	b := botWithIssueURL("https://github.com/hu553in/telekilogram/issues/new")

	got := b.welcomeText()

	if !strings.Contains(got, "[here](https://github.com/hu553in/telekilogram/issues/new)") {
		t.Fatalf("welcomeText() should include issue link, got %q", got)
	}
}

func TestWelcomeTextWithoutIssueURL(t *testing.T) {
	b := botWithIssueURL("")

	got := b.welcomeText()

	if got != welcomeTextBase {
		t.Fatalf("welcomeText() without IssueURL should equal welcomeTextBase, got %q", got)
	}
	if strings.Contains(got, "http") {
		t.Fatalf("welcomeText() without IssueURL should not contain any URL, got %q", got)
	}
}

func TestWithIssueReportLinkIncludesIssueURL(t *testing.T) {
	b := botWithIssueURL("https://github.com/hu553in/telekilogram/issues/new")

	got := b.withIssueReportLink("❌ Couldn't do this.\\.")

	if !strings.Contains(got, "[submit an issue](https://github.com/hu553in/telekilogram/issues/new)") {
		t.Fatalf("withIssueReportLink() should include issue link, got %q", got)
	}
}

func TestWithIssueReportLinkWithoutIssueURL(t *testing.T) {
	b := botWithIssueURL("")
	text := "❌ Some error\\."

	got := b.withIssueReportLink(text)

	if got != text {
		t.Fatalf("withIssueReportLink() without IssueURL should return text unchanged, got %q", got)
	}
}

func TestWithIssueReportLinkEmptyText(t *testing.T) {
	b := botWithIssueURL("https://github.com/hu553in/telekilogram/issues/new")

	got := b.withIssueReportLink("")

	if got != "" {
		t.Fatalf("withIssueReportLink() with empty text should return empty string, got %q", got)
	}
}

func TestWithIssueReportLinkTrimsText(t *testing.T) {
	b := botWithIssueURL("https://example.com/issues")
	text := "  ❌ Error\\.  "

	got := b.withIssueReportLink(text)

	if strings.HasPrefix(got, " ") || strings.HasPrefix(got, "\t") {
		t.Fatalf("withIssueReportLink() should trim leading whitespace from text, got %q", got)
	}
	if !strings.HasPrefix(got, "❌") {
		t.Fatalf("withIssueReportLink() should start with the trimmed text, got %q", got)
	}
}

func TestSplitTelegramTextRespectsLimit(t *testing.T) {
	text := strings.Repeat("a", telegramMessageMaxLength+123)

	chunks := splitTelegramText(text)
	if len(chunks) != 2 {
		t.Fatalf("expected 2 chunks, got %d", len(chunks))
	}

	for i, chunk := range chunks {
		if utf8.RuneCountInString(chunk) > telegramMessageMaxLength {
			t.Fatalf("chunk %d exceeds limit", i)
		}
	}
}

func TestSplitTelegramTextAtExactLimit(t *testing.T) {
	text := strings.Repeat("a", telegramMessageMaxLength)

	chunks := splitTelegramText(text)
	if len(chunks) != 1 {
		t.Fatalf("text at exact limit should not be split, got %d chunks", len(chunks))
	}
	if chunks[0] != text {
		t.Fatal("single chunk should equal original text")
	}
}

func TestSplitTelegramTextEmpty(t *testing.T) {
	chunks := splitTelegramText("")
	if len(chunks) != 1 || chunks[0] != "" {
		t.Fatalf("empty text should produce single empty chunk, got %v", chunks)
	}
}

func TestSplitTelegramTextMultibyteRunes(t *testing.T) {
	// Each '日' is 3 bytes; byte-based splitting would produce many more chunks.
	text := strings.Repeat("日", telegramMessageMaxLength+1)

	chunks := splitTelegramText(text)
	if len(chunks) != 2 {
		t.Fatalf("expected 2 chunks for multibyte text, got %d", len(chunks))
	}
	if utf8.RuneCountInString(chunks[0]) != telegramMessageMaxLength {
		t.Fatalf("first chunk: expected %d runes, got %d", telegramMessageMaxLength, utf8.RuneCountInString(chunks[0]))
	}
	if utf8.RuneCountInString(chunks[1]) != 1 {
		t.Fatalf("second chunk: expected 1 rune, got %d", utf8.RuneCountInString(chunks[1]))
	}
}

func TestSplitTelegramTextPreservesContent(t *testing.T) {
	text := strings.Repeat("hello\n", telegramMessageMaxLength/5)

	chunks := splitTelegramText(text)
	if strings.Join(chunks, "") != text {
		t.Fatal("rejoined chunks should equal original text")
	}
}

func TestFormatMarkdownLinkBasic(t *testing.T) {
	got := formatMarkdownLink("Hello world", "https://example.com")

	if !strings.HasPrefix(got, "[") {
		t.Fatalf("expected link markup starting with '[', got %q", got)
	}
	if !strings.HasSuffix(got, "(https://example.com)") {
		t.Fatalf("expected link markup ending with '(url)', got %q", got)
	}
}

func TestFormatMarkdownLinkEmptyTitle(t *testing.T) {
	url := "https://example.com"
	got := formatMarkdownLink("", url)

	// Empty title → URL is used as title, still formatted as a link.
	if !strings.Contains(got, "("+url+")") {
		t.Fatalf("expected URL in link markup, got %q", got)
	}
	if !strings.HasPrefix(got, "[") {
		t.Fatalf("expected link markup, got %q", got)
	}
}

func TestFormatMarkdownLinkEmptyURL(t *testing.T) {
	got := formatMarkdownLink("Hello", "")

	if strings.Contains(got, "[") || strings.Contains(got, "(") {
		t.Fatalf("empty URL should produce plain text without link markup, got %q", got)
	}
	if got == "" {
		t.Fatal("empty URL should still return escaped title, got empty string")
	}
}

func TestFormatMarkdownLinkBothEmpty(t *testing.T) {
	got := formatMarkdownLink("", "")
	if got != "" {
		t.Fatalf("both empty should return empty string, got %q", got)
	}
}

func TestFormatMarkdownLinkTruncatesLongTitle(t *testing.T) {
	// "A" has no special Markdown chars, so escapedTitle is just "A"s.
	title := strings.Repeat("A", telegramMarkdownLinkMaxLength+10)
	got := formatMarkdownLink(title, "https://example.com")

	aCount := strings.Count(got, "A")
	if aCount != telegramMarkdownLinkMaxLength-3 {
		t.Fatalf("truncated title should have %d 'A's, got %d (full result: %q)",
			telegramMarkdownLinkMaxLength-3, aCount, got)
	}
	if !strings.Contains(got, "(https://example.com)") {
		t.Fatalf("truncated link should still contain URL, got %q", got)
	}
}

func TestFormatPostsAsMessagesRespectsLimit(t *testing.T) {
	b := &Bot{log: slog.Default()}
	var posts []domain.Post

	for i := range 200 {
		posts = append(posts, domain.Post{
			FeedID:    1,
			FeedTitle: "Feed",
			FeedURL:   "https://example.com/feed",
			Title:     strings.Repeat("A", 50),
			URL:       "https://example.com/posts/" + strings.Repeat("1", i%10+1),
		})
	}

	messages := b.formatPostsAsMessages(t.Context(), posts)
	if len(messages) < 2 {
		t.Fatalf("expected multiple digest messages, got %d", len(messages))
	}

	for i, message := range messages {
		if utf8.RuneCountInString(message) > telegramMessageMaxLength {
			t.Fatalf("message %d exceeds limit", i)
		}
	}
}
