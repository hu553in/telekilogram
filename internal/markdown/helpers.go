package markdown

import "strings"

// Taken from https://core.telegram.org/bots/api#markdownv2-style.
const mdV2SpecialChars = `._[](){}#|!+-=*~>` + "`"

func EscapeV2(input string) string {
	lookup := mdV2SpecialCharLookup()
	charsToEscape := 0

	for i := range input {
		if lookup[input[i]] {
			charsToEscape++
		}
	}
	if charsToEscape == 0 {
		return input
	}

	var b strings.Builder
	b.Grow(len(input) + charsToEscape)

	for i := range input {
		c := input[i]
		if lookup[c] {
			b.WriteByte('\\')
		}
		b.WriteByte(c)
	}

	return b.String()
}

func mdV2SpecialCharLookup() [256]bool {
	var m [256]bool
	for _, c := range []byte(mdV2SpecialChars) {
		m[c] = true
	}
	return m
}
