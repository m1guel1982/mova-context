# Mova Context

> **Operational knowledge belongs to the project. Reasoning belongs to the model.**

Docs: **[Español](README.md)** · **[English](README.en.md)**

---

## Index

1. [The convention](#1-the-convention) — the 6 pieces everything depends on
2. [Why Mova Context exists](#2-why-mova-context-exists) — the short story
3. [How it works](#3-how-it-works) — the whole picture in one diagram
4. [Do I need the CLI?](#4-do-i-need-the-cli) — a quick decision table
5. [Tokenomics](#5-tokenomics--the-main-course) — why every token counts, and how Mova controls it
6. [What the CLI brings](#6-what-the-cli-brings) — a summary of what's new
7. [Try it in 2 minutes](#7-try-it-in-2-minutes)
8. [Go deeper](#8-go-deeper)

---

## 1. The convention

**Mova Context is a file convention, not a tool.** Everything you need fits in this structure, and it works with zero installation:

```text
workflow.md                       ← the spec: how the context gets built

agents/[domain]/                  ← who reasons (role, experience)
skills/[domain]/                  ← what it knows (technical or business knowledge)
prompts/[domain]/                 ← what it must do (the task)

projects/[project]/
├── project.json                  ← which agents, skills and prompts to use
└── memory.md                     ← the project's session history
```

With any agent that can read your repository (Claude Code, Cursor, Gemini CLI, Claude Desktop...), all you need is:

```text
Read workflow.md, resolve the project [name], run task [task] and build the context.
```

The agent follows `workflow.md`, resolves `project.json`, loads `agents`/`skills`/`prompts`, injects variables, adds `memory.md`, and assembles the final context. If you're working from a web chat that can't touch your repo (ChatGPT, Claude.ai, Gemini), **`mova run`** produces that exact same context, ready to copy and paste.

**If the `mova` binary vanished tomorrow, the project would keep working exactly the same** — because the knowledge lives in the repository, never in the tool. The CLI only automates tasks: assembling context, managing memory, or exposing it over HTTP/MCP.

---

## 2. Why Mova Context exists

This came from writing the same prompt a thousand times. From re-explaining the project to a model on Monday, and again on Tuesday just because I opened a new chat. From switching from GPT to Claude and feeling a week of context evaporate. At some point I got tired and needed order.

A project's operational knowledge — conventions, business rules, decisions already made, memory of work done — ends up **trapped inside the chat**. And over time the same symptoms always show up:

```text
BEFORE                               MOVA CONTEXT

Context lives in the chat      →      Context lives in the repository
Switching models means          →      Change one line in project.json
  starting over
Every dev explains it           →      A single source of truth
  differently
Decisions get lost              →      memory.md keeps the history
Knowledge depends on            →      Knowledge belongs to the project,
  the provider                          not the provider
```

This isn't magic or an overblown promise: it's simply moving that knowledge from the conversation into the repository, so any model — Claude, GPT, Gemini, Ollama — can read it without you having to explain it all over again.

---

## 3. How it works

```text
                     Mova Context

          agents/  skills/  prompts/
          project.json  memory.md
                 │
                 ▼
           workflow.md
          (the spec)
                 │
      ┌──────────┴──────────┐
      ▼                     ▼
An agent reading         mova run (CLI,
your repository            optional)
      │                     │
      └──────────┬──────────┘
                 ▼
        Assembled context
                 │
                 ▼
 Claude · GPT · Gemini · Ollama
        or any other LLM
```

---

## 4. Do I need the CLI?

| Situation | Need the CLI? |
|---|---|
| You already use Claude Code, Cursor, or an agent that reads the repo | **No.** The agent follows `workflow.md` directly. |
| You want to paste the context into a web chat (Claude.ai, ChatGPT, Gemini) | **Yes.** `mova run` gives it to you ready to paste. |
| You want to call a model's API from a script | **Yes.** Faster than having the model read every file. |
| You want to run a local model (Ollama) | **Yes.** `mova run ... \| ollama run model` in one line. |
| You want to save a session's memory without editing `memory.md` by hand | **Yes.** `mova memory` does it for you. |
| You want to expose the context over HTTP or as an MCP server | **Yes.** `mova http` or `mova mcp start`. |

With or without the CLI, the source of truth never changes: `workflow.md`, `agents/`, `skills/`, `prompts/`, `project.json`, `memory.md`. Skip the CLI and you lose convenience; use it and you gain speed and automation — never the other way around.

---

## 5. Tokenomics — the main course

Every token you send to a model costs something: money if it's Cloud, or accuracy if it's local and the context doesn't fit its window. Mova Context doesn't promise magic — it gives you **real control, before you spend anything**.

### The gate: `budget.max_tokens` stops the run, not just warns

```json
// project.json
{ "budget": { "max_tokens": 8000 } }
```

If the assembled context exceeds that limit, **Mova stops execution before a single token reaches the model** — from `mova chat`, the `chat_completion` MCP tool, or HTTP, always the same way:

```text
ERROR
Current context (14,250 tokens) exceeds the configured limit (8,000).
Suggestion: Use --focus to reduce the included files.
```

This turns cost control into an architectural rule, not a habit someone can forget.

### The report: `mova budget`, zero calls to any provider

`mova budget my-project` calculates, **100% on your machine**, how many tokens the real context (agents + skills + prompt + focus + memory) would use, and how much it would cost across OpenAI, Anthropic, and Google, based on the prices you configure yourself in `config/prices.json`. The report breaks the cost down piece by piece so you know exactly what to trim first:

```text
mova budget my-project my-task --focus
→ mova-budget-report.md
```

**To be clear:** this is an estimate computed with [tiktoken-go](https://github.com/tiktoken-go/tokenizer) (OpenAI's tokenizer), cross-referenced against manual prices. It doesn't replace the real invoice — it's a compass for deciding what to optimize, and the report itself says so three times.

### The learning loop: every real call sharpens the estimate

When `mova chat` or `chat_completion` call a real Cloud provider, that provider reports back how many tokens it actually counted. Mova stores those two numbers — local estimate vs. real — in `mova-token-history.json`, never the content or the prompts:

```json
{ "anthropic": { "total_local_tokens": 120000, "total_api_tokens": 122760 } }
```

Over time, `mova budget` shows you how accurate your estimator is for **that specific project** — your own calibration, not a generic benchmark.

### The automatic cleanup: deduplication across the whole context

If a paragraph got copy-pasted into two agents, or repeats between a skill and a prompt, Mova detects it (identical text only, never code/SQL/JSON) and keeps it just once:

```text
[Dedup] Removed 3 duplicated paragraphs (~450 tokens saved).
```

### In one sentence

In the cloud, excess context means a bigger bill. Locally, excess context means the model silently truncates or degrades. Same problem, two different costs — and Mova applies the same control mechanism to both: measure before spending, stop if it's over, and learn from every real call.

More detail and full examples in [COMMANDS.en.md § mova budget](COMMANDS.en.md#mova-budget--local-token-and-cost-estimation).

---

## 6. What the CLI brings

- **`/save`** — one single command to create or edit any file or folder from the chat. `/save "report.docx"` saves the model's last reply there; the real format (`.md`, `.docx`, `.pdf`, `.xlsx`, `.svg`, source code, ~20 more extensions) is chosen purely by the extension. Same behavior via MCP (`save`) and HTTP (`POST /save`).
- **Natural language in chat** — type *"generate the report at docs/output.pdf"* and it just happens, no commands needed. See [COMMANDS.en.md § natural language](COMMANDS.en.md#creating-files-by-talking-natural-language-in-chat).
- **Office documents and media** — real `.docx`, `.xlsx`, `.pdf` (no external dependencies, just Go's standard library), native SVG, and images via a local diffusion model.
- **`mova chat`** — talk to Ollama, LM Studio, vLLM, OpenAI, Anthropic, or Google from the terminal, with the same context always injected as the system prompt.
- **`mova budget`** — see section 5.

---

## 7. Try it in 2 minutes

```bash
go build -o mova ./src/cli
mova run pruebas-locales
```

There's a full example project at `projects/pruebas-locales/` — inspect its `project.json` or run the command above to see the assembled context.

---

## 8. Go deeper

| I want to... | Document |
|---|---|
| See every command (memory, Focus, MCP, HTTP, tokenomics) | [COMMANDS.en.md](COMMANDS.en.md) |
| Read the full spec that models follow | [workflow.md](../../../workflow.md) |
| Understand the source code (Resolvers, Adapters, how to extend it) | [SOURCE.md](../SOURCE.md) |

---

> **Operational knowledge belongs to the project. Reasoning belongs to the model.**
>
> Mova Context is the convention formed by `workflow.md`, `agents/`, `skills/`, `prompts/`, `project.json`, and `memory.md`. The CLI just automates work on top of that convention — it never replaces it.
