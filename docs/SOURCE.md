# SOURCE — codebase structure (English only)

One binary (`mova`), no build tags, no editions. This document explains how the source is organized, how the two extension points (**Adapters** and **Focus Resolvers**) work, and how to add your own.

## Layout

```text
src/
├── core/                  the engine — zero external dependencies
│   ├── types.go            Project, Task, Adapter-shared structs (mirrors project.json)
│   ├── engine.go           BuildContext() — assembles agents+skills+prompt+memory+focus
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
├── adapters/              alternate storage backends
│   └── db_adapter.go        Postgres/MongoDB Adapter — same interface as file_adapter
│
├── cli/                   the `mova` command — thin dispatcher, no business logic
│   ├── main.go              command switch (run, memory*, list, init, search, mcp)
│   ├── adapter_select.go    decides file vs. db Adapter for a given project
│   ├── memory_mgmt.go       memory-clear / memory-config
│   └── console_*.go         per-OS terminal helpers
│
├── mcp/                   MCP (Model Context Protocol) JSON-RPC layer
│   └── server.go            Process() — same engine, exposed as MCP tools
│
├── http/                  HTTP transport
│   └── server.go            thin wrapper: POSTs JSON-RPC bodies into mcp.Process()
│
└── runtime/               shared bootstrapping
    └── root.go              FindRoot() (locates workflow.md) + AutoDetect()
```

Everything under `core/` has **zero external dependencies** — it only touches the Go standard library. `adapters/` is where anything that needs a third-party driver (a SQL driver, for instance) lives, kept out of `core/` on purpose so the engine stays trivially portable.

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

## Transports (MCP / HTTP) — same engine, different door

`mcp/server.go`'s `Process(adapter, root, req)` is the single dispatcher for MCP JSON-RPC requests (`initialize`, `tools/list`, `tools/call`). `http/server.go` is a thin wrapper: it decodes an HTTP POST body into the same `Request` struct and calls `Process` — no protocol logic of its own, no separate code path from stdio. `cli/main.go`'s `mova mcp start` picks stdio or HTTP purely based on the `--stdio` flag; both call the exact same `Process`.

## `runtime` — shared bootstrapping

`runtime.FindRoot()` walks up from the current directory looking for `workflow.md`, so `mova` works from any subfolder. Resolution order: `MOVA_PROJECT_PATH` (direct override, no search) → `MOVA_PROJECT_ROOT` (search starts there instead of cwd) → current working directory. `runtime.AutoDetect(root)` returns the sole project under `projects/` when there is exactly one, so `[project]` can be omitted on the CLI.

## What's deliberately not here

There is no compiler, no token-optimization pipeline, no licensing tier, and no `-tags premium` build variant. One binary, one behavior. `focus` above is the complete, permanent implementation — not a "free tier" of something bigger.
