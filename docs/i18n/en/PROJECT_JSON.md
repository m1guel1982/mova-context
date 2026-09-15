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
  "llm_profile": { "type": "local", "provider": "ollama", "config": "llama3.2.3b" },
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
| `budget.on_exceed` | `"warn"` or `"block"` when the budget is exceeded. |
| `budget.sanitize.{enabled,dedupe_logs,strip_blank,strip_comments}` | Sanitizer controls. |
| `budget.pii_masking.enabled` | Structural PII masking (see `ARTIFACTS.md`). |
| `debug` | `true` makes every door (chat, CLI, HTTP, MCP) print what it resolved before running a task: repo path, each agent/skill/prompt with its resolved path (or `"inline"`), and `focus`/`exclude` entries with their absolute paths. Defaults to `false`. Never added just by generating a `project.json` with `mova init` — it's manual opt-in. |
| `egress_audit` | Audits the context **already sanitized**, right before it leaves for the LLM, and/or a dry-run switch that stops the call — see the dedicated section below. |

See `docs/i18n/en/AST_FILTER.md` for the exact `file::kind=name` syntax.

## `egress_audit` — pre-provider audit and dry-run

```json
"egress_audit": {
  "dry_run": true,
  "output_file": ".mova/egress_sanitized.md"
}
```

| Key | Type | Default | What it does |
|---|---|---|---|
| `dry_run` | bool | `false` | When `true`: governance/sanitization (and the log, if `output_file` is set) still complete, but **the LLM provider is never called**. The client gets a successful reply saying the dry run finished and no inference happened. |
| `output_file` | string | `""` (disabled) | Where the sanitized-context log gets appended. Independent of `dry_run` — you can audit without dry-running, or dry-run without logging. |

**With the block absent:** `dry_run=false`, `output_file=""` — identical to today's behavior, unchanged.

### Resolving `output_file` (always relative to `project.json`, never the working directory)

- Relative path → resolved against `projects/<project>/`, **not** the directory `mova` was launched from.
  Example: in `projects/02-pii-compliance-governance/project.json`, `"output_file": ".mova/egress_sanitized.md"` writes to `projects/02-pii-compliance-governance/.mova/egress_sanitized.md`.
- Absolute path — Unix (`/var/log/...`), Windows (`C:\...`, `D:\...`, `E:\...`), or UNC (`\\server\share\...`) — used exactly as given, recognized cross-platform regardless of which OS the Mova binary itself runs on (same helper `write_file`/`create_directory` already use).
- If `output_file` ends in `/` or `\` (it names a directory, not a file), the default name **`egress_sanitized.md`** is used inside that directory. A name with no trailing separator (even without an extension, e.g. `.mova`) is honored exactly as given — no extension is forced onto it.
- Missing directories are created automatically (`os.MkdirAll`, standard permissions). The file is always opened in **append** mode: a previous run is never overwritten, and every execution is fenced with its own `execution_id` and `timestamp`.
- If the configured file can't be created or written, `mova` **returns an error and never calls the LLM provider** — identical behavior across CLI/Chat, MCP, and HTTP, because all three doors share one implementation (`models.Session.Send`/`SendStream`).

### What gets written

Only the context **already governed and sanitized** (exactly what would actually be sent to the model) — never the raw, pre-sanitization content. Every block includes at least `execution_id` and `timestamp` (UTC).
