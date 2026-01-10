package feed

import (
	"context"
	"log/slog"
	"sync"
	"telekilogram/internal/summarizer"
	"testing"
	"time"
)

const editedSummary = "edited summary"

type stubSummarizer struct {
	mu      sync.Mutex
	calls   int
	summary string
}

func (s *stubSummarizer) Summarize(
	_ context.Context,
	_ summarizer.Input,
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

type echoCountingSummarizer struct {
	mu    sync.Mutex
	calls int
}

func (s *echoCountingSummarizer) Summarize(
	_ context.Context,
	input summarizer.Input,
) (string, error) {
	s.mu.Lock()
	s.calls++
	s.mu.Unlock()

	return input.Text, nil
}

func (s *echoCountingSummarizer) callCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.calls
}

func TestTelegramSummaryCacheKey(t *testing.T) {
	keyA := telegramSummaryCacheKey(" https://t.me/example/123?single=1 ", " Example post text ")
	keyB := telegramSummaryCacheKey("https://t.me/example/123", "Example post text")

	if keyA == "" || keyB == "" {
		t.Fatalf("expected non-empty cache keys")
	}

	if keyA != keyB {
		t.Fatalf("expected canonicalized cache keys to match, got %q vs %q", keyA, keyB)
	}

	if key := telegramSummaryCacheKey("https://t.me/example/123", " "); key != "" {
		t.Fatalf("expected empty cache key when text is empty, got %q", key)
	}
}

func TestFeedParserSummarizeTelegramPostUsesCache(t *testing.T) {
	stub := &stubSummarizer{summary: "cached summary"}
	parser := NewParser(nil, stub, slog.Default())

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
	parser := NewParser(nil, stub, slog.Default())

	item := channelItem{
		URL:       "https://t.me/example/123",
		Text:      "Example post text",
		published: time.Now().UTC(),
	}

	ctx := context.Background()

	if summary := parser.summarizeTelegramPost(ctx, item); summary != "original summary" {
		t.Fatalf("unexpected initial summary: %q", summary)
	}

	if got := stub.callCount(); got != 1 {
		t.Fatalf("expected summarizer to be called once, got %d", got)
	}

	stub.summary = editedSummary
	edited := item
	edited.Text = "Example post text (edited)"

	if summary := parser.summarizeTelegramPost(ctx, edited); summary != editedSummary {
		t.Fatalf("unexpected edited summary: %q", summary)
	}

	if got := stub.callCount(); got != 2 {
		t.Fatalf("expected summarizer to be called twice after edit, got %d", got)
	}

	stub.summary = "should not be used"
	if summary := parser.summarizeTelegramPost(ctx, edited); summary != editedSummary {
		t.Fatalf("expected cached edited summary, got %q", summary)
	}

	if got := stub.callCount(); got != 2 {
		t.Fatalf("expected cache hit for edited summary, calls %d", got)
	}
}

func TestFeedParserSummarizeTelegramPostsPreservesOrder(t *testing.T) {
	echo := &echoCountingSummarizer{}
	parser := NewParser(nil, echo, slog.Default())

	candidates := []telegramSummarizationCandidate{
		{
			postIndex: 2,
			item:      channelItem{URL: "https://t.me/example/3", Text: "third"},
		},
		{
			postIndex: 0,
			item:      channelItem{URL: "https://t.me/example/1", Text: "first"},
		},
		{
			postIndex: 1,
			item:      channelItem{URL: "https://t.me/example/2", Text: "second"},
		},
	}

	ctx := context.Background()
	summaries := parser.summarizeTelegramPosts(ctx, candidates)

	if got := echo.callCount(); got != len(candidates) {
		t.Fatalf("expected summarizer to be called %d times, got %d", len(candidates), got)
	}

	maxIndex := 0
	for _, candidate := range candidates {
		if candidate.postIndex > maxIndex {
			maxIndex = candidate.postIndex
		}
	}

	titles := make([]string, maxIndex+1)
	for i := range candidates {
		titles[candidates[i].postIndex] = summaries[i]
	}

	want := []string{"first", "second", "third"}
	for i := range want {
		if titles[i] != want[i] {
			t.Fatalf("unexpected title at index %d: got %q want %q", i, titles[i], want[i])
		}
	}
}
