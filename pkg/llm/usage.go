package llm

import (
	"context"
	"sync"
)

// Usage counts the tokens of one or more LLM calls and their estimated cost.
type Usage struct {
	InputTokens  int     `json:"inputTokens"`
	OutputTokens int     `json:"outputTokens"`
	CostUSD      float64 `json:"costUsd"` // 0 when the model has no configured price
	Calls        int     `json:"calls"`
}

// Total is input plus output tokens.
func (u Usage) Total() int { return u.InputTokens + u.OutputTokens }

// Plus returns the sum of two usages.
func (u Usage) Plus(o Usage) Usage {
	return Usage{
		InputTokens:  u.InputTokens + o.InputTokens,
		OutputTokens: u.OutputTokens + o.OutputTokens,
		CostUSD:      u.CostUSD + o.CostUSD,
		Calls:        u.Calls + o.Calls,
	}
}

// usageFrom reads the token counts langchaingo reports per provider
// (PromptTokens/CompletionTokens for OpenAI and Ollama,
// InputTokens/OutputTokens for Anthropic) and prices them.
func usageFrom(info map[string]any, cfg Config) Usage {
	in := firstInt(info, "PromptTokens", "InputTokens")
	out := firstInt(info, "CompletionTokens", "OutputTokens")
	return Usage{
		InputTokens:  in,
		OutputTokens: out,
		CostUSD:      float64(in)*cfg.InputCostPerMillion/1e6 + float64(out)*cfg.OutputCostPerMillion/1e6,
		Calls:        1,
	}
}

func firstInt(m map[string]any, keys ...string) int {
	for _, k := range keys {
		switch v := m[k].(type) {
		case int:
			return v
		case int32:
			return int(v)
		case int64:
			return int(v)
		case float64:
			return int(v)
		}
	}
	return 0
}

// UsageRecorder accumulates the usage of every LLM call made with a context
// that carries it, including calls made inside agent functions.
type UsageRecorder struct {
	mu    sync.Mutex
	total Usage
}

// Add records a call.
func (r *UsageRecorder) Add(u Usage) {
	r.mu.Lock()
	r.total = r.total.Plus(u)
	r.mu.Unlock()
}

// Total returns everything recorded so far.
func (r *UsageRecorder) Total() Usage {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.total
}

type recorderKey struct{}

// WithUsageRecorder returns a context whose LLM calls are recorded by r.
func WithUsageRecorder(ctx context.Context, r *UsageRecorder) context.Context {
	return context.WithValue(ctx, recorderKey{}, r)
}

// RecorderFrom returns the recorder carried by ctx, or nil.
func RecorderFrom(ctx context.Context) *UsageRecorder {
	r, _ := ctx.Value(recorderKey{}).(*UsageRecorder)
	return r
}
