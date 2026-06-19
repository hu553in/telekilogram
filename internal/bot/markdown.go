package bot

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/go-telegram/bot"
)

const telegramMarkdownLinkMaxLength = 512

func formatMarkdownLink(title string, url string) string {
	url = strings.TrimSpace(url)
	title = normalizeMarkdownLinkTitle(title)
	if title == "" {
		title = normalizeMarkdownLinkTitle(url)
	}
	if utf8.RuneCountInString(title) > telegramMarkdownLinkMaxLength {
		title = string([]rune(title)[:telegramMarkdownLinkMaxLength-3]) + "..."
	}

	escapedTitle := bot.EscapeMarkdownUnescaped(title)

	if url == "" {
		return escapedTitle
	}

	return fmt.Sprintf("[%s](%s)", escapedTitle, escapeMarkdownLinkURL(url))
}

func normalizeMarkdownLinkTitle(title string) string {
	return strings.Join(strings.Fields(title), " ")
}

func escapeMarkdownLinkURL(url string) string {
	var escaped strings.Builder

	for _, r := range url {
		// MarkdownV2 link destinations only need these two characters escaped.
		if r == '\\' || r == ')' {
			escaped.WriteRune('\\')
		}
		escaped.WriteRune(r)
	}

	return escaped.String()
}
