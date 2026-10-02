# Model configuration — `max_tokens` (reply cap)

Each model is described in `config/models/<provider>/<model>.json`.

## One standard field: `max_tokens`

`max_tokens` is the **reply** token cap and means the **same for every provider**; Mova translates it to each API's native parameter:

| Provider (`type`) | Native parameter sent | Default when undeclared |
|---|---|---|
| `openai` / OpenRouter / LM Studio / vLLM | `max_tokens` | 512 |
| `anthropic` | `max_tokens` | 1024 |
| `google` | `generationConfig.maxOutputTokens` | 2048 |
| `ollama` | `options.num_predict` | Ollama's own |

`num_predict` is still accepted as an **alias** (if both exist, `max_tokens` wins). Existing files using either work unchanged.


## Reasoning models

Reasoning consumes output tokens. Use a generous `max_tokens` (≥16384; DeepSeek via OpenRouter: 32768) or bound reasoning with the optional `reasoning` field, forwarded as-is to openai-compatible APIs that support it (OpenRouter):

```json
{ "type": "openai", "max_tokens": 32768, "reasoning": { "effort": "low" } }
```

