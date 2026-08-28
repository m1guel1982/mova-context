# mova — Command Guide

> Docs: [Español](COMMANDS.md) · [English](COMMANDS.en.md)

`mova` walks up from the current directory looking for `workflow.md`, so it works from any subfolder of the repo. If `projects/` has a single project, `[project]` is optional. Convention: `[optional]` · `<required>`.

## 1. Quick reference

| Command | Purpose | Example |
|---|---|---|
| `run` | Assembles the context (agents+skills+prompt+focus+memory) | `mova run my-project review-auth` |
| `run --count` | Estimates tokens/cost without running | `mova run --count my-project` |
| `run --diagram` | Generates a visual pipeline diagram | `mova run my-project --diagram` |
| `budget` | Estimates tokens/cost and writes a report | `mova budget my-project --focus` |
| `chat` | Interactive chat with a local or Cloud model | `mova chat my-project review-auth` |
| `memory` | Saves the session to `memory.md` | `mova memory my-project "..."` |
| `memory-read` | Prints the active memory | `mova memory-read my-project --all` |
| `memory-archive` | Archives old entries | `mova memory-archive my-project --days 15` |
| `memory-clear` | Clears memory (full or partial) | `mova memory-clear my-project --yes` |
| `memory-config` | Configures retention/confirmation | `mova memory-config my-project days 45` |
| `list` | Lists all projects | `mova list` |
| `init` | Creates a new project | `mova init my-project` |
| `search` | Searches agents/skills/prompts, no model | `mova search "authentication" software` |
| `config` | Sets the active provider | `mova config ollama` |
| `show config` | Shows active provider/model | `mova show config llama3.1` |
| `install` | Downloads Ollama models | `mova install llama3.1,mistral` |
| `model-list` / `remove` | Lists / removes installed models | `mova remove mistral` |
| `/save` | Creates/edits any file or folder (chat) | `/save "report.pdf"` |
| `/delete` | Deletes files/folders, with confirmation | `/delete "a.txt" "logs/"` |
| `jobs list` / `run` / `start` | Scheduled jobs per project | `mova jobs run my-project 0` |
| `agents list` / `run` | Agents in a multi-agent group | `mova agents run sales seller` |
| `mcp start` | MCP server (stdio or HTTP) | `mova mcp start --stdio` |
| `ui` | Terminal visual interface | `mova ui my-project` |

## 2. `mova run` — assemble the context

```
mova run [project] [task] [--count] [--focus] [--diagram]
```
Combines agents, skills, prompt, memory, and focus, and prints the result to stdout — ready to paste into a chat or send to an API.

`task` is optional: it uses `default_task` if set, or the project's only task if there's just one. **Budget** validation always runs first; if the assembled context exceeds the configured limit, only the error is printed, nothing else:

```text
ERROR
Current context (128,400 tokens) exceeds the configured limit (100,000).
Suggestion: Use --focus to reduce the included files.
```

| Flag | What it does |
|---|---|
| `--count` | doesn't print the context: only estimates tokens/cost (also accepts a multi-agent group) |
| `--focus` | compares the full repo vs. only what `focus` selects |
| `--diagram` | generates a visual diagram instead of running — [§18](#18-visual-diagrams) |

```bash
mova run my-project review-auth
mova run --count my-group              # sums every agent in the group
```

## 3. Focus — working on part of the project

`focus` (in `project.json`, global or per task) limits the context to certain files, folders, or symbols. It's relative to the `"repo"` field. If `task.focus` is set, it **fully replaces** the global `focus`.

It matches by exact name first, and only falls back to partial matching if that finds nothing — case/accent-insensitive, deterministic text search, never a model.

| Item | What it resolves |
|---|---|
| `"manual.md"` | the full file, by name |
| `"src/auth"` | a directory index |
| `"CreateOrder()"` | a function/method/class (`()` = code) |
| `"Article 6"` | a section of a legal document |
| `"## Section"` | a Markdown heading |
| `"table_name"` | a `CREATE TABLE` in a `.sql` file |
| `"**/*"` | the entire repository |
| `"."` | just the project root |
| `"src/", "pkg/"` | specific directories — the usual way to exclude `node_modules`, `vendor`, `.git` |

An item that isn't found shows as `not found: [item]` — never silently skipped.

```json
"tasks": { "review-order": { "focus": ["CreateOrder()", "manual.md", "Article 6"] } }
```

## 4. Memory

```
mova memory         <project> "<answer>"
mova memory-read    [project] [--all | --month YYYY-MM]
mova memory-archive [project] [--days N]
mova memory-clear   [project] [--archived] [--keep-active] [--date D | --from D --to D] [--yes]
mova memory-config  [project] <enable|disable|days N|confirm true|false>
```

`memory` extracts the memory block from a model's response and appends it to `memory.md`; the next `mova run` picks it up automatically.

| Command | Flag | What it does |
|---|---|---|
| `memory-read` | `--all` / `--month` | full history / one archived month |
| `memory-archive` | `--days N` | days to keep active (default 30) |
| `memory-clear` | `--archived` | clears only archived months |
| `memory-clear` | `--keep-active` | deletes archive files, keeps `memory.md` |
| `memory-clear` | `--date` / `--from`/`--to` | a single day or date range |
| `memory-config` | `enable`/`disable` | automatic archiving |
| `memory-config` | `days N` / `confirm` | retention / confirmation on delete |

```bash
mova memory my-project "$(cat answer.txt)"
mova memory-clear my-project --archived --yes
```

## 5. Projects — list, init, search

```
mova list
mova init   [name]
mova search "<query>" [domain]
```
`init` creates a minimal `project.json` + empty `memory.md`. `search` searches agents/skills/prompts by keyword, without using any model.

```bash
mova init my-project
mova search "authentication" software
```

## 6. Models and providers

```
mova config      <provider>
mova show        config [model]
mova install     model1,model2
mova model-list
mova remove      model1,model2
```

`llm_profile` (in `project.json`) is the only thing that changes between models — agents, skills, prompts, memory, and focus never change.

```json
"llm_profile": { "config": "llama3.2.3b" }
```

| Field | Values | What it's for |
|---|---|---|
| `type` | `powerful` (default) \| `local` | `local` adapts formatting so small models follow sequential instructions better |
| `provider` | `ollama`, `google`, `anthropic`, `openai`, `lmstudio`... | a subdirectory of `config/models/` |
| `config` | filename, without `.json` | connection + inference parameters, together |

```json
"llm_profile": {  "config": "claude-sonnet-4-6" }
"llm_profile": {  "config": "gemini-2.5-flash" }
```

Editing a model's `.json` hot-reloads. `install`/`model-list`/`remove` use Ollama's native API; with Cloud or LM Studio/vLLM, the model is installed separately and only its `.json` is created.

```bash
mova install llama3.1,mistral,phi3
mova show config
```

**Remote endpoint (Oracle Cloud, AWS, any private network):** the model's `.json` `base_url` doesn't have to be `localhost` — it can point at a centralized server (see [DEPLOY.md](../en/DEPLOY.md)) over a private network (Tailscale/WireGuard):

```json
{ "base_url": "http://100.x.y.z:11434", "model": "llama3.2:3b" }
```

Nothing else changes: `project.json` still points at the same `llm_profile.config`, and the sanitize/PII Masking pipeline still always runs on the local machine, never on the remote server — see [PROJECT_JSON.md § Distributed architecture](PROJECT_JSON.md#distributed-architecture-remote-endpoints). Ready-to-try example: `config/models/ollama/llama3.2.3b-remote.json`.

## 7. Chat — `mova chat`

```
mova chat [project] [task]
```
Interactive chat with a local or Cloud model. With `[project]`, it loads the same full context `mova run` assembles, as the system message.

| In-chat command | What it does |
|---|---|
| `set -model <name>` | switches model without losing history |
| `/memory` | saves the last exchange to `memory.md` |
| `/budget` | generates `mova-budget-report.md` |
| `/diagram [export] [path]` | generates a visual diagram |
| `/save "path"` / `-d "folder"` / `-c "path"` | save response / create folder / code only |
| `/tools` | lists commands and available tools |
| `exit` / `quit` | ends the session |

```bash
mova chat
> set -model llama3.1
> hi, review the auth module
```

You can also just ask for things in your own words, no commands: *"Generate the audit report at docs/audit.pdf"* creates the file directly — works in English and Spanish, in the same session.

## 8. `save` / `/save` — create or edit files

```
/save ["path/file.ext"]
/save -d "path/folder"
/save [-all | -range N-M] [-c | -text] ["path"]
/save [-overwrite | -no-overwrite | -append] "path"
```
A single entry point: give it `path` (or `directory`) + content, and `mova` picks the right generator based on the extension.

| Flag | Saves... |
|---|---|
| *(none)* | the model's last response |
| `-all` | the whole conversation, as a transcript |
| `-range N-M` | exchanges N through M (1-indexed, inclusive) |
| `-c` / `-text` | code blocks only / text only |

| Flag | Existing file |
|---|---|
| *(none)* / `-overwrite` | overwrites |
| `-no-overwrite` | fails instead of overwriting |
| `-append` | appends to the end |

These combine freely: `/save -all -c "snippets.go"`. Natural language equivalent: *"Overwrite report.pdf"* = `-overwrite`; *"Don't overwrite report.pdf"* = `-no-overwrite`.

**Via MCP/HTTP** — the same `save` tool, with `path`/`directory` + `content`, or `history` + `mode` (`all`/`range`/last exchange):
```bash
curl -X POST http://localhost:3000/mcp -d '{"jsonrpc":"2.0","id":1,"method":"tools/call",
  "params":{"name":"save","arguments":{"project":"my-project","path":"reports/checkout.pdf","content":"Finding 1: ..."}}}'
```

| Extension | How `content` is interpreted |
|---|---|
| `.md`, `.txt`, `.json`, `.yml`, `.csv`, code | written as-is |
| `.docx` | interpreted as Markdown (`#`, `##`, bold, paragraphs) |
| `.pdf` | HTML as-is, or plain text wrapped in `<p>` |
| `.xlsx` | typed `sheets_data` JSON, or plain CSV/TSV |
| `.svg` | valid SVG code |

## 9. `delete` / `/delete`

```
/delete "path1" ["path2" ...]
```
Deletes files and directories, always with prior confirmation. A trailing `/` (`"logs/"`) signals a directory; without it, `mova` checks the filesystem.

```text
> /delete "a.txt" "logs/"
Delete "a.txt"? (Y/N)
```

Via MCP/HTTP: call once without `confirm` to get the exact prompt back, then again with `confirm:true` to execute.

## 10. `workflow.md`

```
read workflow.md
workflow.md <project> [task]
```
`workflow.md` is never opened directly: it first resolves the project, builds its context, and validates it against Budget — only if that passes does the file load.

| Where | What to say/send |
|---|---|
| `mova chat` | `workflow.md my-project` |
| MCP client | ask it to read the workflow — it calls `get_workflow` on its own |
| HTTP | `curl -X POST localhost:3000/workflow -d '{"project":"my-project"}'` |

If the context exceeds Budget, the file doesn't load.

## 11. Autonomous tool-calling from chat

```json
{ "tools": { "enabled": true, "allow": ["save", "read_file"] } }
```
With `tools.enabled`, the model itself can ask `mova` to perform a real action (save, read, edit) during the conversation, on any provider. `allow` restricts to a subset (`save`, `read_file`, `patch_file`, `read_document_layer`); without `allow`, all four are enabled. This is in addition to `/save`, which always works independently of this.

The protocol is plain text, not each provider's native function-calling API — one format (`<<<MOVA_TOOL_CALL>>>{"name":...,"arguments":...}<<<END_MOVA_TOOL_CALL>>>`) works identically for Ollama, Claude, GPT, or Gemini. Small local models (e.g. `lfm2.5-1.2b`) sometimes echo that same shape back as if it were their answer instead of a real call; `mova` detects and discards that residue automatically before showing the reply — in CLI, Chat, MCP, and HTTP alike — so a raw JSON header never shows up where the model's answer should be.

## 12. Office documents, media, text, and code

`save` picks the right generator based on the extension — no need to remember which tool handles which format. `.docx`, `.xlsx`, `.pdf` need no extra packages.

```text
> Generate a simple lease agreement at output/lease.docx
> Build a table of this month's expenses at output/expenses.xlsx
```

Images (`trigger_diffusion_image`): routes the prompt to a local diffusion server compatible with AUTOMATIC1111 — requires that server running separately, with a model installed.

Text and code (`.js`, `.py`, `.go`, `.sql`, `.yaml`... see [SUPPORTED_FORMATS.md](SUPPORTED_FORMATS.md)):

| Tool | What for |
|---|---|
| `read_file` / `write_file` | read / write a file |
| `patch_file` | replaces a single exact occurrence of `search` with `replace` |

`write_file`/`save` validate well-formed `.json`/`.xml`, real `.go` syntax, and consistent `.csv` columns before writing. `patch_file` rejects the change if `search` isn't found, or appears more than once.

## 13. MCP server — `mova mcp start`

```
mova mcp start --stdio
mova mcp start --port 3000
```
Exposes the `mova run` engine over MCP (JSON-RPC 2.0), so a client (Claude Desktop, Cursor) can request the context without any copy-pasting.

```json
{ "mcpServers": { "mova-context": {
  "command": "/path/to/mova", "args": ["mcp", "start", "--stdio"],
  "env": { "MOVA_PROJECT_ROOT": "/path/to/your/mova-context" }
}}}
```

| Tool | Equivalent to |
|---|---|
| `get_full_context` | `mova run [project] [task]` |
| `get_knowledge` | reading a specific agent/skill/prompt |
| `get_memory` / `get_memory_all` | `mova memory-read` |
| `get_workflow` | reading `workflow.md`, validating Budget first |
| `search_context` | `mova search` |
| `chat_completion` | `mova chat` |
| `save` / `delete_path` | create/edit/delete files |
| `estimate_budget` | `mova budget` |
| `generate_diagram` | `mova run --diagram` |
| `list_jobs` / `run_job` | `mova jobs list` / `run` |
| `list_agents` / `run_agent` | `mova agents list` / `run` |

Root resolution order: `MOVA_PROJECT_PATH` → `MOVA_PROJECT_ROOT` → current working directory → binary directory.

## 14. Environment variables

| Variable | Effect |
|---|---|
| `MOVA_DSN` | overrides `project.json.dsn` |
| `MOVA_PROJECT_ROOT` | extra starting point for the upward `workflow.md` search |
| `MOVA_PROJECT_PATH` | uses this path as root directly, no search |

`"repo"` in `project.json` accepts any absolute path — another Windows drive, a Linux mount point, a UNC path, WSL, or a Docker volume. `MOVA_PROJECT_ROOT` is the only thing you need to set separately, so `mova` can find its own root while standing outside it.


## 15. Tokenomics — `mova budget`

```
mova budget [project] [task] [--focus]
```
Estimates tokens and cost of a project's real context, per provider. All computation is local — it never calls an LLM or sends content off your machine. Generates `mova-budget-report.md`.

The report includes: tokenization and automatic deduplication, a breakdown by Agents/Skills/Prompt/Focus/Memory, full repo vs. focus comparison (with `--focus`), the configured limit, and historical accuracy against real Cloud providers (`mova-token-history.json`).

| | `budget.max_tokens` | `num_predict` |
|---|---|---|
| Limits | the context sent to the model | the model's response |
| Enforced by | `mova`, before sending | the provider |
| On exceed | stops the run, zero tokens spent | the provider truncates the response |

```json
{ "budget": { "max_tokens": 6000 } }
```

Pricing in `config/prices.json` (global, hot-reloaded):
```json
{ "providers": { "google": { "models": { "gemini-2.5-flash": { "input": 0.0003, "output": 0.0025 } } } } }
```

```bash
mova budget my-project my-task --focus
```

## 16. Token Firewall

Deterministic, AI-free stages that run automatically on every context assembly — `run`, `chat`, `jobs`, `agents`, MCP:

| Stage | What it does | Default |
|---|---|---|
| **Sanitizer** | collapses repetitive noise (near-identical logs, blank lines, duplicated headers) | on |
| **PII Masking** | pseudonymizes tokens shaped like personal data before sending | **off** |
| **Cache Layout Guard** | orders the prompt into a stable prefix + variable content, for provider prompt caching | on |
| **Circuit Breaker** | stops before sending if `max_tokens_per_run` or `max_monthly_usd` is exceeded | on |

Each stage toggles independently in `"budget"` in `project.json`.

```json
{ "budget": { "pii_masking": { "enabled": true },
  "circuit_breaker": { "max_tokens_per_run": 5000, "max_monthly_usd": 5.0, "on_exceed": "abort" } } }
```

The `mova budget` report always compares tokens/cost with the Token Firewall applied vs. without it.

## 17. Visual diagrams — `mova run --diagram`

```
mova run <project> --diagram [--export svg,png,pdf] [--path ./folder]
```
Generates a real image of a project's or multi-agent group's pipeline: sources → Context Compiler → Token Firewall → agents → jobs → final summary with real tokens/cost.

| Flag | Default | What it does |
|---|---|---|
| `--export` | `svg` | comma-separated list: `svg`, `png`, `pdf` |
| `--path` | current directory | output folder, created if it doesn't exist |

`<project>` can be a normal project, a full multi-agent group, or a single agent within one (`<group>/<agent>`).

```bash
mova run my-group --diagram --export svg,png,pdf --path ./diagrams
```

| Channel | How |
|---|---|
| Chat | `/diagram`, `/diagram png`, `/diagram svg,png ./output` |
| MCP | `generate_diagram` tool with `{"project":"..."}` |
| HTTP | `curl -X POST localhost:3000/diagram -d '{"project":"..."}'` |

## 18. Jobs — scheduled background execution

```
mova jobs list  [project]
mova jobs run   [project] [index|--all]
mova jobs start
```
Reads the `jobs` array from `project.json` and runs it — same engine regardless of channel. `jobs start` starts the scheduler daemon, checking every project once a minute.

```bash
mova jobs run my-project            # all, ignoring the cron
mova jobs run my-project 0          # just the job at index 0
```

With a multi-agent group, `[index]` is read as an agent name instead; to run a job by index within a single agent: `mova jobs run group/agent 0`.

## 19. Multi-agent — agent groups

```
mova agents list <group>
mova agents run  <group> [agent|--all]
```
A group lives under `projects/`; each agent is a normal project, orchestrated by a parent `config.json`.

```bash
mova agents run sales_online              # all, in sequence
mova agents run sales_online seller       # = mova run sales_online/seller
mova chat sales_online seller             # = mova chat sales_online/seller
```

`mova chat <group>` (no agent) lists the available agents instead of opening a chat.

## 20. Visual interface — `mova ui`

```
mova ui [project]
```
A terminal interface that gathers everything the commands in this guide already do behind a menu-driven navigation — it doesn't replace any command; it calls the same internal components.

`↑`/`↓` move · `enter` open/confirm/run · `/` search a list · `ctrl+f` search the open document · `esc` go back · `ctrl+s` save · `ctrl+c` quit.

**Main menu:** Chat · Projects · Workflow.md · Models · Logging · Multi-agent · Search · Logs. The same commands as `mova chat` work inside the TUI's chat.

```bash
mova ui my-group/my-agent
```

## 21. Installation

```
make install
go build -o mova ./src/cli
```
`make install` builds and copies the binary to `$(go env GOPATH)/bin/mova`. With that folder on your `PATH`, `mova` runs from any directory — it always searches upward for `workflow.md` (or uses `MOVA_PROJECT_ROOT`/`MOVA_PROJECT_PATH`).

To install with a graphical interface, no terminal needed: [`installers/README.md`](../../installers/README.md) — double-click installers for Windows, macOS, and Linux, compatible with `make install`.