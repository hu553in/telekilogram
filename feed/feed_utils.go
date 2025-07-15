package feed

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/mmcdole/gofeed"

	"telekilogram/common"
	"telekilogram/model"
)

const MAX_MSG_LENGTH = 4096

var urlRegex = regexp.MustCompile(`https://[^\s]+`)

func FindValidFeedURLs(text string) []string {
	urls := extractURLs(text)
	var feedURLs []string

	for _, u := range urls {
		u = strings.TrimSpace(u)
		if isValidFeedURL(u) {
			feedURLs = append(feedURLs, u)
		}
	}

	return feedURLs
}

func GetFeedTitle(feedURL string) (string, error) {
	parser := &FeedParser{parser: gofeed.NewParser()}

	parsed, err := parser.parser.ParseURL(feedURL)
	if err != nil {
		return feedURL, err
	}

	if parsed.Title == "" {
		return feedURL, nil
	}
	return parsed.Title, nil
}

func FormatPostsAsMessages(posts []model.Post) []string {
	var messages []string
	var currentMessage strings.Builder

	currentMessage.WriteString("📰 *New posts*\n\n")
	headerLength := currentMessage.Len()

	feedGroups := make(map[string][]model.Post)
	feedURLs := make(map[string]string)

	for _, post := range posts {
		feedTitle := post.FeedTitle
		if feedTitle == "" {
			feedTitle = post.FeedURL
		}
		feedGroups[feedTitle] = append(feedGroups[feedTitle], post)
		feedURLs[feedTitle] = post.FeedURL
	}

	for feedTitle, feedPosts := range feedGroups {
		feedHeader := fmt.Sprintf(
			"📌 **[%s](%s)**\n\n",
			common.EscapeMarkdown(feedTitle),
			feedURLs[feedTitle],
		)

		if currentMessage.Len()+len(feedHeader) > MAX_MSG_LENGTH {
			messages = append(messages, currentMessage.String())
			currentMessage.Reset()
			currentMessage.WriteString("📰 *New posts \\(continue\\)*\n\n")
		}

		currentMessage.WriteString(feedHeader)

		for _, post := range feedPosts {
			bulletPoint := fmt.Sprintf(
				"– [%s](%s)\n",
				common.EscapeMarkdown(post.Title),
				post.URL,
			)

			if currentMessage.Len()+len(bulletPoint) > MAX_MSG_LENGTH {
				messages = append(messages, currentMessage.String())
				currentMessage.Reset()
				currentMessage.WriteString("📰 *New posts \\(continue\\)*\n\n")
				currentMessage.WriteString(feedHeader)
			}

			currentMessage.WriteString(bulletPoint)
		}

		currentMessage.WriteString("\n")
	}

	if currentMessage.Len() > headerLength {
		messages = append(messages, currentMessage.String())
	}

	return messages
}

func extractURLs(text string) []string {
	return urlRegex.FindAllString(text, -1)
}

func isValidFeedURL(feedURL string) bool {
	parsedURL, err := url.Parse(feedURL)
	if err != nil {
		return false
	}

	if parsedURL.Scheme != "https" {
		return false
	}

	fp := gofeed.NewParser()
	_, err = fp.ParseURL(feedURL)
	return err == nil
}
