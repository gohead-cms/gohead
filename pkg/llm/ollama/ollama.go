package ollama

import (
	"fmt"

	"github.com/tmc/langchaingo/llms/ollama"
)

// New creates a new Ollama LLM instance with the specified model. serverURL
// is optional and defaults to the local Ollama endpoint.
func New(model, serverURL string) (*ollama.LLM, error) {
	// Ollama typically runs locally and does not require an API key,
	// but it does need a model name.
	if model == "" {
		return nil, fmt.Errorf("Ollama model name cannot be empty")
	}

	opts := []ollama.Option{ollama.WithModel(model)}
	if serverURL != "" {
		opts = append(opts, ollama.WithServerURL(serverURL))
	}
	llm, err := ollama.New(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create Ollama client for model %s: %w", model, err)
	}

	return llm, nil
}
