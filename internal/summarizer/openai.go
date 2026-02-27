package summarizer

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
)

const (
	maxCompletionTokens = 60

	systemPrompt = `Summarize the Telegram post in one ultra-short sentence.

Rules:
- ≤25 words (hard limit 40).
- Include only core idea and critical context (dates, numbers, names, calls to action).
- No lists, no examples — compress into one general statement.
- Neutral tone.
- Remove fillers, emojis, hashtags, links unless essential.
- Output exactly one line in the same language as the input.`
)

// OpenAISummarizer calls OpenAI's Chat Completions API to produce summaries.
type OpenAISummarizer struct {
	client openai.Client
}

// NewOpenAISummarizer builds a new summarizer instance.
func NewOpenAISummarizer(apiKey string) (*OpenAISummarizer, error) {
	return &OpenAISummarizer{
		client: openai.NewClient(option.WithAPIKey(apiKey)),
	}, nil
}

// Summarize produces a single summary suitable for a digest.
func (s *OpenAISummarizer) Summarize(
	ctx context.Context,
	input Input,
) (string, error) {
	text := strings.TrimSpace(input.Text)
	if text == "" {
		return "", errors.New("input is required")
	}

	messages := []openai.ChatCompletionMessageParamUnion{
		openai.SystemMessage(systemPrompt),
	}

	promptBuilder := strings.Builder{}
	if sourceURL := strings.TrimSpace(input.SourceURL); sourceURL != "" {
		promptBuilder.WriteString("Source: ")
		promptBuilder.WriteString(sourceURL)
		promptBuilder.WriteString("\n")
	}
	promptBuilder.WriteString("Content:\n")
	promptBuilder.WriteString(text)

	messages = append(messages, openai.UserMessage(promptBuilder.String()))

	resp, err := s.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model:               openai.ChatModelGPT5Mini2025_08_07,
		Messages:            messages,
		MaxCompletionTokens: openai.Int(maxCompletionTokens),
		ServiceTier:         openai.ChatCompletionNewParamsServiceTierFlex,
		ReasoningEffort:     openai.ReasoningEffortLow,
	})
	if err != nil {
		return "", fmt.Errorf("failed to do request: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", errors.New("chat completion choices are missing")
	}

	summary := strings.TrimSpace(resp.Choices[0].Message.Content)
	if summary == "" {
		return "", errors.New("chat completion choice message content is missing")
	}

	return summary, nil
}
