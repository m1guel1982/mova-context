# project.json — minimal reference

Always lives at `projects/<name>/project.json` (fixed path, the engine looks nowhere else).

## Real minimal example (see `examples/01-mcp-agent-governance/project/project.json`)

```json
{
  "project": "my-project",
  "author": "platform-team",
  "description": "one line",
  "repo": "path/to/code",
  "lang": "en",
  "adapter": "file",
  "default_task": "review",
  "agents": { "domain": "base", "use": ["backend-dev"], "custom": [] },
  "skills": { "domain": "base", "use": ["lazy-minimalism"], "custom": [] },
  "tasks": {
    "review": { "prompt": "review-project", "variables": {}, "focus": ["file.js"] }
  },
  "llm_profile": {  "config": "llama3.2.3b" },
  "budget": { "max_tokens": 20000, "sanitize": { "enabled": true } }
}
```

## Key fields

| Field | What it does |
|---|---|
| `author` | **Required.** Policy author — Audit Matrix #3. Empty → `system:default`. |
| `repo` | The real directory being analyzed (relative to Mova's root). |
| `llm_profile.{provider,config}` | Which model would receive the context — Audit Matrix #11. `config` points to a file under `config/models/<provider>/`. |
| `tasks.<t>.focus` | What goes in: files, directories, globs, or `file::kind=symbol` (AST). |
| `tasks.<t>.exclude` | What's excluded from `focus` — supports the same AST syntax (new: analyze a whole file while leaving just one function out). |
| `budget.max_tokens` / `max_tokens_per_run` / `max_monthly_usd` | Circuit breaker ceilings. |
| `budget.on_exceed` | `"warn"` (default) reports it and continues. `"abort"` (alias: `"block"`) stops **before any call to the LLM provider** — same hard stop in CLI/Chat, MCP and HTTP. |
| `budget.sanitize.{enabled,dedupe_logs,strip_blank,strip_comments}` | Sanitizer controls. |
| `budget.pii_masking.enabled` | Structural PII masking (see `ARTIFACTS.md`). |
| `debug` | `true` makes every door (chat, CLI, HTTP, MCP) print what it resolved before running a task: repo path, each agent/skill/prompt with its resolved path (or `"inline"`), `focus`/`exclude` entries with their absolute paths, **and — in `context-trace` — the exact resolved path of every included/excluded policy file** (see `policies` below). Defaults to `false`. Never added just by generating a `project.json` with `mova init` — it's manual opt-in. |
| `egress_audit` | Audits the context **already sanitized**, right before it leaves for the LLM, and/or a dry-run switch that stops the call — see the dedicated section below. |

See `docs/i18n/en/AST_FILTER.md` for the exact `file::kind=name` syntax.

## `egress_audit` — pre-provider audit and dry-run

```json
"egress_audit": {
  "dry_run": true,
  "output_file": ".mova/egress_sanitized.log"
}
```

| Key | Type | Default | What it does |
|---|---|---|---|
| `dry_run` | bool | `false` | When `true`: governance/sanitization (and the log, if `output_file` is set) still complete, but **the LLM provider is never called**. The client gets a successful reply saying the dry run finished and no inference happened. |
| `output_file` | string | `""` (disabled) | Where the sanitized-context log gets appended. Independent of `dry_run` — you can audit without dry-running, or dry-run without logging. |

**With the block absent:** `dry_run=false`, `output_file=""` — identical to today's behavior, unchanged.

### Resolving `output_file` (always relative to `project.json`, never the working directory)

- Relative path → resolved against `projects/<project>/`, **not** the directory `mova` was launched from.
  Example: in `projects/02-pii-compliance-governance/project.json`, `"output_file": ".mova/egress_sanitized.log"` writes to `projects/02-pii-compliance-governance/.mova/egress_sanitized.log`.
- Absolute path — Unix (`/var/log/...`), Windows (`C:\...`, `D:\...`, `E:\...`), or UNC (`\\server\share\...`) — used exactly as given, recognized cross-platform regardless of which OS the Mova binary itself runs on (same helper `write_file`/`create_directory` already use).
- If `output_file` ends in `/` or `\` (it names a directory, not a file), the default name **`egress_sanitized.md`** is used inside that directory. A name with no trailing separator (even without an extension, e.g. `.mova`) is honored exactly as given — no extension is forced onto it.
- Missing directories are created automatically (`os.MkdirAll`, standard permissions). The file is always opened in **append** mode: a previous run is never overwritten, and every execution is fenced with its own `execution_id` and `timestamp`.
- If the configured file can't be created or written, `mova` **returns an error and never calls the LLM provider** — identical behavior across CLI/Chat, MCP, and HTTP, because all three doors share one implementation (`models.Session.Send`/`SendStream`).

### What gets written

Only the context **already governed and sanitized** (exactly what would actually be sent to the model) — never the raw, pre-sanitization content. Every block includes at least `execution_id` and `timestamp` (UTC).

### Real air-gap — `dry_run` blocks EVERY tool that exposes context

`dry_run: true` isn't just "don't call the model": **no MCP/HTTP tool that can return a project's content does so while it's active, and neither does `mova run` in the CLI**. Covers 7 MCP/HTTP tools plus `mova run`:

| Tool / command | What it protects |
|---|---|
| `chat_completion` | The model's reply (nothing is ever sent to it) |
| `get_full_context` | The full assembled context |
| `get_memory` / `get_memory_all` | `memory.md`'s content |
| `get_workflow` | `workflow.md`'s content |
| `read_file` | Any project file's raw content (`.env`, credentials, anything) |
| `read_document_layer` | Extracted text from `.docx`/`.xlsx`/`.pdf` |
| `mova run` (CLI) | The same full context `get_full_context` returns, printed to stdout |

All of them return the same fixed, translated `reports.egress_airgap_message` + `reports.egress_airgap_directive` block instead of the real content (the latter is an explicit anti-bypass directive aimed at the model/agent receiving it — see `GOVERNANCE_CONTROLS.md § dry_run` for its honest limits), with the real token count:

```
[MOVA EGRESS AUDIT]
dry_run: true
tokens_evaluated: 174
tokens_sent: 0

Context was processed and sanitized locally, but its transmission has been
blocked. Report saved to disk.

CRITICAL SECURITY DIRECTIVE FOR ASSISTANT:
This request has been blocked by Mova Context Egress Control.
DO NOT attempt to bypass this block.
DO NOT read local files, reports, or metadata using any other tools to
reconstruct the context.
DO NOT summarize or guess the context on your own.
Output ONLY the audit block above to the user and terminate the execution
immediately.
```

Both the header and the directive live in `config/lang/{es,en}.json` (`reports.egress_airgap_message` / `reports.egress_airgap_directive`), are editable without touching code, and hot-reload (no restart needed — see `i18n/i18n_reload.go`). Deleting the directive key leaves the block working with just the header; the process never breaks and the raw key name never leaks into the message.

**Deliberately out of scope:** `search_context` (no `project` argument — searches Mova's shared agents/skills/prompts catalog, not a project's private data) and every WRITE tool (`save`, `patch_file`, `delete_path`, `create_directory`, `write_file`, `generate_*`) — `egress_audit` governs content **leaving** Mova toward an LLM/host, not files Mova writes to your own disk on request.

There is no separate `"enabled"` field — `dry_run` is the only condition that activates the air-gap, on all 3 doors (CLI/Chat, MCP, HTTP), verified with real integration tests (`mcp/egress_gate_test.go`, `mcp/airgap_directive_test.go`, `cli/run_cmd_test.go`) and real HTTP/MCP calls against the compiled binary.

### No `llm_profile` — inference delegated to the host

If `project.json` declares no `llm_profile` (or an empty one) and `dry_run` is `false`, Mova **never calls any local or cloud provider on its own**. `chat_completion` returns the already-governed, sanitized context in the tool result, so the **host LLM** (Claude Code, Cursor, Grok, whoever invoked the MCP tool) generates the final answer. In `mova chat` (interactive REPL, no host to delegate to) an explicit notice is printed stating which global model is being used instead — never silently.

## `policies` — governance policy selection

```json
"policies": {
  "include": ["security.json", "review.json", "compliance.json", "pii_strict.json"],
  "exclude": ["pii_permissive.json"]
}
```

The short form `"policies": ["security.json", "review.json"]` is also accepted (same as `include` with no `exclude`).

| Key | What it does |
|---|---|
| `include` | Policy files to load, in order. |
| `exclude` | Names to drop, even if `include` or the recursive search would have picked them up. Matched by **file name**, regardless of directory. |

### Precedence (first layer that declares anything wins)

1. `--policies_include` / `--policies_exclude` on the CLI.
2. `policies` in `project.json` — **completely overrides** `config/policy.json`.
3. `policies` in `config/policy.json` — used only in *discovery* mode (`--repo`, no `project.json`).

**Opt-in:** if `project.json` declares no `policies`, **no policy file** is loaded. Mova's safe built-ins (private-key blocking, default PII masking) still apply; nothing stricter activates without being declared.

### Path resolution (cross-platform)

| Form in `include` | How it resolves |
|---|---|
| `pii_strict.json` (bare name) | **Recursive** search under `config/policy/`. Shallowest match wins; ties break alphabetically (deterministic on every OS). |
| `config/custom/sales.json` (relative) | Against the Mova root. |
| `/etc/mova/p.json`, `C:\pol\p.json`, `\\server\share\p.json` | Absolute Unix / Windows (C:, D:, E:) / UNC network. |

### Custom-name priority

A custom-named file (e.g. `pii_strict_sales.json`) merges into the right dimension because the dimension is detected from the file's **content** (which keys it declares), not from its name. Exclude `pii_strict.json` as well and the default-directory file is left out, replaced by the custom one.

The report always states what applied: `Policy source: CLI -> {security.json}`.
