package openai

import (
	"fmt"
	"os"

	"github.com/tmc/langchaingo/llms/openai"
)

// DefaultModel is used when the agent does not specify one.
const DefaultModel = "gpt-4o"

// New creates an OpenAI LLM instance. The API key comes from the agent
// configuration and falls back to the OPENAI_API_KEY environment variable;
// it is passed to the client directly so that agents with different keys
// never interfere with each other. baseURL is optional.
func New(apiKey, model, baseURL string) (*openai.LLM, error) {
	if apiKey == "" {
		apiKey = os.Getenv("OPENAI_API_KEY")
	}
	if apiKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY is not set and no api_key is configured")
	}
	if model == "" {
		model = DefaultModel
	}
	opts := []openai.Option{openai.WithToken(apiKey), openai.WithModel(model)}
	if baseURL != "" {
		opts = append(opts, openai.WithBaseURL(baseURL))
	}
	llm, err := openai.New(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create OpenAI client: %w", err)
	}
	return llm, nil
}
