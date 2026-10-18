package llm

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUsageFrom(t *testing.T) {
	priced := Config{InputCostPerMillion: 3, OutputCostPerMillion: 15}
	openai := usageFrom(map[string]any{"PromptTokens": 1000, "CompletionTokens": 200, "TotalTokens": 1200}, priced)
	assert.Equal(t, Usage{InputTokens: 1000, OutputTokens: 200, CostUSD: 0.003 + 0.003, Calls: 1}, openai)

	anthropic := usageFrom(map[string]any{"InputTokens": 500, "OutputTokens": 100}, Config{})
	assert.Equal(t, 500, anthropic.InputTokens)
	assert.Equal(t, 100, anthropic.OutputTokens)
	assert.Zero(t, anthropic.CostUSD, "no price, no cost")

	assert.Equal(t, Usage{Calls: 1}, usageFrom(nil, priced), "providers that report nothing")
}

func TestUsageRecorder(t *testing.T) {
	rec := &UsageRecorder{}
	ctx := WithUsageRecorder(context.Background(), rec)
	RecorderFrom(ctx).Add(Usage{InputTokens: 1, OutputTokens: 2, CostUSD: 0.5, Calls: 1})
	RecorderFrom(ctx).Add(Usage{InputTokens: 3, OutputTokens: 4, Calls: 1})
	assert.Equal(t, Usage{InputTokens: 4, OutputTokens: 6, CostUSD: 0.5, Calls: 2}, rec.Total())
	assert.Equal(t, 10, rec.Total().Total())
	assert.Nil(t, RecorderFrom(context.Background()))
}

func TestPricesFollowTheDefaultModel(t *testing.T) {
	SetDefaultConfig(Config{Provider: "openai", Model: "m1", InputCostPerMillion: 1, OutputCostPerMillion: 2})
	defer SetDefaultConfig(Config{})
	assert.Equal(t, 1.0, withDefaults(Config{}).InputCostPerMillion, "helper functions use the default model and its prices")
	assert.Zero(t, withDefaults(Config{Provider: "openai", Model: "other"}).InputCostPerMillion, "another model does not inherit them")
	assert.Equal(t, 9.0, withDefaults(Config{Provider: "openai", Model: "m1", InputCostPerMillion: 9}).InputCostPerMillion)
}
