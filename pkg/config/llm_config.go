package config

import (
	"fmt"
	"os"
	"strings"
)

// ApplyLLMEnv ensures provider libraries that rely on env variables work
// even when credentials are supplied via Viper/Config.
//
// Precedence:
// - If cfg contains an API key, export the provider-specific env.
// - Else, require that the provider env is already set.
func ApplyLLMEnv(llm LLMConfig) error {
	switch strings.ToLower(llm.Provider) {
	case "openai":
		if llm.APIKey != "" {
			if err := os.Setenv("OPENAI_API_KEY", llm.APIKey); err != nil {
				return fmt.Errorf("failed to set OPENAI_API_KEY: %w", err)
			}
			return nil
		}
		return fmt.Errorf("OPENAI_API_KEY is not set")
	case "anthropic":
		if llm.APIKey != "" {
			if err := os.Setenv("ANTHROPIC_API_KEY", llm.APIKey); err != nil {
				return fmt.Errorf("failed to set ANTHROPIC_API_KEY: %w", err)
			}
			return nil
		}
		return fmt.Errorf("ANTHROPIC_API_KEY is not set")
	case "ollama":
		// No API key required.
		return nil

	default:
		return fmt.Errorf("unsupported LLM provider: %s", llm.Provider)
	}
}
