package summarizer

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/responses"
)

const (
	baseMaxOutputTokens  int64 = 512
	limitMaxOutputTokens int64 = 2048

	systemPrompt = `Summarize the Telegram post in one ultra-short sentence.

Rules:
- ≤25 words (hard limit 40).
- Include only core idea and critical context (dates, numbers, names, calls to action).
- No lists, no examples — compress into one general statement.
- Neutral tone.
- Remove fillers, emojis, hashtags, links unless essential.
- Output exactly one line in the same language as the input.`
)

// OpenAISummarizer calls OpenAI's Responses API to produce summaries.
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
		return "", errors.New("input is empty")
	}

	userPromptBuilder := strings.Builder{}
	if sourceURL := strings.TrimSpace(input.SourceURL); sourceURL != "" {
		userPromptBuilder.WriteString("Source:\n")
		userPromptBuilder.WriteString(sourceURL)
		userPromptBuilder.WriteString("\n")
	}
	userPromptBuilder.WriteString("Content:\n")
	userPromptBuilder.WriteString(text)

	maxOutputTokens := baseMaxOutputTokens
	for {
		resp, err := s.client.Responses.New(ctx, responses.ResponseNewParams{
			Model:           openai.ChatModelGPT5Mini2025_08_07,
			ServiceTier:     responses.ResponseNewParamsServiceTierFlex,
			MaxOutputTokens: openai.Int(maxOutputTokens),
			Reasoning: responses.ReasoningParam{
				Effort: openai.ReasoningEffortLow,
			},
			Instructions: openai.String(systemPrompt),
			Input: responses.ResponseNewParamsInputUnion{
				OfString: openai.String(userPromptBuilder.String()),
			},
		})
		if err != nil {
			return "", fmt.Errorf("do request: %w", err)
		}

		if resp.Status == "incomplete" {
			if resp.IncompleteDetails.Reason == "max_output_tokens" && maxOutputTokens < limitMaxOutputTokens {
				maxOutputTokens *= 2
				if maxOutputTokens > limitMaxOutputTokens {
					maxOutputTokens = limitMaxOutputTokens
				}
				continue
			}
			return "", fmt.Errorf(
				"response is incomplete (reason = %s, maxOutputTokens = %d)",
				resp.IncompleteDetails.Reason,
				maxOutputTokens,
			)
		}

		summary := strings.TrimSpace(resp.OutputText())
		if summary == "" {
			return "", fmt.Errorf("output text is missing (status = %s)", resp.Status)
		}
		return summary, nil
	}
}
