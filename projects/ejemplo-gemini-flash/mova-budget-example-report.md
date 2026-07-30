# Mova Budget Report

## Project

Project name: ejemplo-gemini-flash

Task: revisar-backend

## Tokenization

Tool used: tiktoken-go (encoding: cl100k_base)

Token counts are a local estimate computed with tiktoken-go — the same open-source library OpenAI itself publishes. Every calculation in this report runs on this machine: nothing in your project is sent anywhere to produce it.

For OpenAI models, this count is typically an exact match to what the OpenAI API bills you for. For Claude and Gemini, no official local tokenizer is publicly available, so this report reuses the same encoding as a close approximation — real counts from those providers are usually very close, but can differ, especially for non-English text or dense code. See "Historical Token Accuracy" below for this project's own measured difference.

## Token & Cost Breakdown

This is where your token budget is actually going — one row per piece of the context declared in project.json (agents, skills, prompt, focus, memory), plus fixed engine overhead. Use it to see, in both tokens and dollars, exactly what is worth trimming first.

| Component | Tokens | anthropic claude (USD) | google gemini (USD) | openai gpt-5 (USD) |
|---|---|---|---|---|
| Agents | 0 | $0.0000 | $0.0000 | $0.0000 |
| Skills | 0 | $0.0000 | $0.0000 | $0.0000 |
| Prompt | 670 | $0.0020 | $0.0008 | $0.0034 |
| Focus | 321 | $0.0010 | $0.0004 | $0.0016 |
| Memory | 0 | $0.0000 | $0.0000 | $0.0000 |
| Engine overhead | 216 | $0.0006 | $0.0003 | $0.0011 |
| **TOTAL** | **1207** | **$0.0036** | **$0.0015** | **$0.0060** |

Approximate total in CLP (exchange rate 950): $3 CLP (using anthropic claude)

## Budget Limit

Configured limit: 5000 tokens

Current context: 1207 tokens

Usage: 24.1% of the configured limit

Headroom left: 3793 tokens

Status: within budget.

### Why token counts differ between providers

This report uses one encoding (cl100k_base, via tiktoken-go) for every provider, because Claude and Gemini do not publish a local tokenizer. In practice:

- OpenAI/GPT: this estimate is normally an exact or near-exact match, since tiktoken-go is OpenAI's own tokenizer.
- Google Gemini: typically close, small differences come from how Gemini splits certain punctuation, code, and non-English text.
- Anthropic Claude: usually the largest gap, since Claude uses its own (unpublished) tokenizer, which tends to segment text somewhat differently.

See "Historical Token Accuracy" below — once real API calls have been recorded for this project, the deviation for each provider is measured directly instead of assumed.

## Historical Token Accuracy

Mova Budget compares local token estimation (tiktoken-go) with real cloud API usage collected from this project.

| Provider | Average deviation |
|---|---|
| anthropic | No historical data |
| google | -0.3% |
| openai | No historical data |

Historical accuracy is automatically calibrated using previous cloud requests from this project. Actual costs may vary depending on provider billing policies and tokenizer updates.

## Important

These are estimates based on token counts computed locally and on the prices configured in config/prices.json. Real costs can vary depending on provider, model, caching, discounts, and commercial policies. This report is a tool to help you see where your token budget goes and what to optimize — it does not replace checking your actual invoice with each provider, and it is not a guarantee of exact pricing.
