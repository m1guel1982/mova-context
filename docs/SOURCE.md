# SOURCE — codebase structure (English only)

One binary (`mova`), no build tags, no editions. This document explains how the source is organized, how the two extension points (**Adapters** and **Focus Resolvers**) work, and how to add your own.

## Layout

```text
src/
├── core/                  the engine — zero external dependencies
│   ├── types.go            Project, Task, Adapter-shared structs (mirrors project.json)
│   ├── engine.go           BuildContext()/BuildContextSections() — assembles agents+skills+prompt+memory+focus; ResolveTaskName() picks the task (explicit → default_task → the project's one task if it only has one) — the single place that decision is made
│   ├── engine_helpers.go   loadCore, variable injection, focus resolution, LLM profile helpers
│   ├── adapter.go          the Adapter interface (storage abstraction)
│   ├── file_adapter.go     default Adapter: reads agents/skills/prompts/projects from disk
│   ├── file_helpers.go     small file-reading/variable-injection helpers
│   └── focus/              Focus Resolution Engine (see below)
│       ├── engine.go        Engine + Resolver contract, cascade logic
│       ├── match.go         "LIKE simple" matching (case/accent-insensitive)
│       ├── stats.go         scan evidence (files seen/included/excluded)
│       ├── resolvers/       one file per knowledge type (see below)
│       └── render/          turns []ContextBlock into the "## FOCUS" text block
│
├── dedup/                 exact-paragraph deduplication — shared by core and core/focus/render
│   └── dedup.go             Paragraphs() — removes exact repeats given a shared "seen" map, never rewords/summarizes
│
├── adapters/              alternate storage backends
│   └── db_adapter.go        Postgres/MongoDB Adapter — same interface as file_adapter
│
├── documents/              office formats & media — read/generate PDF, DOCX, XLSX, SVG, diffusion images
│   ├── types.go             ResolvePath() — resolves filenames against a project's repo
│   ├── pathresolve.go       ResolveDirectoryPath / ResolveFilePath — cross-platform absolute paths, bare-name disk search with disambiguation, repo default
│   ├── save_service.go      Save() / SaveRequest / IFileWriter / WriterFactory — THE unified entry point behind `/save`, chat, and MCP's `save` tool (see below)
│   ├── save_selection.go    ChatTurn/Exchange/SelectContent/TranscriptText/ExtractCodeBlocks/StripCodeBlocks/ParseRangeToken — the single implementation behind `/save`'s current/`-all`/`-range N-M`/`-c`/`-text` modes, shared by cli/chat_save.go and MCP's `save` tool (`history`/`mode`/`range`/`code_only`/`text_only` args)
│   ├── delete_service.go    Delete()/DeleteRequest/DeleteResult/FormatDeletePrompt — THE unified entry point behind `/delete`, MCP's `delete_path` tool, and `POST /delete`; resolves file-vs-directory via pathresolve.go, never deletes without Confirm:true
│   ├── highlight.go         DetectLanguage()/AutoTagCodeFences() — the single language-detection/auto-tagging implementation shared by chat's renderMarkdown, MCP's chat_completion reply, and therefore HTTP
│   ├── nl_intent.go         DetectSaveIntent() — natural-language "create a file/directory" detector (long, open-ended verb list), shared by cli/nl_save.go and mcp/nl_save.go
│   ├── edit_intent.go       DetectEditIntent() — natural-language "modify an EXISTING file" detector, counterpart to nl_intent.go
│   ├── edit_apply.go        ReadEditableContent/BuildEditPrompt/ExtractEditedContent/ResolveExistingFile — shared plumbing for cli/nl_edit.go and mcp/nl_edit.go
│   ├── difftext.go          DiffLines() — dependency-free LCS line diff, previews a proposed edit before it's written
│   ├── directory.go         create_directory — recursive (mkdir -p), cross-platform
│   ├── textfile.go          read_file / write_file / patch_file — allowlisted text & source-code formats (see SUPPORTED_FORMATS.md); also registers textWriter for every one of those extensions with the WriterFactory above
│   ├── docx.go              generate_word_contract — markdown → real .docx (stdlib only); registers docxWriter for ".docx"
│   ├── xlsx.go              generate_excel_report — typed sheets_data → real .xlsx (stdlib only); registers xlsxWriter (accepts sheets_data JSON **or** plain CSV/TSV) for ".xlsx"
│   ├── pdf.go               generate_pdf_document — HTML/CSS text extraction, pagination, and the SaveService writer adapter for ".pdf" (stdlib only)
│   ├── pdf_writer.go        the low-level PDF 1.4 byte writer (objects/xref/trailer) + WinAnsiEncoding text encoding, split out of pdf.go
│   ├── svg.go               generate_vector_graphic — writes/validates native SVG; registers svgWriter for ".svg"
│   ├── read_layer.go         read_document_layer — text extraction from .docx/.xlsx/.pdf
│   └── diffusion.go         trigger_diffusion_image — routes a prompt to a local diffusion server
│
├── budget/                token/cost estimation — `mova budget`, 100% local, no LLM call
│   ├── prices.go            PricesConfig — reads config/prices.json, hot-reload cache (same mtime pattern as models/config.go)
│   ├── tokencount.go        CountTokens() — wraps github.com/tiktoken-go/tokenizer (embedded, no network)
│   ├── estimate.go          BuildReport() — per-component (agents/skills/prompt/focus/memory) token+cost breakdown, focus comparison
│   ├── limit.go             CheckLimit()/EnforceLimit() — hard "budget": {"max_tokens": N} gate, stops execution before sending to a model or returning a project's context (mova run, mova chat, get_full_context, chat_completion); ResolveTask() always returns a non-nil *Task (a real one, or a zero-value Task{}) so the gate never gets skipped for "no task named"
│   ├── history.go           mova-token-history.json — per-provider {local,api} token accumulators only, no prompts/content ever stored; HistoryPath() resolves the path from project.json's "token_history_path" (a single string)
│   ├── report.go            RenderMarkdown()/WriteReport() — mova-budget-report.md (English only, see SUPPORTED_FORMATS.md); BudgetReportPath() resolves the destination from project.json's "budget_path" (config/prices.json no longer has "report_path" — see Recent changes); budgetLimitSection/budgetRecommendations/providerComparisonNote — the enriched percent-used/diff/cross-provider explanation
│   └── workflow.go          LoadWorkflow() — the Budget-gated pipeline behind "lee/ejecuta workflow.md": resolve project → core.ResolveWorkflowPath → core.BuildContextSections (Dedup+Focus already inside it) → CountTokens → EnforceLimit; only on success is workflow.md's content returned. Shared by cli/workflow_cmd.go, mcp/server.go's `get_workflow`, and http's `/workflow`
│
├── models/                local + Cloud LLM providers (Ollama, LM Studio, vLLM, OpenAI, Anthropic, Google...)
│   ├── types.go             ModelConfig — ONE struct, ONE file per model (config/models/<provider>/<config>.json): connection (base_url/api_key/timeout) AND inference params (temperature/num_predict/...) together, no separate provider-level config.json anymore
│   ├── config.go            loads config/models/, active provider/model (config/models/active.json — just a {provider,config} pointer), hot-reload cache; ResolveConfigProvider() — finds which provider folder owns a given config filename, so llm_profile.provider is optional
│   ├── http_client.go       one shared *http.Client (connection pool + keep-alive)
│   ├── provider.go          Provider/StreamProvider interfaces + NewProvider() dispatch (purely on ModelConfig.Type) + small shared helpers
│   ├── provider_gemini.go   Google Gemini's native REST API ("type": "google"/"gemini")
│   ├── provider_ollama.go   Ollama's native /api/chat, also the default for any unrecognized local backend ("type": "ollama" or anything else)
│   ├── provider_openai.go   any /v1/chat/completions-shaped API: OpenAI, LM Studio, vLLM, TGI... ("type": "openai"/"openai-compatible")
│   ├── provider_anthropic.go Anthropic's /v1/messages shape ("type": "anthropic"/"claude")
│   ├── provider_http.go     postJSON/postJSONStream — generic HTTP plumbing shared by provider_ollama.go and provider_openai.go
│   ├── manage.go            Install/Remove/ListInstalled (Ollama's /api/pull|tags|delete) — seedConnection() borrows a sibling model's connection details, since there's no per-provider config.json to read anymore
│   ├── usage.go             UsageInfo/UsageFor/FormatLine — the `[Tokens] used / max context window` line shared by cli and mcp
│   └── chat.go              Session — used by `mova chat`, MCP and HTTP alike; SwitchProvider() applies a project's llm_profile as a session-local override
│
├── cli/                   the `mova` command — thin dispatcher, no business logic
│   ├── main.go              command switch (run, memory*, list, init, search, mcp, chat...)
│   ├── help.go              usage text for `mova` with no arguments
│   ├── adapter_select.go    decides file vs. db Adapter for a given project
│   ├── memory_mgmt.go       memory-clear / memory-config
│   ├── models_cmd.go        config / show config / install / model-list / remove
│   ├── budget_cmd.go        mova budget — thin wrapper over mova.local/budget.BuildReport
│   ├── run_cmd.go           mova run — builds a project's context and enforces Budget before printing anything (see budget/limit.go)
│   ├── chat_cmd.go          mova chat — REPL loop only (/memory, /budget, /save, /tools dispatch; natural-language create/edit dispatch — see chat_helpers.go/chat_save.go/nl_save.go/nl_edit.go)
│   ├── chat_helpers.go      applyProjectLLMProfile, sendWithTools (model-driven tool-calling loop), printTokenUsage, providerLabel
│   ├── chat_save.go         /memory, /budget, /save command implementations
│   ├── nl_save.go           natural-language file/directory creation for `mova chat` (no `/save` required) — see documents.DetectSaveIntent
│   ├── nl_edit.go           natural-language file EDITING for `mova chat` — Claude Console-style: shows a diff, asks "Apply this change? (y/n)" (and "apply to ALL?" for multi-file requests) before writing anything; chatFileState tracks the last file touched this session
│   └── console_*.go         per-OS terminal helpers
│
├── mcp/                   MCP (Model Context Protocol) JSON-RPC layer
│   ├── server.go            Process() — same engine, exposed as MCP tools
│   ├── context_tool.go      get_full_context tool — the MCP/HTTP equivalent of `mova run`; enforces Budget before returning anything
│   ├── chat_tool.go         chat_completion tool — wraps models.Session; same tool-calling loop as cli/chat_cmd.go (sendWithToolsMCP); prints the `[Tokens]` usage line
│   ├── nl_save.go           natural-language file/directory creation for chat_completion (no `/save` required) — mirrors cli/nl_save.go
│   ├── nl_edit.go           natural-language file EDITING for chat_completion — mirrors cli/nl_edit.go, but since MCP/HTTP calls are stateless there's no y/n prompt: an explicit `apply_edits` argument (default false = propose only) replaces it
│   ├── agent_tools.go       lets chat (CLI + MCP) call a whitelisted subset of tools mid-conversation via one provider-agnostic, marker-delimited plain-text protocol — AgentToolNames/ToolsSystemPrompt/ParseAgentToolCall/RunAgentTool/RunFileTool
│   ├── budget_tool.go       estimate_budget tool — wraps mova.local/budget.BuildReport
│   ├── documents_tool.go    the executeTool switch for save (unified) + read_document_layer / generate_* / trigger_diffusion_image / create_directory / write_file / patch_file / read_file (legacy, kept for compatibility) tools
│   └── documents_tool_helpers.go  boolArg/hasArg/resolveSmartDir/resolveSmartFile/repoFor/formatAmbiguousMessage/parseSheetsData/loadDiffusionConfig — split out of documents_tool.go
│
├── http/                  HTTP transport
│   └── server.go            thin wrapper: POSTs JSON-RPC bodies into mcp.Process()
│
└── runtime/               shared bootstrapping
    └── root.go              FindRoot() (locates workflow.md) + AutoDetect()
```

Everything under `core/` has **zero external dependencies** — it only touches the Go standard library. `adapters/` is where anything that needs a third-party driver (a SQL driver, for instance) lives, kept out of `core/` on purpose so the engine stays trivially portable. `models/` follows the same discipline: standard library only, one shared `http.Client`, no per-request allocation of transports — it's built to be called a very large number of times without degrading.

## The engine, in one paragraph

`core.BuildContext(adapter, root, projectName, taskName)` is the single entry point. It reads `project.json` via `Adapter.GetProject`, resolves which agents/skills/prompt to load (domain + `i18n/[lang]` + fallback to `en`), injects `variables` (project-level, then task-level overrides), appends `memory.md` if present, resolves `focus` if the task or project declares any, and returns the finished context as a string. `mova run`, the MCP `get_full_context` tool, and the HTTP transport all call exactly this function — there is no second code path.

## Extension point 1 — Adapter (storage backend)

```go
// core/adapter.go
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

The engine only ever talks to this interface — it never knows whether data comes from Markdown files on disk (`core.FileAdapter`, the default) or a database (`adapters.DBAdapter`, Postgres/MongoDB today).

**To add a new backend** (MySQL, SQLite, Redis, a headless CMS, whatever):

1. Implement `core.Adapter` in a new file under `adapters/` (or your own package).
2. Wire it up in `cli/adapter_select.go` — the only file allowed to import both `core` and `adapters`, so `core` never has to know a second backend exists.
3. Nothing in `core/` changes.

## Extension point 2 — Focus Resolvers

`focus` (the `project.json` key, or `workflow.md`'s `## FOCUS` section for a model reading it directly) lets a task target specific files, symbols, or document sections instead of the whole repo. It's resolved by a small, deterministic cascade — **no LLM call, no probabilistic matching, same input always gives the same output.**

```go
// core/focus/engine.go
type Resolver interface {
    Match(ctx Context, target string) bool
    Resolve(ctx Context, target string) ([]ContextBlock, error)
}
```

`Engine.Resolve` tries each registered `Resolver` in priority order. A resolver's `Match` is a cheap "am I a candidate for this target?" check; `Resolve` does the real work and can decline (return `ErrNotFound`) without stopping the cascade — the engine just moves to the next resolver.

Default cascade (`core/focus/render.DefaultResolvers()`), in priority order:

| Resolver | Handles |
|---|---|
| `FileResolver` / `DirectoryResolver` | exact file or folder paths |
| `JSONResolver` | a node inside a `.json` file |
| `SQLResolver` | a `CREATE TABLE ...;` definition by table name |
| `CodeSymbolResolver` | a function/class declaration (`func Foo()`, brace/indent matching) |
| `MarkdownResolver` | a heading/section (`## Some Section`) |
| `LegalResolver` | hierarchical legal documents (Título/Capítulo/Artículo/Inciso) |
| `MemoryResolver` | dated chronological blocks in `memory.md`-style files |
| `FallbackResolver` | bounded excerpt around the first occurrence — last resort, never a whole file |

All text matching goes through `focus.LikeContains` (`core/focus/match.go`) — case- and accent-insensitive, so `"articulo 3"` matches `"## Artículo 3"`.

**To add a new resolver** (a new document format, a new symbol type, anything):

1. Implement `Match`/`Resolve` in a new file under `core/focus/resolvers/`.
2. Register it in `DefaultResolvers()` (`core/focus/render/render.go`) at the position that makes sense in the cascade.
3. Nothing else changes — `Engine`, the other resolvers, and `BuildContext` are unaware of the addition (Open/Closed).

## Extension point 3 — Model Providers (local + Cloud inference)

```go
// models/provider.go
type Usage struct {
    PromptTokens     int
    CompletionTokens int
}

type Provider interface {
    Chat(ctx context.Context, model string, mc *ModelConfig, messages []ChatMessage) (string, Usage, error)
}
```

`NewProvider(mc *ModelConfig)` picks an implementation based on `ModelConfig.Type`: `"ollama"` (native `/api/chat`), `"openai-compatible"` (`/v1/chat/completions` — LM Studio, vLLM, TGI, and also the **real OpenAI and Google Gemini Cloud APIs**, both of which speak this exact format), or `"anthropic"` (`/v1/messages` — Claude's own shape, headers, and system-prompt field). `models.Session` (used by `mova chat`, the `chat_completion` MCP tool, and therefore HTTP) only ever talks to this interface — it doesn't know or care which engine, local or Cloud, is actually running behind `config/models/<provider>/<config>.json`. Every implementation returns real token `Usage` when the server reports it (`prompt_eval_count`/`eval_count` for Ollama, `usage.prompt_tokens`/`completion_tokens` for OpenAI-compatible, `usage.input_tokens`/`output_tokens` for Anthropic) — this is what closes `mova.local/budget`'s Feedback Loop (see below).

**To add a new provider type**: implement `Provider` in a new file under `models/`, add a `case` in `NewProvider`, done — `Session`, the CLI, MCP and HTTP are all unaware of the addition. No changes to `core`, `budget`, `cli`, or `mcp` are ever needed to add a provider.

**Single source of truth per model — one file, not three.** `config/models/<provider>/<config>.json` is the ONLY place a model's configuration lives: connection (`base_url`/`host`/`port`/`api_key`/`timeout_seconds`) AND inference parameters (`temperature`/`num_ctx`/`num_predict`/the real `model` tag/...) in the same JSON (`models.ModelConfig`, `models/types.go`). There used to be a second file, `config/models/<provider>/config.json`, holding just the connection fields, shared by every model under that provider — that file is gone. Two reasons: it was a second place the exact same connection data could drift out of sync with a model's own file, and — the bug that actually motivated removing it — Windows forbids `:` in filenames, so a model file named after an Ollama tag like `llama3.2:3b.json` can't exist there; people worked around it by naming the file `llama3.2.3b.json` while `project.json`'s `llm_profile` kept pointing at `llama3.2:3b`, and `models.FindModelFile` (which only ever matched by filename) would silently fail to resolve it. Now `llm_profile.config` and the file name are the same string by construction — see the next paragraph — so that class of bug can't recur.

Model configuration is read through `ConfigCache` (`models/config.go`): every read stats the file's mtime and reloads only if it changed, so editing a `.json` by hand takes effect on the very next message — no restart, no watcher goroutine, no extra dependency. `config/models/active.json` (`models.ActiveState`) is the global default provider/model, but it is deliberately just a *pointer* — `{"provider": "...", "config": "..."}` — never a copy of any connection or inference data, so there is nothing there that could go stale either.

For providers with zero models configured yet (a fresh `mova config lmstudio` before writing the first `.json`), `models.SetActiveProvider` only requires the `config/models/<provider>/` directory to exist — it no longer depends on a connection file being there first. `mova install` (Ollama-only; see `manage.go`'s doc comment) resolves connection details for a brand-new model by borrowing them from any sibling model already configured for that provider (`seedConnection`), or bootstraps a plain `http://localhost:11434` Ollama default if the provider has no models at all yet.

### Where the LLM provider is resolved (single source of truth)

`project.json`'s `llm_profile` (`{type, provider, config}`) is the one place that decides which provider/model a project's chat uses — `config` is nothing more than the filename (no `.json`) under `config/models/<provider>/` holding that model's full configuration (see above). **`provider` is optional.** When it's omitted, `models.ResolveConfigProvider(root, config)` scans every folder under `config/models/` for the one that has `<config>.json` and uses that — the provider's real identity then comes from that single resolved file's own `"type"` field (`"google"`, `"anthropic"`, `"openai-compatible"`, `"ollama"`, ...), never duplicated in project.json too. `provider` still works exactly as before when set explicitly (needed only if the same `config` filename exists under more than one provider folder — resolution would otherwise be ambiguous). `cli/chat_cmd.go`'s `applyProjectLLMProfile` and `mcp/chat_tool.go`'s equivalent block both do this resolution, then call `Session.SwitchProvider(provider, config)` — a session-local override that never touches `config/models/active.json` (the global default used when no project is given). HTTP gets this for free, same as everything else, because it's a thin wrapper over the same `mcp.Process()`. `workflow.md`'s "RESOLUCIÓN DE PROVEEDOR LLM" section documents this same resolution order for anyone reading the workflow spec directly. See `docs/i18n/en/COMMANDS.md`'s "`llm_profile`" section for the full list of `type` values a model config file supports and what each does.

## Chat tool-calling — letting the model create/edit files mid-conversation

`project.json`'s optional `"tools": {"enabled": true, "allow": [...]}` (`core.ToolsConfig`) turns `mova chat` and the MCP `chat_completion` tool into a small agent: the model can ask, in plain text, to run a whitelisted tool (currently `save`, `read_file`, `patch_file`, `read_document_layer` — see `mcp.AgentToolNames`), and get the *real* result back as a new turn, instead of just describing in prose what it would do (which is all `mova chat` ever did before, for any provider — small local models included).

This is deliberately **not** wired through any provider's native function-calling API (OpenAI/Anthropic/Ollama `tools` parameters differ from each other, and small local models like `llama3.2:3b` frequently don't support them at all). Instead, `mcp/agent_tools.go` defines one plain-text, provider-agnostic protocol: `ToolsSystemPrompt` appends a short instructions block (which tools, which arguments) to the chat's system prompt; the model replies with a `<<<MOVA_TOOL_CALL>>> {"name":"...","arguments":{...}} <<<END_MOVA_TOOL_CALL>>>` block; `ParseAgentToolCall` extracts it; `RunAgentTool` executes it (reusing `documentTool`, the exact same dispatcher `tools/call` uses) and returns the real result as the next user turn — `cli/chat_cmd.go`'s `sendWithTools` and `mcp/chat_tool.go`'s `sendWithToolsMCP` are the (intentionally small, ~20-line) identical loops that drive this on each door, capped at `mcp.MaxAgentToolTurns` round-trips.

Separately, `/save` (see below) is a **deterministic** slash command a human types directly — it never depends on the model cooperating with the protocol above, works with any provider (including one that can't reliably follow instructions at all), and is always available once a project is loaded, the same way `/memory` and `/budget` always are. `mcp.RunFileTool` is the unguarded counterpart `RunAgentTool` uses for autonomous, model-triggered calls (which stay gated behind `"tools.enabled"`).

## save — the single unified way to create or edit a file or directory

Before this, creating a Word doc, a PDF, an Excel report, an SVG, or a plain text file each meant calling a differently-named tool with a differently-named content argument (`markdown_content`, `layout_html_css`, `sheets_data`, `content`...) — easy for a human or a model to get wrong, and exactly the kind of mismatch that used to produce an empty `.docx` or a `generate_pdf_document: no se encontró texto en layout_html_css` error when the wrong key was used.

`documents/save_service.go` replaces all of that with one call: `documents.Save(root, documents.SaveRequest{Path, Directory, Content, Append, Overwrite, Repo})`. Internally:

- **`IFileWriter`** is the only extension point: `Write(path string, opts SaveOptions) error`.
- **`WriterFactory`** (`RegisterWriter`/`WriterFor`) is a plain `map[string]IFileWriter` keyed by lowercased extension — no `if`/`switch` to extend.
- Each existing generator registers its own adapter from an `init()` right next to it — `docx.go`'s `docxWriter`, `pdf.go`'s `pdfWriterAdapter`, `xlsx.go`'s `xlsxWriter`, `svg.go`'s `svgWriter`, and `textfile.go`'s `textWriter` (registered once per entry in the existing `supportedExtensions` list, so every plain-text/source-code extension gets it for free, no new type needed).
- **Adding a new output format** is exactly: write one small `IFileWriter` implementation, call `RegisterWriter(".ext", myWriter{})` from its own `init()`. `Save`, `/save`, the `save` MCP tool, and `POST /save` never change — Open/Closed by construction.

Two format-specific fixes landed as part of this, both in the Writer adapters, **not** in the underlying generator (so the legacy `generate_pdf_document`/`generate_word_contract` tools keep their original, stricter contracts for existing callers):

- **PDF**: `pdf.go`'s `contentAsHTML` auto-wraps plain text/Markdown in `<p>` tags before handing it to `GeneratePDFDocument`, so `/save`'ing a `.pdf` with prose or Markdown (not hand-written HTML) no longer produces the old "no se encontró texto" error — `GeneratePDFDocument` was already correct, the empty/untagged input reaching it was the actual bug.
- **Excel**: `xlsx.go`'s `sheetsFromContent` accepts the original typed `sheets_data` JSON shape *or* plain CSV/TSV text (auto-typing each cell as a number or string), so a model that just answers with a table doesn't need to learn `SheetsData`'s JSON shape to produce a working `.xlsx`.
- **Word**: `GenerateWordContract` itself was already correct (verified: real content lands in `word/document.xml`) — the "empty file" symptom came from a caller sending content under the wrong argument name (`content` instead of `markdown_content`, say). `/save`'s single `content` field removes that whole failure mode.

`/save` is reachable from all three doors identically:

- **Chat** (`cli/chat_cmd.go`, `runChatSave`): `/save "docs/informe.md"` saves the model's last reply there (format from the extension); `/save -d "docs/backend"` only creates a directory. Deterministic — doesn't require `"tools.enabled"`.
- **MCP** (`mcp/documents_tool.go`'s `"save"` case): `{"path"|"directory", "content", "append", "overwrite", "project"}` — the same tool the model itself calls autonomously when chat tool-calling is on (see above).
- **HTTP**: `POST /save` with the same JSON body, internally repackaged as a `tools/call` for `"save"` (see Transports section above) — zero duplicated logic.

The legacy per-format tools (`create_directory`, `write_file`, `generate_word_contract`, `generate_pdf_document`, `generate_vector_graphic`, `generate_excel_report`) are unchanged and still work, for backward compatibility with existing MCP clients/scripts — `save` is additive, not a breaking replacement.

## Documents — office formats & media

`documents/` implements file/directory creation and office-format generation. Since the `save` unification (see above), every capability below is reachable two ways: through the single `save`/`/save` entry point (recommended — picks the right one by extension automatically), or directly by name (kept for backward compatibility):

| Tool | Does |
|---|---|
| `save` | **the unified entry point** — `path`'s extension picks the right generator below automatically; `directory` instead of `path` only creates a folder |
| `create_directory` *(legacy)* | creates a directory recursively (`mkdir -p` semantics), cross-platform — path resolved via the same absolute/search/relative/default logic as every file tool below |
| `read_file` | reads any plain-text/source file as-is, no extension restriction |
| `write_file` *(legacy)* | creates or fully overwrites a file — extension must be on the `write_file`/`patch_file` allowlist (see `docs/SUPPORTED_FORMATS.md`), otherwise returns an English "Unsupported file type" error. `.json`/`.xml`/`.go`/`.csv` are validated for correctness before writing |
| `patch_file` | surgically replaces one exact, unique occurrence inside an existing file — rejects the edit if the match is missing, ambiguous, or the extension isn't on the allowlist |
| `read_document_layer` | extracts the plain-text layer from `.docx`, `.xlsx`, or `.pdf` |
| `generate_word_contract` *(legacy)* | compiles markdown into a real `.docx` |
| `generate_excel_report` *(legacy)* | compiles typed `sheets_data` (JSON: string/number/boolean cells) into a real `.xlsx` |
| `generate_pdf_document` *(legacy)* | compiles HTML/CSS layout text into a real `.pdf` |
| `generate_vector_graphic` *(legacy)* | writes/validates native SVG for diagrams and architecture maps |
| `trigger_diffusion_image` | routes a prompt to a local diffusion server (AUTOMATIC1111-compatible `/sdapi/v1/txt2img`) configured at `config/models/diffusion/config.json` — the diffusion server is a different kind of backend (an image server, not an LLM) so it keeps its own small connection file; it was never part of the LLM `config.json`-per-provider convention that `config/models/ollama`/`config/models/lmstudio` used to have (see Extension point 3) |

`write_file`/`patch_file`'s allowlist (`supportedExtensions` in `documents/textfile.go`) covers plain text, config formats, and ~20 programming languages. **See `docs/SUPPORTED_FORMATS.md` for the exact, authoritative list** — every format `chat`/MCP/HTTP can create, edit, or read, in one place.

`.docx` and `.xlsx` are written by hand as OOXML (a ZIP of XML parts) using only `archive/zip` and the standard library — no third-party office-format dependency, same zero-dependency discipline as `core/`. `.pdf` is written the same way: raw PDF 1.4 objects, no rendering engine. `read_document_layer`'s PDF path is a best-effort text extractor (FlateDecode streams + `Tj`/`TJ` operators) — it reliably reads PDFs this tool generates and most simple text PDFs, but it is not a full PDF parser (no font/CMap tables), so scanned or exotically-encoded PDFs may return nothing.

Generated files are resolved through `documents.ResolvePath(root, repo, filename)` — the same "generated files live under the project's `repo`" rule `workflow.md` already defines for source code, not inside `mova-context` itself.

## Budget — local token/cost estimation (`mova budget`)

`budget/` implements token/cost estimation as a single capability, reachable identically from the CLI (`mova budget`), MCP stdio/HTTP (the `estimate_budget` tool, `mcp/budget_tool.go`), and the chat REPL (`/budget`) — one `budget.BuildReport()` function, three doors, exactly the pattern established by every other capability in this codebase.

`BuildReport` calls `core.BuildContextSections()` — the *same* assembly `mova run`, `get_full_context`, and `chat_completion` use, just kept split into pieces instead of concatenated. This is the whole point: whatever `agents`/`skills`/`focus`/`memory` a project actually declares in `project.json` is exactly what gets tokenized and priced — tune `project.json` to optimize the reported cost, there's no separate "budget mode" configuration to keep in sync.

Each piece (Agents, Skills, Prompt, Focus, Memory, plus fixed engine overhead for the header/instruction boilerplate) is tokenized independently with `github.com/tiktoken-go/tokenizer` (embedded vocabularies, no network call) and priced against `config/prices.json` — hot-reloaded on every run via the same mtime-cache pattern as `models/config.go`, so editing prices never needs a rebuild. Component token counts always sum exactly to the report's total (see `budget/estimate.go`'s tests) — there is never a "total" that doesn't reconcile with its own breakdown.

`--focus` additionally walks the entire `repo` unfiltered (`core/focus/resolvers.WalkAllFiles` — the same utility the Premium edition's semantic resolver reuses) and compares it against the already-computed Focus section text, to show the token/cost difference `focus` actually makes. If the project has no `focus` configured, `--focus` returns a clear error rather than a meaningless 0-vs-0 comparison.

The report (`mova-budget-report.md`, written via `documents.WriteFile` — the same writer every other create/edit tool uses) is deliberately English-only and jargon-free, since whoever reads a cost report may not read the rest of Mova Context's Spanish CLI output. It states plainly, in three places, that these are local estimates, not an exact bill — see `docs/i18n/es/README.md`/`docs/i18n/en/README.md` for the same honesty framing in the project's own docs.

### Automatic deduplication (real optimization, not a suggestion)

`core.BuildContextSections` shares one `dedup.Paragraphs` "seen" map across Agents→Skills→Prompt→Focus→Memory, in that order — an exact-duplicate paragraph anywhere in that sequence survives only once. `dedup/dedup.go` is a small leaf package with no dependencies (used by both `core` and `core/focus/render`, which can't import each other), so this logic exists in exactly one place. `ContextSections.DuplicatesRemoved`/`DuplicatesRemovedChars` carry the count into `budget.BuildReport`'s "Deduplication" section and into the `[Dedup] Removed N duplicated paragraph(s) (~M tokens saved)` message both `cli/chat_cmd.go` and `mcp/chat_tool.go` print — the `~M tokens` figure is a `chars/4` approximation (documented as such), kept dependency-free rather than re-tokenizing removed text.

### Budget as an enforced limit, not just a report

`project.json`'s `"budget": {"max_tokens": N}` (task-level overrides project-level, see `core.ResolveBudget`) is checked two different ways: `budget.CheckLimit` for the report's informational "Budget Limit" section, and `budget.EnforceLimit` — the hard gate — called **before context is ever printed or sent anywhere**, at every door that assembles a project's context: `cli/run_cmd.go` (`mova run`), `cli/chat_cmd.go` (`mova chat`), `mcp/context_tool.go` (`get_full_context`), and `mcp/chat_tool.go` (`chat_completion`). All four return the exact same `ERROR / Current context (...) exceeds .../ Suggestion: Use --focus...` text and stop — nothing is printed, nothing is sent to a model, nothing reaches whoever asked (a human at a terminal, an MCP client, a raw HTTP caller). `mova budget` itself never calls this gate (it never sends anything to a model, it only reports); the gate only lives at the doors that actually produce or forward context, so CLI/MCP/HTTP behave identically. The enriched "Budget Limit" section in `mova-budget-report.md` (percent used, over/under-limit difference, which component to trim first, and a plain-language explanation of why token counts differ by provider) is built by `budget.budgetLimitSection`/`budgetRecommendations`/`providerComparisonNote` in `budget/report.go`.

### Token usage display (`[Tokens] ... used / ... max context window`)

After every chat response — `mova chat`, MCP `chat_completion`, and therefore HTTP — a one-line `[Tokens]` summary is printed, the same minimal info tools like Cline show: how many tokens *this* request used, and the active model's maximum context window. `models/usage.go`'s `UsageFor(sess, mc, fallbackEstimate)` builds it: it prefers the real `Usage.PromptTokens` the provider reported (see Extension point 3) and falls back to a local tiktoken-go estimate only if the provider didn't report one; the context window comes from `ModelConfig.ContextWindow` (0 if that model's config doesn't declare one, in which case the line says so instead of printing a bogus percentage). `cli/chat_helpers.go`'s `printTokenUsage` and `mcp/chat_tool.go`'s `writeTokenUsage` are the two call sites — both call the same `UsageFor`/`FormatLine`, so the line reads identically everywhere. This is purely informational and never blocks anything — that's `budget.EnforceLimit`'s job, and it already ran before the request was sent.

### Natural-language file/directory creation (no `/save` required)

`/save` is still there and still works exactly as before, but it's no longer the *only* way to get a file or folder out of a chat. `documents/nl_intent.go`'s `DetectSaveIntent(text)` is a small, dependency-free heuristic (a long, deliberately open-ended list of creation verbs — "genera"/"crea"/"hazme"/"elabora"/"escribe"/"redacta"/"prepara"/"arma"/"construye"/"produce"/"guarda" in Spanish, "create"/"generate"/"make"/"build"/"write"/"draft"/"produce"/"prepare"/"put together"/"save" in English, no cap on phrasing or repetition — paired with either a directory keyword or a path-shaped token ending in a file extension) shared by both `cli/nl_save.go` (`mova chat`) and `mcp/nl_save.go` (`chat_completion`, and therefore HTTP). A plain message like `Genera reporte.pdf` or `Crea el directorio docs/out` is detected automatically: directories are created immediately (they don't need the model's reply); files are still sent through as an ordinary chat turn so the model can produce content, and the reply is then saved to each detected path exactly the way `/save` would — just without typing the command. `Crea el directorio X y genera Y` does both in one message. Which `documents` writer actually handles a file (Word/PDF/Excel/SVG/plain text) is decided purely by its extension, same as `/save` and `save`/MCP — see "save — the single unified way..." above; this feature only decides *whether* to call `Save`, not *how*.

### Natural-language file EDITING (Claude Console-style review before writing)

The counterpart to file creation: `documents/edit_intent.go`'s `DetectEditIntent(text)` recognizes a similarly long, open-ended list of modify verbs ("modifica"/"edita"/"cambia"/"actualiza"/"corrige"/"arregla"/"repara"/"reemplaza"/"ajusta"/"revisa"/"agrega"/"quita"/"elimina"/"borra" in Spanish, "modify"/"edit"/"change"/"update"/"fix"/"correct"/"repair"/"replace"/"adjust"/"revise"/"alter"/"rewrite"/"refactor" in English). What tells "edit" apart from "create" is deliberately NOT the verb list (the two lists don't overlap, but that's incidental) — it's **existence**: `documents.ResolveExistingFile` only treats a message as an edit if the named path (or, with no path mentioned, `chatFileState.lastFile` — the last file this chat created or edited, see `cli/nl_edit.go`) already exists on disk. A request naming something that isn't there yet gets pointed at the creation phrasing instead, never silently misfired as an edit of nothing.

The flow, shared by `cli/nl_edit.go` (`mova chat`) and `mcp/nl_edit.go` (`chat_completion`/HTTP): read the file's current content (`documents.ReadEditableContent` — the raw file for text/code, `ReadDocumentLayer`'s extracted text for `.docx`/`.pdf`/`.xlsx`), ask the model for the complete new content (`documents.BuildEditPrompt`), clean up its reply (`documents.ExtractEditedContent` strips a stray code fence defensively), then compute a real line diff (`documents.DiffLines` — see below) **before writing anything**. This is a deliberate design choice, not a shortcut: it regenerates full content rather than computing a binary patch, so `.docx`/`.pdf` edits preserve the *content* but re-lay-out through the same generator `save_service.go` already uses (`docx.go`/`pdf.go`), not a byte-for-byte patch of the original file; `.xlsx` is the least reliable of the three for this reason (`ReadDocumentLayer`'s sheet extraction is a best-effort text view). For source code and plain text, `documents.PatchFile` (exact, unique search/replace — already used by the `patch_file` MCP tool) remains the more surgical option when precision at the byte level matters more than natural-language convenience.

Confirmation is the one place the two doors genuinely differ, because one has an interactive terminal and the other doesn't: `mova chat` asks **"Apply this change? (y/n)"** per file, and — with more than one file targeted by one message — asks once **"Apply to ALL N files? (y/n)"** first, falling back to per-file confirmation on "n" (`cli/nl_edit.go`'s `readYesNo`). `chat_completion` over MCP/HTTP has no terminal to prompt, so it takes an explicit `apply_edits` argument instead (default `false`: propose the diff, write nothing; `true`: apply immediately) — see the tool's description in `mcp/server.go`.

`documents/difftext.go`'s `DiffLines` is a small, dependency-free LCS-based line diff (classic O(n·m) dynamic programming, bounded by `maxDiffCells` against pathologically large files) — only changed lines are shown (`-`/`+`, unchanged lines omitted), capped at `maxDiffLinesShown` so a sweeping rewrite doesn't flood the chat. This diff is what makes the edit reviewable and precise instead of "trust the model, hope for the best."

### Feedback Loop — calibrating the estimate against real Cloud usage

`budget/history.go`'s `mova-token-history.json` (next to `project.json`, or at `Project.TokenHistoryPath`) stores exactly two accumulators per provider (`total_local_tokens`, `total_api_tokens`) — no prompts, no replies, no per-request log. `cli/chat_cmd.go`'s `recordRealUsage`/`mcp/chat_tool.go`'s `recordRealUsageMCP` call `budget.RecordUsage` right after a successful `Session.Send` whenever the provider returned real `Usage` (see Extension point 3 above) — this is the only writer. `budget.BuildReport` reads it back through `TokenHistory.DeviationPercent` (`(API-Local)/Local*100`) to populate "Historical Token Accuracy"; a provider with no recorded calls yet shows `No historical data`, never an error or a fabricated number.

## Transports (MCP / HTTP) — same engine, different door

`mcp/server.go`'s `Process(adapter, root, req)` is the single dispatcher for MCP JSON-RPC requests (`initialize`, `tools/list`, `tools/call`). `http/server.go` is a thin wrapper: it decodes an HTTP POST body into the same `Request` struct and calls `Process` — no protocol logic of its own, no separate code path from stdio. It also exposes one convenience REST-shaped route, `POST /save`, which just repackages its JSON body as a `tools/call` for the `save` tool and calls `Process` — same underlying `documents.Save`, zero duplicated logic. `cli/main.go`'s `mova mcp start` picks stdio or HTTP purely based on the `--stdio` flag; both call the exact same `Process`. The `chat_completion` tool (`mcp/chat_tool.go`) is registered the same way as every other tool — it goes through `models.Session`, so `mova chat`, stdio MCP clients (Claude Desktop, Cursor) and raw HTTP calls (curl, Postman) all drive the exact same code. `get_full_context` (`mcp/context_tool.go`) is the MCP/HTTP equivalent of `mova run` — same `core.BuildContextSections` assembly, same `budget.EnforceLimit` gate, before anything is returned.

## `runtime` — shared bootstrapping

`runtime.FindRoot()` walks up from the current directory looking for `workflow.md`, so `mova` works from any subfolder. Resolution order: `MOVA_PROJECT_PATH` (direct override, no search) → `MOVA_PROJECT_ROOT` (search starts there instead of cwd) → current working directory. `runtime.AutoDetect(root)` returns the sole project under `projects/` when there is exactly one, so `[project]` can be omitted on the CLI.

## Troubleshooting — what to check, by area

- **Budget** (limit not enforced, wrong token count, missing report section): confirm `project.json` (or the task) actually has a `"budget": {"max_tokens": N}` block — `core.ResolveBudget` returns 0 (no gate) if neither has one. Check `budget/limit.go`'s `EnforceLimit` is actually being called from the door you're using (`run_cmd.go`, `chat_cmd.go`, `mcp/context_tool.go`, `mcp/chat_tool.go` — all four, nowhere else). For a wrong count, check which `llm_profile.config` is active — `budget.CountTokens`'s model hint picks the tokenizer encoding from it.
- **Commands** (`mova run`/`mova chat` behaving differently than expected): both now print the same `[Project]/[Context]/[Focus]` status lines and run the same Budget gate before printing/sending anything — see `cli/run_cmd.go` and `cli/chat_cmd.go`. If one shows a status line the other doesn't, that's a bug, not an intentional difference.
- **Documents / `/save`**: every generate/write path funnels through `documents.Save` (`documents/save_service.go`) — check `WriterFor`/`RegisteredExtensions` first if a format seems unsupported, and `documents/pathresolve.go` for "which folder did you mean?" disambiguation issues.
- **Automatic file/directory generation** (natural language not detected, or a directory/file created in the wrong place): check `documents/nl_intent.go`'s `DetectSaveIntent` — it needs a creation verb AND either a directory keyword or a path with a recognizable extension in the same clause (clauses split on " y "/" and "). Windows-style paths (`c:/...`) only resolve on a Windows host — see `documents/pathresolve.go`'s cross-platform guard, which is what a message like "uses Windows format but this server runs on linux" is telling you.
- **Automatic file editing** (edit not detected, wrong file targeted, or nothing gets applied): check `documents/edit_intent.go`'s `DetectEditIntent` first — same clause-based rule as creation, but "edit" additionally requires the target to already exist (`documents.ResolveExistingFile`); if it doesn't, that's why it fell through as a plain chat message instead. No path mentioned falls back to `chatFileState.lastFile` (CLI only — MCP/HTTP has no session state across calls, so always name the file explicitly there). Nothing gets written on `mova chat` until you answer "y"; on MCP/HTTP, until the call includes `"apply_edits": true`.
- **CLI**: `cli/chat_helpers.go` (session/provider plumbing), `cli/chat_save.go` (`/memory`, `/budget`, `/save`), `cli/nl_save.go` (natural-language save) — split out of `chat_cmd.go` to stay under 300 lines each; the REPL loop itself only dispatches.
- **MCP**: `mcp/server.go`'s `Process` dispatch table is the first place to check a tool is actually registered; `mcp/nl_save.go` mirrors `cli/nl_save.go` for `chat_completion`.
- **HTTP**: `http/server.go` has no logic of its own — if HTTP behaves differently from MCP stdio for the same tool/arguments, the bug is in how the HTTP body is being decoded into a `Request`, not in the tool itself.

## What's deliberately not here

There is no compiler, no token-optimization pipeline, no licensing tier, and no `-tags premium` build variant. One binary, one behavior. `focus` above is the complete, permanent implementation — not a "free tier" of something bigger.

## Recent changes — unified delete, extended `/save`, syntax highlighting, Budget-gated `workflow.md`, project.json path config

This section documents one specific change set, on top of everything above, which is still the accurate description of the rest of the architecture.

### Why

Five requirements arrived together, all sharing one constraint: **Chat, MCP, and HTTP must behave identically**, with no per-door logic and no duplicated implementations. That constraint is exactly the architecture this project already had for `save`/`chat_completion`/`estimate_budget` — so the work was to extend that same pattern, not invent a new one.

### New files

| File | What it is |
|---|---|
| `documents/delete_service.go` | `Delete()`/`DeleteRequest`/`DeleteResult`/`FormatDeletePrompt` — the single implementation behind `/delete`, MCP's `delete_path`, and `POST /delete` |
| `documents/save_selection.go` | `SelectContent()`/`GroupExchanges()`/`TranscriptText()`/`ExtractCodeBlocks()`/`StripCodeBlocks()`/`ParseRangeToken()` — the single implementation behind `/save`'s current/`-all`/`-range`/`-c`/`-text` modes |
| `documents/highlight.go` | `DetectLanguage()`/`AutoTagCodeFences()` — the single language-detection implementation behind "code always comes back highlighted" |
| `documents/highlight_test.go`, additions to `budget/*_test.go` | unit coverage for the above |
| `budget/workflow.go` | `LoadWorkflow()` — the Budget-gated pipeline behind "lee/ejecuta workflow.md" |
| `cli/delete_cmd.go` | `/delete`'s interactive Y/N loop — the only door-specific code delete needed (MCP/HTTP use `confirm:true` instead, same convention `apply_edits` already established) |
| `cli/workflow_cmd.go` | recognizes every workflow.md phrasing the spec lists and calls `budget.LoadWorkflow` |
| `docs/i18n/{en,es}/PROJECT_JSON.md` | new project.json field reference, both languages |

### Modified files (non-doc)

- **`core/types.go`** — added `Project.WorkflowPath`/`Project.BudgetPath`/`Project.TokenHistoryPath` (all plain `string`, new JSON keys `workflow_path`/`budget_path`/`token_history_path`) and `ResolveWorkflowPath`. `Project.Repo` is unchanged — still the project's one repository; there is no `repos` array (see **Follow-up simplification** below for why).
- **`budget/history.go`** — added `HistoryPath` (resolves `token_history_path`, or `projects/<project>/mova-token-history.json` by default); `RecordUsage`'s call sites unchanged otherwise.
- **`budget/prices.go`** — removed `ReportPath` field and function. `config/prices.json` is model prices only now.
- **`budget/report.go`** — `WriteReport`'s signature changed from `(root string, prices *PricesConfig, report *Report)` to `(root, project string, proj *core.Project, report *Report)`; added `BudgetReportPath` (resolves `budget_path`, or `projects/<project>/mova-budget-report.md` by default). Every existing caller (`cli/budget_cmd.go`, `mcp/budget_tool.go`, `cli/chat_save.go`) updated.
- **`mcp/server.go`** — `get_workflow`'s tool schema and case now require `project` and route through `budget.LoadWorkflow`; `delete_path` registered (schema + dispatch).
- **`mcp/documents_tool.go`**, **`mcp/documents_tool_helpers.go`** — `delete_path` case added; `save` case accepts `history`/`mode`/`range`/`code_only`/`text_only`; added `pathsArg` (for `delete_path`'s own "one or more files to remove in one call" — unrelated to project.json config) and `chatTurnsArg`.
- **`mcp/chat_tool.go`** — `chat_completion`'s reply now passes through `documents.AutoTagCodeFences` before returning.
- **`http/server.go`** — added `/delete` and `/workflow`, both thin wrappers over the matching MCP tool call (same pattern `/save` already used) — no HTTP-specific logic.
- **`cli/chat_cmd.go`** — dispatches `/delete` and, before natural-language edit/save detection, `handleWorkflowCommand`.
- **`cli/chat_save.go`** — `-all`/`-range N-M`/`-text` flags added; the file's own code/text/range logic was removed and replaced with calls into `documents/save_selection.go` (see "Removed" below); `renderMarkdown` now calls `documents.AutoTagCodeFences` first.
- **`config/prices.json`** — `"report_path"` key removed.

### Follow-up simplification: single paths, no multi-repo

An earlier iteration of this work made `repo` (via a new `repos` array + `RepoConfig`), `workflow_path`, `budget_path`, and `token_history_path` all accept EITHER a single string or an array (a `core.StringList` type that decoded both shapes), so more than one repository or more than one report destination could be configured per project. That was deliberately reverted, on request, in favor of what's actually shipped: every one of those fields is a **plain single string**, full stop.

Why the simpler version won: a project genuinely only ever needs ONE `workflow.md`, ONE Budget report destination, and ONE token-history file — the "more than one" cases that seemed to justify an array turned out, in every real example, to be "more than one DIRECTORY inside the same repo", which `focus` already solves (see [PROJECT_JSON.md](PROJECT_JSON.md)'s "Working on part of `repo`" section) without adding a second config shape to every `*_path` field. `StringList`/`RepoConfig`/`Project.Repos`/`Project.Repositories()`/`HistoryPaths`/`RecordUsageAll`/`BudgetReportPaths` (the array-returning plural forms) existed briefly and were removed — `core/types.go`, `budget/history.go`, and `budget/report.go` now only expose the single-value `HistoryPath`/`BudgetReportPath` functions described above. If a genuine need for more than one repository per project shows up later, re-introducing `RepoConfig` is straightforward (it's preserved in this document's history), but it should wait for that real need rather than being spec'd in ahead of it.

### Removed

- `budget.ReportPath` (function) and `PricesConfig.ReportPath` (field) — superseded by `budget.BudgetReportPath`, driven by `project.json`'s `budget_path` instead of `config/prices.json`.
- `cli/chat_save.go`'s private `exchangePairs`/`transcriptText`/`stripCodeBlocks`/`extractCodeBlocks`/`buildSaveContent`/`parseRangeToken`/`splitFirstToken`(range parsing part) — moved into `documents/save_selection.go` as the shared implementation; `cli/chat_save.go` now only adapts `models.ChatMessage` into `documents.ChatTurn` and calls the shared functions.
- `core.StringList`/`core.RepoConfig`/`Project.Repos`/`Project.Repositories()`, and `budget.HistoryPaths`/`RecordUsageAll`/`BudgetReportPaths` — the array-based multi-repo/multi-path support described in **Follow-up simplification** above, shipped briefly and then removed in favor of single-value fields.

Nothing else was removed. `mcp/server.go`'s legacy per-format tools (`write_file`, `generate_word_contract`, `generate_pdf_document`, `generate_excel_report`, `generate_vector_graphic`, `trigger_diffusion_image`) are untouched — they were already superseded by `save` before this change set, and removing them wasn't asked for and isn't required by anything above.

### Compatibility

Every change here is additive at the `project.json` level: a project that never set `workflow_path`/`budget_path`/`token_history_path` behaves exactly as before — the default output paths (`projects/<project>/mova-budget-report.md`, `projects/<project>/mova-token-history.json`) match what `HistoryPath`/the old `ReportPath` already defaulted to for the one example project that had a custom `report_path` configured. `/save` with no flags still saves just the last response, unchanged; `/save -c`/`-d`/`-append`/`-overwrite`/`-no-overwrite` are unchanged in behavior, only refactored to share their filtering logic with the new modes.

### Execution flow (workflow.md)

```
"lee workflow.md" / "workflow.md <project> [task]"
        │
        ▼
budget.LoadWorkflow(adapter, root, project, task, explicitPath, modelHint)
        │
        ├─ 1. adapter.GetProject(project)            — resolve project.json
        ├─ 2. resolveTask(proj, task)                 — resolve the task, if named
        ├─ 3. core.ResolveWorkflowPath(root, proj, …)  — which workflow.md file
        ├─ 4. core.BuildContextSections(...)           — agents+skills+prompt+focus+memory
        │      (Dedup and Focus already run INSIDE this call — nothing extra to invoke)
        ├─ 5. os.ReadFile(path)                        — read workflow.md's bytes
        ├─ 6. CountTokens(context + workflow.md)        — estimate
        ├─ 7. CheckLimit / EnforceLimit                 — the Budget gate
        │      ├─ over limit  → return {Log, no Content}, err = the ERROR/Suggestion block
        │      └─ within limit → return {Log, Content = workflow.md's text}
        ▼
Chat prints Log + Content and folds Content into sess.System (available to later turns)
MCP/HTTP return Log + Content (or Log + the error) as the tool result text
```

### Bug fixed in this session: the Budget gate was silently skipped with no task

`cli/run_cmd.go`, `cli/chat_cmd.go`, `mcp/context_tool.go` (`get_full_context`), and `mcp/chat_tool.go` (`chat_completion`) all had the same shape of bug: `if t, ok := proj.Tasks[resolvedTask]; ok { EnforceLimit(...) }` — the Budget check only ran when `resolvedTask` matched a real entry in `proj.Tasks`. A project with no `default_task` (or a task name that didn't match), asked to just "read the whole project" with no task named, skipped the Budget gate entirely — the exact scenario a person means by "lee `<project>`" (no task) reaching Claude Console, Codex, Gemini, or any other MCP client.

Fixed by centralizing task resolution in two small, exported functions instead of four ad-hoc copies:

- **`core.ResolveTaskName(proj, taskName)`** — explicit `taskName`, else `proj.DefaultTask`, else the project's one task if it only declares a single one, else `""`. Used by `core.BuildContextSections` itself (so the auto-pick behavior lives in one place) and by every Budget-gate call site, so the task the Budget check validates is always exactly the task the context was actually built from.
- **`budget.ResolveTask(proj, taskName)`** — looks up that name in `proj.Tasks`, or returns a pointer to a zero-value `Task{}` (never `nil`) so `EnforceLimit` always has something to call, even for "no task at all" (which correctly falls back to the project-level `budget`).

All four call sites now do `t := budget.ResolveTask(proj, core.ResolveTaskName(proj, taskName)); if err := budget.EnforceLimit(proj, t, tokens); err != nil { ... }` unconditionally — the gate can no longer be skipped. Verified end-to-end with a real compiled binary and a project with `budget.max_tokens: 5` and no `default_task`: `mova run`, `mova chat`'s "workflow.md `<project>`", the `get_workflow`/`get_full_context` MCP tools (over real stdio JSON-RPC), and `POST /workflow` (over a real HTTP server) all correctly returned the ERROR/Suggestion block instead of the content.

Also fixed in the same pass: `mcp/server.go`'s generic tool-error wrapper used to append "Please use 'list_projects' to see valid projects." to EVERY tool error, including Budget errors — misleading, since a Budget-exceeded error has nothing to do with the project name being wrong. It now only appends that suggestion to errors that don't already carry their own explanation (i.e. anything without a `"\nSuggestion:"` block).

### How to extend this further

- **A new `/save` selection mode**: add a case to `documents.SelectionMode`/`SelectContent` in `save_selection.go` — every door picks it up automatically through `flags.mode()` (CLI) or the `mode` JSON argument (MCP/HTTP); no per-door change needed.
- **A new workflow.md trigger phrase**: add it to `cli/workflow_cmd.go`'s regexes — MCP/HTTP already accept any `project`/`task`/`workflow` argument combination, so no change needed there.
- **A genuine need for more than one repository per project**: don't reach for an array field on `Project` directly — model it explicitly (a `RepoConfig`-shaped type, scoped tasks/focus per repo) once there's a real multi-repo use case driving the design, rather than speculatively supporting it now (see **Follow-up simplification** above).



## Recent changes — Job Engine, Logging, Multiagent orchestrator

This section documents one specific change set, on top of everything
above, which is still the accurate description of the rest of the
architecture.

### Why

Three requirements arrived together: (1) run scheduled, unattended
work (audits, reports, cleanup, memory archiving) on a cron schedule;
(2) an opt-in logging system, disabled by default; (3) orchestrate
several related agents (each an ordinary project) from one parent
config file. All three share the same constraint every prior change
set in this document shared: **CLI, chat, HTTP, and MCP must behave
identically**, through one shared implementation — no per-door logic.

### New packages

| Package | What it is |
|---|---|
| `jobs/` | The Job Engine. `cron.go` (dependency-free 5-field cron parser/matcher), `engine.go` (`RunJob`/`RunProjectJobs`/`RunJobByIndex` — the one execution flow), `actions_tasks.go`/`actions_save.go`/`actions_memory.go`/`actions_delete.go`/`actions_budget.go` (one file per action, each reusing an existing component — `core.BuildContextSections`, `documents.Save`, `Adapter.AppendMemory`/`ArchiveMemory`, `documents.Delete`, `budget.BuildReport`), `scheduler.go` (`RunScheduler`/`RunDueJobs` — the `mova jobs start` daemon) |
| `orchestrator/` | The multiagent orchestrator. `config.go` (`LoadGroupConfig` — reads `projects/<group>/config.json`, auto-discovers agent subdirectories when `"agents"` is omitted), `run.go` (`RunAgent`/`RunGroup` — sequential by design, see "Extensibility" below) |
| `logging/` | The logging system. `config.go` (`LoadConfig` — reads `config/log/logging.json`, always returns a safe disabled default), `level.go` (debug/info/warning/error), `rotate.go` (rotation interval + retention cleanup, pure functions), `logger.go` (`Open`/`Logger` — the file writer, rotation-on-write), `default.go` (`SetDefault`/`L()` — the process-wide default every package logs through) |

### Modified files (non-doc)

- **`core/types.go`** — added `Project.Jobs []JobSpec` (new `jobs` JSON
  key) and the `JobSpec`/`JobMemoryArchive`/`JobBudget` types, kept in
  `core` (not `jobs`) so `core.Project` never has to import the `jobs`
  package — `jobs` imports `core`, never the reverse (same rule
  `Adapter`/`adapters` already follows).
- **`budget/gated_context.go`** (new file) — `BuildGatedContext`
  extracts the exact "assemble context, then apply the Budget gate"
  sequence `cli/run_cmd.go`'s `runProject` used to inline, so
  `orchestrator.RunAgent` (which needs to do precisely that, once per
  agent) reuses it instead of a second copy. **`cli/run_cmd.go`** was
  refactored to call it too — this is the one non-additive change in
  this set, made because the alternative was duplicating the exact
  business logic the spec's "no duplicar lógica de negocio" rule
  forbids.
- **`cli/main.go`** — added the `jobs`/`agents` dispatch cases; opens
  the shared `logging.Logger` once at startup (`logging.Open` +
  `logging.SetDefault`), logs the invoked command line.
- **`cli/jobs_cmd.go`**, **`cli/agents_cmd.go`** (new files) — CLI
  argument parsing only; all business logic lives in `jobs`/`orchestrator`.
- **`cli/help.go`** — added `mova jobs`/`mova agents` to the usage text.
- **`mcp/job_tool.go`**, **`mcp/agent_group_tool.go`** (new files) —
  `"list_jobs"`/`"run_job"`/`"list_agents"`/`"run_agent"` tools.
- **`mcp/server.go`** — registered the four tools above (schema in
  `tools()`, dispatch in `executeTool`'s switch); opens the shared
  Logger in `StartStdio`; logs each tool call.
- **`http/server.go`** — added `/jobs/run` and `/agents/run`, both thin
  wrappers over the matching MCP tool call (same pattern `/save`/`/delete`/`/workflow`
  already used) — no HTTP-specific logic; opens the shared Logger in
  `StartServer`.
- **`config/log/logging.json`** (new file) — the default logging
  config, `"enabled": false`.

### Execution flow — a job run

```
mova jobs run <project> [index]  /  POST /jobs/run  /  MCP "run_job"  /  mova jobs start (cron match)
        │
        ▼
jobs.RunJob(adapter, root, project, proj, spec)
        │
        ├─ 1. runTasks     — core.BuildContextSections per task (or every task for "*"),
        │                    same budget.EnforceLimit gate `mova run` applies, per task
        ├─ 2. runSave      — documents.Save(spec.Save with {date} expanded, accumulated task output)
        ├─ 3. runMemory    — Adapter.AppendMemory(spec.Memory with {date}/{time} expanded)
        ├─ 4. runMemoryArchive — Adapter.ArchiveMemory(days, or core.RetentionDays(proj.Archive))
        ├─ 5. runDelete    — glob-expand spec.Delete, then documents.Delete(Confirm: true)
        ├─ 6. runBudget    — budget.BuildReport + budget.WriteReport (spec.Budget.Focus)
        ▼
*jobs.Result{Project, Steps, Errors} — printed by CLI, returned as MCP/HTTP tool text
```

### Execution flow — a multiagent group run

```
mova agents run <group> [agent]  /  POST /agents/run  /  MCP "run_agent"
        │
        ▼
orchestrator.RunGroup(adapter, root, group, only, task)
        │
        ├─ 1. orchestrator.LoadGroupConfig(root, group)   — read projects/<group>/config.json
        │        (auto-discovers agent subdirectories if "agents" is omitted)
        ├─ 2. for each agent (sequentially):
        │        orchestrator.RunAgent → budget.BuildGatedContext(adapter, root,
        │        "<group>/<agent>", task)  — the SAME function `mova run` uses; an
        │        agent is never a special case, only a project name with a "/" in it
        ▼
[]orchestrator.AgentResult{Agent, Project, Text, Tokens, Err} — one per agent, in order
```

Nothing new was needed in `core/file_adapter.go` for nested project
paths — `GetProject`/`ListProjects` already resolve any path under
`projects/` relative to that directory, so `"ventas_online/vendedor"`
was already a valid project name before this change set; the
orchestrator package's only real job is reading `config.json` and
looping.

### Extensibility — what to touch for a new capability

- **A new Job Engine action** (e.g. a future `"notify"`): add the field
  to `core.JobSpec` (`core/types.go`), add one `actions_notify.go` file
  in `jobs/` following the exact shape of `actions_save.go`/`actions_memory.go`
  (a `run<Name>(jc *jobContext, res *Result)` function that reuses an
  existing component), and add one `run<Name>(jc, res)` call to
  `jobs.RunJob`'s fixed sequence in `jobs/engine.go`. Nothing else in
  the engine changes — `RunProjectJobs`, `RunJobByIndex`, `RunDueJobs`,
  the CLI/MCP/HTTP doors, and every other action are untouched.
- **A new job action that also needs a CLI/MCP/HTTP-visible result**:
  the existing `Result.Steps`/`Result.Errors` already carry through to
  every door via `jobs.Result` — no new plumbing needed unless the
  action needs its own dedicated argument (in which case, add it next
  to `index` in `mcp/job_tool.go`'s `runJobTool` and `cli/jobs_cmd.go`'s
  `runJobsRun`).
- **A new multiagent capability** (e.g. parallel execution): the single
  place to change is `orchestrator.RunGroup`'s for-loop in
  `orchestrator/run.go` — see the comment there for exactly how
  (goroutine per agent + `sync.WaitGroup`) without touching
  `RunAgent`, `GroupConfig`, or any caller's contract.
- **A new logging category**: no code change needed — any string is a
  valid category key in `config/log/logging.json`'s `"categories"`
  object; call `logging.L().Info("your-category", "...")` from
  wherever the trace should come from. Document the new category in
  `config/log/README.en.md`/`README.es.md`'s parameter table for
  consistency (not required for it to function).

### Troubleshooting — Jobs / Orchestrator / Logging

- **A job doesn't fire on schedule**: check `jobs.ParseSchedule`
  accepted the `"schedule"` string (invalid cron syntax is reported as
  a per-job error by `RunDueJobs`, not a crash) and that `mova jobs
  start` is actually running — the daemon only checks while its
  process is alive; a missed minute while the process was down is
  never "caught up" (same as system cron).
- **A job's `"save"` produced an unexpected file format**: the format
  is picked by `documents.Save` from the path's extension, same as
  `/save` — if the extension doesn't match a registered writer, check
  `documents.RegisteredExtensions`.
- **An agent isn't found under a group**: `orchestrator.LoadGroupConfig`
  only auto-discovers subdirectories that contain their own
  `project.json` — a directory without one is silently skipped, not an
  error.
- **Logging produces nothing**: `config/log/logging.json`'s `"enabled"`
  is `false` by default — check that first, then `"level"` (a `"debug"`
  message is dropped if `"level"` is `"info"` or higher) and
  `"categories"` (an explicit, non-empty object only logs the
  categories set to `true`).

### Compatibility

Every change here is additive at the `project.json` level: a project
that never declares `"jobs"` behaves exactly as before, and
`"agents"`/multiagent groups are opt-in (a plain `projects/<name>/project.json`
with no sibling `config.json` is just a regular project, unaffected).
`cli/run_cmd.go`'s refactor to call `budget.BuildGatedContext` is
behavior-preserving — the same status lines print in the same order,
the same Budget gate runs before the same output, verified against the
existing `budget`/`core` test suite plus a full `go build ./...` of the
whole module.

## Recent changes — Terminal UI (`mova ui`)

### Why

A single, low-friction way to browse and edit everything Mova Context
already manages (project.json, workflow.md, model/log configs, memory,
jobs, multiagent groups, reports, logs, execution history) without
memorizing a growing list of flags — and without adding a second
implementation of anything the CLI already does correctly.

### New files (all package `main`, i.e. `cli/` — not a separate
top-level package)

The TUI is presentation, not business logic — like `chat_cmd.go`,
`jobs_cmd.go`, `agents_cmd.go` before it, it belongs in `cli/` so it can
call `must`/`die`/`consolePrint`/`newAdapter`/`sendWithTools` directly
(package `main` can't be imported, so a separate `mova.local/tui`
package would have forced either duplicating those helpers or exporting
CLI internals — both worse than keeping it in the same package).

| File | What it is |
|---|---|
| `cli/ui_cmd.go` | Entry point: `runUI(root, project)`, wires `tea.NewProgram` around `tuiApp` |
| `cli/tui_app.go` | Root model — a screen STACK (push/pop), routes `Update`/`View` to whichever screen is on top |
| `cli/tui_style.go` | Every Lip Gloss style, in one place |
| `cli/tui_menu.go` | ONE generic list-based menu screen, reused by nearly every section |
| `cli/tui_mainmenu.go` | Builds the main menu's items |
| `cli/tui_fileview.go` | ONE generic file viewer/editor (textarea), reused for project.json, workflow.md, memory.md, model configs, logging.json |
| `cli/tui_projectpicker.go` | Lists projects (`core.Adapter.ListProjects`) and routes to any destination screen |
| `cli/tui_dashboard.go` | Per-project menu (project.json, memory.md, Jobs, Reports, history) |
| `cli/tui_reports.go` | Generic directory listing, reused for reports/ and memory-archive/ |
| `cli/tui_textscreen.go` | Read-only screen for in-memory text (job/agent run output — no backing file) |
| `cli/tui_jobs.go` | Lists + runs a project's jobs via `jobs.RunJobByIndex` |
| `cli/tui_agents.go` | Lists groups/agents + runs them via `orchestrator.RunGroup` |
| `cli/tui_models.go` | Lists/edits `config/models/**/*.json` |
| `cli/tui_logs.go` | Tails the active log file (`logging.LoadConfig`'s path), auto-refreshing |
| `cli/tui_chat.go` | Chat screen — same session/tool-loop setup as `runChat` |
| `cli/tui_paths.go` | Small path helpers shared by the screens above |

### Modified files

- **`cli/chat_helpers.go`** — `sendWithTools` gained one new parameter,
  `emit func(string)`. `nil` preserves the exact original behavior
  (writes straight to the terminal via `consolePrint`, as `mova chat`'s
  REPL always has). This is the one change that made the TUI's chat
  screen possible without a second copy of the model-call/tool-loop
  logic: it passes its own `emit` (a no-op — the TUI shows replies once
  complete, not token-by-token, to avoid writing to stdout mid-render
  and corrupting the Bubble Tea screen) instead of `consolePrint`.
- **`cli/chat_cmd.go`** — its one call site now passes `nil` explicitly;
  behavior is unchanged.
- **`cli/main.go`** — bootstraps logging then calls `dispatch(root)`.
- **`cli/dispatch.go`** — added the `"ui"` case.
- **`src/go.mod`** — added `github.com/charmbracelet/bubbletea`,
  `github.com/charmbracelet/lipgloss`, `github.com/charmbracelet/bubbles`
  as direct requires, same single-block style the existing four
  dependencies already used. No `go.sum` entries were hand-added: this
  project's `go.sum` was already incomplete for `glamour`'s own
  transitive dependencies before this change, meaning `go build` already
  relied on Go's standard "fetch + verify + add missing go.sum entries
  automatically" behavior — the same mechanism now covers the three new
  dependencies too, with zero extra steps for double-click installers or
  a plain `go build` (both already just call `go build`/`go install`).

### Extensibility — adding a new TUI section

Almost always: call `newMenuScreen(title, items, help)` with a new
`menuItem` whose `onSelect` pushes an existing screen constructor
(`newFileScreen`, `newDirListScreen`, `newTextScreen`) pointed at the
right path — see any entry in `tui_mainmenu.go` or `tui_dashboard.go`
for the pattern. A genuinely new interaction (not view/edit/list/run)
gets its own `tuiScreen` implementation, following the shape of
`tui_jobs.go` or `tui_chat.go` (small, single-purpose, calling straight
into an existing package — never re-implementing what that package
already does).

### Troubleshooting

- **`mova ui` fails to start**: it needs a real terminal (TTY) — running
  it with stdin/stdout redirected to a file or pipe will fail the same
  way any Bubble Tea program would.
- **First build after pulling this change is slower / needs internet**:
  expected — Go is downloading the three new dependencies the first
  time, exactly like it already did for `glamour`. Subsequent builds
  use the local module cache.
- **A file edited in the TUI didn't save**: only `Ctrl+S` writes to
  disk; navigating away (`Esc`) without saving discards the in-memory
  edit, by design, so accidental navigation never loses — or corrupts —
  a file silently.

## Recent changes — installers open a ready-to-use console

### Why

The double-click installers (`installers/windows/`, `installers/macos/`,
`installers/linux/`) already copied the `mova` binary and updated
`PATH`, but still left one manual step: closing/reopening a terminal
for that `PATH` change to take effect, then navigating to a useful
directory. This change removes that step — each installer now finishes
by asking which console to leave open, already on `PATH` and already
in the Mova root, so `mova` works the moment the installer's console
appears (or the current window hands off to an interactive shell).

### What changed (installer scripts only — no Go source touched)

- **`installers/windows/install.ps1`** — after the existing
  build/copy/`PATH` steps, prompts `[1] PowerShell (default) / [2] CMD
  / [3] Don't open one` and launches the chosen console via
  `Start-Process`, with `$env:PATH`/`set PATH=` extended inline for
  that specific process (a brand-new process isn't guaranteed to see a
  registry `PATH` change made moments earlier in the same login
  session) and its working directory set to the repo root.
- **`installers/macos/install.command`** — after the existing steps,
  prompts `[1] Continue right here (default) / [2] Open a new Terminal
  window / [3] Don't open one`. Option 1 exports `PATH` for the
  current shell and `exec`s a fresh login shell in place (same
  window, same process slot) — the simplest form of "hand off a
  ready-to-use console" since `.command` files already run inside
  Terminal.app. Option 2 uses `osascript` to open a second Terminal
  window via AppleScript, falling back to option 1's behavior if
  AppleScript automation isn't available.
- **`installers/linux/install.sh`** — same three-way prompt as macOS.
  Option 2 tries `$TERMINAL` first, then auto-detects one of
  `x-terminal-emulator`, `gnome-terminal`, `konsole`, `xfce4-terminal`,
  `tilix`, `alacritty`, `kitty`, `xterm` (Linux has no single "the"
  terminal emulator, unlike Windows/macOS), falling back to option 1's
  in-place `exec` if none is found.
- **Both `install.command` and `install.sh`** — also fixed `mktemp`'s
  temp-binary template to `mktemp "${TMPDIR:-/tmp}/mova.XXXXXX"` (the
  positional-template form works identically on BSD/macOS and GNU/Linux
  `mktemp`; the previous `-t mova` form is BSD-only and fails on GNU
  systems with "too few X's in template").

### Compatibility

Purely additive to the installer scripts — the build/copy/`PATH` logic
from the earlier installers section is untouched, so a person who
chooses "Don't open one" gets exactly the previous behavior (a note to
open a new terminal manually). No Go source, `go.mod`, or `Makefile`
target changed; `make install`/`make build-all` behave exactly as
before.

### Generating installers for a new release

See [`installers/README.md` § Generating installers for a new
version](../installers/README.md#generating-installers-for-a-new-version) —
the scripts are version-independent (no version number or file list
hardcoded anywhere in them), so a new release only means optionally
running `make build-all` to refresh `dist/`'s prebuilt binaries before
packaging; the installer scripts themselves never need to change.

## Recent changes — installers set MOVA_PROJECT_ROOT (work from any drive)

### Why

A person can clone/install Mova Context in one place (e.g. `C:\mova`)
while the actual codebase a project audits/works on lives somewhere
completely different (e.g. `D:\my-app`). The "different codebase
location" half of that was already fully supported — every project.json
`"repo"` accepts an absolute path, and every subsystem that touches
project files already resolves it correctly (see the `filepath.IsAbs`/
`documents.IsAbsCrossPlatform` checks across `core/focus/`,
`documents/pathresolve.go`, `documents/types.go`, `budget/`, `jobs/`).
What was missing was the OTHER half: finding Mova's *own* `workflow.md`
when the person is standing inside that external folder and just types
`mova` — `runtime.FindRoot()` only searches upward from the current
directory (plus the binary's own directory as a last resort), neither
of which reaches a separate drive/volume.

`MOVA_PROJECT_ROOT` already existed as an escape hatch for this
(documented for MCP clients launching `mova` from an unrelated working
directory — see `runtime/root.go`), but nothing set it automatically,
so a person would hit "No Mova project was found" the first time they
tried this from outside the repo, with no obvious fix short of reading
`runtime/root.go`'s error message closely.

### What changed (installer scripts + docs only — no Go source touched)

- **`installers/windows/install.ps1`** — after adding `<GOPATH>\bin` to
  `PATH`, also sets the user-level `MOVA_PROJECT_ROOT` environment
  variable to this repo's path via the registry (`[Environment]::
  SetEnvironmentVariable(...)`), only if it isn't already set. The
  console opened afterward (PowerShell/CMD) also gets it applied inline
  for that immediate session, same reasoning as the `PATH` inline
  application already documented above.
- **`installers/macos/install.command`**, **`installers/linux/install.sh`**
  — same idea via `~/.zshrc`/`~/.bashrc`: append `export
  MOVA_PROJECT_ROOT="$REPO_ROOT"` if not already present, export it in
  the current session immediately, and include it in the "open a new
  terminal window" path too. Also switched the "continue in this same
  window" handoff from `exec "$SHELL" -l` (login shell — doesn't
  necessarily source `.bashrc`, depending on whether `.bash_profile`
  chains to it) to `exec "$SHELL" -i` (interactive shell — and since
  `PATH`/`MOVA_PROJECT_ROOT` are already exported in the running
  installer process before the `exec`, they carry over via the
  replaced process's inherited environment regardless of which
  rc-file the new shell does or doesn't source).
- **Documentation** — `docs/i18n/{en,es}/COMMANDS.md` § Environment
  variables gained a full "working across different drives/locations"
  explanation with a worked Windows/Linux/macOS example;
  `docs/i18n/{en,es}/PROJECT_JSON.md`'s `repo` field description now
  points to it; `docs/i18n/{en,es}/README.md` gained a short callout;
  `installers/README.md` documents the new install step.

### How this was verified

A real project (`project.json` with `"repo"` set to an absolute path
entirely outside the Mova repo tree, e.g. `/tmp/external-drive/app`)
was run end-to-end from a working directory that has nothing to do
with either location:

1. `mova run <project>` — confirmed the assembled context correctly
   included `focus` content read from the external path.
2. `mova jobs run <project>` (with a `"save"` action) — confirmed the
   report was written *inside the external folder*
   (`/tmp/external-drive/app/reports/...`), not under the Mova root.
3. Without `MOVA_PROJECT_ROOT` set, the exact same commands correctly
   fail with "No Mova project was found" — confirming the fix actually
   addresses a real gap, not a false alarm.
4. With `MOVA_PROJECT_ROOT` set (as the installers now do
   automatically), both commands above succeed from a brand-new,
   non-login interactive shell — the realistic "open a new terminal
   window" scenario — with zero manual configuration.

### Compatibility

Purely additive: existing installs where `mova` is already run from
inside the repo (the original, still-supported workflow) are
unaffected — `MOVA_PROJECT_ROOT` is only ever consulted as an *extra*
starting point in `runtime.FindRoot()`'s search order, never a
replacement for the existing upward search. A person who already has
`MOVA_PROJECT_ROOT` set to something else on purpose is left alone
(the installer only sets it when absent). No Go source, `go.mod`, or
`Makefile` target changed.

## Recent changes — UNC paths, WSL, and Docker (verified, docs only)

### Why

Following the `MOVA_PROJECT_ROOT` work above, the natural next question
was whether the same "external `repo` path" support extends to Windows
network shares (UNC, `\\server\share`), WSL, and Docker — all common
ways a real team's codebase ends up somewhere Mova's own install
doesn't naturally reach.

### What was found

**No code change was needed — this was already correctly implemented.**
Verified by reading Go's own standard library source
(`internal/filepathlite/path_windows.go`'s `IsAbs`/`volumeNameLen`,
and `Clean`'s UNC-preserving logic) rather than assuming: on Windows,
`filepath.IsAbs` explicitly treats a UNC prefix (`\\server\share`) as
a volume name and correctly reports it as absolute, and `filepath.Join`/
`filepath.Clean` preserve that prefix through every join/clean
operation instead of stripping or mis-parsing it. Every place across
the codebase that resolves a project's `"repo"` — `core/focus/`,
`budget/report.go`, `budget/history.go`, `jobs/actions_delete.go`,
`documents/types.go`, `cli/tui_paths.go`, `cli/tui_logs.go` — already
uses `filepath.IsAbs`/`filepath.Join` consistently (audited with a
project-wide `grep` for every absolute-path check in the repo), so a
UNC `"repo"` works today with zero additional code.

Two related scenarios turned out to be pure usage patterns of that
same already-verified behavior, needing documentation rather than code:

- **WSL**: a Linux `mova` inside WSL2 reaching a Windows drive is just
  a normal Linux absolute path (`/mnt/d/...`, auto-mounted by WSL2). A
  Windows `mova.exe` reaching into a WSL distro's filesystem does so
  via a UNC path (`\\wsl$\Ubuntu\...` / `\\wsl.localhost\Ubuntu\...`)
  — the exact same UNC support above, no separate case.
- **Docker**: a bind-mounted host directory is simply a native
  absolute path from the container's point of view — the exact same
  "external absolute `repo`" behavior already verified for the
  `MOVA_PROJECT_ROOT` change, just inside a container's filesystem
  namespace instead of a second drive.

The one honest limitation documented alongside this: a UNC or
drive-letter path in `"repo"` (or typed in chat/MCP/HTTP — see
`documents.IsAbsCrossPlatform`/`normalizeAbsPath` in
`documents/pathresolve.go`) only resolves when Mova itself runs as a
**Windows** binary — there is no OS-level UNC support on Linux/macOS,
so reaching a Windows share from those requires mounting it first
(`mount -t cifs`, or Finder on macOS) and using the resulting native
path instead. `normalizeAbsPath` already surfaces this as a clear,
actionable error rather than silently writing to the wrong place.

### Where this is documented

`docs/i18n/{en,es}/COMMANDS.md` § Environment variables, under
"Network shares — UNC paths", "WSL", and "Docker / containers" — each
with a worked example (`project.json` snippet and, for Docker, the
matching `docker run` invocation).

### Compatibility

Documentation-only change — no Go source, `go.mod`, `Makefile`, or
installer script was touched.

## Recent changes — Token Firewall (Sanitizer, Cache Layout Guard, Circuit Breaker, Context Cache)

### Why

Every existing cost control in Mova (`max_tokens`, the Budget gate)
answers "is this context too big?" — none of them answered "is this
context bigger than it needs to be?", "is this prompt shaped so a
provider's own caching can actually discount it?", or "has this
project's spend quietly crossed a line no one is watching?". The Token
Firewall answers those three questions, deterministically, with no AI
involved and no new external dependency — consistent with every other
Mova subsystem's design (Job Engine, Focus, Budget itself).

### Architecture — one pipeline, one chokepoint

```
budget.BuildGatedContext(adapter, root, project, task)
        │
        ├─ core.BuildContextSections   (unchanged — the existing assembly)
        │
        ├─ [1] SanitizeCached          (contextcache.go → sanitize.ApplyFocus/ApplyMemory)
        ├─ [2] CheckCircuitBreaker     (spend.go, reads/writes mova-spend.json)
        ├─ [3] EnforceLimit            (limit.go — the pre-existing max_tokens gate, unchanged)
        │
        ▼
GatedContext{Text, Tokens, Sanitize, CircuitBreaker, Err}
```

`BuildGatedContext` was already the single function `mova run`
(`cli/run_cmd.go`) and `orchestrator.RunAgent` shared (see the earlier
"budget.BuildGatedContext" change). This change set additionally
**consolidated three call sites that had their own copy of "build then
gate"** onto the same function: `cli/chat_cmd.go`'s `runChat`,
`cli/tui_chat.go`'s `newChatScreen`, `jobs/actions_tasks.go`'s
`runTasks`, and `mcp/chat_tool.go`'s `chat_completion` tool. Every one
of those five call sites — CLI, TUI, Jobs, MCP, and the original
`mova run`/orchestrator pair — now shares the exact same Token Firewall
automatically, with zero duplicated pipeline logic anywhere.

The **Cache Layout Guard** is deliberately NOT part of
`BuildGatedContext`'s output text — reordering Header-first `Full()`
into static-prefix-first only matters for an actual model call, so it
stays a separate function (`budget.LayoutForCache`) that only the four
chat-sending call sites (CLI, TUI, MCP, and — for the preview shown in
`mova-budget-report.md` — `BuildReport`) call. `mova run`'s printed
output and the report's other sections keep `Full()`'s original,
more-readable Header-first order untouched.

### New packages/files

| File | What it is |
|---|---|
| `sanitize/sanitize.go` | Text-level rules: `dedupeRepeatedLines` (timestamp-aware — strips a leading `YYYY-MM-DD HH:MM:SS`-style prefix for COMPARISON only, so real logs where every line's timestamp differs are still recognized as repeats), blank-line collapsing, optional comment-block stripping |
| `sanitize/dedupe.go` | `DedupeLeadingBlocks` — cross-file leading-block dedup (license headers, import blocks) shared across several Focus files |
| `sanitize/sections.go` | `Apply`/`ApplyFocus`/`ApplyMemory` — wires the rules above onto a `*core.ContextSections`, parsing/reassembling `core/engine.go`'s `"FOCUS:<path>"` marker convention **including its `"\n\n---\n## FOCUS\n"` preamble**, which an earlier version of this file's `splitFocusBlocks` silently dropped on rebuild (a real bug caught during testing — see "What was found during review" below) |
| `budget/cachelayout.go` | `LayoutForCache` — builds the static-prefix/dynamic-tail layout + a stability fingerprint (sha256, truncated for readability) |
| `budget/spend.go` | `SpendState`/`CheckCircuitBreaker`/`RecordSpend`/`EstimateCostFor` — the spend-governance gate, backed by `mova-spend.json` |
| `budget/contextcache.go` | `SanitizeCached` — per-project local memoization of the Sanitizer's result, backed by `mova-context-cache.json` |
| `budget/report_pipeline.go` | The report sections these stages add: Sanitizer, Cache Layout Guard, Circuit Breaker, Token Firewall Summary (before/after) |
| `core/budget_config.go` | Extended with the new `BudgetConfig` fields and their `core.XEnabled(cfg)` reader functions (all default-true — see "Configuration model" below) |
| `models/types.go`, `models/chat.go` | `ChatMessage.CacheBoundary`/`Session.CacheBoundary` — an inert `int` field every provider except Anthropic ignores entirely |
| `models/provider_anthropic.go` | `systemField` — builds Anthropic's two-content-block `system` array with `cache_control` when a boundary is set; a plain string (unchanged) otherwise |

### Configuration model — default-on, explicit opt-out

Every Token Firewall field defaults to **enabled** (`nil` in Go, absent
in JSON, or an explicit `true` all mean "on") — the opposite default
from `core.ToolsEnabled` (tools default OFF), because these stages are
safety/efficiency defaults, not opt-in extras. Each is independently a
`*bool` in `core.BudgetConfig`, read through a matching
`core.XEnabled(cfg)` function (`CacheGuardEnabled`,
`CircuitBreakerEnabled`, `TokenEstimationEnabled`,
`DetailedReportsEnabled`, `ContextCacheEnabled`, `SanitizerEnabled`) —
one place per toggle to ask "is this on for this project?", instead of
every caller re-deriving the nil-means-true logic itself.

### What was found during review (fixed, not just noted)

A dedicated review pass — running the real binary against the shipped
example (`projects/ejemplo-token-firewall/`), not just reading the code
— caught two real bugs before release:

1. **Log dedup missed the realistic case.** The first
   `dedupeRepeatedLines` implementation compared lines byte-for-byte,
   which correctly found duplicates in a synthetic test but found
   **zero** in a realistic log where every line has a unique timestamp
   (otherwise identical). Fixed by comparing lines with a leading
   timestamp pattern stripped, while still keeping (and printing) the
   original, timestamped line. Verified: 47 of 48 near-identical log
   lines now collapse correctly.
2. **`splitFocusBlocks` dropped the `"## FOCUS"` header on rebuild.**
   `core/engine.go` prepends `"\n\n---\n## FOCUS\n"` before the first
   `"FOCUS:<path>"` marker; the original split/rejoin logic didn't
   account for that preamble and silently discarded it when
   `ApplyFocus` reassembled `sections.Focus`, corrupting the actual
   prompt sent to the model. Fixed by capturing and preserving the
   preamble explicitly (`splitFocusBlocks` now returns it alongside the
   parsed blocks). Verified with `mova run`'s actual printed output —
   the `## FOCUS` header survives byte-for-byte.

Both were caught specifically because the review used the real,
end-to-end example rather than unit-level reasoning alone — the
practical argument for shipping a working example alongside every
feature in this codebase, not just documentation describing one.

### Performance (measured, not assumed)

Back-to-back real runs of the shipped example, same machine: ~69ms for
a cold run (no Context Cache entry yet), ~57ms once the Context Cache
is warm. Most of that remaining time is process startup and file I/O —
the Sanitizer itself, on content this size, runs in single-digit
microseconds (pure string/regex operations, no I/O, no network).

### Extensibility

- **A new Sanitizer rule**: add a function to `sanitize/sanitize.go`
  following `dedupeRepeatedLines`'s shape, call it from `Text` gated by
  a new `Config` field, and add the matching field to
  `core.SanitizeConfig` (`core/budget_config.go`). Nothing else changes
  — `ApplyFocus`/`ApplyMemory`/`Apply` and every caller stay the same.
- **A new provider's cache mechanism**: only `models/provider_<name>.go`
  needs a `cache_control`-equivalent, reading `ChatMessage.CacheBoundary`
  the same way `provider_anthropic.go` does — the layout stage
  (`budget.LayoutForCache`) never needs to know which provider will
  read its output.
- **A new Circuit Breaker ceiling**: add a field to `BudgetConfig`, a
  check in `spend.go`'s `CheckCircuitBreaker`, and a line in
  `circuitBreakerMessage` — `GatedContext`/the report already surface
  whatever `CircuitBreakerResult` carries.

### Compatibility

A `project.json` that never touches any Token Firewall field behaves
exactly as it always did, with one exception made deliberately and
documented above: **the new fields default to enabled**, meaning a
project written before this change set gets Sanitizer/Cache Layout
Guard/Circuit Breaker-mechanism/Context Cache automatically, at their
conservative defaults (Sanitizer's `strip_comments` off, no ceilings
configured so the Circuit Breaker has nothing to enforce, Cache Layout
Guard reordering the prompt with zero behavioral downside even where a
provider doesn't cache). Nothing that previously worked stops working;
`mova run`'s printed output, `mova-token-history.json`, and every
existing report section are unchanged in format and meaning.
