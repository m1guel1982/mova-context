# Mova Context — Technical Architecture Reference

**One binary. One behavior.** `mova` ships as a single Go executable. Every capability is reachable
identically from four doors — **CLI**, **Chat (REPL)**, **MCP** (stdio/HTTP), and **HTTP REST** — because
all four call the exact same underlying function. There is never a second, door-specific implementation
of business logic.

## 1. System map

```text
src/
├── core/          Engine — zero external dependencies (stdlib only)
│   └── focus/       Focus/Exclude resolution engine (incl. AST, §3)
├── dedup/         Exact-paragraph deduplication
├── adapters/      Alternate storage backends (Postgres/MongoDB)
├── documents/     Save service · Office formats (docx/xlsx/pdf/svg)
├── sanitize/      Context Governance: Sanitizer, secret detection, PII Masking
├── budget/        Token/cost estimation, circuit breaker, context cache
├── models/        LLM providers — local + cloud, single source of truth per model
├── orchestrator/  Multi-agent orchestrator — groups of projects run as agents
├── diagram/       Renders a project's real pipeline as SVG/PNG/PDF
├── trace/         `context-trace` engine — pre-inference context/budget/governance audit (§4)
├── cli/           `mova` command — thin dispatcher (CLI + `mova chat` REPL)
├── mcp/           MCP JSON-RPC layer — tools/list, tools/call
├── http/          HTTP transport — thin wrapper over mcp.Process()
└── runtime/       FindRoot()/AutoDetect() — shared bootstrapping
```

**Out of scope, already removed:** `jobs/` (cron scheduler) and the terminal UI (`mova ui`) — cut to
specialize on pre-inference governance instead of staying a "does everything" tool. If you see old notes
mentioning either, they no longer exist in this code.

## 2. Core execution pipeline

```
core.BuildContext(adapter, root, project, task)
    1. Adapter.GetProject(name)        — reads project.json, always fresh
    2. ResolveTaskName(...)            — explicit → default_task → sole task → ""
    3. resolve agents/skills/prompt    — domain + i18n/[lang] + fallback "en"
    4. inject variables                — project-level, then task-level overrides
    5. append memory.md (if present)
    6. resolve `focus`/`exclude`       — incl. AST syntax (`file::kind=name`)
    7. dedup.Paragraphs (Agents→Skills→Prompt→Focus→Memory)
    ▼
finished context (string)
```

`mova run`, `get_full_context` (MCP/HTTP), and `mova chat` all call **exactly this function** — no second
code path.

## 3. Focus / Exclude — AST on both sides (new)

`project.json`'s `"focus"` and `"exclude"` are lists of targets (files, directories, globs, or
`file::kind=symbol` via Tree-Sitter). Previously only `focus` supported AST; `exclude` now does too
(`ApplyAstSymbolExcludes`, `core/focus/resolvers/ast_symbol.go`) — lets you analyze a whole file while
leaving out just one legacy function/symbol (see the `03-tokenomics-context-trace` example).

## 4. `context-trace` — pre-inference audit engine

`src/trace/` is the single engine behind `mova context-trace`/`mova trace` (CLI), `/context-trace` (Chat),
the `context_trace` MCP tool, and `POST /api/v1/context-trace` (HTTP).

**Two modes, two token-counting paths:**
- **Project mode** (`AnalyzeLocal`) — never counts tokens itself; calls `budget.BuildReport`, the same
  function `mova budget` calls.
- **Discovery mode** (`AnalyzeRemote`, remote repo, no `project.json`) — counts tokens file by file so it
  can build the "tokens by directory" breakdown.

**Audit identity** (new — answers Audit Matrix questions #3, #10, #11, see `README.md`):
`Data.AgentClient` / `Data.TargetModel` / `Data.PolicyAuthor`, computed in `trace.applyAuditIdentity` and
`core.ResolvePolicyAuthor`/`core.TargetModelFor`. Never left empty — see `docs/i18n/en/ARTIFACTS.md`.

## 5. Transports — same engine, different door

CLI, Chat, MCP (stdio/HTTP), and HTTP REST all call `mcp.Process()` or the same `core`/`trace`/`budget`
functions. The only difference between doors is **who's** calling — see `AgentClient` above.

## 6. Diagrams

`mova run <project> --diagram [--export svg,png,pdf]` (`diagram/build.go` + `svg.go`) renders a project's
real pipeline from its `project.json` — nothing invented. Accepts `--path <file>.<format>` as a full
destination (not just a directory) when a single format is requested.

## 7. Extensibility

Four Open/Closed extension points: **Adapters** (`core.Adapter`, under `adapters/`), **Focus Resolvers**
(`core/focus/resolvers/`), **Model Providers** (`models/`, `Provider` interface), **Save Writers**
(`documents/`, `RegisterWriter(".ext", ...)`). Each: implement an interface, register it, nothing else
changes.

## 8. Deliberately not here

No custom compiler, no licensing tier, no `premium` build tags. `focus`/`exclude` is the complete,
permanent implementation. Every capability is additive at the `project.json` level.
