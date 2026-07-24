# SOURCE — codebase structure (English only)

One binary (`mova`), no build tags, no editions. This document explains how the source is organized, how the two extension points (**Adapters** and **Focus Resolvers**) work, and how to add your own.

## Layout

```text
src/
├── core/                  the engine — zero external dependencies
│   ├── types.go            Project, Task, Adapter-shared structs (mirrors project.json)
│   ├── engine.go           BuildContext()/BuildContextSections() — assembles agents+skills+prompt+memory+focus
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
│   ├── limit.go             CheckLimit()/EnforceLimit() — hard "budget": {"max_tokens": N} gate, stops execution before sending to a model or returning a project's context (mova run, mova chat, get_full_context, chat_completion)
│   ├── history.go           mova-token-history.json — per-provider {local,api} token accumulators only, no prompts/content ever stored
│   └── report.go            RenderMarkdown()/WriteReport() — mova-budget-report.md (English only, see SUPPORTED_FORMATS.md); budgetLimitSection/budgetRecommendations/providerComparisonNote — the enriched percent-used/diff/cross-provider explanation
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
