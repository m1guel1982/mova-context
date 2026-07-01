# Mova Context

**A portable operational knowledge layer for AI-assisted projects.**

> **Operational knowledge belongs to the project. Reasoning belongs to the model.**

Works with any tool or model capable of reading text files.

---

## The problem

When a project uses AI, operational knowledge ends up scattered across chats, provider-specific configurations, isolated documents, and individual team members' memory.

Over time, familiar problems emerge:

- technical decisions that are hard to trace
- conventions explained repeatedly
- context rebuilt from scratch every session
- dependency on specific tools or providers

**Project context should live with the project.**

---

## The idea

Mova Context keeps operational knowledge in versionable files that travel alongside the repository.

```text
Project
│
├── Code
├── Conventions
├── Memory
├── Operational rules
└── Shared context
```

Models may change. Project context remains.

---

## What it is

An organizational convention for managing operational knowledge in AI-assisted projects.

It aims to enable:

- context reuse
- knowledge preservation
- team collaboration
- portability across tools
- decision traceability

---

## What it is not

- Not an AI framework
- Not a runtime
- Not an automation platform
- Not a replacement for Claude, GPT, Gemini, or any model

---

## Structure

```text
mova-context/
│
├── README.md                    ← language selector
├── workflow.md                  ← single entry point
│
├── docs/i18n/{es,en}/           ← bilingual documentation
│
├── agents/{domain}/i18n/{lang}/ ← who the model behaves as
├── skills/{domain}/i18n/{lang}/ ← what the model knows
├── prompts/{domain}/i18n/{lang}/← what the model must do
│
├── projects/{PROJECT}/
│   ├── project.json             ← source of truth
│   └── memory.md                ← session history
│
├── adapters/                    ← filesystem · postgresql · mongodb
├── schema/                      ← database schemas
├── cli/                         ← command-line tool
└── mcp/                         ← MCP integration
```

---

## project.json — source of truth

```json
{
  "project": "my-project",
  "description": "Project description",
  "repo": ".",
  "lang": "en",
  "adapter": "file",
  "llm": "claude",
  "default_task": "my-task",

  "variables": {
    "company": "Acme Corp"
  },
  "agents": { "domain": "software", "use": ["backend-dev"] },
  "skills": { "domain": "software", "use": ["api-security"] },

  "tasks": {
    "my-task": {
      "prompt": "review-project",
      "variables": { "module": "auth" }
    }
  }
}
```

One file controls everything. Everything else is Markdown.

---

## Supported LLMs

The project works identically with any model. Only `project.json` changes.

| Field | Value |
|-------|-------|
| `"llm": "claude"` | Anthropic Claude |
| `"llm": "gpt"` | OpenAI GPT |
| `"llm": "gemini"` | Google Gemini |
| `"llm": "ollama"` | Ollama (local) |
| `"llm": "openrouter"` | OpenRouter (multi-model) |

Code, agents, skills, and prompts never change.

---

## Adapters

| Adapter | Description |
|---------|-------------|
| `"adapter": "file"` | Markdown files (default) |
| `"adapter": "postgresql"` | PostgreSQL database |
| `"adapter": "mongodb"` | MongoDB database |

Only the adapter changes. The workflow stays the same.

---

## CLI

```bash
mova list                                    # list available projects
mova run [project] [task]                    # generate context
mova memory [project] "llm response"         # update memory
mova init [name]                             # create new project
mova mcp start                               # start MCP server
```

---

## Official examples

- [Chilean Privacy Law 21.719](../../examples/i18n/en/privacy-law/)
- [Enterprise Omnichannel](../../examples/i18n/en/omnichannel/)

---

## Documentation

- [workflow.md](workflow.md) — full operational guide
- [architecture.md](architecture.md) — philosophy and principles
- [cli.md](cli.md) — CLI reference
- [mcp.md](mcp.md) — MCP integration
- [adapters.md](adapters.md) — storage adapters
- [schema.md](schema.md) — database schema

---

## Core principle

> **Operational knowledge belongs to the project. Reasoning belongs to the model.**

Models will change. Providers will change. Tools will change.

The accumulated knowledge of the project should remain under the control of the team that builds it.
