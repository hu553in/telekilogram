package feed

import "testing"

func TestTelegramMessageCanonicalURL(t *testing.T) {
	raw := "https://t.me/example/123?single=1"
	got := TelegramMessageCanonicalURL(raw)
	want := "https://t.me/example/123"
	if got != want {
		t.Fatalf("canonicalized URL mismatch: got %q want %q", got, want)
	}
}

func TestTelegramMessageCanonicalURLInvalid(t *testing.T) {
	raw := "::not a url::"
	if got := TelegramMessageCanonicalURL(raw); got != raw {
		t.Fatalf("expected invalid URLs to be returned verbatim, got %q", got)
	}
}

func TestTelegramMessageCanonicalURLTrimsWhitespace(t *testing.T) {
	raw := "  https://t.me/example/123  "
	got := TelegramMessageCanonicalURL(raw)
	want := "https://t.me/example/123"
	if got != want {
		t.Fatalf("expected trimmed URL, got %q", got)
	}
}
