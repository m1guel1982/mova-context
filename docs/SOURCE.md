# Mova Context — Architecture & Technical Reference

**One binary. One behavior.** `mova` ships as a single Go executable with no build tags, no editions, no `-tags premium` variant. Every capability described below is reachable identically from four doors — **CLI**, **Chat (REPL)**, **MCP** (stdio/HTTP), and **HTTP REST** — because every door calls into exactly the same underlying function. There is never a second, door-specific implementation of business logic.

---

## 1. System Map

```text
src/
├── core/          Engine — zero external dependencies (stdlib only)
│   └── focus/       Focus Resolution Engine (Extension Point 2)
├── dedup/         Exact-paragraph deduplication (shared leaf package)
├── adapters/      Alternate storage backends (Postgres/MongoDB)
├── documents/     Save Service · Delete Service · Office formats (docx/xlsx/pdf/svg) · Path Resolution
├── sanitize/      Token Firewall: Sanitizer, PII Masking
├── budget/        Token/cost estimation, Budget gate, Token Firewall pipeline, Feedback Loop
├── models/        LLM Providers — local + Cloud, single source of truth per model
├── jobs/          Job Engine — cron scheduler, unattended actions
├── orchestrator/  Multiagent orchestrator — groups of projects run as agents
├── logging/       Opt-in structured logging (disabled by default)
├── diagram/       Renders a project's real execution pipeline as SVG/PNG/PDF
├── cli/           `mova` command — thin dispatcher + Terminal UI (`mova ui`)
├── mcp/           MCP JSON-RPC layer — tools/list, tools/call
├── http/          HTTP transport — thin wrapper over mcp.Process()
└── runtime/       FindRoot() / AutoDetect() — shared bootstrapping
```

| Layer | Dependency discipline |
|---|---|
| `core/` | Standard library only — the engine must stay trivially portable |
| `adapters/` | The only place a third-party driver (SQL, etc.) is allowed to live |
| `models/` | Standard library only, one shared `http.Client` — built for high call volume without degrading |
| `documents/` | `.docx`/`.xlsx` hand-written as OOXML (ZIP of XML) via `archive/zip`; `.pdf` as raw PDF 1.4 objects — no office-format or rendering-engine dependency |

---

## 2. Core Engine & Execution Pipeline

```
core.BuildContext(adapter, root, projectName, taskName)
        │
        ├─ 1. Adapter.GetProject(name)          — read project.json
        ├─ 2. ResolveTaskName(proj, taskName)    — explicit → default_task → sole task → ""
        ├─ 3. resolve agents/skills/prompt        — domain + i18n/[lang] + fallback "en"
        ├─ 4. inject variables                    — project-level, then task-level overrides
        ├─ 5. append memory.md (if present)
        ├─ 6. resolve `focus` (if task/project declares it) — see §3.2
        ├─ 7. dedup.Paragraphs across Agents→Skills→Prompt→Focus→Memory (one shared "seen" map)
        ▼
finished context (string)
```

`mova run`, the MCP `get_full_context` tool, and the HTTP transport all call **exactly this function** — no second code path exists.

**`ResolveTaskName(proj, taskName)`** is the single place task selection happens: explicit name → `proj.DefaultTask` → the project's one task if it declares only one → `""`.

### Adapter interface (storage abstraction)

```go
type Adapter interface {
    GetKnowledge(kind, domain, lang, name string) (string, error)
    GetProject(name string) (*Project, error)
    ListProjects() ([]ProjectSummary, error)
    GetMemory(project string) (string, error)
    GetMemoryAll(project string) (string, error)
    AppendMemory(project, entry string) error
    ArchiveMemory(project string, keepDays int) error
    DeleteMemory(project string, req MemoryDeleteRequest) (int, error)
    Search(query, domain string) ([]SearchResult, error)
}
```

The engine never knows whether data comes from disk (`core.FileAdapter`, default) or a database (`adapters.DBAdapter`). Deduplication (`dedup.Paragraphs`) lives in its own dependency-free leaf package because `core` and `core/focus/render` cannot import each other, yet both need it.

---

## 3. Architecture Extension Points

The codebase has four Open/Closed extension seams. Each follows the same rule: implement an interface, register it, and nothing upstream changes.

### 3.1 Adapters — storage backend

| Step | Action |
|---|---|
| 1 | Implement `core.Adapter` under `adapters/` (e.g. MySQL, SQLite, Redis) |
| 2 | Wire it in `cli/adapter_select.go` — the only file allowed to import both `core` and `adapters` |
| 3 | `core/` never changes |

### 3.2 Focus Resolvers — targeted context instead of the whole repo

`focus` (`project.json` key, or `workflow.md`'s `## FOCUS` section) lets a task target specific files, symbols, or document sections. Resolution is a **deterministic cascade — no LLM call, no probabilistic matching, same input always produces the same output.**

```go
type Resolver interface {
    Match(ctx Context, target string) bool
    Resolve(ctx Context, target string) ([]ContextBlock, error)
}
```

`Engine.Resolve` tries each registered `Resolver` in priority order; `Resolve` may decline (`ErrNotFound`) without breaking the cascade.

| Resolver (priority order) | Handles |
|---|---|
| `FileResolver` / `DirectoryResolver` | Exact file or folder paths |
| `JSONResolver` | A node inside a `.json` file |
| `SQLResolver` | A `CREATE TABLE ...;` definition by table name |
| `CodeSymbolResolver` | A function/class declaration (brace/indent matching) |
| `MarkdownResolver` | A heading/section (`## Some Section`) |
| `LegalResolver` | Hierarchical legal documents (Title/Chapter/Article/Clause) |
| `MemoryResolver` | Dated chronological blocks in `memory.md`-style files |
| `FallbackResolver` | Bounded excerpt around the first match — last resort, never a whole file |

All matching goes through `focus.LikeContains` — case- and accent-insensitive.

**To add a resolver:** implement `Match`/`Resolve` under `core/focus/resolvers/`, register it in `DefaultResolvers()` at the right cascade position. Nothing else changes.

`.pdf`/`.docx` focus files are read through `documents.ReadDocumentLayer` (real text extraction), never raw `os.ReadFile` — a fixed bug where binary container bytes were dumped straight into the assembled context.

### 3.3 Model Providers — local + Cloud inference

```go
type Usage struct { PromptTokens, CompletionTokens int }

type Provider interface {
    Chat(ctx context.Context, model string, mc *ModelConfig, messages []ChatMessage) (string, Usage, error)
}
```

`NewProvider(mc)` dispatches purely on `ModelConfig.Type`:

| `Type` | Shape | Examples |
|---|---|---|
| `"ollama"` | Native `/api/chat` | Ollama, and the default for any unrecognized local backend |
| `"openai-compatible"` | `/v1/chat/completions` | LM Studio, vLLM, TGI, real OpenAI, real Google Gemini |
| `"anthropic"` | `/v1/messages` | Claude — own headers and system-prompt field |
| `"google"` / `"gemini"` | Google's native REST API | — |

`models.Session` only talks to the `Provider` interface — it neither knows nor cares whether the engine behind it is local or Cloud. Every implementation returns real `Usage` when the server reports it, which feeds the **Feedback Loop** (§6.3).

**To add a provider type:** implement `Provider` under `models/`, add one `case` in `NewProvider`. `Session`, CLI, MCP, HTTP stay unaware.

### 3.4 Save Writers — output-format extension point

```go
type IFileWriter interface { Write(path string, opts SaveOptions) error }
```

`WriterFactory` is a plain `map[string]IFileWriter` keyed by lowercased extension — no `if`/`switch` chain. Each generator self-registers from its own `init()`: `docxWriter`, `pdfWriterAdapter`, `xlsxWriter`, `svgWriter`, `textWriter` (one entry per allow-listed extension).

**To add an output format:** write one `IFileWriter`, call `RegisterWriter(".ext", myWriter{})` from `init()`. `Save`, `/save`, the `save` MCP tool, and `POST /save` never change.

---

## 4. Unified Save Service & Natural-Language File Creation/Editing

### 4.1 `save` — one call replaces per-format tools

Before unification, creating a Word doc, PDF, Excel report, SVG, or text file each meant a differently-named tool with a differently-named content argument (`markdown_content`, `layout_html_css`, `sheets_data`, `content`) — a common source of empty-file bugs.

```go
documents.Save(root, documents.SaveRequest{
    Path, Directory, Content, Append, Overwrite, Repo,
})
```

Reachable identically from all three doors:

| Door | Entry point |
|---|---|
| Chat | `/save "docs/report.md"` (last reply, format from extension); `/save -d "docs/backend"` (directory only) |
| MCP | `"save"` tool — `{path\|directory, content, append, overwrite, project}` — same tool the model calls autonomously (§5) |
| HTTP | `POST /save` — repackaged internally as a `tools/call`, zero duplicated logic |

Legacy per-format tools (`create_directory`, `write_file`, `generate_word_contract`, `generate_pdf_document`, `generate_vector_graphic`, `generate_excel_report`) remain fully functional for backward compatibility — `save` is additive.

| Tool | Does |
|---|---|
| `save` | **Unified entry point** — extension picks the generator automatically |
| `create_directory` *(legacy)* | Recursive `mkdir -p`, cross-platform |
| `read_file` | Reads any plain-text/source file, no restriction |
| `write_file` *(legacy)* | Create/overwrite — allow-listed extensions only; `.json`/`.xml`/`.go`/`.csv` validated before writing |
| `patch_file` | Surgical replace of one exact, unique occurrence — rejects if missing/ambiguous |
| `read_document_layer` | Extracts plain text from `.docx`/`.xlsx`/`.pdf` |
| `generate_word_contract` *(legacy)* | Markdown → real `.docx` |
| `generate_excel_report` *(legacy)* | Typed `sheets_data` JSON → real `.xlsx` |
| `generate_pdf_document` *(legacy)* | HTML/CSS layout text → real `.pdf` |
| `generate_vector_graphic` *(legacy)* | Native SVG for diagrams |
| `trigger_diffusion_image` | Prompt → local AUTOMATIC1111-compatible diffusion server |

### 4.2 Natural-language file creation (no `/save` required)

`documents.DetectSaveIntent(text)` — a dependency-free heuristic requiring a creation verb (open-ended list: create/generate/make/build/write/draft/produce/prepare/save, and Spanish equivalents) **plus** either a directory keyword or a path-shaped token with a recognizable extension, in the same clause (clauses split on "and"/" y "). Directories are created immediately; files are sent through as a normal chat turn so the model produces content, then saved automatically. Shared by `cli/nl_save.go` and `mcp/nl_save.go`.

### 4.3 Natural-language file editing (review-before-write)

`documents.DetectEditIntent(text)` recognizes an open-ended list of modify verbs (modify/edit/change/update/fix/correct/repair/replace/adjust/revise/alter/rewrite/refactor). What separates "edit" from "create" is **not** the verb list — it's **existence**: `ResolveExistingFile` only treats a message as an edit if the named path (or, with none named, the session's last-touched file) already exists on disk.

```
1. ReadEditableContent    — raw file (text/code) or ReadDocumentLayer's extracted text (docx/pdf/xlsx)
2. BuildEditPrompt        — ask the model for the complete new content
3. ExtractEditedContent   — strip a stray code fence defensively
4. DiffLines (LCS)        — compute a real line diff BEFORE writing anything
5. Confirm, then write
```

Full content is regenerated (not a binary patch) so `.docx`/`.pdf` edits re-lay-out through the same generator `save_service.go` uses. For source code / plain text where byte-level precision matters more, `patch_file` remains the surgical option.

**`documents.DiffLines`** — dependency-free **LCS (Longest Common Subsequence) line diff**, classic O(n·m) dynamic programming, bounded by `maxDiffCells` against pathological inputs; only changed lines are shown (`-`/`+`), capped at `maxDiffLinesShown`.

**Confirmation differs by door** because only one has a terminal:

| Door | Confirmation |
|---|---|
| `mova chat` | *"Apply this change? (y/n)"* per file; *"Apply to ALL N files? (y/n)"* first when multiple files are targeted, falling back to per-file on "n" |
| MCP / HTTP | Explicit `apply_edits` argument — `false` (default) proposes the diff only, `true` applies immediately |

---

## 5. Autonomous Chat Tool-Calling Protocol

`project.json`'s optional `"tools": {"enabled": true, "allow": [...]}` turns `mova chat` and the MCP `chat_completion` tool into a small agent that can invoke a whitelisted tool (`save`, `read_file`, `patch_file`, `read_document_layer`) and receive the **real result**, instead of only describing in prose what it would do.

**Deliberately not wired through any provider's native function-calling API** — OpenAI/Anthropic/Ollama `tools` parameters differ, and small local models (e.g. `llama3.2:3b`) frequently don't support them at all. Instead, one plain-text, provider-agnostic protocol:

```
1. ToolsSystemPrompt        — appends an instructions block (tools + arguments) to the system prompt
2. Model replies:
   <<<MOVA_TOOL_CALL>>> {"name":"...","arguments":{...}} <<<END_MOVA_TOOL_CALL>>>
3. ParseAgentToolCall        — extracts the block
4. RunAgentTool               — executes via the same dispatcher `tools/call` uses
5. Real result returned as the next user turn
```

`sendWithTools` (CLI) and `sendWithToolsMCP` (MCP) are two small (~20-line), identical loops driving this, capped at `MaxAgentToolTurns` round-trips.

`/save` remains a **deterministic** slash command a human types directly — it never depends on the model cooperating with the protocol above, and works even with providers with no tool-following ability at all.

---

## 6. Budget, Deduplication, and the Token Firewall

### 6.1 Budget report (`mova budget`)

One `budget.BuildReport()`, three doors: `mova budget` (CLI), `estimate_budget` (MCP/HTTP), `/budget` (chat REPL). It calls `core.BuildContextSections()` — the exact same assembly `mova run` uses, split into pieces instead of concatenated, so whatever a project declares is exactly what gets priced.

Each piece (Agents, Skills, Prompt, Focus, Memory, + fixed overhead) is tokenized independently via `github.com/tiktoken-go/tokenizer` (embedded vocabularies, no network) and priced against `config/prices.json` (hot-reloaded by mtime, same pattern as model configs). Component counts always sum exactly to the total.

`--focus` walks the full unfiltered repo and compares it against the computed Focus section, to show the token/cost delta `focus` actually makes.

### 6.2 Automatic deduplication

`core.BuildContextSections` shares one `dedup.Paragraphs` "seen" map across **Agents → Skills → Prompt → Focus → Memory**, in that order — an exact-duplicate paragraph anywhere in the sequence survives only once. `~M tokens saved` is a `chars/4` approximation, documented as such.

### 6.3 Budget as an enforced limit — the gate

`project.json`'s `"budget": {"max_tokens": N}` (task-level overrides project-level) is checked two ways:

- `budget.CheckLimit` — informational, feeds the report's "Budget Limit" section.
- `budget.EnforceLimit` — the **hard gate**, called before context is ever printed or sent, at every door that assembles context.

If exceeded, all doors return the identical `ERROR / ... exceeds .../ Suggestion: Use --focus...` text and stop — nothing reaches a model or a caller.

**Fixed bug — the gate was silently skippable.** Four call sites used to gate only `if t, ok := proj.Tasks[resolvedTask]; ok { EnforceLimit(...) }` — a project with no `default_task` and no task named skipped the gate entirely. Fixed by centralizing resolution:

- `core.ResolveTaskName(proj, taskName)` — the single task-picking logic (§2).
- `budget.ResolveTask(proj, taskName)` — looks up `proj.Tasks`, or returns a pointer to a **zero-value `Task{}`, never `nil`**, so `EnforceLimit` always has something to call and correctly falls back to the project-level budget.

Every gate call site now does:
```go
t := budget.ResolveTask(proj, core.ResolveTaskName(proj, taskName))
if err := budget.EnforceLimit(proj, t, tokens); err != nil { ... }
```
unconditionally — the gate can no longer be bypassed.

### 6.4 Feedback Loop — calibrating estimates against real usage

`mova-token-history.json` stores exactly two accumulators per provider (`total_local_tokens`, `total_api_tokens`) — no prompts, no replies. Written once, right after a successful send that returned real `Usage`. `TokenHistory.DeviationPercent = (API-Local)/Local*100` populates "Historical Token Accuracy"; no data yet → `No historical data`, never a fabricated number.

### 6.5 Token Firewall — is context bigger than it needs to be?

Budget answers *"is this context too big?"*. The Token Firewall answers three further questions, all **deterministically, no AI involved**:

```
budget.BuildGatedContext(adapter, root, project, task)
        │
        ├─ core.BuildContextSections     (unchanged assembly)
        ├─ [1]  SanitizeCached           — text-level noise reduction
        ├─ [1b] PII Masking (optional)   — off by default
        ├─ [2]  CheckCircuitBreaker      — spend-governance gate
        ├─ [3]  EnforceLimit             — the pre-existing max_tokens gate
        ▼
GatedContext{Text, Tokens, Sanitize, CircuitBreaker, PII, Err}
```

`BuildGatedContext` is the **single** function shared by `mova run`, `orchestrator.RunAgent`, `mova chat`, the TUI chat screen, `jobs/actions_tasks.go`, and `chat_completion` — five call sites, one pipeline, zero duplicated logic.

| Stage | File | What it does |
|---|---|---|
| Sanitizer | `sanitize/sanitize.go` | `dedupeRepeatedLines` (timestamp-aware: strips a leading timestamp for comparison only), blank-line collapsing, optional comment stripping |
| Cross-file dedup | `sanitize/dedupe.go` | `DedupeLeadingBlocks` — license headers, import blocks across Focus files |
| PII Masking | `sanitize/pii.go` | `MaskPII` — scores each token via `wordShapeScore` (digit ratio, separators, `@`, length, case) + `entropyScore` (normalized Shannon entropy); tokens above `MinScore` become deterministic `[PII_xxxxxxxx]` (FNV-1a hash). Language-agnostic — no word lists |
| Circuit Breaker | `budget/spend.go` | `CheckCircuitBreaker`/`RecordSpend`, backed by `mova-spend.json` |
| Context Cache | `budget/contextcache.go` | `SanitizeCached` — per-project memoization of the Sanitizer's result |
| Cache Layout Guard | `budget/cachelayout.go` | `LayoutForCache` — reorders into a static-prefix/dynamic-tail shape + a stability fingerprint (sha256), so a provider's own prompt caching can actually discount it. **Not** part of `BuildGatedContext`'s text — only the four chat-sending call sites (+ report preview) apply it; `mova run`'s printed output keeps the original, more-readable order |

**Configuration** — every Token Firewall toggle is a `*bool` on `core.BudgetConfig`, read through `core.XEnabled(cfg)` (nil, absent, or `true` all mean **on**) — opposite default from `tools.enabled` (off by default), because these are safety/efficiency defaults, not opt-in extras. **PII Masking is the one exception: off by default**, because it changes actual content.

Anthropic's provider is the only one that reads `ChatMessage.CacheBoundary` (`models/provider_anthropic.go`'s `systemField` builds the two-block `system` array with `cache_control`); every other provider ignores the field entirely.

---

## 7. Path Resolution & Cross-Platform Engine

### 7.1 Root discovery

```
runtime.FindRoot()
   1. MOVA_PROJECT_PATH   — direct override, no search
   2. MOVA_PROJECT_ROOT   — search starts there instead of cwd
   3. current working directory — walk upward looking for workflow.md
```
`runtime.AutoDetect(root)` returns the sole project under `projects/` when exactly one exists, so `[project]` can be omitted.

### 7.2 File/directory resolution

`documents.ResolvePath(root, repo, filename)` / `ResolveDirectoryPath` / `ResolveFilePath` — cross-platform absolute-path handling, bare-name disk search with disambiguation, and a repo default. Generated files live under a project's `repo`, never inside the tool's own installation.

### 7.3 UNC, WSL, Docker — verified, not assumed

Audited directly against Go's standard library (`internal/filepathlite`): `filepath.IsAbs` correctly treats a UNC prefix (`\\server\share`) as an absolute volume name, and `filepath.Join`/`Clean` preserve it through every operation. Every code path resolving `"repo"` (`core/focus/`, `budget/report.go`, `budget/history.go`, `jobs/actions_delete.go`, `documents/types.go`, `cli/tui_paths.go`, `cli/tui_logs.go`) already uses `filepath.IsAbs`/`filepath.Join` consistently.

| Scenario | Behavior |
|---|---|
| **UNC** (`\\server\share`) | Works natively on a Windows `mova.exe` build |
| **WSL** | Linux `mova` → Windows drive: ordinary `/mnt/d/...` path. Windows `mova.exe` → WSL distro: UNC path (`\\wsl$\Ubuntu\...`) — same mechanism, no special case |
| **Docker** | A bind-mounted host directory is a native absolute path from the container's point of view — identical to the `MOVA_PROJECT_ROOT` case |
| **Limitation** | UNC/drive-letter paths only resolve on a **Windows** `mova` binary — no OS-level UNC support on Linux/macOS; mount first (`mount -t cifs`, Finder), then use the native mounted path. `normalizeAbsPath` surfaces this as a clear error |

### 7.4 Working from a different drive (`MOVA_PROJECT_ROOT`)

A project's `repo` can already point anywhere absolute; what was missing was Mova finding **its own** `workflow.md` from an unrelated directory. Installers now set `MOVA_PROJECT_ROOT` automatically (registry on Windows, shell rc-file on macOS/Linux) — purely additive: an existing value is never overwritten, and the upward search from cwd still runs first.

---

## 8. Single Source of Truth: Model Resolution & Hot-Reloading

**One file per model, not three.** `config/models/<provider>/<config>.json` (`models.ModelConfig`) holds connection (`base_url`/`host`/`port`/`api_key`/`timeout_seconds`) **and** inference parameters (`temperature`/`num_ctx`/`num_predict`/model tag) together — no separate provider-level `config.json`. That file was removed because (a) it could drift out of sync with a model's own file, and (b) Windows forbids `:` in filenames, so a model tag like `llama3.2:3b` couldn't safely have a matching filename — `llm_profile.config` and the filename are now the same string by construction, so that class of bug cannot recur.

**Hot reload** — `ConfigCache` (`models/config.go`) stats each file's mtime on every read and reloads only on change: editing a `.json` by hand takes effect on the very next message, no restart, no watcher goroutine. `config/models/active.json` is deliberately just a pointer `{"provider", "config"}` — never a copy of connection/inference data — so nothing there can go stale.

### Provider resolution order

`project.json`'s `llm_profile = {type, provider, config}` decides which model a project's chat uses. **`provider` is optional:**

```
1. If provider is set        → use it directly
2. If provider is omitted    → models.ResolveConfigProvider scans every config/models/<provider>/
                                folder for the one owning "<config>.json"; that file's own "type"
                                field becomes the resolved provider identity (never duplicated
                                in project.json)
```
`provider` is only required when the same `config` filename exists under more than one provider folder (otherwise ambiguous). Resolution happens identically in `cli/chat_cmd.go` and `mcp/chat_tool.go`, then calls `Session.SwitchProvider` — a **session-local** override that never touches the global `active.json`.

---

## 9. Job Engine & Multiagent Orchestrator

### 9.1 Job Engine (`jobs/`)

| File | Role |
|---|---|
| `cron.go` | Dependency-free 5-field cron parser/matcher |
| `engine.go` | `RunJob` / `RunProjectJobs` / `RunJobByIndex` — the one execution flow |
| `actions_*.go` | One file per action (`tasks`, `save`, `memory`, `delete`, `budget`) — each reuses an existing component |
| `scheduler.go` | `RunScheduler` / `RunDueJobs` — the `mova jobs start` daemon |

```
jobs.RunJob(adapter, root, project, proj, spec)
   1. runTasks          — BuildContextSections per task (or all, for "*"), same EnforceLimit gate as `mova run`
   2. runSave            — documents.Save with {date} expanded
   3. runMemory           — Adapter.AppendMemory with {date}/{time} expanded
   4. runMemoryArchive     — Adapter.ArchiveMemory
   5. runDelete             — glob-expand, documents.Delete(Confirm: true)
   6. runBudget               — budget.BuildReport + WriteReport
   ▼
jobs.Result{Project, Steps, Errors}
```

**Adding a new action:** add a field to `core.JobSpec`, one `actions_<name>.go` file following the existing shape, one call in `RunJob`'s fixed sequence. Nothing else in the engine changes.

### 9.2 Multiagent orchestrator (`orchestrator/`)

`config.go`'s `LoadGroupConfig` reads `projects/<group>/config.json`, auto-discovering agent subdirectories (any folder containing its own `project.json`) when `"agents"` is omitted.

```
orchestrator.RunGroup(adapter, root, group, only, task)
   1. LoadGroupConfig
   2. for each agent, sequentially:
        RunAgent → budget.BuildGatedContext(..., "<group>/<agent>", task)
        — the SAME function `mova run` uses; an agent is just a project name with a "/" in it
   ▼
[]AgentResult{Agent, Project, Text, Tokens, Err}
```

Sequential by design; parallel execution (goroutine + `sync.WaitGroup` per agent) is a documented, isolated extension point in the `RunGroup` for-loop only.

### 9.3 Logging (`logging/`)

Opt-in, **disabled by default** (`config/log/logging.json`: `"enabled": false`). `level.go` (debug/info/warning/error), `rotate.go` (interval + retention, pure functions), `logger.go` (`Open`/`Logger`, rotation-on-write), `default.go` (`SetDefault`/`L()` — the process-wide default every package logs through). Any string is a valid category key — no code change needed to add one.

---

## 10. Transports — MCP / HTTP, same engine, different door

`mcp.Process(adapter, root, req)` is the single dispatcher for `initialize` / `tools/list` / `tools/call`. `http/server.go` decodes an HTTP POST body into the same `Request` struct and calls `Process` — **no protocol logic of its own**. Convenience REST routes (`POST /save`, `/delete`, `/workflow`, `/jobs/run`, `/agents/run`) are thin wrappers that repackage a JSON body as a `tools/call`.

```
mova mcp start [--stdio]
        │
        ├─ stdio  ──► mcp.Process
        └─ http   ──► http.Server ──► mcp.Process   (identical result)
```

`get_full_context` (MCP/HTTP) = `mova run`; `chat_completion` = `mova chat`, minus the terminal.

---

## 11. Diagrams — visualizing a project's real pipeline

`mova run <project> --diagram [--export svg,png,pdf]` and the `generate_diagram` MCP tool render a project's actual pipeline (sources → Context Compiler → Token Firewall incl. PII Masking → agents → jobs → interfaces → real token/cost metrics), built **entirely from real `project.json`/`config.json` plus a live `orchestrator.Count` result** — nothing invented, nothing drawn for an unconfigured stage.

| File | Role |
|---|---|
| `diagram/model.go` | `Data` + sub-structs (`Firewall`, `AgentNode`, `JobNode`, `Metrics`) |
| `diagram/build.go` | `BuildDiagram` — delegates every number to `orchestrator.Count`, reads structure from `core.Project` |
| `diagram/svg.go` | Dark-themed, color-coded vertical flow SVG layout engine |
| `diagram/png.go` | Rasterizes via `oksvg`/`rasterx` (pure Go); labels drawn in a second pass with `image/font` (ASCII-only bitmap face, so accents are transliterated for this raster layer only — the SVG export keeps full Unicode) |
| `diagram/pdf.go` | Wraps the raster into a standalone one-page PDF — a separate, minimal writer from `documents/pdf_writer.go` on purpose |
| `diagram/export.go` | `Export(data, formats, outDir, baseName)` — cross-platform via `os.MkdirAll` only |

---

## 12. Terminal UI (`mova ui`)

Presentation only — lives in `cli/` (package `main`) so it can call existing CLI helpers directly. Root model is a **screen stack** (push/pop); one generic menu screen and one generic file-viewer/editor are reused across nearly every section.

| Screen file | Role |
|---|---|
| `tui_app.go` | Root model — screen stack, routes Update/View |
| `tui_menu.go` | Generic list-based menu, reused everywhere |
| `tui_fileview.go` | Generic file viewer/editor (textarea) — project.json, workflow.md, memory.md, model/log configs |
| `tui_projectpicker.go`, `tui_dashboard.go` | Project navigation |
| `tui_jobs.go`, `tui_agents.go`, `tui_models.go`, `tui_logs.go`, `tui_chat.go` | Section-specific screens, each calling straight into `jobs`/`orchestrator`/`models`/`logging`/chat — never re-implementing what that package does |

Only `Ctrl+S` writes to disk; `Esc` discards in-memory edits by design.

## 13. Component Index (quick reference)

| Directory | Owns |
|---|---|
| `core/` | Engine, types, Adapter interface, Focus engine |
| `dedup/` | Exact-paragraph deduplication |
| `adapters/` | DB-backed Adapter implementations |
| `documents/` | Save/Delete services, path resolution, office-format generators |
| `sanitize/` | Sanitizer + PII Masking |
| `budget/` | Token/cost estimation, Budget gate, Token Firewall pipeline, Feedback Loop |
| `models/` | LLM provider abstraction + single-source-of-truth model config |
| `jobs/` | Cron-scheduled unattended actions |
| `orchestrator/` | Multiagent groups |
| `logging/` | Opt-in structured logging |
| `diagram/` | Pipeline visualization (SVG/PNG/PDF) |
| `cli/` | Command dispatch + Terminal UI |
| `mcp/` | MCP JSON-RPC tool layer |
| `http/` | HTTP transport |
| `runtime/` | Root discovery |

## 14. Troubleshooting Matrix

| Area | Check |
|---|---|
| Budget not enforced | `project.json`/task has a `"budget": {"max_tokens": N}` block; `EnforceLimit` is called from the door in use |
| Wrong token count | Which `llm_profile.config` is active — it picks the tokenizer encoding |
| Format "unsupported" | `WriterFor`/`RegisteredExtensions` in `save_service.go` |
| NL creation not detected | `DetectSaveIntent` needs a creation verb **and** a directory keyword or extension in the same clause |
| NL edit not detected / wrong target | `DetectEditIntent` + `ResolveExistingFile` — the file must already exist; no path given falls back to the session's last file (CLI only) |
| Nothing written after an edit | `mova chat` needs a "y" answer; MCP/HTTP needs `"apply_edits": true` |
| Job doesn't fire | Valid cron string, and `mova jobs start` actually running — a missed minute is never caught up |
| Agent not found under a group | `LoadGroupConfig` silently skips subdirectories with no `project.json` |
| Logging silent | `"enabled"` defaults to `false`; then check `"level"` and `"categories"` |
| `mova ui` fails to start | Needs a real TTY — stdin/stdout redirection breaks any Bubble Tea program |
| HTTP behaves differently than MCP stdio | `http/server.go` has no logic of its own — the bug is in request decoding, not the tool |

---

## 15. Deliberately Not Here

No compiler, no token-optimization pipeline, no licensing tier, no `-tags premium` build. `focus` is the complete, permanent implementation, not a limited edition of something larger. Every "Recent changes" capability above is additive at the `project.json` level — a project that never touches a new field behaves exactly as it did before that feature shipped.