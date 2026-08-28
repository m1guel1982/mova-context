# PROJECT.JSON — field reference

Every project lives in `projects/<name>/project.json`. This is the single source of configuration for that project — model prices (`config/prices.json`) and the optional PII Masking thresholds (`config/policy.json`) are the only things that live elsewhere, since both are global configuration shared by every project (see [COMMANDS.md §15](COMMANDS.md#15-tokenomics--mova-budget) and [COMMANDS.md §20](COMMANDS.md#20-token-firewall)).

## Minimal example

```json
{
  "project": "my-project",
  "repo": "examples/my-project-repo",
  "lang": "en",
  "adapter": "file",
  "default_task": "review",

  "agents": { "domain": "base", "use": ["backend-dev"], "custom": [] },
  "skills": { "domain": "base", "use": ["lazy-minimalism"], "custom": [] },

  "tasks": {
    "review": { "prompt": "review-project" }
  },

  "llm_profile": {  "config": "llama3.2.3b" }
}
```

## All fields

| Field | Type | What it's for |
|---|---|---|
| `project` | string | the project's name — matches the folder under `projects/` |
| `description` | string | free text, shown in `mova projects` |
| `repo` | string | the project's single repository. One repository per project, one path — see **More than one directory?** below if you're tempted to add a second |
| `lang` | string | `"es"`, `"en"`, ... — which prompt/agent/skill language variant to load |
| `llm_profile` | object | `{  "config" }` — the modern way to pick a model (see [COMMANDS.md §6](COMMANDS.md#6-models-and-providers)) |
| `default_task` | string | task used when none is given on the command line |
| `variables` | object | `{name: value}` injected into prompts/agents/skills |
| `agents` / `skills` | object | `{ "domain", "use": [...], "custom": [...] }` |
| `tasks` | object | named tasks — each may override `prompt`/`agents`/`skills`/`variables`/`focus`/`exclude`/`budget` |
| `archive` | object | memory management config (see [COMMANDS.md §4](COMMANDS.md#4-memory)) |
| `focus` | array | files/dirs/symbols to work on instead of the whole repo — see **Working on part of `repo`** below. Also supports cross-platform absolute host paths (`"C:\\example\\file.py"`, `"/mnt/data"`) alongside repo-relative paths and globs — see **Absolute paths in `focus`** below |
| `exclude` | array | files/dirs/patterns `focus` must NEVER read, even if requested by exact name — same syntax as `focus` (name, relative path, cross-platform absolute path, glob). See **Excluding from `focus` (`exclude`)** below |
| `focus_display_limit` | number | how many `focus` file/directory names the `[Focus] Selected ...` line (`mova chat`, chat_completion, Mova UI) shows before collapsing the rest into a `+N` badge. Defaults to `2` — see **The `[Focus] Selected ...` line** below |
| `budget` | object | `{ "max_tokens": N }` — the context ceiling (see [COMMANDS.md §15](COMMANDS.md#15-tokenomics--mova-budget)) |
| `workflow_path` | string | where `workflow.md` lives for this project — see **Workflow** below |
| `budget_path` | string | where `mova-budget-report.md` is written — see **Budget report** below |
| `token_history_path` | string | where `mova-token-history.json` is written — see **Token history** below |
| `tools` | object | `{ "enabled": true/false }` — lets chat call file/document tools mid-conversation |
| `jobs` | array | scheduled background jobs (cron `schedule` + actions) — see **Jobs** below |

`repo`, `workflow_path`, `budget_path`, and `token_history_path` are all single strings — on purpose. One value per field keeps a project.json easy to read and easy to reason about; there is no array form for any of them.

`repo` doesn't have to live near Mova's own install — it accepts an absolute path pointing anywhere, including a different drive/volume entirely (`"repo": "D:\\my-app"` on Windows, `"repo": "/mnt/data/my-app"` on Linux, `"repo": "/Volumes/Data/my-app"` on macOS), which every focus/save/delete/jobs action already resolves correctly. See [COMMANDS.md § Working across different drives/locations](COMMANDS.md#working-across-different-drivesocations-windowslinuxmacos) for the full explanation and how to run `mova` from inside that external folder.

## Working on part of `repo`, instead of a second repository

If part of your work only touches one folder inside `repo` — one service in a monorepo, one module, one directory — that's what `focus` is for, not a second `repo` entry (there isn't one):

```json
{
  "project": "my-monorepo-project",
  "repo": "examples/my-monorepo",
  "tasks": {
    "review-api": { "prompt": "review-api", "focus": ["services/api/"] },
    "review-web": { "prompt": "review-ui", "focus": ["services/web/"] }
  }
}
```

Each task scopes itself to its own directory with `focus` — see [COMMANDS.md's Focus section](COMMANDS.md#3-focus--scoping-context-to-part-of-the-repo) for the full glob/directory syntax (`"**/*"`, `"."`, `"src/"`, ...). This keeps `project.json` simple (one `repo`, one path) while still letting each task work on just its own slice of the repository.

If your work genuinely spans two UNRELATED repositories — different owners, different release cycles, never touched by the same task — the simpler and more honest setup is two separate projects (`projects/service-a/project.json`, `projects/service-b/project.json`), each with its own `repo`, `workflow_path`, `budget_path`, and `token_history_path`. See **More than one project** below.

## Absolute paths in `focus` (Windows / Linux / macOS)

Besides repo-relative paths, a `focus` (or `memory`) target can be an absolute filesystem path, to work with a file or folder that lives **entirely outside `repo`**:

```json
{
  "focus": [
    "C:\\ejemploPython\\testSentence.py",
    "d:\\test\\test.py",
    "/mnt/archivo.java",
    "/mnt"
  ]
}
```

Works identically no matter which OS `mova` runs on — a Windows drive letter (`C:\...`) or a UNC path (`\\server\share`) resolves directly; a Unix-style `/something` is tried first as a real absolute path (used as such if it exists on disk), and falls back to the historical behavior (relative to `repo`'s root — the long-standing `"/src"` convention keeps working exactly as before) if it doesn't. An absolute glob also works (`"/mnt/**/*.java"`).

## Excluding from `focus` (`exclude`)

`exclude` is `focus`'s counterpart: a list of targets, with the **same cross-platform syntax** (bare name, relative path, absolute host path, glob), but for EXCLUSION. Any file or directory matching an `exclude` pattern is never read — even if `focus` requests it by its exact name — and therefore never makes it into `mova-context-cache.json`.

```json
{
  "focus": ["."],
  "exclude": [
    "node_modules",
    ".git",
    "C:\\secrets",
    "D:\\private",
    "/mnt/sensitive-data",
    "*.env"
  ]
}
```

Supported forms, same as `focus`:

| Pattern | What it excludes |
|---|---|
| `"node_modules"`, `".git"` | that folder/file by NAME, at any level of the tree — no need to know the full path |
| `"src/secrets"` | that specific path, relative to `repo` |
| `"C:\\secrets"`, `"/mnt/data"` | that absolute host path (and everything under it, if it's a directory) |
| `"*.env"`, `"**/*.pem"` | any file matching the glob, regardless of directory |

A task-level `tasks.<name>.exclude` overrides (does not merge with) the project-level `exclude` — same rule as `focus`. If `exclude` isn't declared, `focus` behaves exactly as it did before this key existed (on top of the always-on default exclusion of `.git`/`node_modules`/`vendor`/`dist`/`build`/`__pycache__`/`.venv`/`venv`/`.idea`/`.vscode`).

## The `[Focus] Selected ...` line

When `focus` is configured, `mova chat`, the `chat_completion` tool (MCP/HTTP), and Mova UI show a status line summarizing what's about to be analyzed — for example:

```
[Focus] Selected 3 items (45 file(s) total): server.js, backend-test.py 📎+1.
```

It distinguishes file from directory, counts real resolved files (not configured targets), and lists up to `focus_display_limit` names before collapsing the rest into the `📎+N` badge:

```json
{ "focus_display_limit": 4 }
```

Defaults to `2` when not declared. Any configured value is respected as-is — exceeding it always shows the `+N`, whatever the limit is.

## Workflow example

```json
{
  "project": "my-project",
  "repo": "examples/my-project-repo",
  "workflow_path": "workflow.md"
}
```

Saying "lee workflow.md", "ejecuta workflow.md", "workflow.md my-project", etc. resolves this project, validates its Budget, and only then loads the file — see [COMMANDS.md §9](COMMANDS.md#workflowmd--budget-gated-execution). Without `workflow_path`, Mova looks for a plain `workflow.md` at the Mova root, same as before this field existed.

## Budget example

```json
{
  "budget": { "max_tokens": 8000 },
  "budget_path": "mova-budget-report.md"
}
```

Without `budget_path`, the report is written to `projects/<project>/mova-budget-report.md`.

## Token history example

```json
{
  "token_history_path": "mova-token-history.json"
}
```

Without it, the file is written to `projects/<project>/mova-token-history.json`.

## Tasks example

```json
{
  "tasks": {
    "review": { "prompt": "review-project" },
    "audit": {
      "prompt": "audit-consent-flow",
      "agents": ["security-architect"],
      "variables": { "module": "checkout" },
      "focus": ["checkout.html"],
      "budget": { "max_tokens": 4000 }
    }
  }
}
```

## Agents / Skills / Prompts example

```json
{
  "agents": { "domain": "base", "use": ["backend-dev", "qa-engineer"], "custom": [] },
  "skills": { "domain": "base", "use": ["lazy-minimalism"], "custom": ["my-custom-skill"] }
}
```

Prompts are referenced by name from each task's `"prompt"` field (see the **Tasks** example above) — there's no separate top-level `prompts` field; a prompt is chosen per task.

## More than one project — agents/skills/prompts shared, nothing duplicated

There is no such thing as one `project.json` holding several projects — each project is its own folder with its own file:

```text
projects/
├── project-a/
│   └── project.json
└── project-b/
    └── project.json
```

**`projects/project-a/project.json`**
```json
{
  "project": "project-a",
  "repo": "examples/project-a-repo",
  "lang": "en",
  "agents": { "domain": "base", "use": ["backend-dev"], "custom": [] },
  "skills": { "domain": "base", "use": ["lazy-minimalism"], "custom": [] },
  "tasks": { "review": { "prompt": "review-project" } },
  "budget_path": "mova-budget-report.md",
  "token_history_path": "mova-token-history.json"
}
```

**`projects/project-b/project.json`**
```json
{
  "project": "project-b",
  "repo": "examples/project-b-repo",
  "lang": "en",
  "agents": { "domain": "base", "use": ["security-architect"], "custom": [] },
  "skills": { "domain": "base", "use": ["lazy-minimalism"], "custom": ["my-custom-skill"] },
  "tasks": { "audit": { "prompt": "audit-consent-flow" } },
  "budget_path": "mova-budget-report.md",
  "token_history_path": "mova-token-history.json"
}
```

Agents/skills/prompts are global resources (they live once, under `agents/`, `skills/`, `prompts/` at the Mova root); each project only *references* them by name in `use`/`prompt`. If `project-a` and `project-b` both use `"lazy-minimalism"`, it's the exact same skill file for both — nothing is copied per project. Each project keeps its own `budget_path`/`token_history_path`/`workflow_path`, so their Budget reports and token history never mix.

## Budget (and the Token Firewall)

`"budget"` (project- or task-level — a task's own `budget` replaces the
project's, same rule as `focus`) always accepted `max_tokens`, a hard
content-size ceiling. Since the Token Firewall, it also accepts every
field below — a set of deterministic, zero-AI stages that reduce what
gets sent to a model and govern what it costs, running automatically
before that gate. **Every stage is enabled by default, EXCEPT
`pii_masking`** — set the matching field to `false` (or, for
`pii_masking`, to `true` to opt in) to change just that one:

```json
{
  "budget": {
    "max_tokens": 20000,
    "max_tokens_per_run": 8000,
    "max_monthly_usd": 15.00,
    "on_exceed": "warn",
    "sanitize": { "enabled": true, "dedupe_logs": true, "strip_blank": true, "strip_comments": false },
    "pii_masking": { "enabled": false },
    "cache_hint": true,
    "circuit_breaker": true,
    "token_estimation": true,
    "detailed_reports": true,
    "context_cache": true
  }
}
```

| Field | Type | Default | What it does |
|---|---|---|---|
| `max_tokens` | number | none | Hard ceiling on the assembled context — unchanged since before the Token Firewall existed |
| `max_tokens_per_run` | number | none (0 = no ceiling) | Circuit Breaker: aborts/warns if a single run's token count exceeds this |
| `max_monthly_usd` | number | none (0 = no ceiling) | Circuit Breaker: aborts/warns if this project's tracked spend for the current calendar month reaches this |
| `on_exceed` | `"warn"` \| `"abort"` | `"warn"` | What the Circuit Breaker does when a ceiling above is hit |
| `sanitize` | object | enabled, conservative | The Sanitizer's own settings — see below |
| `pii_masking` | object | **disabled** (`{"enabled": false}` or absent) | OPTIONAL structural PII pseudonymization stage (word shape + Shannon entropy, no word lists) — set `{"enabled": true}` to explicitly opt this project in. Its thresholds/weights live in `config/policy.json`, not here. See [COMMANDS.md § Token Firewall § PII Masking](COMMANDS.md#token-firewall) and `skills/base/i18n/en/compliance/pii-context-reduction.md` — **not** a legal anonymization or Ley 21.719/GDPR compliance guarantee |
| `cache_hint` | boolean | `true` | Enables the Cache Layout Guard (reorders the system prompt for provider prompt-caching) — set `false` to disable |
| `circuit_breaker` | boolean | `true` | Enables the Circuit Breaker mechanism itself, independent of whether a ceiling is configured — set `false` to disable even with ceilings still set |
| `token_estimation` | boolean | `true` | Uses the real tiktoken tokenizer — set `false` for a fast chars/4 approximation instead (a performance trade-off, not a savings feature) |
| `detailed_reports` | boolean | `true` | Includes the full breakdown (per-file tokens, before/after comparison) in mova-budget-report.md — set `false` for just the totals |
| `context_cache` | boolean | `true` | Enables Mova's own local memoization of Sanitizer results (mova-context-cache.json) — saves wall-clock time on repeat runs with unchanged files |

### `sanitize` — the Sanitizer's own settings

| Field | Type | Default | What it does |
|---|---|---|---|
| `enabled` | boolean | `true` | Master switch for the whole Sanitizer stage |
| `dedupe_logs` | boolean | `true` | Collapses 3+ near-identical consecutive lines (ignoring a leading timestamp) into the first occurrence + a counter — the "50 lines of INFO 200 OK" case |
| `strip_blank` | boolean | `true` | Collapses runs of 3+ blank lines to 1 |
| `strip_comments` | boolean | `false` | Removes comment-only blocks of 5+ lines — off by default, since a task about documentation needs them intact |

See [COMMANDS.md § Token Firewall](COMMANDS.md#token-firewall) for the
full explanation of each stage, how the Cache Layout Guard behaves per
provider (Claude, GPT, Gemini, Ollama...), and a complete worked
example with real measured savings.

## Jobs

`jobs` is an array of scheduled background jobs, run by the Job Engine
(`mova.local/jobs`) — the same executor `mova jobs run`, the "run_job"
MCP tool, `POST /jobs/run`, and the `mova jobs start` daemon all share.
Each entry combines a cron `schedule` with one or more independent
actions:

```json
{
  "jobs": [
    {
      "comment": "Nightly checkout/cookies audit",
      "schedule": "0 2 * * *",
      "tasks": ["auditar-checkout", "auditar-cookies"],
      "save": "reports/auditoria_{date}.pdf",
      "budget": { "focus": true },
      "memory": "Auditoría de checkout y cookies realizada ({date})"
    },
    {
      "comment": "Monthly memory archive, no tasks",
      "schedule": "0 3 1 * *",
      "memory_archive": { "days": 30 }
    },
    {
      "comment": "Run every task defined in this project",
      "schedule": "0 4 * * *",
      "tasks": ["*"],
      "save": "reports/auditoria_completa_{date}.pdf"
    },
    {
      "comment": "Clean up temp files",
      "schedule": "0 5 * * *",
      "delete": ["reports/temp_*.csv", "logs/draft.md"]
    }
  ]
}
```

| Field | Type | What it does |
|---|---|---|
| `comment` | string | free text, never interpreted — for humans reading project.json |
| `schedule` | string | 5-field cron (`min hour dom month dow`) — see README.md § Cron for examples |
| `tasks` | array of strings | task names from this project's `tasks`, or `["*"]` to run all of them |
| `save` | string | output path; `{date}` expands to `YYYY-MM-DD`. Format is picked from the extension (`.md`, `.pdf`, `.docx`...), same as chat's `/save` |
| `memory` | string | text appended to `memory.md` via `AppendMemory`; supports `{date}` and `{time}` |
| `memory_archive` | object | `{ "days": N }` — archives memory entries older than N days (defaults to the project's own `archive` retention when omitted) |
| `delete` | array of strings | glob patterns (relative to `repo`), e.g. `"reports/temp_*.csv"` |
| `budget` | object | `{ "focus": true }` — also writes a `mova-budget-report.md`, like `mova budget --focus`. Distinct from the top-level `budget` field (a token ceiling): this one is an ACTION, not a gate |

Every field is independent — a job can declare any subset of
`tasks`/`save`/`memory`/`memory_archive`/`delete`/`budget`. All
declared actions for a job run in this fixed order: tasks → save →
memory → memory_archive → delete → budget, regardless of the order
they're written in JSON.

Run jobs on demand, ignoring `schedule`, with `mova jobs run <project>`
(or `mova jobs run <project> <index>` for a single job) — see
COMMANDS.md § Jobs.

## Multiagent (agent groups)

A **group** is a directory under `projects/` holding several
independent agents, each an ordinary project with its own
`project.json`:

```text
projects/
    ventas_online/
        config.json
        vendedor/
            project.json
        atencionCliente/
            project.json
        soporte/
            project.json
```

`projects/ventas_online/config.json` is the parent/orchestrator file:

```json
{
  "group": "ventas_online",
  "description": "Sales, support, and customer-care agents",
  "agents": ["vendedor", "atencionCliente", "soporte"]
}
```

| Field | Type | What it's for |
|---|---|---|
| `group` | string | display name (defaults to the directory name if omitted) |
| `description` | string | free text |
| `agents` | array of strings | subdirectory names, each with its own `project.json`. If omitted, every subdirectory containing a `project.json` is auto-discovered |

Each agent is addressed as `<group>/<agent>` — an ordinary project name
that every existing command already understands (`mova run
ventas_online/vendedor`, `mova budget ventas_online/vendedor`, `mova
jobs run ventas_online/vendedor`, ...). `mova agents run ventas_online`
runs every agent sequentially through the same assemble+Budget-gate
pipeline `mova run` uses for any project — see COMMANDS.md § Multiagent.

## Diagram (optional)

Visual diagram preferences (`mova run <project> --diagram`) — see
[COMMANDS.md § Visual diagrams](COMMANDS.md#21-visual-diagrams--mova-run-project---diagram)
for the full command. Absent = defaults apply (`verbose`, export to
`svg`).

```json
{
  "diagram": {
    "detail_level": "simple",
    "export_formats": ["svg", "png"]
  }
}
```

| Field | Type | Default | What it does |
|---|---|---|---|
| `detail_level` | `"simple"` \| `"verbose"` | `"verbose"` | Diagram detail level — the CLI's `--diagram` can override this per run |
| `export_formats` | array of strings | `["svg"]` | Default formats when `mova run --diagram` is called without `--export` |

## Distributed architecture (remote endpoints)

`llm_profile` makes no distinction between a model running on `localhost` and one running on another machine — the only field that changes is `base_url`, inside the file `llm_profile.config` points to (`config/models/<provider>/<file>.json`), never `project.json` itself. This is what enables a centralized deployment (see [DEPLOY.md](DEPLOY.md)): a single instance — for example a Docker container on Oracle Cloud or AWS running `mova mcp start` + Ollama — serves as an inference coprocessor for several local clients, without any of them needing to run the model on their own.

```json
{
  "base_url": "http://100.x.y.z:11434",
  "model": "llama3.2:3b",
  "timeout_seconds": 300
}
```

`100.x.y.z` is deliberately a **private-network** address (Tailscale, WireGuard, or a cloud provider's own virtual network) — never an unprotected public IP: see DEPLOY.md § Network security for why. The repository ships a complete, ready-to-try example under `config/models/ollama/llama3.2.3b-remote.json` and `projects/ejemplo-ley21719-pii-context/ai-privacy-reviewer/project_remote.json` — the same `project_local.json` / `project_cloud.json` / `project_remote.json` convention the Ley 21.719 example already uses to switch between a local model, a Cloud model, and a remote model without changing a single line of code.

**Strict separation of responsibilities — why the remote server never sees the repository:**

| Always happens on the CLIENT (this machine) | Always happens on the remote SERVER |
|---|---|
| Reading `project.json`, `agents/`, `skills/`, `prompts/`, `memory.md` | None of the above — the remote server neither has nor needs the repository |
| Sanitization (`budget.sanitize`) and deduplication | — |
| PII Masking (`budget.pii_masking`) | — |
| Building the final context (the complete Token Firewall pipeline) | — |
| Sending the final, ready payload to `base_url` | Receiving the payload and running inference — a stateless coprocessor that never persists any project content |

This separation isn't a configurable option: it's how the pipeline is built (`budget.BuildGatedContext` always runs locally, before `models.Session.Send` makes the outbound HTTP call) — the same engine serves a model on `localhost` and one on Oracle Cloud identically, with no separate code path between the two.
