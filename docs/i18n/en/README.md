# Mova Context

> **Operational knowledge belongs to the project. Reasoning belongs to the model.**

Docs: **[English](README.md)** · **[Español](../es/README.md)**

---

# Philosophy

All project knowledge lives in version-controlled text files.

The CLI (`mova`) simply automates repetitive tasks.

If the `mova` executable disappeared tomorrow, your project would continue to work because the knowledge remains in the repository—not inside a tool.

---

## The Problem

When you use AI in a project, much of the operational knowledge—conventions, business rules, architectural decisions, and session history—ends up **trapped inside chat conversations**.

Over time, the same problems always appear:

- You have to explain the project again in every conversation.
- You switch models or providers and lose the context.
- Every team member explains the project differently.
- No one remembers exactly what was decided weeks ago.

Mova Context turns that operational knowledge into a first-class part of your repository.

Instead of living inside conversations with an LLM, it lives in version-controlled text files that any model can use without you having to explain everything again to Claude, GPT, Gemini, Ollama, or any other model.

---

## The Essentials

**Mova Context is a file convention, not a tool.**

Everything you need is this structure:

```text
workflow.md                       ← specification describing how to build the context

agents/[domain]/                  ← who reasons (role, expertise)
skills/[domain]/                  ← what the model knows (technical/business knowledge)
prompts/[domain]/                 ← what the model should do (task)

projects/[project]/
├── project.json                  ← which agents, skills and prompts to use
└── memory.md                     ← project session history
```

You can use all of this **without installing anything**.

You only need an AI agent capable of reading your repository (Claude Code, Cursor, Claude Desktop, Gemini CLI, etc.), or you can even copy the files manually into a chat.

For example:

```text
Read workflow.md, resolve project [name], execute task [task], and build the context.
```

An agent capable of accessing the repository can follow `workflow.md` to assemble the context automatically.

By following that specification, it:

- resolves the project defined in `project.json`
- loads the appropriate `agents`, `skills`, and `prompts`
- injects the required variables
- includes the project's memory
- builds the final context

If you're working from a web chat (ChatGPT, Claude.ai, or Gemini), where the model cannot access your repository directly, **`mova run`** generates that exact same context, ready to copy and paste.

**The CLI (`mova`) is not required for Mova Context to work. It simply automates context assembly and additional tasks such as memory management, HTTP, and MCP integration.**

---

## How It Works

```text
                     Mova Context

          agents/
          skills/
          prompts/
          project.json
          memory.md
                 │
                 ▼
           workflow.md
          (specification)
                 │
      ┌──────────┴──────────┐
      │                     │
      ▼                     ▼
Repository-aware       mova run (CLI)
AI agent               (optional)
      │                     │
      └──────────┬──────────┘
                 ▼
        Assembled Context
                 │
                 ▼
 Claude • GPT • Gemini • Ollama
      or any other LLM
```

---

## When Should You Use the CLI?

| Situation | Do You Need the CLI? |
|---|---|
| You already use Claude Code, Cursor, or another repository-aware AI agent | **No.** The agent follows `workflow.md` directly. |
| You want to paste the context into a web chat (Claude.ai, ChatGPT, Gemini) | **Yes.** `mova run` generates the assembled context, ready to copy and paste. |
| You want to call an LLM API from a script or automation | **Yes.** It's faster than having the model read all the files itself. |
| You want to run a local model (Ollama) | **Yes.** `mova run ... \| ollama run model` in a single command. |
| You want to save session memory without editing `memory.md` manually | **Yes.** `mova memory` updates the file automatically. |
| You want to expose the context through HTTP or as an MCP server | **Yes.** `mova http` or `mova mcp start`. |

**In short:**

Without the CLI, you lose convenience.

With the CLI, you gain speed, automation, and integrations.

The single source of truth is always:

- `workflow.md`
- `agents/`
- `skills/`
- `prompts/`
- `project.json`
- `memory.md`

Never the executable.

---

## Before vs Mova Context

```text
BEFORE                              MOVA CONTEXT

Context lives inside chats     →     Context lives inside the repository

Changing models means          →     Change a single line in project.json
starting over

Every developer explains       →     A single source of truth
the project differently

Important decisions            →     memory.md preserves project history
are lost

Knowledge depends              →     Knowledge belongs to the project,
on the provider                      not the provider
```

---

## Install the CLI (Optional)

```bash
go build -o mova ./src/cli
```

See **[COMMANDS.md](COMMANDS.md)** for all available commands (`run`, `memory`, `search`, `focus`, `mcp`, `http`, etc.).

---

## Minimal Example

A complete sample project is available at:

```text
projects/pruebas-locales/
```

You can inspect its `project.json` or generate the project context by running:

```bash
mova run pruebas-locales
```

---

## Learn More

| I want to... | Document |
|---|---|
| View all commands (Memory, Focus, MCP, HTTP included) | [COMMANDS.md](COMMANDS.md) |
| Read the complete specification followed by every model | [workflow.md](../../../workflow.md) |
| Understand the source code (Resolvers, Adapters, and extensibility) | [SOURCE.md](../SOURCE.md) |

---

> **Operational knowledge belongs to the project. Reasoning belongs to the model.**
>
> Mova Context is the convention formed by `workflow.md`, `agents/`, `skills/`, `prompts/`, `project.json`, and `memory.md`.
>
> The CLI simply automates working with that convention—it does not replace it.