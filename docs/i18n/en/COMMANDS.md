# COMMANDS — command guide

> Docs: [English](COMMANDS.md) · [Español](../es/COMMANDS.md)

The CLI (`mova`) is a complement — everything it does can also be done by asking a model to read `workflow.md` directly. See [README.md](README.md#the-essentials-before-anything-else).

`mova` walks up directories automatically until it finds `workflow.md`, so it works from any subfolder of the repo. If there's a single project under `projects/`, `[project]` is optional — it's auto-detected.

## All commands

```text
mova run           [project] [task]        generate context for LLM
mova memory        [project] "response"    save session to memory.md
mova memory-read   [project]               print active memory
  --all                                    include archives
  --month 2024-01                          specific archive month
mova memory-archive [project]              archive old entries
  --days N                                 keep N days active (default 30)
mova list                                  list all projects
mova init          [name]                  create project
mova search        "query" [domain]        search knowledge
mova mcp start                             start MCP server
  --port 3000                              run as HTTP server (default)
  --stdio                                  run as Stdio server (for Claude/Cursor)
mova memory-clear  [project]               delete ALL memory
  --archived                               delete only archived months
  --keep-active                            delete archives, keep memory.md
  --date 2024-06-15                        delete a specific day
  --from 2024-06-01 --to 2024-06-30        delete date range
  --yes                                    skip confirmation
mova memory-config [project] [action] [value]
  enable | disable                         toggle auto-archive
  days N                                   set retention days (1, 10, 30, 90...)
  confirm true|false                       toggle confirmation on delete
```

## `mova run [project] [task]`

Assembles agents + skills + prompt + memory + focus, and prints it to stdout — ready to paste into a chat or send to an API.

```bash
mova run my-project review-auth
```

If the task has `focus` set (in `project.json`, global or per-task), that section is automatically appended to the end of the context — see below.

## `mova memory [project] "LLM response"`

Extracts the ` ```memory ` block from a model's response and appends it to `memory.md`.

```bash
mova memory my-project "$(cat response.txt)"
```

Next time you run `mova run my-project`, that memory shows up in the context automatically.

## `mova memory-read [project] [--all] [--month YYYY-MM]`

```bash
mova memory-read my-project --all
mova memory-read my-project --month 2024-01
```

## `mova memory-archive [project] [--days N]`

Moves entries older than `N` days (default 30) out of `memory.md`, grouped by month.

```bash
mova memory-archive my-project --days 15
```

## `mova memory-clear [project] [flags]`

Asks for confirmation unless you pass `--yes`.

```bash
mova memory-clear my-project --archived --yes
```

## `mova memory-config [project] [action] [value]`

```bash
mova memory-config my-project days 45
```

## `mova list` / `mova init [name]`

```bash
mova list
mova init my-project
```

`init` creates `projects/my-project/project.json` (minimal template) and an empty `memory.md`.

## `mova search "query" [domain]`

Searches agents, skills, and prompts — by keyword, no model required.

```bash
mova search "authentication" software
```

## FOCUS — working on a specific part of the project

`focus` (set in `project.json`, global or per task) tells the engine to work only on certain files, folders, or symbols — instead of the whole repo. It works the same with or without the CLI: if a model reads `workflow.md` directly, the spec's `## FOCUS` section explains exactly how to resolve it.

**Important:** `focus` is relative to `project.json`'s `"repo"` field, not to the `mova-context` root. If `"repo": "examples/my-repo"`, an item `"manual.md"` is searched for inside `examples/my-repo/`, not the Mova Context project root.

If `task.focus` is set, it **replaces** the project's global `focus` entirely (the two lists are never merged).

### How each item matches — same idea as SQL LIKE

Each `focus` item is resolved through a cascade of resolvers (file → code symbol → Markdown section → legal article → memory → fallback). All of them use the same two-pass criteria, equivalent to SQL's `LIKE`:

| Pass | SQL equivalent | When it's used |
|---|---|---|
| 1 — Exact | `WHERE name = 'CreateOrder'` (word-boundary match) | Always tried first — highest priority |
| 2 — LIKE / contains | `WHERE name ILIKE '%CreateOrder%'` | Only if pass 1 found nothing — case- and accent-insensitive |

You don't declare which pass to use — the engine tries pass 1 and automatically falls back to pass 2 if there's no result. Both passes are case/accent-insensitive (`articulo 6` finds `Artículo 6`). It never uses an LLM or meaning-based heuristics — this is deterministic text search: same input, same result, every time.

### Supported item types

| Item in `focus` | What it resolves |
|---|---|
| `"manual.md"` | the full file, found by name anywhere in the repo |
| `"src/auth"` | a directory index (not the contents of every file in it) |
| `"CreateOrder()"` | the function/method/class — the `()` syntax tells the engine it's a code symbol, not a file |
| `"Article 6"` | a section of a legal/structured document (Title, Chapter, Section, Article, Clause) |
| `"## Some heading"` or `"Some heading"` | a Markdown heading |
| `"table_name"` | the `CREATE TABLE ...;` definition in a `.sql` file |

### Real example

```json
"tasks": {
  "review-order": {
    "prompt": "review-project",
    "focus": [
      "CreateOrder()",
      "manual.md",
      "Article 6"
    ]
  }
}
```

```bash
mova run my-project review-order
```

Resulting context (excerpt):

```text
---
## FOCUS
FOCUS:CreateOrder()
  (src/orders.go)
func CreateOrder(clientID string, amount float64) (string, error) {
    ...
}

FOCUS:manual.md
  (manual.md)
# Operations Manual
...

FOCUS:Article 6
  (manual.md)
### Article 6 — Order cancellation
An order can only be cancelled if it hasn't been dispatched yet.
```

If an item isn't found in either pass, it shows up as `not found: [item]` instead of being silently dropped — so you know immediately if a name is misspelled or the file isn't inside the configured `repo`.

### If your context comes back with nothing from focus — checklist

1. Does `"repo"` in `project.json` point to a folder that **actually exists** and contains the files you're targeting? (`focus` never searches outside `repo`)
2. Does the code symbol end in `()` (`"CreateOrder()"`) so the engine knows it's a function, not a file?
3. Does the `task` you ran define its own `focus`? If so, that one replaces the global one — check which is actually in effect.
4. Are you running a `mova` binary built **before** this fix? If the context shows no `## FOCUS` section at all (not even a `not found:`), rebuild it: `go build -o mova ./src/cli`.

## `mova mcp start` — expose Mova Context as a server

Same engine as `mova run`, exposed over the MCP protocol (JSON-RPC 2.0) — so a client (Claude Desktop, Cursor) can request context on its own, without you copy-pasting anything.

**Stdio mode** (used by Claude Desktop / Cursor):

```bash
mova mcp start --stdio
```

Typical MCP client config:

```json
{
  "mcpServers": {
    "mova-context": {
      "command": "/path/to/mova",
      "args": ["mcp", "start", "--stdio"],
      "env": { "MOVA_PROJECT_ROOT": "/path/to/your/mova-context" }
    }
  }
}
```

**HTTP mode** (to test with curl/Postman, or wire into your own backend):

```bash
mova mcp start --port 3000
```

```bash
curl -X POST http://localhost:3000/rpc \
  -H "content-type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"get_full_context","arguments":{"project":"my-project","task":"review-auth"}}}'
```

### Tools available via MCP

| Tool | Equivalent to |
|---|---|
| `get_full_context` | `mova run [project] [task]` |
| `get_knowledge` | reading a single agent/skill/prompt |
| `get_memory` | `mova memory-read [project]` |
| `get_memory_all` | `mova memory-read [project] --all` |
| `get_workflow` | reading `workflow.md` |
| `search_context` | `mova search "query" [domain]` |

## Environment variables

```bash
MOVA_ADAPTER=db MOVA_DSN=postgres://user:pass@host/db mova run my-project
```

| Variable | Effect |
|---|---|
| `MOVA_ADAPTER` | Overrides `project.json.adapter` (`file` / `db`) |
| `MOVA_DSN` | Overrides `project.json.dsn` |
| `MOVA_PROJECT_ROOT` | Extra starting point for the `workflow.md` upward search |
| `MOVA_PROJECT_PATH` | Uses this path as the root directly, no search |

### Root resolution and MCP clients

MCP clients (Claude Desktop, Cursor) launch `mova` from a directory that's usually not your project — that's why the config example above sets `MOVA_PROJECT_ROOT`. Resolution order: `MOVA_PROJECT_PATH` (direct) → `MOVA_PROJECT_ROOT` (search upward from there) → current working directory → the binary's own directory.

## `llm_profile` — which model the context is handed to

`llm_profile` (in `project.json`) is the only thing that changes when you switch from one model/provider to another. Agents, skills, prompts, memory, and `focus` never change — the same `mova run` produces the same context no matter which model is going to read it.

```json
"llm_profile": {
  "type": "local",
  "provider": "ollama",
  "model": "llama3.2:3b",
  "base_url": "http://localhost:11434"
}
```

| Field | Values | What it's for |
|---|---|---|
| `type` | `"powerful"` (default) \| `"local"` | With `"local"` the engine adapts formatting: dash lists become numbered, `INSTRUCTIONS:` gets prepended — small local models follow explicit, sequential instructions better. With `"powerful"` content is delivered unchanged. |
| `provider` | `"claude"` \| `"gpt"` \| `"gemini"` \| `"ollama"` \| any string | Informational — shown in the context header (`Profile: local/ollama:llama3.2:3b`). Doesn't change what gets generated, except through `type`. |
| `model` | exact model name | Same as `provider`: informational, useful to know which model a given `contexto.txt` was generated for. |
| `base_url` | server URL | Needed for `ollama` or another OpenAI-compatible server running locally — not used by the assembly engine itself, it's there for your own script to know where to send the context. |

### Simple form (legacy)

If you don't need `base_url` or an explicit `model`, the `llm` field (a plain string) still works and is automatically translated into an `llm_profile`:

```json
"llm": "ollama"
```

is equivalent to:

```json
"llm_profile": { "type": "local", "provider": "ollama" }
```

Automatically recognized as local: `ollama`, `llama`, `mistral`, `deepseek`, `qwen`, `gemma`, `phi`. Anything else (`claude`, `gpt`, `gemini`, or a custom value) is treated as `"powerful"`.

### Switching providers without touching anything else

```json
// Claude / GPT / Gemini — via API or pasted into a web chat
"llm_profile": { "type": "powerful", "provider": "claude", "model": "claude-sonnet-4-6" }

// Local Ollama
"llm_profile": { "type": "local", "provider": "ollama", "model": "llama3.2:3b", "base_url": "http://localhost:11434" }
```

```bash
mova run my-project my-task > contexto.txt
ollama run llama3.2:3b < contexto.txt
```

## Building the CLI

```bash
go build -o mova ./src/cli
```

No editions or special build flags — one binary, every command above.
