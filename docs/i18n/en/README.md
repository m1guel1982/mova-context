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
7. [Quick Installation & Test (2 Minutes)](#7-quick-installation--test-2-minutes)
8. [Go deeper](#8-go-deeper)
9. [Job Engine, Cron & Multiagent](#9-job-engine-cron--multiagent)
10. [Visual interface — `mova ui`](#10-visual-interface--mova-ui)

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

## 5. Tokenomics 

### The analogy: the airport scale

Before a flight, you weigh your suitcase at home on a bathroom scale. It gives you a rough idea, but it isn't the official scale. At the airport, the suitcase gets weighed for real — and that's when the gap between your guess and the actual weight shows up. If you knew in advance how far off your home scale usually is (say, "it always reads 3% under the real weight"), you could adjust your packing next time and stop getting surprised at the counter.

**Mova Tokenomics does exactly that, but with tokens instead of kilos:**

| In the analogy | In Mova |
|---|---|
| Bathroom scale at home | Local estimate computed with `tiktoken-go`, before anything is sent to any provider |
| Official scale at the airport | Real token count returned by the provider (Anthropic, OpenAI, Google) when the API is actually called |
| Airline's weight limit | `budget.max_tokens` in your `project.json` |
| The notebook where you write "home said X, airport said Y" | The `mova-token-history.json` file that lives in your project |

And because that notebook is yours — not a generic average from thousands of other people's suitcases — the calibration Mova learns is specific to **your project**: its language mix, its code, its documents.

**Why this matters for your wallet (and your sanity):** every token you send to a Cloud model costs money; if the model is local, a context that doesn't fit the window silently truncates or degrades. Mova attacks both problems with the same mechanism: **measure before spending, stop if it's over, and learn from every real call.**

### The Token Firewall — the newest, biggest lever

Everything below in this section was already true before this feature
existed. The Token Firewall adds three more deterministic, zero-AI
stages that run automatically, in front of every one of them — same
`mova run`, `mova chat`, jobs, MCP, the TUI, no new command:

| Stage | What it does | Real, measured result (shipped example) |
|---|---|---|
| **Sanitizer** | Collapses repeated log lines, blank-line runs, and duplicated file headers — before anything is counted | 2,737 → 1,764 tokens: **35.6% fewer tokens, same information** |
| **Cache Layout Guard** | Reorders the prompt so its first tokens are a byte-stable prefix, which is what lets Claude/GPT/Gemini's own prompt caching actually trigger | ~1,050 of 1,167 static-prefix tokens estimated reusable on a cache hit (~90%, Anthropic's own published discount) |
| **Circuit Breaker** | Per-run and monthly USD ceilings, checked BEFORE anything is sent — `"on_exceed": "abort"` stops execution, not just a warning after the bill arrives | Verified: a run that would exceed its ceiling never reaches the model |

**Every stage is on by default**, independently toggleable in
`project.json`, and every number above is real — measured by actually
running the example shipped in this repository
(`projects/ejemplo-token-firewall/`), not projected for documentation.
See [COMMANDS.md § Token Firewall](COMMANDS.md#token-firewall) for the
full mechanics, how caching behaves per provider, and the complete
walkthrough.

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

#### The report, explained simply

When you run the command above, you get a `mova-budget-report.md` file. In plain terms, it tells you three things:

| Report section | What it tells you |
|---|---|
| **Token & Cost Breakdown** | How much each piece of your context (agents, skills, prompt, focus, memory) weighs, in tokens and in dollars — so you know what to trim first. |
| **Budget Limit** | Your configured limit vs. what you're actually using, in tokens and as a percentage. |
| **Historical Token Accuracy** | How close your local estimator gets to reality, measured against your own past real calls to each provider. |

Real example (trimmed from an actual case): a project with a 5,000-token limit uses 1,207 tokens (24.1% of the limit) and would cost between USD 0.0015 and USD 0.0060 depending on the provider — a fraction of a cent. With that, you already know, **before spending anything**, whether to send it to OpenAI, Anthropic, or Google, and how much headroom you have before hitting the limit.

### The learning loop: every real call sharpens the estimate

This is likely the single most important feature in Tokenomics — and the one most worth understanding in depth.

**What gets saved, and where.** Every time `mova chat` or the `chat_completion` MCP tool makes a real call to a Cloud provider (Anthropic, OpenAI, or Google), the API response comes back with a usage field indicating how many tokens the provider actually counted — not an estimate, the exact number you get billed for. Mova takes that real number, together with the local estimate it had computed with `tiktoken-go` *before* the request was sent, and adds both to two running totals stored in a local file inside your project: `mova-token-history.json`. **It never stores the content, the prompts, or the responses** — only two numbers per provider:

```json
{ "anthropic": { "total_local_tokens": 120000, "total_api_tokens": 122760 } }
```

**How the deviation is calculated.** With those two running totals, the formula is simple and fully transparent — no black box:

```text
deviation % = (total_api_tokens − total_local_tokens) / total_local_tokens × 100
```

With the numbers above: `(122,760 − 120,000) / 120,000 × 100 = +2.3%`. In other words: for *this specific project*, Mova's local estimator underestimates real Anthropic usage by 2.3%, on average.

**Why it's a running total instead of a call-by-call log.** Every real call you make adds to those two totals, so the deviation shown isn't based on the last request alone (which can be noisy: a very short prompt, an unusual case), but is a weighted average across all your accumulated real usage. The more real calls you make, the sharper — and more reliable — the number becomes for that specific project, with its own language mix, code, and context density.

**Example with Google (taken straight from this project's own report):**

```json
{ "google": { "total_local_tokens": 1205, "total_api_tokens": 1201 } }
```

`(1,201 − 1,205) / 1,205 × 100 = −0.33%` → that's exactly how `mova budget` arrives at the **"−0.3%"** shown in the *Historical Token Accuracy* section of the report. It's not a made-up figure or a generic benchmark pulled from the internet: it's direct math on your own calls.

**How the file evolves over time.** As more real calls accumulate, the deviation stops jumping around and settles into a reliable number:

| After... | `total_local_tokens` | `total_api_tokens` | Accumulated deviation |
|---|---|---|---|
| 1st real call | 1,000 | 1,030 | +3.0% |
| 5 real calls | 5,400 | 5,505 | +1.9% |
| 20 real calls | 21,800 | 22,190 | +1.8% |

**What Mova does with that number.** `mova budget` uses this accumulated deviation to show you, alongside every new estimate, how far off it might be from reality — and over time it tells you, for that specific project, whether your estimator tends to run low or high with each provider, so you can set `budget.max_tokens` with a real margin instead of guessing blind.

**What happens if you've never called a real provider.** If `total_local_tokens` is 0 (you've never made a real call with `mova chat` to that provider), the report shows `No historical data` — Mova doesn't invent a deviation without real data behind it.

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

## 7. Quick Installation & Test (2 Minutes)

You can install and start using **mova** in just a few minutes using whichever method you prefer.

### Option A — Automatic Installer (Recommended)

1. Go to the `installers/` folder in the repository.
2. Run the installer for your operating system:

| Operating System | Installer |
|------------------|-----------|
| Windows | `install.bat` |
| macOS | `install.command` |
| Linux | `install.sh` |

The installer will:

- Build the executable.
- Install it on your system.
- Automatically configure your **PATH**.

Once the installation is complete, open any terminal (CMD, PowerShell, Git Bash, Bash, Zsh, etc.) and run:

```bash
mova run local-tests
```

You can also run:

```bash
mova
```

or launch the interactive interface:

```bash
mova ui
```

---

### Option B — Using the Makefile (Developers)

If you already have **Go** and **make** installed:

**Install globally:**

```bash
make install
mova run local-tests
```

**Build locally only (`dist/`):**

```bash
make build
./dist/mova run local-tests
```

**Build binaries for all supported platforms:**

```bash
make build-all
```

---

### Option C — Manual Build

If you prefer to build directly with Go:

```bash
go build -o mova ./src/cli
./mova run local-tests
```

---

### Working with Projects Located Anywhere

Your project does **not** need to be inside the **mova** repository.

The `repo` command accepts absolute paths, allowing you to work with projects located on:

- Another drive (Windows).
- Another mount point (Linux/macOS).
- Any directory on your system.

The installers automatically configure everything so **mova** works from any project location without additional setup.

> See **COMMANDS.md** → **Working Across Different Drives/Locations** for more details.
---

## 8. Go deeper

| I want to... | Document |
|---|---|
| See every command (memory, Focus, MCP, HTTP, tokenomics) | [COMMANDS.en.md](COMMANDS.en.md) |
| Read the full spec that models follow | [workflow.md](../../../workflow.md) |
| Understand the source code (Resolvers, Adapters, how to extend it) | [SOURCE.md](../SOURCE.md) |

---

---

## 9. Job Engine, Cron & Multiagent

### Scheduling work: the Job Engine

A project can declare **jobs** in its `project.json` — scheduled,
unattended runs that build context, save reports, update memory, clean
up files, and produce budget reports, all on a cron schedule:

```json
{
  "jobs": [
    {
      "schedule": "0 2 * * *",
      "tasks": ["auditar-checkout", "auditar-cookies"],
      "save": "reports/auditoria_{date}.pdf",
      "budget": { "focus": true },
      "memory": "Auditoría de checkout y cookies realizada"
    }
  ]
}
```

Run it on demand with `mova jobs run <project>`, or start the daemon
that checks every project once a minute with `mova jobs start`. See
[PROJECT_JSON.md § Jobs](PROJECT_JSON.md#jobs) for every field and
[COMMANDS.en.md § Jobs](COMMANDS.en.md) for every command.

### Understanding `schedule` (Cron)

`schedule` uses standard 5-field cron syntax:

```text
schedule: "0 2 * * *"
           │ │ │ │ │
           │ │ │ │ └─ day of week (0-6, 0 = Sunday, * = every day)
           │ │ │ └─── month (1-12, * = every month)
           │ │ └───── day of month (1-31, * = every day)
           │ └─────── hour (0-23)
           └───────── minute (0-59)
```

A few easy examples:

| `schedule` | Runs... |
|---|---|
| `"0 2 * * *"` | every day at 2:00 AM |
| `"30 8 * * 1"` | every Monday at 8:30 AM |
| `"0 0 1 * *"` | at midnight on the 1st of every month |
| `"*/15 * * * *"` | every 15 minutes |
| `"0 9-17 * * 1-5"` | every hour, 9 AM to 5 PM, Monday through Friday |

### Multiagent: several agents under one group

A directory under `projects/` can hold several independent agents,
each an ordinary project, orchestrated by a parent `config.json`:

```text
projects/
    ventas_online/
        config.json          ← the orchestrator
        vendedor/project.json
        atencionCliente/project.json
        soporte/project.json
```

```bash
mova agents run ventas_online          # every agent, sequentially
mova agents run ventas_online vendedor # just one agent
```

Each agent keeps its own memory, budget, focus, tasks, and jobs — see
[PROJECT_JSON.md § Multiagent](PROJECT_JSON.md#multiagent-agent-groups).

---

## 10. Visual interface — `mova ui`

Everything above is also reachable from a simple, lightweight terminal
interface:

```bash
mova ui
```

One single command, navigated with the arrow keys and Enter: chat,
`project.json`, `workflow.md`, model configuration, logging, memory,
jobs, multiagent, reports, and logs, all from the same place — without
adding a new command per feature. The interface replaces nothing: it
calls the exact same components `mova chat`, `mova jobs run`, and
`mova agents run` already use. See
[COMMANDS.md § Visual interface](COMMANDS.md#19-visual-interface--mova-ui)
for the full detail.

---

> **Operational knowledge belongs to the project. Reasoning belongs to the model.**
>
> Mova Context is the convention formed by `workflow.md`, `agents/`, `skills/`, `prompts/`, `project.json`, and `memory.md`. The CLI just automates work on top of that convention — it never replaces it.
