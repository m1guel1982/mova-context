# COMMANDS — command guide

> Docs: [Español](COMMANDS.md) · [English](COMMANDS.en.md)

The CLI (`mova`) is a convenience layer — everything it does you can also ask a model to do by reading `workflow.md` directly. See [README.en.md](README.en.md#1-the-convention).

`mova` walks up directories automatically until it finds `workflow.md`, so it works from any subfolder of the repo. If there's only one project under `projects/`, `[project]` is optional — it's auto-detected.

## Index

1. [Quick reference](#1-quick-reference) — every command in one table
2. [Assembling context — `mova run`](#2-assembling-context--mova-run)
3. [Focus — working on one part of the project](#3-focus--working-on-one-part-of-the-project)
4. [Memory](#4-memory)
5. [Projects — list, init, search](#5-projects--list-init-search)
6. [Models and providers](#6-models-and-providers)
7. [Chatting with a local or Cloud model](#7-chatting-with-a-local-or-cloud-model)
8. [Creating files by talking: natural language in chat](#8-creating-files-by-talking-natural-language-in-chat)
9. [`save` — create or edit any file or directory](#9-save--create-or-edit-any-file-or-directory)
10. [Autonomous tool-calling from chat](#10-autonomous-tool-calling-from-chat)
11. [Office documents and media (PDF, Word, Excel, SVG, images)](#11-office-documents-and-media-pdf-word-excel-svg-images)
12. [Text and code files](#12-text-and-code-files)
13. [MCP server — `mova mcp start`](#13-mcp-server--mova-mcp-start)
14. [Environment variables](#14-environment-variables)
15. [Tokenomics — `mova budget`](#15-tokenomics--mova-budget)
16. [Global CLI installation](#16-global-cli-installation)

---

## 1. Quick reference

```text
mova run           [project] [task]         builds the context for the LLM
mova memory        [project] "reply"        saves the session to memory.md
mova memory-read   [project]                prints the active memory
  --all                                     include archived files
  --month 2024-01                           a specific archived month
mova memory-archive [project]               archives old entries
  --days N                                  days to keep active (default 30)
mova memory-clear  [project]                deletes ALL memory
  --archived                                deletes only archived months
  --keep-active                             deletes archive files, keeps memory.md
  --date 2024-06-15                         deletes a specific day
  --from 2024-06-01 --to 2024-06-30         deletes a date range
  --yes                                     skips the confirmation prompt
mova memory-config [project] [action] [value]
  enable | disable                          toggles auto-archiving
  days N                                    retention days (1, 10, 30, 90...)
  confirm true|false                        toggles delete confirmation

mova list                                   lists all projects
mova init          [name]                   creates a project
mova search        "query" [domain]         searches the knowledge base, no model needed

mova config        <provider>               sets the active provider (ollama, lmstudio...)
mova show          config [model]           shows the active provider, or a model's config
mova install       llama3.1,mistral         installs models (with a progress bar)
mova model-list                             lists installed models
mova remove        llama3.1,mistral         removes installed models

mova chat          [project] [task]         interactive chat with a local or Cloud model
  set -model <name>                         switches model without losing history
  /memory                                   saves the last exchange to memory.md
  /budget                                   generates mova-budget-report.md for the active project
  /save "path/file.ext"                     saves the last reply there (format by extension)
  /save -d "path/folder"                    creates only a directory
  /save -c "src/index.js"                   saves ONLY the source code blocks from the last reply
  /tools                                    lists available commands and tools
  exit | quit                               ends the session

mova budget        [project] [task]         estimates tokens and cost, 100% local
  --focus                                   compares the full repo vs. only what focus selects

mova mcp start                              starts the MCP server
  --port 3000                               as an HTTP server (default)
  --stdio                                   as a Stdio server (for Claude/Cursor)
```

---

## 2. Assembling context — `mova run`

Assembles agents + skills + prompt + memory + focus and prints it to stdout — ready to paste into a chat or send to an API.

```bash
mova run my-project review-auth
```

`task` is optional: if the project sets `"default_task"`, that's used; if the project declares exactly ONE task and neither is given, that one task is used automatically — no friction for the common case of one project, one task. With two or more tasks and neither an explicit one nor a `default_task`, Mova still can't guess which one and asks, listing the available tasks.

**The Budget gate always runs first, no matter which of the cases above applies** — even "just the project name, no task at all". If the assembled context (agents+skills+prompt+focus+memory) exceeds the project/task's configured `budget`, nothing is printed except the same ERROR/Suggestion block `mova budget` shows:

```text
$ mova run my-project
ERROR

Current context (128,400 tokens) exceeds the configured limit (100,000).

Suggestion:
Use --focus to reduce the included files.
```

This is exactly what MCP's `get_full_context` tool and `chat_completion` do too — see [§10](#10-mcp-model-context-protocol) — so whether the context is being read by `mova run`, `mova chat`, or an external model over MCP (Claude Console, Codex, Gemini, or any MCP client), the same Budget check runs before it, every time, with no extra step to remember.

If the task has `focus` (in `project.json` or global), that section is automatically appended at the end — see the next section.

---

## 3. Focus — working on one part of the project

`focus` (in `project.json`, global or per task) tells the engine to work only on certain files, folders, or symbols, instead of the whole repo. It works the same with or without the CLI: if a model reads `workflow.md` directly, the `## FOCUS` section of the spec explains exactly how to resolve it.

**Important:** `focus` is relative to `project.json`'s `"repo"` field, never to the `mova-context` root. If `task.focus` is defined, it **fully replaces** the project's global `focus` (the two lists are never merged).

### How each item matches — just like SQL `LIKE`

| Pass | SQL equivalent | When it's used |
|---|---|---|
| 1 — Exact | `WHERE name = 'CreateOrder'` (word-boundary) | Always tried first |
| 2 — Contains | `WHERE name ILIKE '%CreateOrder%'` | Only if pass 1 found nothing |

Case/accent-insensitive in both passes (`articulo 6` matches `Artículo 6`). It never uses an LLM — it's deterministic text search: same input, same result, every time.

### Supported item types

| Item in `focus` | What it resolves |
|---|---|
| `"manual.md"` | the full file, found by name anywhere in the repo |
| `"src/auth"` | the directory index |
| `"CreateOrder()"` | the function/method/class (`()` tells the engine it's code, not a file) |
| `"Article 6"` | the section of a legal/structured document (Title, Chapter, Section, Article, Clause) |
| `"## Some heading"` | a Markdown heading |
| `"table_name"` | the `CREATE TABLE ...;` definition in a `.sql` file |

### Example

```json
"tasks": {
  "review-order": {
    "prompt": "review-project",
    "focus": ["CreateOrder()", "manual.md", "Article 6"]
  }
}
```

```bash
mova run my-project review-order
```

If an item isn't found in either pass, it shows up as `not found: [item]` — it's never silently skipped.

### Whole-repo and directory-level focus

Besides individual files/symbols/sections, `focus` also accepts glob patterns and bare directory paths:

**The entire repository:**

```json
"focus": ["**/*"]
```

Activates the `GlobResolver`: walks every file and directory under `repo`, recursively. Useful when a task genuinely needs the whole codebase and you'd rather be explicit about it than leave `focus` unset:

```json
"tasks": {
  "revisar-backend": {
    "prompt": "review-project",
    "focus": ["**/*"]
  }
}
```

**Just the project root:**

```json
"focus": ["."]
```
or
```json
"focus": ["./"]
```

Activates the `DirectoryResolver` on `repo`'s root — a directory index of whatever's directly there, via the same resolver used for any other directory.

**Specific directories:**

```json
"focus": ["src/", "pkg/", "cmd/"]
```

Each path resolves through the `DirectoryResolver`. This is the usual way to keep heavy directories — `node_modules`, `vendor`, `.git`, `dist`, `build` — out of the assembled context: list only the directories that actually matter instead of `"**/*"`, and anything not listed is simply never walked.

### Checklist if `focus` comes out empty

1. Does `"repo"` point to a folder that exists and contains what you're looking for? (`focus` never searches outside `repo`)
2. Does the code symbol end with `()`?
3. Does the `task` have its own `focus`? If so, it replaces the global one.
4. Is your binary older than this fix? Rebuild: `go build -o mova ./src/cli`.

---

## 4. Memory

```bash
mova memory my-project "$(cat reply.txt)"
```

Extracts the ` ```memory ` block from a model's reply and appends it to `memory.md`. Next time you run `mova run my-project`, that memory shows up in the context automatically.

```bash
mova memory-read my-project --all
mova memory-read my-project --month 2024-01

mova memory-archive my-project --days 15     # moves old entries out of memory.md, grouped by month

mova memory-clear my-project --archived --yes   # asks for confirmation unless --yes is passed

mova memory-config my-project days 45
```

---

## 5. Projects — list, init, search

```bash
mova list                       # lists all projects
mova init my-project            # creates project.json (minimal template) + empty memory.md
mova search "authentication" software   # keyword search over agents/skills/prompts
```

`search` doesn't use any model — it's keyword search over the repository's knowledge.

---

## 6. Models and providers

`llm_profile` (in `project.json`) is the only thing that changes when you switch models or providers. Agents, skills, prompts, memory, and focus **never** change — the same `mova run` produces the same context no matter which model will read it.

```json
"llm_profile": { "type": "local", "provider": "ollama", "config": "llama3.2.3b" }
```

| Field | Values | What it's for |
|---|---|---|
| `type` | `"powerful"` (default) \| `"local"` | With `"local"` the engine adapts the format (numbered lists, an `INSTRUCTIONS:` prefix) so smaller models follow sequential instructions better. With `"powerful"` the content is delivered untouched. |
| `provider` | `"ollama"` \| `"google"` \| `"anthropic"` \| `"openai"` \| `"lmstudio"` \| any string | A subdirectory of `config/models/` — that's where all of this model's configuration comes from. |
| `config` | a filename, without `.json` | Points to `config/models/<provider>/<config>.json` — the ONE file holding connection (`base_url`, `api_key`, `timeout_seconds`) **and** inference parameters (`temperature`, `num_predict`, the real model tag) together. |

**A single source of truth.** `llm_profile.config` and the filename are the same string, by construction — there are no two identifiers that can drift apart (the classic bug was naming the file differently from Ollama's real tag, e.g. because Windows doesn't allow `:` in filenames).

```json
// config/models/ollama/llama3.2.3b.json — EVERYTHING in one file
{
  "provider": "ollama", "type": "ollama",
  "base_url": "http://localhost:11434", "timeout_seconds": 300,
  "model": "llama3.2:3b",
  "top_k": 40, "top_p": 0.9, "num_ctx": 4096,
  "temperature": 0, "num_predict": 512,
  "context_window": 131072, "repeat_penalty": 1.1
}
```


### Switching providers without touching anything else

```json
// Claude — Cloud, via API
"llm_profile": { "type": "powerful", "provider": "anthropic", "config": "claude-sonnet-4-6" }

// Gemini Flash — Cloud, via API
"llm_profile": { "type": "powerful", "provider": "google", "config": "gemini-2.5-flash" }

// Local Ollama
"llm_profile": { "type": "local", "provider": "ollama", "config": "llama3.2.3b" }
```

```bash
mova run my-project my-task > context.txt
ollama run llama3.2:3b < context.txt
```

### `config/models/` structure

```text
config/models/
├── active.json              ← pointer: {"provider", "config"} currently selected (never a data copy)
├── ollama/
│   ├── llama3.2.3b.json     ← connection + parameters for THIS model, all together
│   └── mistral.json
├── google/gemini-2.5-flash.json
├── anthropic/claude-sonnet-4-6.json
└── lmstudio/                ← provider selected, no model yet
```

Editing any `<model>.json` by hand hot-reloads: the next chat message (or the next MCP/HTTP call) already uses the new values, with nothing to restart.

```bash
mova config ollama                    # sets the active provider
mova show config                      # what provider/model am I using?
mova show config llama3.1             # full config for a specific model
mova install llama3.1,mistral,phi3    # downloads Ollama models (with progress)
mova model-list                       # models installed for the active provider
mova remove mistral                   # removes an installed model
```

`install`/`model-list`/`remove` use Ollama's native API. With LM Studio/vLLM/Cloud, you install or subscribe to the model through its own tool and just create its `.json`. `mova install`ing a new model copies the connection settings from any existing sibling model for that provider.

### Real Cloud providers (OpenAI, Anthropic, Google)

`config/models/<provider>/<config>.json` is the only place you need to touch. **OpenAI and Google Gemini** expose an `"openai-compatible"` endpoint (the same format Mova already used for LM Studio/vLLM):

```json
// config/models/openai/gpt-5.json
{
  "provider": "openai", "type": "openai-compatible",
  "base_url": "https://api.openai.com", "api_key": "sk-...",
  "model": "gpt-5", "temperature": 0.2, "num_predict": 1024
}
```

```json
// config/models/google/gemini-2.5-flash.json
{
  "provider": "google", "type": "openai-compatible",
  "base_url": "https://generativelanguage.googleapis.com/v1beta/openai",
  "api_key": "AIza...", "model": "gemini-2.5-flash",
  "temperature": 0.2, "num_predict": 1024
}
```

**Anthropic (Claude)** has different headers and response shape, so it uses its own native type:

```json
// config/models/anthropic/claude-sonnet-4-6.json
{
  "provider": "anthropic", "type": "anthropic",
  "base_url": "https://api.anthropic.com", "api_key": "sk-ant-...",
  "model": "claude-sonnet-4-6", "temperature": 0.2, "num_predict": 1024
}
```

Adding a new Cloud provider in the future is exactly this: a new `.json`, and if its API isn't OpenAI-compatible, a new implementation of the `Provider` interface (`models/provider.go`) — never touching `core`, `budget`, `cli`, or `mcp`. Mova calls these providers directly over HTTP, with no dependency on Claude Desktop, Claude Console, Codex, or Gemini CLI.

---

## 7. Chatting with a local or Cloud model

```bash
mova chat
> set -model llama3.1
✓ model switched to: llama3.1 (provider: ollama)
> hi, review the auth module
[llama3.1] ...
> set -model mistral
✓ model switched to: mistral (provider: ollama)
> keep going with that
[mistral] ...          # chat history is preserved across model switches
> exit
```

If you pass `[project]` (and optionally `[task]`), the chat loads the **same full context** that `mova run` builds (agents+skills+prompt+memory+focus) as the system message. Commands available inside chat, always:

| Command | What it does |
|---|---|
| `/memory` | saves the last exchange to `memory.md` |
| `/budget` | generates `mova-budget-report.md` for the active project |
| `/save "path/file.ext"` | saves the last reply there — format by extension (see [§9](#9-save--create-or-edit-any-file-or-directory)) |
| `/save -d "path/folder"` | creates only a directory |
| `/tools` | lists commands + the tools the model can invoke (if the project enabled them) |

### Via MCP / HTTP — the `chat_completion` tool

The same chat session is an MCP tool, and therefore also HTTP:

```bash
mova mcp start --port 3000
```

```bash
curl -X POST http://localhost:3000/mcp -H "content-type: application/json" -d '{
  "jsonrpc":"2.0","id":1,"method":"tools/call",
  "params":{"name":"chat_completion","arguments":{
    "model":"llama3.1","project":"my-project","task":"review-auth",
    "message":"what would you check first?"
  }}
}'
```

`model`, `project`, and `task` are all optional — without `project` it's an "empty" chat with that model; without `model` it uses whatever is active in `active.json`.

### How it all fits together

```text
config/models/<provider>/<config>.json ──► ConfigCache (hot-reloaded)
      (connection + parameters, one single file)
        ┌───────────────────────────────────┬───────────────────────────────┐
        ▼                                   ▼                               ▼
   mova chat (REPL)               chat_completion MCP tool             (same tool)
        │                                   │                          via HTTP /mcp
        └──────────────────────► models.Session.Send() ◄────────────────────┘
                                            │
                                  models.Provider (interface)
                                    ├─ ollamaProvider   → POST /api/chat
                                    ├─ openAIProvider   → POST /v1/chat/completions (LM Studio, vLLM, OpenAI, Gemini)
                                    └─ anthropicProvider → POST /v1/messages (Claude)
```

A single shared HTTP client (keep-alive + connection pooling) serves every call, built to handle high request volume.

---

## 8. Creating files by talking: natural language in chat

Besides `/save`, you can simply **ask for it in plain words**, in the same chat message — no command to memorize.

```text
> Generate the audit report at docs/audit.pdf
[the model writes the report, and the reply is saved automatically to docs/audit.pdf]

> Create the folder reports/2026 and generate the summary at reports/2026/summary.docx
[the folder is created, then the .docx is saved with the reply]

> Save the analysis to output/analysis.txt
[saved as-is, without needing a model call if you already have the text — see below]
```

### How it works under the hood

It's a **regex-based heuristic detector**, not a language model or a separate AI — it lives in `documents/nl_intent.go` and is shared by `mova chat` and `chat_completion` (MCP/HTTP). There's nothing mysterious about it: the message is split into clauses on the word **and**/**y**, and each clause is checked for a creation verb:

| English | Spanish |
|---|---|
| generate | genera, generar, generame/generáme |
| create | crea, crear, creame/creáme |
| save | guarda, guardar |
| make | — |

You can use these as many times as you like in one message, mixing English and Spanish in the same session.

If it finds the verb, it looks for **either** (a) a directory keyword plus a path, **or** (b) a path-shaped token ending in an extension (`.pdf`, `.md`, `.docx`, etc.):

- **Directory only** ("Create the folder X") → created instantly, no model call.
- **File only** ("Generate report.pdf") → the message does go to the model (it needs to generate the content), and the reply is saved automatically at that path.
- **Both in one message** ("Create the folder X and generate Y") → the folder first, then the file.

### Real limits (so you don't expect more than what's there)

- It needs the verb **and** (a folder keyword or a path with an extension) in the **same clause** — *"Generate something interesting about X"* triggers nothing, because there's no recognizable path or extension.
- It only understands **and**/**y** as a connector between two requests — it doesn't handle more complex constructions ("first... then...", commas, etc.).
- It's deterministic: same text, same result, every time — it doesn't interpret intent beyond the pattern.

`/save` still works exactly as always, for when this heuristic isn't enough or you want exact control over the path.

---

## 9. `save` — create or edit any file or directory

There used to be a different tool for each format (`generate_word_contract`, `generate_pdf_document`, `generate_excel_report`, `write_file`...), each with its own argument name for the content — easy to mix up, and the real cause of bugs like an empty `.docx`. `save` replaces all of that with a single entry point: you give it `path` (or `directory`) + `content`, and Mova internally picks the right generator based on the extension.

**From chat:**

```text
> Audit the checkout flow and put together the fixes report
[llama3.1] (the full report reply)

> /save "reports/checkout-fixed.md"
[Save] ✓ file saved: examples/example-law21719-repo/reports/checkout-fixed.md

> /save -d "carpeta  dos"
[Save] ✓ directory created: examples/example-law21719-repo/carpeta  dos
```

`/save -d "carpeta  dos"` creates exactly that directory name, including the double space, on Windows, Linux, and macOS alike — nothing about the name is normalized or trimmed.

By default `/save` uses the model's LAST reply as content. Several flags change WHICH text gets saved and how:

| Flag | Saves... |
|---|---|
| *(none)* | the last model response (unchanged default) |
| `-all` | the full conversation so far, as a `### You` / `### Assistant` transcript |
| `-range N-M` | exchanges N through M, 1-indexed inclusive, as the same transcript format |
| `-c` | only the fenced code blocks (` ``` `) from whatever was selected above — any language: Go, Python, Java, C#, Rust, JavaScript, TypeScript, Kotlin, SQL, YAML, JSON, XML, Bash, and more. No extra prose is saved. |
| `-text` | the opposite of `-c` — only the prose, with code blocks removed |

And separately, how an existing file is handled:

| Flag | Effect |
|---|---|
| *(none)* | overwrites an existing file (unchanged default) |
| `-overwrite "notes.txt"` | explicit overwrite — same as the default, useful for scripts that always want to state their intent |
| `-no-overwrite "notes.txt"` | fails with a clear message instead of overwriting if the file already exists |
| `-append "notes.txt"` | appends the selected content to the end of the existing file |

These combine freely: `/save -all -c "src/all_snippets.go"` saves every code block from the whole conversation; `/save -range 2-4 -text "summary.md"` saves only the prose from exchanges 2–4.

**Natural language works too**, for overwrite/no-overwrite, in either language:

```text
> Sobreescribe reporte.pdf
```

is exactly `/save -overwrite "reporte.pdf"`, and

```text
> No sobreescribas reporte.pdf
```

is exactly `/save -no-overwrite "reporte.pdf"`. Same detector, same result, whether you type the flag or say it naturally — see `documents/save_modifiers.go`.

**Via MCP / HTTP** — the same tool, reachable over stdio, HTTP, or `chat_completion`:

```bash
mova mcp start --port 3000
```

```bash
curl -X POST http://localhost:3000/mcp -H "content-type: application/json" -d '{
  "jsonrpc":"2.0","id":1,"method":"tools/call",
  "params":{"name":"save","arguments":{
    "project":"my-project",
    "path":"reports/checkout-fixed.pdf",
    "content":"Finding 1: ...\nFinding 2: ..."
  }}
}'
```

```json
{"name":"save","arguments":{"project":"my-project","directory":"reports/2026"}}
```

MCP/HTTP callers that hold their own conversation (a chat UI, a script) can use the exact same current/range/full-conversation/code-only/text-only selections chat's `/save` supports, by passing `history` instead of `content`:

```json
{"name":"save","arguments":{
  "history":[{"role":"user","content":"..."},{"role":"assistant","content":"..."}],
  "mode":"range",
  "range":"2-4",
  "code_only":true,
  "path":"src/snippets.go"
}}
```

`mode` is `"all"`, `"range"` (with `range: "N-M"`), or omitted for just the last exchange — identical selection logic to chat's `-all`/`-range` (see `documents/save_selection.go`; there is only one implementation, shared by every door).

**Via direct HTTP** — a dedicated `POST /save` endpoint, same JSON body, same internal logic with no duplication:

```bash
curl -X POST http://localhost:3000/save -H "content-type: application/json" -d '{
  "path":"reports/checkout-fixed.xlsx",
  "content":"item,risk\ngrouped consent,high"
}'
```

### `save` arguments

| Argument | What it's for |
|---|---|
| `path` | the file to create/edit — its extension picks the format. Resolved with the usual smart logic: an absolute path as-is, a relative path under the project's `repo`, or a bare name (searches for matches, asks if there's more than one) |
| `directory` | instead of `path` — just creates that folder (and any missing parents); no Writer is involved |
| `content` | text/Markdown/HTML/CSV — the internal Writer decides how to convert it (see table below) |
| `history` | instead of `content` — a JSON array of `{"role","content"}` messages to select from (see above) |
| `mode` | `"all"` \| `"range"` \| omitted (last exchange) — only used with `history` |
| `range` | `"N-M"`, 1-indexed inclusive — only used with `mode:"range"` |
| `code_only` / `text_only` | booleans — same as chat's `-c`/`-text` |
| `overwrite` | explicit `false` → `save` refuses the request instead of overwriting. Without this argument, the default (overwrite) behavior stays the same |
| `append` | `true` → appends the new content to the end of the existing file (text formats) |
| `project` | resolves relative paths against that project's `repo` |

### How `save` interprets `content` based on extension

| Extension | What `save` does |
|---|---|
| `.md`, `.txt`, `.json`, `.yml`/`.yaml`, `.xml`, `.csv`, and ~20 more code languages | written as-is |
| `.docx` | interpreted as Markdown (`#`/`##`/`###`, **bold**, paragraphs) |
| `.pdf` | if it already looks like HTML it's used as-is; if it's plain text or Markdown, it's wrapped in `<p>` tags before generating the PDF |
| `.xlsx` | accepts typed `sheets_data` JSON, **or** plain CSV/TSV text (a single "Sheet1" sheet, each cell auto-typed) |
| `.svg` | expects valid SVG code |

### Syntax highlighting

Whenever chat, MCP, or HTTP produce source code, any fenced code block left without a language tag gets one added automatically — the same detector (`documents.DetectLanguage`/`AutoTagCodeFences`) runs everywhere, so a Go block, a SQL query, or a YAML file all get highlighted correctly regardless of which door produced them. In the terminal, `mova chat` then renders that Markdown (including the highlighting) with `glamour`; over MCP/HTTP, the response is returned as correctly-tagged Markdown for whatever client renders it.

### `delete` — remove files and directories

One unified command removes files and directories, identically from chat, MCP, and HTTP — nothing is ever deleted without confirmation.

```text
> /delete "reports/old-draft.md"
Delete "old-draft.md"? (Y/N)
y
✓ deleted: examples/example-law21719-repo/reports/old-draft.md
```

```text
> /delete "a.txt" "b.txt" "logs/"
Delete "a.txt"?
(Y/N)
y
Delete "b.txt"?
(Y/N)
n
Delete "logs/"?
(Y/N)
y
✓ deleted: .../a.txt
⚠ not found, skipped: b.txt
✓ deleted: .../logs
```

A trailing `/` (`"logs/"`) is a hint that the target is a directory; without one, `/delete` asks the filesystem what it actually is rather than guessing.

**Via MCP / HTTP** — since there's no terminal to type Y/N into, confirmation is explicit: call once without `confirm` to get back the exact prompt text, then call again with `confirm:true` once the person has agreed — the same convention `chat_completion`'s `apply_edits` already uses for natural-language edits.

```json
{"name":"delete_path","arguments":{"paths":"a.txt,b.txt,logs/","project":"my-project"}}
```
```json
{"name":"delete_path","arguments":{"paths":"a.txt,b.txt,logs/","project":"my-project","confirm":true}}
```

```bash
curl -X POST http://localhost:3000/delete -H "content-type: application/json" -d '{
  "path":"reports/old-draft.md","confirm":true
}'
```

### `workflow.md` — Budget-gated execution

`workflow.md` is never opened directly. Saying any of the following resolves the project, builds its context, and validates the result against its configured Budget FIRST — only if that check passes is `workflow.md` actually loaded:

```text
lee workflow.md
leer workflow.md
ejecuta workflow.md
run workflow.md
execute workflow.md
workflow.md <project>
workflow.md <project> <task>
```

**Simplest possible usage, one line, same everywhere:**

| Where | What to say/send |
|---|---|
| `mova chat` (this CLI) | type `workflow.md my-project` |
| Claude Console, Codex, Gemini, or any MCP client | ask it to read the workflow — the model calls the `get_workflow` tool with `{"project":"my-project"}` on its own; no special phrasing needed on your end |
| HTTP | `curl -X POST http://localhost:3000/workflow -d '{"project":"my-project"}'` |

One call, one door, always Budget-checked first — there's no separate "check the budget, then load the file" dance to remember on any of them.


```text
> workflow.md my-project
[Project] Loading project configuration...
[Project] Using configured provider...
[Context] Building context...
[Dedup] No duplicates found.
[Focus] No focus configured — using the full agents/skills/prompt context.
[Workflow] Loaded /path/to/workflow.md (4,102 tokens).

(workflow.md's content, rendered)
```

If the estimated context (agents + skills + prompt + focus + `workflow.md` itself) exceeds the project/task's configured Budget, `workflow.md` is **not** loaded — the same ERROR/Suggestion block `mova budget` already shows is printed instead:

```text
[Project] Loading project configuration...
[Project] Using configured provider...
[Context] Building context...
[Dedup] Removed 3 duplicated paragraph(s) (512 chars).
[Focus] No focus configured — using the full agents/skills/prompt context.

ERROR

Current context (128,400 tokens) exceeds the configured limit (100,000).

Suggestion:
Use --focus to reduce the included files.
```

The same pipeline runs from MCP (`get_workflow` tool) and HTTP (`POST /workflow`):

```json
{"name":"get_workflow","arguments":{"project":"my-project","task":"revisar-backend"}}
```
```bash
curl -X POST http://localhost:3000/workflow -H "content-type: application/json" -d '{
  "project":"my-project"
}'
```

Which `workflow.md` file gets used comes from `project.json`'s `workflow_path` (see [PROJECT_JSON.md](PROJECT_JSON.md)); once configured, Mova always uses exactly that file and never searches for another.



Add this to `project.json` so the model itself (local Ollama, Gemini, Claude, GPT — any of them) can ask Mova to perform a real action during the conversation, instead of just describing in text what it would do:

```json
{ "tools": { "enabled": true } }
```

With that, `mova chat` and `chat_completion` append a simple plain-text protocol to the system message, which works the same way for any provider (it doesn't rely on native "function calling", which not every provider supports the same way, and which a small model like `llama3.2:3b` often handles poorly): the model replies with `<<<MOVA_TOOL_CALL>>> {"name":"save", "arguments":{...}} <<<END_MOVA_TOOL_CALL>>>`, Mova actually executes it, and feeds the real result back as a new turn so it can finish its reply.

```json
{ "tools": { "enabled": true, "allow": ["save", "read_file"] } }
```

`allow` is optional — restricts to a subset (`save`, `read_file`, `patch_file`, `read_document_layer`); if omitted, all four are enabled. This is **in addition to** `/save`: `/save` always works without depending on `tools.enabled` or the model cooperating (it's the deterministic path); `tools.enabled` is for when you want the model itself to decide, autonomously, when to create or edit something.

---

## 11. Office documents and media (PDF, Word, Excel, SVG, images)

With `save` you don't need to remember which tool generates which format — the extension in `path` is enough. No extra packages required: `.docx`, `.xlsx`, and `.pdf` are written by hand using Go's standard library. Only `trigger_diffusion_image` needs a separate diffusion server.

### Simple, natural-language examples

```text
> Generate a simple lease agreement at output/contract.docx
[Save] ✓ Word document generated: output/contract.docx

> Put together a table of this month's expenses at output/expenses.xlsx
[Save] ✓ spreadsheet generated: output/expenses.xlsx

> Write a one-page executive summary at reports/summary.pdf
[Save] ✓ PDF generated: reports/summary.pdf

> Save the meeting notes to notes/meeting-july-23.txt
[Save] ✓ file saved: notes/meeting-july-23.txt
```

### MCP/HTTP examples

Generating a Word document with `save` and reading it back:

```bash
mova mcp start --port 3000
```

```bash
curl -X POST http://localhost:3000/mcp -H "content-type: application/json" -d '{
  "jsonrpc":"2.0","id":1,"method":"tools/call",
  "params":{"name":"save","arguments":{
    "path":"output/contract.docx",
    "content":"# Contract\n\nThis is a **test** paragraph."
  }}
}'
```

```bash
curl -X POST http://localhost:3000/mcp -H "content-type: application/json" -d '{
  "jsonrpc":"2.0","id":2,"method":"tools/call",
  "params":{"name":"read_document_layer","arguments":{"filename":"output/contract.docx"}}
}'
```

An Excel sheet from plain CSV (no typed JSON needed if you don't need it):

```json
{"name":"save","arguments":{"path":"output/report.xlsx","content":"Item,Amount\nCoffee,4.5\nTea,3.2"}}
```

Or with the usual typed `sheets_data` (avoids type ambiguity):

```json
{"name":"generate_excel_report","arguments":{
  "filename":"output/report.xlsx",
  "sheets_data":{"Expenses":[
    [{"type":"string","value":"Item"},{"type":"string","value":"Amount"}],
    [{"type":"string","value":"Coffee"},{"type":"number","value":4.5}]
  ]}
}}
```

All tools resolve `path`/`filename` relative to the project's `repo` (if `project` is passed).

### Installing required packages

```bash
# .docx, .xlsx, .pdf and .svg: NO extra dependency —
# generated with Go's standard library.
go build -o mova ./src/cli
```

`read_document_layer` over `.pdf` is best-effort (extracts text from FlateDecode streams); scanned PDFs or exotic encodings might not return text.

### Local model for images (`trigger_diffusion_image`)

This tool doesn't generate images itself: it routes the prompt to a local diffusion server compatible with the **AUTOMATIC1111** API (`/sdapi/v1/txt2img`), configured in `config/models/diffusion/config.json` — a separate file, because a diffusion server has no inference parameters like `temperature`/`num_predict` to merge in. You need that server running separately, with a diffusion model installed (Stable Diffusion 1.5, SDXL, via AUTOMATIC1111 or ComfyUI in API mode). Mova only makes the HTTP call and saves the resulting PNG.

---

## 12. Text and code files

Three tools cover plain text, config, and source code — `.js`, `.ts`, `.py`, `.go`, `.cs`, `.java`, `.php`, `.rb`, `.rs`, `.c`, `.cpp`, `.h`, `.kt`, `.swift`, `.sh`, `.html`, `.css`, `.sql`, `.csv`, `.toml`, `.ini`, `.env`, `.log`, and more (full list in [SUPPORTED_FORMATS.md](SUPPORTED_FORMATS.md)):

| Tool | What it's for 
|---|---|---|
| `read_file` | reads a file
| `write_file` | writes a file 
| `patch_file` | replaces one single exact occurrence of `search` with `replace`

Requesting an unsupported extension (`.exe`, `.bin`) returns:

```
Unsupported file type: .exe. Supported extensions: .c, .cpp, .cs, .css, .csv, ...
```

`write_file`/`save` validate `.json`/`.xml` (well-formed), `.go` (real syntax via `go/parser`), and `.csv` (consistent columns) before writing. `patch_file` rejects the change if `search` doesn't appear, or appears more than once — it never risks an ambiguous edit.

### Example — "create a file with such and such" from chat

If you ask *"create a NOTES.md in my project with the current status"*, the assistant resolves the path against the project's `repo` and calls `save` — the file shows up in the real directory, not in an ephemeral chat:

```bash
curl -X POST http://localhost:3000/mcp -H "content-type: application/json" -d '{
  "jsonrpc":"2.0","id":1,"method":"tools/call",
  "params":{"name":"save","arguments":{
    "project":"my-project","path":"NOTES.md",
    "content":"# Project notes\n\nStatus: in progress"
  }}
}'
```

```json
{"name":"patch_file","arguments":{
  "project":"my-project","filename":"NOTES.md",
  "search":"Status: in progress","replace":"Status: complete"
}}
```

```json
{"name":"read_file","arguments":{"project":"my-project","filename":"NOTES.md"}}
```

Same pattern for `.json`, `.yml`, `.xml`, `.docx`, `.pdf`, or `.xlsx` — only `path`/`filename` and `content` change; `save` picks the right generator on its own. If chat gives you an absolute path (e.g. `E:/other-projects/README.md`), all these tools use it directly, without going through `repo`.

---

## 13. MCP server — `mova mcp start`

The same engine behind `mova run`, exposed over the MCP protocol (JSON-RPC 2.0) — so a client (Claude Desktop, Cursor) can request the context on its own, with nothing to copy or paste.

**Stdio mode** (used by Claude Desktop / Cursor):

```bash
mova mcp start --stdio
```

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

**HTTP mode** (for curl/Postman or your own backend):

```bash
mova mcp start --port 3000
```

```bash
curl -X POST http://localhost:3000/rpc -H "content-type: application/json" -d '{
  "jsonrpc":"2.0","id":1,"method":"tools/call",
  "params":{"name":"get_full_context","arguments":{"project":"my-project","task":"review-auth"}}
}'
```

### Tools available via MCP

| Tool | Equivalent to |
|---|---|
| `get_full_context` | `mova run [project] [task]` |
| `get_knowledge` | reading a specific agent/skill/prompt |
| `get_memory` | `mova memory-read [project]` |
| `get_memory_all` | `mova memory-read [project] --all` |
| `get_workflow` | reading `workflow.md`, but only after resolving the project and validating its Budget first (see [§9](#9-save--create-or-edit-any-file-or-directory)) |
| `search_context` | `mova search "query" [domain]` |
| `chat_completion` | `mova chat [project] [task]` |
| `save` | create/edit any file or directory (see [§9](#9-save--create-or-edit-any-file-or-directory)) |
| `delete_path` | delete one or more files/directories, with confirmation (see [§9](#9-save--create-or-edit-any-file-or-directory)) |
| `estimate_budget` | `mova budget [project] [task]` |

### Root resolution for MCP clients

MCP clients (Claude Desktop, Cursor) launch `mova` from a directory that usually isn't your project — that's why the example above sets `MOVA_PROJECT_ROOT`. Resolution order: `MOVA_PROJECT_PATH` (direct) → `MOVA_PROJECT_ROOT` (upward search from there) → current working directory → binary's directory.

---

## 14. Environment variables

```bash
MOVA_ADAPTER=db MOVA_DSN=postgres://user:pass@host/db mova run my-project
```

| Variable | Effect |
|---|---|
| `MOVA_ADAPTER` | Overrides `project.json.adapter` (`file` / `db`) |
| `MOVA_DSN` | Overrides `project.json.dsn` |
| `MOVA_PROJECT_ROOT` | Extra starting point for the upward `workflow.md` search |
| `MOVA_PROJECT_PATH` | Uses this path as the root directly, with no search |

---

## 15. Tokenomics — `mova budget`

Estimates how many tokens a project's real context would use (agents + skills + prompt + focus + memory — the same thing `mova run` assembles) and how much it would cost across every provider in `config/prices.json`. **The whole calculation is local**: it doesn't call any LLM or external API, it doesn't send a single line of the project off your machine, it uses no database, and it never stores prompts or content anywhere.

```bash
mova budget my-project
mova budget my-project my-task --focus
```

Generates `mova-budget-report.md` (configurable path via `\"budget_path\"` in `project.json` — see [PROJECT_JSON.md](PROJECT_JSON.md); defaults to `projects/<project>/mova-budget-report.md`) — always in simple English, so whoever pays the bill understands it regardless of the rest of Mova Context's language. Reachable identically from the CLI, MCP (`estimate_budget`), and the chat REPL (`/budget`):

```json
{"name":"estimate_budget","arguments":{"project":"my-project","task":"my-task","focus":"true"}}
```

The report includes, in this order:

- **Tokenization** — which tiktoken-go encoding was used.
- **Deduplication** — how many identical paragraphs were removed and how many tokens that saved.
- **Token & Cost Breakdown** — one row per Agents/Skills/Prompt/Focus/Memory/Overhead, in tokens and USD per provider, plus the total.
- **Context Optimization** — only with `--focus`: the full unfiltered repo vs. only what `focus` selects.
- **Budget Limit** — only if `project.json` defines a `"budget"`.
- **Historical Token Accuracy** — the feedback loop (see below).
- **Important** — the reminder that this is an estimate, not an invoice.

### Automatic deduplication

Mova deduplicates identical paragraphs across the **entire** assembled context — not just inside `focus`, but across Agents+Skills+Prompt+Focus+Memory together. It's never a rewording or a summary — only identical text (whitespace-normalized), never code/SQL/JSON:

```text
[Dedup] Removed 3 duplicated paragraphs (~450 tokens saved).
```

Shows up the same way in `mova chat`, in `chat_completion`, and in the report's "Deduplication" section — it runs on every context assembly, with nothing to configure.

### Two limits that sound alike but aren't the same

| | `budget.max_tokens` (`project.json`) | `num_predict` (`config/models/<provider>/<config>.json`) |
|---|---|---|
| Limits | the **assembled context** — what's sent TO the model | the model's **reply** — what it generates back |
| Who enforces it | Mova, BEFORE sending anything | the provider itself, as a request parameter |
| If it's exceeded | Mova stops execution with an error, zero tokens spent | the provider simply cuts the reply short there |

```json
// project.json
{ "budget": { "max_tokens": 6000 } }
```

```json
// config/models/google/gemini-2.5-flash.json
{ "num_predict": 1024 }
```

### Budget as a rule, not a suggestion

A limit in `project.json` (or per task, which overrides the project's) is validated **before sending any context** to a model — from `mova chat`, `chat_completion`, and therefore HTTP, always the same way:

```json
{ "budget": { "max_tokens": 8000 } }
```

```text
ERROR
Current context (14,250 tokens) exceeds the configured limit (8,000).
Suggestion: Use --focus to reduce the included files.
```

`mova budget` (the report, which never sends anything to any model) only shows it as informational under "Budget Limit" — the actual execution stop happens in `mova chat`/`chat_completion`, before a single real token is spent.

### Feedback loop — closing the circle with reality

Every time `mova chat` or `chat_completion` send the context to a **real Cloud provider** (OpenAI, Anthropic, Google) and that provider reports how many tokens it actually counted, Mova accumulates that difference in `mova-token-history.json` (next to `project.json` by default, or at the path set by `"token_history_path"` in `project.json`; see [PROJECT_JSON.md](PROJECT_JSON.md)). The file **only** stores two numbers per provider — never prompts, replies, or content:

```json
{
  "anthropic": { "total_local_tokens": 120000, "total_api_tokens": 122760 },
  "google": { "total_local_tokens": 85000, "total_api_tokens": 85150 }
}
```

`mova budget` reads this file and computes `(API - Local) / Local * 100` per provider, showing the average deviation in "Historical Token Accuracy" — a provider with no data yet shows `No historical data`, never an error. The more real calls made, the more accurate the number gets for **that specific project** — your own calibration, not a generic benchmark.

### Counting tool

All token counting is done with **[tiktoken-go](https://github.com/tiktoken-go/tokenizer)**, embedded (no network calls), the same tokenizer OpenAI publishes and uses in its own API. For OpenAI models, the count is usually exact. For Claude and Gemini there's no official public local tokenizer, so the same encoding is reused as an approximation — the real-world gap tends to be small, and the feedback loop above narrows it over time, per project.

### Configuring prices (`config/prices.json`)

`config/prices.json` holds **only** model prices — shared, global configuration for every project. Where each project writes its own `mova-budget-report.md` or `mova-token-history.json` is project-specific configuration, and lives exclusively in that project's `project.json` (`budget_path`, `token_history_path` — see [PROJECT_JSON.md](PROJECT_JSON.md)), never here.

```json
{
  "currency": "USD",
  "exchange_rate_clp": 950,
  "unit": "per_1k_tokens",
  "providers": {
    "google": { "models": { "gemini-2.5-flash": { "input": 0.0003, "output": 0.0025 } } }
  }
}
```

Hot-reloaded (same mechanism as `config/models/`). Adding a new provider or model is just JSON, never code. The example values need to be updated with each provider's real, current pricing.

### Full example — Gemini Flash + budget + tools + real files

`projects/example-gemini-flash/` brings all of this together in a single `project.json`:

```json
{
  "project": "example-gemini-flash",
  "repo": "examples/example-gemini-flash-repo",
  "default_task": "review-backend",
  "agents": { "domain": "base", "use": ["backend-dev", "security-architect"] },
  "skills": { "domain": "base", "use": ["api-security"] },
  "tasks": {
    "review-backend": {
      "prompt": "review-project",
      "variables": { "PROJECT": "backend-api", "REVIEW_TYPE": "full" },
      "focus": ["server.js"]
    }
  },
  "llm_profile": { "type": "powerful", "provider": "google", "config": "gemini-2.5-flash" },
  "budget": { "max_tokens": 6000 },
  "tools": { "enabled": true }
}
```

```bash
mova config google
mova chat example-gemini-flash review-backend
> Audit server.js and prioritize the findings
[gemini-2.5-flash] (finds the hardcoded secret, missing validation, the unauthenticated endpoint...)
> /save -d "reports"
[Save] ✓ directory created: examples/example-gemini-flash-repo/reports
> /save "reports/backend-audit.md"
[Save] ✓ file saved: examples/example-gemini-flash-repo/reports/backend-audit.md
> exit
```

Before sending anything to Gemini, `budget.max_tokens: 6000` was already validated against the real context (`mova budget example-gemini-flash review-backend` shows it without spending a token: in this example it's ~1,600, well under the ceiling). After the first real call, `mova-token-history.json` starts accumulating Gemini's deviation from the local estimate, visible on the next `mova budget` run.

---

## 16. Global CLI installation

```bash
make install
```

Builds and copies the binary to `$(go env GOPATH)/bin/mova` — the same folder `go install` uses on Linux, macOS, and Windows. With that folder in your `PATH`, `mova` runs from any directory. It never depends on where the binary lives: it always searches upward for `workflow.md` from the current directory (or from `MOVA_PROJECT_ROOT`/`MOVA_PROJECT_PATH`), never storing absolute paths.

```text
ERROR
No Mova project was found (looking for workflow.md, the project root marker).
Suggestion: Run "mova init" or move to a Mova project directory.
```

```bash
go build -o mova ./src/cli
```

No special editions or build flags — a single binary, every command in this guide.
