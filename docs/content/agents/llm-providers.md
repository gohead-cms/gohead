# LLM providers

| `provider` | Models | Key | `base_url` |
|------------|--------|-----|------------|
| `openai` | any chat model (default `gpt-4o`) | `api_key` or `OPENAI_API_KEY` | any OpenAI-compatible endpoint (Azure OpenAI, vLLM, LiteLLM, …) |
| `anthropic` | any Claude model | `api_key` or `ANTHROPIC_API_KEY` | optional proxy URL |
| `ollama` | a local model, `model` required | none | Ollama server, default `http://localhost:11434` |

## Per-agent configuration

```json
"llm_config": {
  "provider": "anthropic",
  "model": "claude-sonnet-4-5",
  "api_key": "sk-ant-…"
}
```

Keys are passed to the client of that agent only, so agents with different keys or providers
can run side by side in the same worker. Stored keys are masked in API responses.

## Instance defaults

The [`llm`](../getting-started/configuration.md#llm-defaults) block of the configuration
fills whatever an agent leaves empty:

```yaml
llm:
  provider: openai
  model: gpt-4o-mini
  api_key: ""          # or OPENAI_API_KEY
  base_url: ""
  input_cost_per_million: 0.15   # USD, optional: used to compute run costs
  output_cost_per_million: 0.60
```

- An agent whose `provider` matches the default inherits the missing `model`, `api_key` and
  `base_url`, and the default prices if it also uses the default model.
- An agent on another provider inherits nothing, so keys never leak across providers.
- The `llm.*` [helper functions](functions.md#llm-helper-functions) always use these defaults.

## Ollama example

```json
"llm_config": { "provider": "ollama", "model": "llama3.1", "base_url": "http://ollama:11434" }
```

Tool calling needs a model that supports it (Llama 3.1+, Qwen 2.5, Mistral…).

## Costs and limits

Each run makes up to `max_turns` LLM calls, and replays at most the last 40 messages of the
agent's history. Event-driven agents on busy collections can be expensive: use cheaper models
for classification and tagging, and narrow `events` to what you need.

Every run records the tokens the provider reports, including calls made by `llm.*` helper
functions. GoHead ships no price list: set `input_cost_per_million` and
`output_cost_per_million` in `llm_config` (or in the defaults) to see costs, and use
[`limits`](runs-and-limits.md#limits) to cap runs, tokens and spend per agent.
