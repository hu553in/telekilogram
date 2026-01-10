package feed_test

import (
	"telekilogram/internal/feed"
	"testing"
)

func TestTelegramMessageCanonicalURL(t *testing.T) {
	raw := "https://t.me/example/123?single=1"
	got := feed.TelegramMessageCanonicalURL(raw)
	want := "https://t.me/example/123"
	if got != want {
		t.Fatalf("canonicalized URL mismatch: got %q want %q", got, want)
	}
}

func TestTelegramMessageCanonicalURLInvalid(t *testing.T) {
	raw := "::not a url::"
	if got := feed.TelegramMessageCanonicalURL(raw); got != raw {
		t.Fatalf("expected invalid URLs to be returned verbatim, got %q", got)
	}
}

func TestTelegramMessageCanonicalURLTrimsWhitespace(t *testing.T) {
	raw := "  https://t.me/example/123  "
	got := feed.TelegramMessageCanonicalURL(raw)
	want := "https://t.me/example/123"
	if got != want {
		t.Fatalf("expected trimmed URL, got %q", got)
	}
}

func TestTelegramChannelCanonicalURLTrimsSlug(t *testing.T) {
	got := feed.TelegramChannelCanonicalURL("  example  ")
	want := "https://t.me/s/example"
	if got != want {
		t.Fatalf("expected trimmed slug, got %q", got)
	}
}

func TestTelegramChannelCanonicalURLEmptySlug(t *testing.T) {
	if got := feed.TelegramChannelCanonicalURL("   "); got != "" {
		t.Fatalf("expected empty slug to return empty URL, got %q", got)
	}
}
