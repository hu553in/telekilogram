package bot

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
	"telekilogram/internal/domain"
)

const telegramMessageMaxLength = 4096

type feedGroupKey struct {
	ID    int64
	title string
	URL   string
}

func (b *Bot) SendNewPosts(ctx context.Context, chatID int64, posts []domain.Post) error {
	if len(posts) == 0 {
		return nil
	}

	var errs []error
	messages := b.formatPostsAsMessages(ctx, posts)

	for _, message := range messages {
		if err := b.sendMessageWithKeyboard(chatID, message, b.returnKeyboard); err != nil {
			errs = append(errs, fmt.Errorf("send message with keyboard: %w", err))
		}
	}

	return errors.Join(errs...)
}

func (b *Bot) formatPostsAsMessages(ctx context.Context, posts []domain.Post) []string {
	var messages []string
	var currentMessage strings.Builder

	currentMessage.WriteString("📰 *New posts*\n\n")
	headerLength := currentMessage.Len()

	feedGroups := make(map[feedGroupKey][]domain.Post)

	for _, post := range posts {
		normalized, ok := b.normalizePost(ctx, post)
		if !ok {
			continue
		}

		key := feedGroupKey{
			ID:    normalized.FeedID,
			title: normalized.FeedTitle,
			URL:   normalized.FeedURL,
		}

		feedGroups[key] = append(feedGroups[key], normalized)
	}

	feedGroupKeySeq := maps.Keys(feedGroups)
	feedGroupKeys := slices.SortedFunc(
		feedGroupKeySeq,
		func(a, b feedGroupKey) int { return cmp.Compare(a.ID, b.ID) },
	)

	for _, key := range feedGroupKeys {
		feedPosts := feedGroups[key]

		feedHeader := fmt.Sprintf("📌 *[%s](%s)*\n\n", escapeMarkdownV2(key.title), key.URL)
		firstBulletPoint := fmt.Sprintf("– [%s](%s)\n\n", escapeMarkdownV2(feedPosts[0].Title), feedPosts[0].URL)

		if currentMessage.Len()+
			len(feedHeader)+
			len(firstBulletPoint) > telegramMessageMaxLength {
			messages = append(messages, currentMessage.String())
			currentMessage.Reset()
			currentMessage.WriteString("📰 *New posts \\(continue\\)*\n\n")
		}

		currentMessage.WriteString(feedHeader)

		for _, post := range feedPosts {
			bulletPoint := fmt.Sprintf("– [%s](%s)\n\n", escapeMarkdownV2(post.Title), post.URL)

			if currentMessage.Len()+len(bulletPoint) > telegramMessageMaxLength {
				messages = append(messages, currentMessage.String())
				currentMessage.Reset()
				currentMessage.WriteString("📰 *New posts \\(continue\\)*\n\n")
				currentMessage.WriteString(feedHeader)
			}

			currentMessage.WriteString(bulletPoint)
		}
	}

	if currentMessage.Len() > headerLength {
		messages = append(messages, currentMessage.String())
	}
	return messages
}

func (b *Bot) normalizePost(ctx context.Context, post domain.Post) (domain.Post, bool) {
	normalized := post

	normalized.Title = strings.TrimSpace(post.Title)
	normalized.URL = strings.TrimSpace(post.URL)
	normalized.FeedTitle = strings.TrimSpace(post.FeedTitle)
	normalized.FeedURL = strings.TrimSpace(post.FeedURL)

	switch {
	case normalized.FeedURL == "" && normalized.URL != "":
		normalized.FeedURL = normalized.URL
	case normalized.URL == "" && normalized.FeedURL != "":
		normalized.URL = normalized.FeedURL
	case normalized.URL == "" && normalized.FeedURL == "":
		b.log.WarnContext(ctx, "Skipping post with empty URLs",
			"feedID", post.FeedID,
			"title", normalized.Title)

		return domain.Post{}, false
	}

	if normalized.FeedTitle == "" {
		b.log.WarnContext(ctx, "Empty feed title",
			"feedID", post.FeedID,
			"feedURL", normalized.FeedURL,
			"postURL", normalized.URL)

		normalized.FeedTitle = normalized.FeedURL
		if normalized.FeedTitle == "" {
			normalized.FeedTitle = normalized.URL
		}
	}

	return normalized, true
}
