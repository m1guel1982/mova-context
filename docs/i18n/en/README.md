# Mova Context

> **The universal context engine for AI: audit, protect, and visualize what actually reaches your LLM — local or cloud, in one command.**

Docs: **[Español](../es/README.md)** · **[English](README.md)**

---

```text
Sources  ──▶  Privacy Firewall (PII)  ──▶  Agents  ──▶  Auditable diagram
(data)          (Sanitizer + PII Masking)  (your logic)   (SVG / PNG / PDF)
```

- ✅ **Know exactly what data leaves your machine** — before it leaves, not after a leak.
- ✅ **One command generates a real diagram of your context architecture** — living auto-documentation, not a drawing someone made once and let go stale.
- ✅ **Works the same from CLI, Chat, MCP, or HTTP** — same engine, same result, zero integration friction.

---

## Index

1. [The diagram, in action](#1-the-diagram-in-action) — the artifact that summarizes the whole project
2. [Why Mova Context exists](#2-why-mova-context-exists) — the two problems it solves
3. [Quick installation & test (2 minutes)](#3-quick-installation--test-2-minutes)
4. [The four doors](#4-the-four-doors) — CLI, Chat, MCP, HTTP
5. [How it works](#5-how-it-works) — the complete map in one diagram
6. [The convention](#6-the-convention) — the 6 pieces everything depends on
7. [Token Firewall & Tokenomics](#7-token-firewall--tokenomics) — the protection and cost-control layer
8. [Job Engine, Cron & Multiagent](#8-job-engine-cron--multiagent)
9. [Visual interface — `mova ui`](#9-visual-interface--mova-ui)
10. [Do I need the CLI?](#10-do-i-need-the-cli) — decision table
11. [Go deeper](#11-go-deeper)

---

## 1. The diagram, in action — Cloud & Local in the same multi-agent group

This is the actual output of running `mova run <project> --diagram` in this repository — a customer-data audit scenario (Chile's Ley 21.719) featuring three parallel agents (`data-analyst`, `purpose-analyst`, and `ai-privacy-reviewer`).

The diagram demonstrates how Mova Context combines local and cloud-based models within a single execution pipeline:

![Multi-agent diagram example mixing Cloud and Local models](../assets/example-cloud-local.png)

- **Hybrid execution per agent:** While `data-analyst` and `purpose-analyst` run locally via Ollama (`llama3.2:3b`), `ai-privacy-reviewer` operates via Google Cloud (`gemini-3-flash-preview`).
- **Targeted PII protection:** The Token Firewall enables `PII Masking: true` specifically for the agent transmitting data externally (`ai-privacy-reviewer`), pseudonymizing sensitive details (`78/1694 token(s) pseudonymized`) before leaving the machine, whereas local agents keep `PII Masking: false`.
- **Detailed cost breakdown in `FINAL SUMMARY`:**
  - **Local agents:** Display `(local — no cost)`, confirming zero-billed on-premise execution.
  - **Cloud agent:** Computes exact API usage and costs (`ai-privacy-reviewer: 7503 tok, cheapest: google/gemini $0.0094`).
```bash
# Cloud
mova run ejemplo-ley21719-pii-context --diagram --export svg,png,pdf --path ./diagrams

# Local — same command, same project, only project.json changes
```

The example project ships both variants ready to test: `projects/ejemplo-ley21719-pii-context/ai-privacy-reviewer/project.json` (local, active by default) and `project_cloud.json` (the Cloud alternative) — switching between the two only requires renaming the active file to `project_local.json` and renaming `project_cloud.json` to `project.json`, without touching any other file or a single line of code.

**Read in three parts, in either diagram:**

- **`SOURCES` (what goes in):** the real files that build the context — a customer JSON, a PDF, a DOCX, a technical log. Every source is a real project file, never invented data.
- **`TOKEN FIREWALL` (what gets cleaned):** the Sanitizer removes repeated noise before counting a single token; PII Masking, when enabled, replaces personal-data-shaped tokens with deterministic pseudonyms (`[PII_a1b2c3d4]`) before anything leaves the machine.
- **`AGENTS` & `METRICS` (what gets delivered):** each agent shows its real model, whether it's local or cloud, and whether PII protection is on — and at the end, the summary: tokens before, tokens after, reduction percentage, and real cost (only if the model is billed; a local model never shows a cost figure).

One command. No extra code. The result is a file that can be attached to an audit ticket, shown in a compliance meeting, or committed to the repository itself as proof of what the system does, refreshed on every run.

---

## 2. Why Mova Context exists

Anyone integrating a language model into a real system eventually runs into two questions without a simple answer:

**a) What sensitive data is being sent to an external model (or even a local one)?**
Names, emails, ID numbers, histories — they often end up inside a prompt without anyone reviewing them one by one. There's no simple way to know, before sending, what personal-data-shaped information is traveling in that context.

**b) How do you document and show someone else how context flows through an AI system?**
A hand-drawn architecture diagram goes stale the following week. Explaining it in a meeting from configuration files is slow and unreliable.

Mova Context solves both with the same mechanism: a context pipeline that always passes through a sanitization and data-protection layer before reaching the model, and that can turn into an auditable image with a single command, no matter how large the project gets.

This doesn't replace a privacy policy or a legal review — it's a technical tool that makes visible and auditable what, in most AI systems today, stays hidden inside an API call.

---

## 3. Quick installation & test (2 minutes)

### Option A — Automatic installer (recommended)

1. Go to the `installers/` folder in the repository.
2. Run the installer for your operating system:

| OS | Installer |
|---|---|
| Windows | `install.bat` |
| macOS | `install.command` |
| Linux | `install.sh` |

The installer compiles the executable, installs it, and configures `PATH` automatically. Once done, from any terminal:

```bash
mova run pruebas-locales --diagram --export png
```

You can also open the interactive interface with `mova ui`, or just run `mova` to see available commands.

### Option B — Makefile (if you already have Go)

```bash
make install
mova run pruebas-locales
```

Or just build locally without installing:

```bash
make build
./dist/mova run pruebas-locales
```

### Option C — Manual build

```bash
go build -o mova ./src/cli
./mova run pruebas-locales
```

### Working with projects in any folder

A project doesn't need to live inside the Mova repository. `project.json`'s `"repo"` field accepts an absolute path to anywhere (another drive on Windows, another mount point on Linux/macOS) — double-click installers already configure everything for this to work with no extra steps. See [COMMANDS.md § Working across different drives/locations](COMMANDS.md#working-across-different-driveslocations-windowslinuxmacos).

---

## 4. The four doors

The same auditing and context-protection logic works exactly the same regardless of how you reach it — one engine behind four different ways to use it.

### MCP Server — for Cursor, VS Code, Claude Desktop

One configuration step. From then on, every context your editor builds for the model passes through Mova's privacy firewall first, in real time, while you code — without changing your usual workflow. See [COMMANDS.md § MCP Server](COMMANDS.md#13-mcp-server--mova-mcp-start).

### HTTP API — the ultralight Go middleware

Sits in front of calls to any LLM. No heavy dependencies, starts in milliseconds. It's the missing piece for a team that already has an AI pipeline in production and needs to audit it without rewriting it — one `POST` call is enough to sanitize and diagram a request.

```bash
curl -X POST http://localhost:3000/diagram -d '{"project": "my-project", "export": "svg"}'
```

### CLI — the terminal copilot

One command generates the architecture diagram for the context system. Drops straight into CI/CD: every commit can automatically leave behind an updated snapshot of what data flows where — auto-documentation that never goes stale because it regenerates itself.

```bash
mova run my-project --diagram --export png --path ./docs/diagrams
```

### Chat — the diagnostic console

Before taking a prompt to production, test it here: see the assembled context exactly as it would reach the model, check whether the privacy firewall flagged anything, tune parameters instantly — without leaving the terminal.

```bash
mova chat my-project
> /diagram
```

All four channels share the same assembly engine, the same privacy firewall, and the same diagram generator — only the door changes. See [COMMANDS.md § Visual diagrams](COMMANDS.md#21-visual-diagrams--mova-run-project---diagram) for a one-line example per channel.

---

## 5. How it works

```text
                     Mova Context

          agents/  skills/  prompts/
          project.json  memory.md
                 │
                 ▼
           workflow.md
        (the specification)
                 │
      ┌──────────┴──────────┐
      ▼                     ▼
An agent reading         mova run (CLI,
your repository           Chat, MCP, HTTP)
      │                     │
      └──────────┬──────────┘
                 ▼
      Token Firewall (Sanitizer,
      PII Masking, Cache Guard)
                 │
                 ▼
        Audited context
                 │
                 ▼
 Claude · GPT · Gemini · Ollama
      or any other LLM
```

---

## 6. The convention

**At its core, Mova Context is a file convention — not a mandatory tool.** Everything it needs fits into this structure, and works with nothing installed:

```text
workflow.md                       ← specification: how context gets built

agents/[domain]/                  ← who reasons (role, expertise)
skills/[domain]/                  ← what it knows (technical or business knowledge)
prompts/[domain]/                 ← what it must do (the task)

projects/[project]/
├── project.json                  ← which agents, skills, and prompts to use
└── memory.md                     ← the project's session history
```

With any agent that can read a repository (Claude Code, Cursor, Gemini CLI, Claude Desktop...), it's enough to ask:

```text
Read workflow.md, resolve project [name], run task [task], and build the context.
```

The agent follows `workflow.md`, resolves `project.json`, loads `agents`/`skills`/`prompts`, injects variables, adds `memory.md`, and assembles the final context. When working from a web chat that can't touch the repository (ChatGPT, Claude.ai, Gemini), **`mova run`** generates that exact same context, ready to copy and paste.

**If the `mova` binary disappeared tomorrow, the project would keep working exactly the same** — because the knowledge lives in the repository, never in the tool. The CLI only automates tasks: assembling context, auditing and protecting data, generating diagrams, managing memory, or exposing everything over HTTP/MCP.

This started from writing the same prompt a thousand times, from explaining a project to a model on Monday and explaining it again on Tuesday in a different chat, from switching providers and feeling a week of work vanish. A project's operational knowledge — conventions, business rules, decisions already made — shouldn't stay trapped inside a conversation:

```text
BEFORE                               MOVA CONTEXT

Context lives in the chat      →      Context lives in the repository
Switching models                →      Changing one line in project.json
  means starting over
Every person explains it        →      One single source of truth
  differently
Decisions get lost               →      memory.md keeps the history
No one knows what data left      →      The diagram always shows it
```

---

## 7. Token Firewall & Tokenomics

Beyond auditing and protecting data, Mova Context also controls — deterministically, with no AI involved — how much context gets sent and what it costs.

### The Token Firewall

Three automatic stages that run ahead of every execution (`mova run`, `mova chat`, jobs, MCP, HTTP, the TUI itself), with no extra command:

| Stage | What it does | Real, measured result (example included) |
|---|---|---|
| **Sanitizer** | Collapses repeated log lines, runs of blank lines, and duplicated file headers — before counting anything | 2,737 → 1,764 tokens: **35.6% fewer tokens, same information** |
| **PII Masking** (optional) | Replaces structurally personal-data-shaped tokens with deterministic pseudonyms, before counting or sending anything | See section 1 — the diagram itself shows how many tokens were protected |
| **Cache Layout Guard** | Reorders the prompt so its first tokens form a stable, byte-for-byte prefix — what actually triggers Claude/GPT/Gemini's prompt caching | ~1,050 of 1,167 static-prefix tokens estimated reusable on a cache hit |
| **Circuit Breaker** | Per-run and monthly USD limits, checked **before** anything is sent | A run that would exceed its limit never reaches the model |

Every stage is on by default (except PII Masking, which requires explicit per-project opt-in — see [PROJECT_JSON.md § Budget](PROJECT_JSON.md#budget-and-the-token-firewall)), can be turned off independently, and every number in the table is real, measured by actually running the example included in this repository. See [COMMANDS.md § Token Firewall](COMMANDS.md#20-token-firewall) for the full mechanics.

### The gate: `budget.max_tokens`

```json
{ "budget": { "max_tokens": 8000 } }
```

If the assembled context exceeds that limit, Mova stops execution before a single token goes out to the model — from `mova chat`, the `chat_completion` MCP tool, or HTTP, always the same. Cost control becomes an architectural rule, not a habit someone can forget.

### The report: `mova budget`, zero calls to any provider

`mova budget my-project` calculates, 100% on your own machine, how many tokens the real context would use and what it would cost on OpenAI, Anthropic, and Google, based on `config/prices.json`. It's an estimate computed with [tiktoken-go](https://github.com/tiktoken-go/tokenizer), cross-checked against manually maintained prices — it doesn't replace the real invoice, it's a compass for deciding what to optimize.

### The learning loop: every real call refines the estimate

Every time a real call reaches a provider, Mova compares the local estimate against the real count the API returns, and stores that deviation in `mova-token-history.json` — never the content, just two numbers per provider. Over time, the estimate calibrates specifically to each project: its language mix, its code, its documents. See [COMMANDS.md § mova budget](COMMANDS.md#15-tokenomics--mova-budget) for the full mechanics with real examples.

---

## 8. Job Engine, Cron & Multiagent

### Scheduling work: the Job Engine

A project can declare **jobs** in its `project.json` — scheduled, unattended runs that assemble context, audit data, save reports, update memory, and generate diagrams, all on a cron schedule:

```json
{
  "jobs": [
    {
      "schedule": "0 2 * * *",
      "tasks": ["audit-checkout", "audit-cookies"],
      "save": "reports/audit_{date}.pdf",
      "budget": { "focus": true },
      "memory": "Checkout and cookies audit completed"
    }
  ]
}
```

Runs on demand with `mova jobs run <project>`, or the daemon that checks every project once a minute starts with `mova jobs start`. See [PROJECT_JSON.md § Jobs](PROJECT_JSON.md#jobs) for every field.

`schedule` uses standard 5-field cron syntax (minute, hour, day of month, month, day of week) — for example, `"0 2 * * *"` runs daily at 2:00 AM, and `"*/15 * * * *"` runs every 15 minutes.

### Multiagent: several agents under one group

A directory under `projects/` can hold several independent agents, each an ordinary project, orchestrated by a parent `config.json`:

```text
projects/
    online_sales/
        config.json          ← the orchestrator
        salesperson/project.json
        customerCare/project.json
        support/project.json
```

```bash
mova agents run online_sales          # every agent, in sequence
mova agents run online_sales salesperson  # a single agent
mova run online_sales --diagram       # the whole group's diagram
```

Each agent keeps its own memory, budget, focus, tasks, jobs, and — as shown in section 1 — its own data-protection status. See [PROJECT_JSON.md § Multiagent](PROJECT_JSON.md#multiagent-agent-groups).

---

## 9. Visual interface — `mova ui`

Everything above can also be used from a simple, lightweight terminal interface:

```bash
mova ui
```

One command, navigated with arrows and Enter: chat, `project.json`, `workflow.md`, model configuration, memory, jobs, multiagent groups, search with exact file/line navigation, and diagrams — all from the same place, no new command per feature. The interface replaces nothing: it calls the exact same components `mova chat`, `mova jobs run`, and `mova run --diagram` already use. See [COMMANDS.md § Visual interface](COMMANDS.md#19-visual-interface--mova-ui).

---

## 10. Do I need the CLI?

| Situation | CLI? |
|---|---|
| Already using Claude Code, Cursor, or another agent that reads the repository | **No.** The agent follows `workflow.md` directly. |
| Want to paste the context into a web chat (Claude.ai, ChatGPT, Gemini) | **Yes.** `mova run` delivers it ready to copy. |
| Need to audit what personal data travels to the model | **Yes.** `mova run --diagram` shows it in one image. |
| Want to call a model's API from a script | **Yes.** Faster than making the model read every file. |
| Want to run a local model (Ollama) | **Yes.** `mova run ... \| ollama run model` in one line. |
| Want to expose context over HTTP or as an MCP server | **Yes.** `mova http` or `mova mcp start`. |

With or without the CLI, the source of truth never changes: `workflow.md`, `agents/`, `skills/`, `prompts/`, `project.json`, `memory.md`. Without the CLI, convenience is lost; with it, speed, automatic auditing, and data protection by default are gained — never the other way around.

---

## 11. Go deeper

| Looking for... | Document |
|---|---|
| Every command (memory, Focus, MCP, HTTP, diagrams, tokenomics) | [COMMANDS.md](COMMANDS.md) |
| The full specification models follow | [workflow.md](../../../workflow.md) |
| The source code (Resolvers, Adapters, how to extend it) | [SOURCE.md](../SOURCE.md) |

---

> **Operational knowledge belongs to the project. Reasoning belongs to the model.**
>
> Mova Context is the convention formed by `workflow.md`, `agents/`, `skills/`, `prompts/`, `project.json`, and `memory.md`. The CLI only automates working with that convention — it never replaces it.
