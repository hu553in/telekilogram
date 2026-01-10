package summarizer

import (
	"context"
)

// Input describes the payload for a summary request.
type Input struct {
	// Text contains the original plain text to summarise.
	Text string
	// SourceURL is optional metadata that helps the model reference the origin.
	SourceURL string
}

// Summarizer produces a single summary for a given input text.
type Summarizer interface {
	Summarize(ctx context.Context, input Input) (string, error)
}
