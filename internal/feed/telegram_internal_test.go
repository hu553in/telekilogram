package feed

import (
	"slices"
	"testing"
)

func TestFindTelegramChannelURLCandidates(t *testing.T) {
	text := "check t.me/thestrikemch and t.me/s/another_channel, plus duplicate t.me/thestrikemch."

	got := findTelegramChannelURLCandidates(text)
	want := []string{
		"https://t.me/thestrikemch",
		"https://t.me/s/another_channel",
	}

	if !slices.Equal(got, want) {
		t.Fatalf("unexpected candidates: got %q want %q", got, want)
	}
}

func TestFindTelegramChannelURLCandidatesTrimsTrailingPunctuation(t *testing.T) {
	got := findTelegramChannelURLCandidates("try t.me/thestrikemch), then continue.")
	want := []string{"https://t.me/thestrikemch"}

	if !slices.Equal(got, want) {
		t.Fatalf("unexpected candidates: got %q want %q", got, want)
	}
}

func TestFindTelegramChannelURLCandidatesSupportsSPath(t *testing.T) {
	got := findTelegramChannelURLCandidates("read t.me/s/another_channel for updates")
	want := []string{"https://t.me/s/another_channel"}

	if !slices.Equal(got, want) {
		t.Fatalf("unexpected candidates: got %q want %q", got, want)
	}
}

func TestFindTelegramChannelURLCandidatesDeduplicatesAcrossForms(t *testing.T) {
	text := "first t.me/thestrikemch then https://t.me/thestrikemch and again t.me/thestrikemch."

	got := findTelegramChannelURLCandidates(text)
	want := []string{"https://t.me/thestrikemch"}

	if !slices.Equal(got, want) {
		t.Fatalf("unexpected candidates: got %q want %q", got, want)
	}
}

func TestFindTelegramChannelURLCandidatesPreservesFirstSeenOrder(t *testing.T) {
	text := "links: t.me/second_one t.me/first_one t.me/second_one t.me/s/third_one"

	got := findTelegramChannelURLCandidates(text)
	want := []string{
		"https://t.me/second_one",
		"https://t.me/first_one",
		"https://t.me/s/third_one",
	}

	if !slices.Equal(got, want) {
		t.Fatalf("unexpected candidates: got %q want %q", got, want)
	}
}

func TestFindTelegramChannelURLCandidatesAcceptsHTTPAndHTTPS(t *testing.T) {
	text := "check http://t.me/thestrikemch and https://t.me/s/another_channel"

	got := findTelegramChannelURLCandidates(text)
	want := []string{
		"https://t.me/thestrikemch",
		"https://t.me/s/another_channel",
	}

	if !slices.Equal(got, want) {
		t.Fatalf("unexpected candidates: got %q want %q", got, want)
	}
}

func TestFindTelegramChannelURLCandidatesIgnoresTooShortNames(t *testing.T) {
	text := "ignore t.me/abcd and keep t.me/abcde"

	got := findTelegramChannelURLCandidates(text)
	want := []string{"https://t.me/abcde"}

	if !slices.Equal(got, want) {
		t.Fatalf("unexpected candidates: got %q want %q", got, want)
	}
}

func TestFindTelegramChannelURLCandidatesExtractsPrefixFromTrailingUnderscore(t *testing.T) {
	// "channel_" is not a valid Telegram username (trailing underscore), but the regex
	// still extracts "channel" from it — this is intentional; downstream validation
	// (via fetch) will reject invalid channels.
	text := "t.me/channel_ and t.me/channel_ok"

	got := findTelegramChannelURLCandidates(text)
	want := []string{"https://t.me/channel", "https://t.me/channel_ok"}

	if !slices.Equal(got, want) {
		t.Fatalf("unexpected candidates: got %q want %q", got, want)
	}
}

func TestFindTelegramChannelURLCandidatesExtractsSlugFromSubpath(t *testing.T) {
	// "share" is extracted from "t.me/share/url" — downstream validation will reject
	// it if it's not a real channel. Invalid prefixes like "+" are never extracted.
	text := "ignore t.me/+abcdef, t.me/share/url, and t.me/"

	got := findTelegramChannelURLCandidates(text)
	want := []string{"https://t.me/share"}

	if !slices.Equal(got, want) {
		t.Fatalf("unexpected candidates: got %q want %q", got, want)
	}
}

func TestFindTelegramChannelURLCandidatesHandlesWrappedLinks(t *testing.T) {
	text := "(t.me/thestrikemch), [t.me/s/another_channel]."

	got := findTelegramChannelURLCandidates(text)
	want := []string{
		"https://t.me/thestrikemch",
		"https://t.me/s/another_channel",
	}

	if !slices.Equal(got, want) {
		t.Fatalf("unexpected candidates: got %q want %q", got, want)
	}
}

func TestFindTelegramChannelURLCandidatesReturnsEmptyForNoMatches(t *testing.T) {
	got := findTelegramChannelURLCandidates("there is nothing useful here")

	if len(got) != 0 {
		t.Fatalf("unexpected candidates: got %q want empty", got)
	}
}

func TestFindTelegramChannelURLCandidatesCaseInsensitive(t *testing.T) {
	// Mixed-case input should be normalised to lowercase.
	got := findTelegramChannelURLCandidates("visit T.ME/TheStrikeMch for news")
	want := []string{"https://t.me/thestrikemch"}

	if !slices.Equal(got, want) {
		t.Fatalf("unexpected candidates: got %q want %q", got, want)
	}
}

func TestFindTelegramChannelURLCandidatesMaxLengthName(t *testing.T) {
	// 32-char slug: 1 letter + 30 middle chars + 1 alphanumeric = max allowed.
	slug := "a" + "bcdefghij012345678901234567890" + "z" // 32 chars
	got := findTelegramChannelURLCandidates("t.me/" + slug)
	want := []string{"https://t.me/" + slug}

	if !slices.Equal(got, want) {
		t.Fatalf("32-char slug should be accepted: got %q want %q", got, want)
	}
}
