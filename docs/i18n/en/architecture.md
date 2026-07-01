# Architecture — Mova Context

## Core principle

> **Operational knowledge belongs to the project. Reasoning belongs to the model.**

Mova Context does not dictate how the model reasons. It organizes what knowledge the model receives.

---

## The four design principles

Each principle lives in a two-line core file. They can be replaced freely.

### YAGNI → applied to Agents

*You Aren't Gonna Need It*

The agent makes no assumptions about future needs. It creates no abstractions, endpoints, or structures unless the task explicitly requests them.

File: `agents/base/i18n/en/yagni-core.md`

### KISS → applied to Skills

*Keep It Simple, Stupid*

Each skill solves one thing, in the most direct way possible, using what already exists before proposing something new.

File: `skills/base/i18n/en/kiss-dry-core.md`

### DRY → applied to Skills

*Don't Repeat Yourself*

A rule or instruction exists in one place. Everything else references it, not copies it.

### Occam's Razor → applied to Prompts

Among multiple valid solutions, the model chooses the simplest. No unnecessary explanatory prose.

File: `prompts/base/i18n/en/ockham-core.md`

---

## Replacing a principle

For a specific project (without affecting the global repo):

1. Create `agents/custom/i18n/en/my-rule.md` with your own rule
2. In `project.json`, add your file to agents and remove the original
3. The rest of the repo is unchanged

To change globally: edit the core file directly.

---

## Data flow

```text
User
  │
  └─→ Read workflow.md → [PROJECT] → [TASK]
          │
          └─→ Read project.json
                │
                ├─→ Resolve lang, llm, adapter
                │
                ├─→ Load agents (who the model is)
                ├─→ Load skills (what the model knows)
                ├─→ Load prompt (what the model must do)
                └─→ Read memory.md (history)
                          │
                          └─→ Inject {{VARIABLES}}
                                    │
                                    └─→ LLM executes
                                              │
                                              └─→ Update memory.md
```

---

## Separation of concerns

| Component | Responsibility |
|-----------|----------------|
| `project.json` | Source of truth. Configuration and orchestration |
| `workflow.md` | Orchestrator. Reads config and directs the flow |
| `agents/` | Model personality and constraints |
| `skills/` | Domain-specific knowledge |
| `prompts/` | Concrete task instructions |
| `memory.md` | Session history |

---

## Portability

The same project works with:

- Any LLM (Claude, GPT, Gemini, Ollama, local)
- Any adapter (files, PostgreSQL, MongoDB)
- Any language (en, es, fr, pt, and any other)
- Any domain (software, legal, callcenter, healthcare, etc.)

Only `project.json` changes. Knowledge stays the same.

---

## Extensibility

Add a language → create folder `i18n/{lang}/`

Add a domain → create folder `{domain}/i18n/{lang}/`

Add an adapter → implement in `adapters/{name}/`

Add a project → create `projects/{name}/project.json`

No change requires modifying the system core.
