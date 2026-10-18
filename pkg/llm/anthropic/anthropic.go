package anthropic

import (
	"fmt"
	"os"

	"github.com/tmc/langchaingo/llms/anthropic"
)

// New creates an Anthropic LLM instance. The API key comes from the agent
// configuration and falls back to the ANTHROPIC_API_KEY environment variable;
// it is passed to the client directly so that agents with different keys
// never interfere with each other. model and baseURL are optional.
func New(apiKey, model, baseURL string) (*anthropic.LLM, error) {
	if apiKey == "" {
		apiKey = os.Getenv("ANTHROPIC_API_KEY")
	}
	if apiKey == "" {
		return nil, fmt.Errorf("ANTHROPIC_API_KEY is not set and no api_key is configured")
	}
	opts := []anthropic.Option{anthropic.WithToken(apiKey)}
	if model != "" {
		opts = append(opts, anthropic.WithModel(model))
	}
	if baseURL != "" {
		opts = append(opts, anthropic.WithBaseURL(baseURL))
	}
	llm, err := anthropic.New(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create Anthropic client: %w", err)
	}
	return llm, nil
}
