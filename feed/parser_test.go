package feed

import (
	"context"
	"sync"
	"testing"
	"time"

	"telekilogram/summarizer"
)

const editedSummary = "edited summary"

type stubSummarizer struct {
	mu      sync.Mutex
	calls   int
	summary string
}

func (s *stubSummarizer) Summarize(
	ctx context.Context,
	input summarizer.Input,
) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++

	return s.summary, nil
}

func (s *stubSummarizer) callCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.calls
}

func TestTelegramSummaryCacheKey(t *testing.T) {
	keyA := telegramSummaryCacheKey(
		" https://t.me/example/123?single=1 ",
		" Example post text ",
	)
	keyB := telegramSummaryCacheKey(
		"https://t.me/example/123",
		"Example post text",
	)

	if keyA == "" || keyB == "" {
		t.Fatalf("expected non-empty cache keys")
	}

	if keyA != keyB {
		t.Fatalf(
			"expected canonicalized cache keys to match, got %q vs %q",
			keyA,
			keyB,
		)
	}

	if key := telegramSummaryCacheKey("https://t.me/example/123", " "); key != "" {
		t.Fatalf("expected empty cache key when text is empty, got %q", key)
	}
}

func TestFeedParserSummarizeTelegramPostUsesCache(t *testing.T) {
	stub := &stubSummarizer{summary: "cached summary"}
	parser := NewFeedParser(nil, stub)

	item := channelItem{
		URL:       "https://t.me/example/123",
		Text:      "Example post text",
		published: time.Now().UTC(),
	}

	ctx := context.Background()

	first := parser.summarizeTelegramPost(ctx, item)
	second := parser.summarizeTelegramPost(ctx, item)

	if first != "cached summary" {
		t.Fatalf("unexpected first summary: %q", first)
	}

	if second != "cached summary" {
		t.Fatalf("unexpected second summary: %q", second)
	}

	if got := stub.callCount(); got != 1 {
		t.Fatalf("expected summarizer to be called once, got %d", got)
	}
}

func TestFeedParserSummarizeTelegramPostEditedTextBypassesCache(t *testing.T) {
	stub := &stubSummarizer{summary: "original summary"}
	parser := NewFeedParser(nil, stub)

	item := channelItem{
		URL:       "https://t.me/example/123",
		Text:      "Example post text",
		published: time.Now().UTC(),
	}

	ctx := context.Background()

	if summary := parser.summarizeTelegramPost(
		ctx,
		item,
	); summary != "original summary" {
		t.Fatalf("unexpected initial summary: %q", summary)
	}

	if got := stub.callCount(); got != 1 {
		t.Fatalf("expected summarizer to be called once, got %d", got)
	}

	stub.summary = editedSummary
	edited := item
	edited.Text = "Example post text (edited)"

	if summary := parser.summarizeTelegramPost(
		ctx,
		edited,
	); summary != editedSummary {
		t.Fatalf("unexpected edited summary: %q", summary)
	}

	if got := stub.callCount(); got != 2 {
		t.Fatalf("expected summarizer to be called twice after edit, got %d", got)
	}

	stub.summary = "should not be used"
	if summary := parser.summarizeTelegramPost(
		ctx,
		edited,
	); summary != editedSummary {
		t.Fatalf("expected cached edited summary, got %q", summary)
	}

	if got := stub.callCount(); got != 2 {
		t.Fatalf("expected cache hit for edited summary, calls %d", got)
	}
}
