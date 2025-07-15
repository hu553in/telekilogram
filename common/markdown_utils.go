package common

import "strings"

// taken from https://core.telegram.org/bots/api#markdownv2-style
var MARKDOWN_SPECIAL_CHARS = [18]string{
	"_", "*", "[", "]", "(", ")",
	"~", "`", ">", "#", "+", "-",
	"=", "|", "{", "}", ".", "!",
}

func EscapeMarkdown(input string) string {
	escaped := input
	for _, char := range MARKDOWN_SPECIAL_CHARS {
		escaped = strings.ReplaceAll(escaped, char, "\\"+char)
	}
	return escaped
}
