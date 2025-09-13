package ollama

import (
	"fmt"

	"github.com/tmc/langchaingo/llms/ollama"
)

// New creates a new Ollama LLM instance with the specified model.
func New(model string) (*ollama.LLM, error) {
	// Ollama typically runs locally and does not require an API key,
	// but it does need a model name.
	if model == "" {
		return nil, fmt.Errorf("Ollama model name cannot be empty")
	}

	llm, err := ollama.New(ollama.WithModel(model))
	if err != nil {
		return nil, fmt.Errorf("failed to create Ollama client for model %s: %w", model, err)
	}

	return llm, nil
}
