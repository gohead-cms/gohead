package anthropic

import (
	"fmt"
	"os"

	"github.com/tmc/langchaingo/llms/anthropic"
)

// New creates a new Anthropic LLM instance.
// It relies on the ANTHROPIC_API_KEY environment variable.
func New() (*anthropic.LLM, error) {
	if os.Getenv("ANTHROPIC_API_KEY") == "" {
		return nil, fmt.Errorf("ANTHROPIC_API_KEY environment variable not set")
	}

	// The anthropic.New() function automatically reads the API key from the environment variable.
	llm, err := anthropic.New()
	if err != nil {
		return nil, fmt.Errorf("failed to create Anthropic client: %w", err)
	}

	return llm, nil
}
