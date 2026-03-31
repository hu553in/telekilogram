package summarizer

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"telekilogram/internal/config"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/responses"
)

// OpenAISummarizer calls OpenAI's Responses API to produce summaries.
type OpenAISummarizer struct {
	client openai.Client
	cfg    config.OpenAIConfig
}

// NewOpenAISummarizer builds a new summarizer instance.
func NewOpenAISummarizer(apiKey string, cfg config.OpenAIConfig) (*OpenAISummarizer, error) {
	return &OpenAISummarizer{
		client: openai.NewClient(option.WithAPIKey(apiKey)),
		cfg:    cfg,
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

	maxOutputTokens := s.cfg.BaseMaxOutputTokens
	for {
		resp, err := s.client.Responses.New(ctx, responses.ResponseNewParams{
			Model:           s.cfg.AIModel,
			ServiceTier:     responses.ResponseNewParamsServiceTier(s.cfg.ServiceTier),
			MaxOutputTokens: openai.Int(maxOutputTokens),
			Reasoning: responses.ReasoningParam{
				Effort: openai.ReasoningEffort(s.cfg.ReasoningEffort),
			},
			Instructions: openai.String(s.cfg.SystemPrompt),
			Input: responses.ResponseNewParamsInputUnion{
				OfString: openai.String(userPromptBuilder.String()),
			},
		})
		if err != nil {
			return "", fmt.Errorf("do request: %w", err)
		}

		if resp.Status == "incomplete" {
			if resp.IncompleteDetails.Reason == "max_output_tokens" && maxOutputTokens < s.cfg.LimitMaxOutputTokens {
				maxOutputTokens *= s.cfg.MaxOutputTokensGrowthFactor
				if maxOutputTokens > s.cfg.LimitMaxOutputTokens {
					maxOutputTokens = s.cfg.LimitMaxOutputTokens
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
