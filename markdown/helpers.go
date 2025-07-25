package markdown

import "strings"

// taken from https://core.telegram.org/bots/api#markdownv2-style
var markdownSpecialChars = []string{
	"_", "*", "[", "]", "(", ")",
	"~", "`", ">", "#", "+", "-",
	"=", "|", "{", "}", ".", "!",
}

func EscapeV2(input string) string {
	escaped := input
	for _, char := range markdownSpecialChars {
		escaped = strings.ReplaceAll(escaped, char, "\\"+char)
	}

	return escaped
}
