package summarizer

import (
	"context"
	"fmt"
	"strings"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
)

const (
	temperature         = 0.2
	maxCompletionTokens = 150

	systemPrompt = `Summarize Telegram channel post
	in a single mid-length sentence,
	covering only the main idea so the user understands it
	and can decide whether to open the full post.
	Keep only critical context (dates, numbers, names, calls to action).
	Do not enumerate any lists or examples -
	summarize them as a single general statement.
	Remove fillers/emojis/hashtags/links unless essential.
	Stay neutral and objective.
	The output must be in one line of a single mid-length sentence,
	in the same language as the input
	(you must check input language by reading the entire text, not first words).
	Max sentence length is 500 characters (but try to keep it much shorter).`
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
		client: openai.NewClient(
			option.WithAPIKey(cfg.APIKey),
		),
	}, nil
}

// Summarize produces a single summary suitable for a digest.
func (s *OpenAISummarizer) Summarize(
	ctx context.Context,
	input Input,
) (string, error) {
	text := strings.TrimSpace(input.Text)
	if text == "" {
		return "", fmt.Errorf("input is required")
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
		Model:               openai.ChatModelGPT4_1Mini,
		Messages:            messages,
		Temperature:         openai.Float(temperature),
		MaxCompletionTokens: openai.Int(maxCompletionTokens),
	})
	if err != nil {
		return "", fmt.Errorf("failed to do request: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("chat completion choices are missing")
	}

	summary := strings.TrimSpace(resp.Choices[0].Message.Content)
	if summary == "" {
		return "", fmt.Errorf("chat completion choice message content is missing")
	}

	return summary, nil
}
