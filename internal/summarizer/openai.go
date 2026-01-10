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

	systemPrompt = `Summarize Telegram channel post in one ultra-short sentence
	using the fewest words that still convey the core point (aim for ≤25 words;
	never exceed 40). Stop as soon as the main idea is clear and the
	requirements below are met. Include only critical context (dates, numbers,
	names, calls to action). Do not enumerate lists or examples - condense
	them into one general statement. Stay neutral and objective. Strip
	fillers/emojis/hashtags/links unless essential. Output must be a single
	line in the same language as the input (you must check input language by
	reading the entire text, not first words).`
)

// OpenAIConfig contains configuration for the OpenAI-backed summarizer.
type OpenAIConfig struct {
	APIKey string
}

// OpenAISummarizer calls OpenAI's Chat Completions API to produce summaries.
type OpenAISummarizer struct {
	client openai.Client
}

// NewOpenAISummarizer builds a new summarizer instance.
func NewOpenAISummarizer(cfg OpenAIConfig) (*OpenAISummarizer, error) {
	return &OpenAISummarizer{
		client: openai.NewClient(option.WithAPIKey(cfg.APIKey)),
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
		Model:               openai.ChatModel("gpt-5.1"),
		Messages:            messages,
		MaxCompletionTokens: openai.Int(maxCompletionTokens),
		ServiceTier:         openai.ChatCompletionNewParamsServiceTierFlex,
		ReasoningEffort:     openai.ReasoningEffort("none"),
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
