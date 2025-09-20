package openai

import (
	"fmt"
	"os"

	"github.com/tmc/langchaingo/llms/openai"
)

// New creates a new OpenAI LLM instance.
// It relies on the OPENAI_API_KEY environment variable.
func New() (*openai.LLM, error) {
	if os.Getenv("OPENAI_API_KEY") == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY environment variable not set")
	}

	// The openai.New() function automatically reads the API key from the environment variable.
	llm, err := openai.New(openai.WithModel("gpt-4o"))
	if err != nil {
		return nil, fmt.Errorf("failed to create OpenAI client: %w", err)
	}

	return llm, nil
}
