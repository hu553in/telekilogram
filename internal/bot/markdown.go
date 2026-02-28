package bot

import (
	"strings"
)

// Taken from https://core.telegram.org/bots/api#markdownv2-style.
const mdV2SpecialChars = `._[](){}#|!+-=*~>` + "`"

//nolint:gochecknoglobals // Lookup table meant to be immutable.
var mdV2Lookup = func() [256]bool {
	var m [256]bool
	for i := range len(mdV2SpecialChars) {
		m[mdV2SpecialChars[i]] = true
	}
	return m
}()

func escapeMarkdownV2(input string) string {
	charsToEscape := 0

	for i := range len(input) {
		if mdV2Lookup[input[i]] {
			charsToEscape++
		}
	}

	if charsToEscape == 0 {
		return input
	}

	var b strings.Builder
	b.Grow(len(input) + charsToEscape)

	for i := range len(input) {
		c := input[i]
		if mdV2Lookup[c] {
			b.WriteByte('\\')
		}
		b.WriteByte(c)
	}

	return b.String()
}
