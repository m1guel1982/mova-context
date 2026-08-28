# Mova Context — Architecture & Technical Reference

**One binary. One behavior.** `mova` ships as a single Go executable with no build tags, no editions. Every capability is reachable identically from four doors — **CLI**, **Chat (REPL)**, **MCP** (stdio/HTTP), and **HTTP REST** — because every door calls into exactly the same underlying function. There is never a second, door-specific implementation of business logic.

---

## 1. System Map

```text
src/
├── core/          Engine — zero external dependencies (stdlib only)
│   └── focus/       Focus Resolution Engine (Extension Point 2 — see §3.2, §16)
├── dedup/         Exact-paragraph deduplication (shared leaf package)
├── adapters/      Alternate storage backends (Postgres/MongoDB)
├── documents/     Save Service · Office formats (docx/xlsx/pdf/svg) · Path Resolution
├── sanitize/      Token Firewall: Sanitizer, PII Masking
├── budget/        Token/cost estimation, Budget gate, Token Firewall pipeline (incl. contextcache.go — §17)
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
| `documents/` | `.docx`/`.xlsx`/`.pdf` hand-written — no office-format dependency |

---

## 2. Core Engine & Execution Pipeline

```
core.BuildContext(adapter, root, projectName, taskName)
        ├─ 1. Adapter.GetProject(name)          — read project.json (fresh, every call — see §17)
        ├─ 2. ResolveTaskName(proj, taskName)    — explicit → default_task → sole task → ""
        ├─ 3. resolve agents/skills/prompt        — domain + i18n/[lang] + fallback "en"
        ├─ 4. inject variables                    — project-level, then task-level overrides
        ├─ 5. append memory.md (if present)
        ├─ 6. resolve `focus` (if declared)        — see §16 (multi-target, dirs, globs, absolute paths)
        ├─ 7. dedup.Paragraphs across Agents→Skills→Prompt→Focus→Memory
        ▼
finished context (string) + sections.FocusItems (§16.3)
```

`mova run`, `get_full_context` (MCP/HTTP), and `mova chat`/`chat_completion` all call **exactly this function** (via `budget.BuildGatedContext`, §6.5) — no second code path exists.

### Adapter interface (storage abstraction)
```go
type Adapter interface {
    GetKnowledge(kind, domain, lang, name string) (string, error)
    GetProject(name string) (*Project, error)
    ListProjects() ([]ProjectSummary, error)
    GetMemory(project string) (string, error)
    AppendMemory(project, entry string) error
    Search(query, domain string) ([]SearchResult, error)
}
```
The engine never knows whether data comes from disk (`core.FileAdapter`, default) or a database (`adapters.DBAdapter`). `core.ProjectJSONFingerprint(root, project)` (§17) returns `ok=false` for DB-backed projects — there's no `project.json` file to watch, and that's a normal, silent no-op, never an error.

---

## 3. Architecture Extension Points

Four Open/Closed seams. Each: implement an interface, register it, nothing upstream changes.

**3.1 Adapters** — implement `core.Adapter` under `adapters/`, wire in `cli/adapter_select.go`.

**3.2 Focus Resolvers** — `focus` (`project.json` key) targets specific files/dirs/symbols instead of the whole repo. Deterministic cascade, no LLM call. See **§16** for the full resolver table and this round's fixes.

**3.3 Model Providers** — implement `Provider` under `models/`, add one `case` in `NewProvider`.
```go
type Provider interface {
    Chat(ctx context.Context, model string, mc *ModelConfig, messages []ChatMessage) (string, Usage, error)
}
```

**3.4 Save Writers** — implement `IFileWriter`, `RegisterWriter(".ext", myWriter{})` from `init()`.

---

## 4. Unified Save Service & Natural-Language Editing

`documents.Save(root, SaveRequest{Path, Directory, Content, Append, Overwrite, Repo})` — one call, reachable from `/save` (chat), the `save` MCP tool, and `POST /save`.

Natural-language creation (`DetectSaveIntent`) needs a creation verb **and** a directory keyword/extension in the same clause. Natural-language editing (`DetectEditIntent` + `ResolveExistingFile`) only fires when the named file **already exists**; otherwise it's creation. Edits go through `ReadEditableContent → BuildEditPrompt → ExtractEditedContent → DiffLines (LCS) → confirm → write`. Confirmation: `mova chat` asks *"Apply this change? (y/n)"*; MCP/HTTP use an explicit `apply_edits` argument.

---

## 5. Autonomous Chat Tool-Calling Protocol

`project.json`'s `"tools": {"enabled": true, "allow": [...]}` lets `mova chat`/`chat_completion` invoke whitelisted tools via a plain-text protocol (not a provider's native function-calling API, which small local models often don't support):
```
<<<MOVA_TOOL_CALL>>> {"name":"...","arguments":{...}} <<<END_MOVA_TOOL_CALL>>>
```
`sendWithTools` (CLI) / `sendWithToolsMCP` (MCP) drive identical loops, capped at `MaxAgentToolTurns`.

---

## 6. Budget, Deduplication, and the Token Firewall

**6.1 Budget report** (`mova budget` / `estimate_budget` / `/budget`) — one `budget.BuildReport()`, tokenized per-section via `tiktoken-go/tokenizer`, priced against `config/prices.json`.

**6.2 Deduplication** — `core.BuildContextSections` shares one `dedup.Paragraphs` "seen" map across Agents→Skills→Prompt→Focus→Memory.

**6.3 Budget gate** — `project.json`'s `"budget": {"max_tokens": N}` is enforced by `budget.EnforceLimit`, called at every door before context is ever sent. `budget.ResolveTask` always returns a non-nil `*Task` (zero-value if none matched), so the gate can never be silently skipped.

**6.4 Feedback Loop** — `mova-token-history.json` stores per-provider token accumulators (no prompts/replies) to calibrate estimates against real `Usage`.

**6.5 Token Firewall pipeline:**
```
budget.BuildGatedContext(adapter, root, project, task)
        ├─ core.BuildContextSections
        ├─ [1]  SanitizeCached          — §6.5 table + §17 (hot invalidation)
        ├─ [1b] PII Masking (optional, off by default)
        ├─ [2]  CheckCircuitBreaker     — spend-governance gate
        ├─ [3]  EnforceLimit            — §6.3
        ▼
GatedContext{Text, Tokens, Sections, Sanitize, CircuitBreaker, Err}
```
**Single function**, shared by `mova run`, `orchestrator.RunAgent`, `mova chat`, the TUI chat screen, `jobs/actions_tasks.go`, and `chat_completion` — one pipeline, zero duplicated logic.

| Stage | File | What it does |
|---|---|---|
| Sanitizer | `sanitize/sanitize.go` | Repeated-line dedup, blank-line collapsing, optional comment stripping |
| PII Masking | `sanitize/pii.go` | `MaskPII` — deterministic `[PII_xxxxxxxx]` tokens, language-agnostic |
| Circuit Breaker | `budget/spend.go` | `CheckCircuitBreaker`/`RecordSpend`, backed by `mova-spend.json` |
| Context Cache | `budget/contextcache.go` | `SanitizeCached` — memoizes the Sanitizer's result; hot-invalidates on any `project.json` change (§17) |
| Cache Layout Guard | `budget/cachelayout.go` | Reorders into static-prefix/dynamic-tail + sha256 fingerprint for provider-side prompt caching |

Every Token Firewall toggle is a `*bool` on `core.BudgetConfig` — nil/absent/`true` all mean **on** (PII Masking is the one opt-in exception).

---

## 7. Path Resolution & Cross-Platform Engine

`runtime.FindRoot()` — `MOVA_PROJECT_PATH` → `MOVA_PROJECT_ROOT` → walk up from cwd looking for `workflow.md`. `documents.ResolvePath`/`ResolveDirectoryPath`/`ResolveFilePath` handle cross-platform absolute paths and bare-name disk search for the **Save Service**. `core/focus/resolvers/fsutil.go` handles the equivalent for **Focus targets** — see §16.2 (this round added host-absolute-path support there specifically).

UNC (`\\server\share`), WSL, and Docker bind-mounts are supported via `filepath.IsAbs`/`filepath.Join` throughout — verified against Go's stdlib, not assumed.

---

## 8. Single Source of Truth: Model Resolution & Hot-Reloading

`config/models/<provider>/<config>.json` holds connection + inference parameters together — one file per model. `ConfigCache` (`models/config.go`) stats each file's mtime on every read and reloads only on change — the original hot-reload pattern this round's project.json watching (§17) follows. `project.json`'s `llm_profile = {type, provider, config}` picks the model; `provider` is optional and auto-resolved by scanning `config/models/*/` for the owning folder.

---

## 9. Job Engine & Multiagent Orchestrator

**Jobs** (`jobs/`): `RunJob(adapter, root, project, proj, spec)` runs `runTasks → runSave → runMemory → runMemoryArchive → runDelete → runBudget` in a fixed sequence — `mova jobs start` is the cron daemon.

**Orchestrator** (`orchestrator/`): `RunGroup` reads `projects/<group>/config.json`, then calls `budget.BuildGatedContext(..., "<group>/<agent>", task)` per agent sequentially — the **same** function `mova run` uses; an agent is just a project name with a `/` in it.

---

## 10. Transports — MCP / HTTP, same engine, different door

`mcp.Process(adapter, root, req)` is the single dispatcher for `tools/list`/`tools/call`. `http/server.go` decodes an HTTP POST into the same `Request` struct and calls `Process` — no protocol logic of its own. `get_full_context` = `mova run`; `chat_completion` = `mova chat` minus the terminal — **both are stateless: every call re-reads `project.json` fresh**, so they never need the hot-reload logic §17 adds for the long-lived `mova chat` REPL/TUI.

---

## 11. Diagrams & Terminal UI

`mova run <project> --diagram [--export svg,png,pdf]` / `generate_diagram` render a project's real pipeline from `project.json` + a live `orchestrator.Count` — nothing invented. `mova ui` (`cli/tui_*.go`) is presentation-only, a screen stack over the same CLI helpers; `tui_chat.go` is where this round's hot-reload (§17) and `[Focus] Selected ...` line (§18) are wired for the interactive TUI.

---

## 16. Focus Targeting — multi-target, directories, globs, absolute paths

`project.json`'s `"focus"` (and `"memory"`) is a **list** of targets — files, directories, glob patterns, or absolute host paths, resolved by the cascade below. `"exclude"` (§16.4) is its mirror for exclusion.

### 16.1 Resolver cascade (`core/focus/resolvers/`, order matters)

| Order | Resolver | Handles |
|---|---|---|
| 1 | `FileResolver` | An exact file — repo-relative **or absolute host path** (new, see 16.2) |
| 2 | `DirectoryResolver` | A directory's full contents — repo-relative **or absolute host path** |
| 3 | `JSONResolver` | A node inside a `.json` file (`file.json#dot.path`) |
| 4 | `SQLResolver` | A `CREATE TABLE ...;` by name |
| 5 | `CodeSymbolResolver` | A function/class declaration |
| 6 | `MarkdownResolver` | A heading/section |
| 7 | `LegalResolver` | Título/Capítulo/Artículo hierarchies |
| 8 | `MemoryResolver` | Dated chronological blocks |
| 9 | `GlobResolver` | `"."`, `"**/*"`, `"**/*.ext"`, or an absolute glob (`/mnt/**/*.java`) |
| 10 | `FallbackResolver` | Bounded excerpt — last resort |

**Fixed this round** (`code_symbol.go`, `sql.go`, `markdown.go`, `legal.go`, `memory.go`): resolvers 5–8 (and SQL) previously had `Match() bool { return true }` — an unconditional catch-all that ran **before** `GlobResolver` in the cascade. A target like `"."` or `"**/*.go"` was being intercepted by `CodeSymbolResolver`'s LIKE fallback pass, which does a raw substring match — `"."` trivially matches almost any line of code, producing exactly one spurious match instead of reaching `GlobResolver` and recursing the whole repo/pattern. **Fix:** each of these five `Match()` now returns `false` immediately when `isGlobPattern(target)` is true (see `file.go`), so `"."` and any glob always resolve deterministically via `GlobResolver`, never via a heuristic catch-all. **File to touch if this regresses again:** `core/focus/resolvers/{code_symbol,sql,markdown,legal,memory}.go`'s `Match` functions.

Multi-target lists (`["server.js", "backend-test.py"]`, or `["."]`, or `["src"]`) were already looped correctly by `core/focus/render/render.go`'s `renderFocusContext` and deduplicated via `included map[string]bool` — no code path stopped after the first item.

### 16.2 Absolute host paths (Windows / Linux / macOS) — new this round

`focus`/`memory` targets like `"C:\ejemploPython"`, `"C:\ejemploPython\testSentence.py"`, `"d:\test\test.py"`, `"/mnt/archivo.java"`, `"/mnt"` now resolve as literal filesystem paths **outside the repo**, on whichever OS the binary runs on. **File: `core/focus/resolvers/fsutil.go`** (new functions) + `file.go` (call sites):

- `isWindowsDriveAbs`/`isUNCPath` — a drive letter (`C:\...`) or UNC path (`\\server\share`) is unambiguous on any host OS: tried directly, no fallback.
- A leading `/` (`"/mnt"`, but also the historical `"/src"` convention) is **ambiguous** — resolved as absolute **only if it exists on disk as such** (`os.Stat`); otherwise it falls back to the pre-existing "relative to repo root" behavior, so **no existing `project.json` changes behavior**.
- `normalizeHostPath` converts `\` → `/` so a Windows-style path string can still be walked correctly by Go's stdlib when the binary itself runs on Linux/macOS (and is a no-op on Windows, which accepts both separators).
- `GlobResolver` additionally supports an **absolute glob root** (`splitAbsoluteGlobRoot`): `/mnt/**/*.java` walks from `/mnt` instead of the repo root, if `/mnt` exists.
- `relOrBase` (labeling) now shows the **full absolute path** as `Source` for anything outside the repo, instead of silently truncating to just the filename.

**File to touch for a new absolute-path rule:** `core/focus/resolvers/fsutil.go`.

### 16.3 `FocusItem` — resolved-target metadata (new this round)

`core/focus/stats.go` adds `FocusItem{Name, Kind ("file"|"dir"), Files int}` and `ScanStats.Items []FocusItem`, populated once per **target** (never per individual file inside a directory) by `render.go`, using the resolvers' own `Match()` (`GlobResolver`/`DirectoryResolver`) to decide `Kind` — **not** a block-count heuristic, which mislabels a directory that happens to resolve to exactly one file. `core.ContextSections.FocusItems` carries this out of `core/engine.go`. **Files: `core/focus/stats.go`, `core/focus/render/render.go`, `core/engine.go`.**

### 16.4 `exclude` — targets `focus` must never read (new this round)

`project.json`'s `"exclude"` (+ task-level override, same "task wins if non-empty" rule as `focus`) is `focus`'s mirror image: **same cross-platform syntax** (bare name, repo-relative path, absolute host path, glob), but for exclusion. Anything matching `exclude` never resolves — even an explicit `focus` target naming it exactly — so it never reaches `SanitizeCached`/`mova-context-cache.json` (§17) in the first place.

- **`core/focus/resolvers/exclude.go`** (new file) — `excludeMatcher`: `bareNames` (any-depth name match, e.g. `"node_modules"`), `absPaths`/`repoPaths` (exact path + everything under it), `globs` (`filepath.Match`). Built once per `Resolve()` call, never per file.
- **`core/types.go`** — `Project.Exclude`/`Task.Exclude`; **`core/engine_helpers.go`** — `resolveTaskExclude`/`ResolveExclude` (mirrors `resolveTaskFocus`/`ResolveFocus`).
- **`core/focus/engine.go`** — `Context.Exclude []string` carries the raw patterns (kept in `core/focus` to avoid `focus`↔`resolvers` import cycle; the actual matching logic lives in `resolvers/exclude.go`, which already owns every path-normalization helper it needs).
- **Call sites, all in `core/focus/resolvers/file.go` + `fsutil.go`:** `FileResolver.candidatePath` and `DirectoryResolver.resolvePath` reject an explicitly-excluded target outright; `DirectoryResolver.Resolve`/`GlobResolver.Resolve`/`walkFiles` check `skipDirOrExcluded` (dir pruning) and `excludesPath` (per-file) during the walk, so a nested exclude match inside an otherwise-included directory is caught too.
- **`core/focus/render/render.go`** — `RenderFocusContext`/`WithEngine`/`WithSeen` all gained an `exclude []string` parameter (5th positional arg); **`core/engine.go`**'s one real caller passes `resolveTaskExclude(proj, &task)`.

**Security fixes discovered and closed this round, same files (`core/focus/resolvers/file.go`):**
- `repoRelativePath` didn't confirm the joined path stayed inside `repoPath` — a `focus`/`exclude` target like `"../../etc/passwd"` could escape the repo entirely. Now rejected (`filepath.Rel` check) unless it's an intentional, existence-checked absolute path (§16.2).
- `DirectoryResolver.Resolve`/`GlobResolver.Resolve` walked with raw `filepath.WalkDir`, **not** honoring `ctx.SkipDir`'s default exclusion (`.git`, `node_modules`, ...) — a bare `"focus": ["."]` could leak `.git/config` credentials or `node_modules/` contents into the LLM context. Both now call `skipDirOrExcluded` during the walk, same as every other resolver always did via `walkFiles`.

---

## 17. Hot Reload — `project.json` changes without restarting

Every **stateless** door (`mova run`, `mova jobs`, HTTP, MCP) already re-reads `project.json` fresh on every single invocation — nothing to fix there. The one long-lived process that used to freeze its context at startup was **`mova chat`**'s REPL and the **TUI chat screen**, both of which built `budget.BuildGatedContext` and baked the resulting system prompt into the session **once**, then reused it for every turn, so editing `focus`/`sanitize`/any other field in `project.json` mid-session had no effect until restart.

```
cli/chat_helpers.go: refreshProjectContext(root, project, task, sess, proj, adapter, lastSignature, emit)
   1. projectSignature(proj) vs. lastSignature  — cheap comparison, no I/O beyond a stat+hash
   2. unchanged → no-op, return immediately (never rebuilds on every turn)
   3. changed   → re-read project.json, re-run BuildGatedContext, replace sess.System, print [Project] notice
```

**This function already existed but was never called** — wired in for the first time this round:
- `cli/chat_cmd.go` — called once per turn, right before dispatching the user's line, in `runChat`'s REPL loop.
- `cli/tui_chat.go` — called once per turn, right before `chatSendCmd`; `chatScreen.signature` field added, initialized in `newChatScreen`.

**Cache-file safety net** (`budget/contextcache.go`): `invalidateOnProjectChange(root, project, *contextCacheFile)` compares `core.ProjectJSONFingerprint`'s sha256 against a new `ProjectHash` field persisted in `mova-context-cache.json`; any drift wipes every cached entry **before** individual Focus/Memory hashes are even compared, so a config change with no visible text difference still forces a fresh write. Runs at the top of `SanitizeCached`, covering every door — not just `mova chat`.

**Files touched:** `core/file_adapter.go` (`ProjectJSONPath`, `ProjectJSONFingerprint`), `budget/contextcache.go` (`invalidateOnProjectChange`, `contextCacheFile.ProjectHash`), `cli/chat_cmd.go`, `cli/tui_chat.go`. **To extend hot reload to a new long-lived surface:** call `refreshProjectContext` (or replicate its signature-check pattern) once per "turn" of that surface's loop.

---

## 18. The `[Focus] Selected ...` status line

Previously `fmt.Sprintf("[Focus] Selected %d file(s).", strings.Count(sections.Focus, "FOCUS:"))` — counted **configured targets**, not resolved files, never distinguished file vs. directory, and never listed names.

**Now:** `core.FormatFocusSelection(sections.FocusItems, core.FocusDisplayLimit(proj))` (`core/engine.go`) builds e.g. `[Focus] Selected 3 items (45 file(s) total): server.js, backend-test.py 📎+1.` — real file/dir counts (§16.3), correct pluralization/kind label, and a name list capped by `project.json`'s `"focus_display_limit"` (`core/types.go`, default `2` — **any** configured value is respected as-is; exceeding it always collapses the rest into `📎+N`).

**Call sites (identical output everywhere), files:**
- `cli/chat_helpers.go` — `printContextSummary` (console) + `refreshProjectContext`'s reload notice.
- `cli/chat_cmd.go`, `cli/run_cmd.go` — pass `proj` into `printContextSummary`.
- `mcp/chat_tool.go` — `writeContextSummary` (identical text, written into the tool's returned string instead of stdout) — same output over MCP **and** HTTP, since HTTP wraps MCP (§10).

**To change the message format:** edit `core.FormatFocusSelection`/`focusKindLabel` in `core/engine.go` — every door updates automatically.

---

## 19. Troubleshooting Matrix

| Area | Check |
|---|---|
| Budget not enforced | `project.json`/task has `"budget": {"max_tokens": N}`; `EnforceLimit` is called from the door in use |
| `focus` target not found | Cascade order (§16.1); a catch-all resolver could theoretically still intercept a new symbol-like target — check `isGlobPattern` guards first |
| `"."` / glob not recursing whole repo | Regression of the §16.1 fix — check `Match()` in `code_symbol.go`/`sql.go`/`markdown.go`/`legal.go`/`memory.go` |
| Absolute path (`C:\...`, `/mnt/...`) not resolving | Confirm it exists on **this host's** disk (`os.Stat`) — §16.2 never invents a path; also check `normalizeHostPath` |
| `mova chat` not picking up a `project.json` edit | Confirm `refreshProjectContext` is actually being called in the loop (§17) — it's a no-op if `projectSignature` didn't change |
| `[Focus] Selected ...` shows wrong count/names | Check `FocusItem` population in `render.go` (§16.3), not the display line itself in `engine.go` (§18) |
| `exclude` not blocking a target | Check pattern classification in `newExcludeMatcher` (§16.4) — a bare name only matches by NAME anywhere, a path must match exactly or as a prefix |
| A `focus`/`exclude` target with `"../"` reads nothing | Expected — path-traversal containment in `repoRelativePath` (§16.4); use an absolute host path (§16.2) if the target is genuinely outside the repo |
| Format "unsupported" | `WriterFor`/`RegisteredExtensions` in `save_service.go` |
| Job doesn't fire | Valid cron string, `mova jobs start` running — a missed minute is never caught up |
| `mova ui` fails to start | Needs a real TTY |
| HTTP behaves differently than MCP stdio | `http/server.go` has no logic of its own — the bug is in request decoding, not the tool |

---

## 20. Deliberately Not Here

No compiler, no token-optimization pipeline, no licensing tier, no `-tags premium` build. `focus` is the complete, permanent implementation. Every capability above is additive at the `project.json` level — a project that never touches a new field (`focus_display_limit`, an absolute path, a glob) behaves exactly as it did before that feature shipped.
