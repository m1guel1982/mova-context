# Logging configuration — `config/log/logging.json`

Mova Context's logging system is **disabled by default**. Nothing is
written to disk until you explicitly set `"enabled": true` in
`config/log/logging.json`. Every entrypoint (CLI, chat, HTTP, MCP)
reads this same file through one shared component
(`mova.local/logging`), so behavior is identical no matter how Mova is
started.

If the file is missing, unreadable, or contains invalid JSON, Mova
silently falls back to "logging disabled" — a broken logging config
never stops Mova from running.

## File location

```
config/log/logging.json
```

## Full example

```json
{
  "enabled": true,
  "structured": false,
  "level": "info",
  "categories": {
    "jobs": true,
    "orchestrator": true,
    "memory": true,
    "save": true,
    "delete": true,
    "budget": true,
    "chat": true,
    "mcp": true,
    "http": true,
    "cli": true
  },
  "file": {
    "path": "logs/mova.log",
    "auto_create": true
  },
  "rotation": {
    "mode": "daily",
    "custom_days": 7
  },
  "retention": {
    "policy": "daily",
    "custom_days": 30
  }
}
```

## Parameters

### `enabled`
- **Description**: master switch for the whole logging system.
- **Allowed values**: `true` | `false`
- **Default**: `false`
- **Example**: `"enabled": true`
- **Recommendation**: keep this `false` in a fresh checkout; turn it on
  only for the environment(s) where you actually want a trail (e.g. a
  server running `mova jobs start`).

### `structured`
- **Description**: switches the on-disk line format between plain text
  and one JSON object per line.
- **Allowed values**: `true` (JSON lines) | `false` (plain text)
- **Default**: `false`
- **Example (plain)**: `2026-07-30 02:00:01 [info] [jobs] finished job for project=ventas_online/vendedor (3 step(s))`
- **Example (structured)**: `{"time":"2026-07-30T02:00:01Z","level":"info","category":"jobs","message":"finished job..."}`
- **Recommendation**: use `true` if you plan to feed logs into a
  log-aggregation tool (Loki, ELK, Datadog...); plain text is easier to
  `tail -f` by hand.

### `level`
- **Description**: minimum severity written to the log file. Anything
  below this level is silently dropped.
- **Allowed values**: `"debug"` | `"info"` | `"warning"` | `"error"`
  (increasing severity, in that order)
- **Default**: `"info"`
- **Example**: `"level": "warning"` — only warnings and errors are kept.
- **Recommendation**: use `"debug"` only while troubleshooting; it is
  noisy. `"info"` is the right default for normal operation.

### `categories`
- **Description**: per-subsystem on/off switches. An **absent or empty**
  `categories` object means "log every category" once `enabled` is
  `true`. An **explicit** object only logs the categories set to `true`
  — set every category to `false` to disable all logging output while
  leaving `enabled: true` (useful for a temporary blanket mute without
  losing the rest of the config).
- **Allowed keys**: `jobs`, `orchestrator`, `memory`, `save`, `delete`,
  `budget`, `chat`, `mcp`, `http`, `cli` (any subset)
- **Allowed values per key**: `true` | `false`
- **Default**: all categories enabled (empty object)
- **Example**: `{"jobs": true, "memory": true}` — only Job Engine and
  Memory traces are written; everything else is silent.
- **Recommendation**: start with everything enabled; narrow down once
  you know which subsystem you actually need to watch.

### `file.path`
- **Description**: where the active log file is written. Relative
  paths resolve against the Mova root (the directory containing
  `workflow.md`); absolute paths are used as-is.
- **Allowed values**: any valid filesystem path
- **Default**: `"logs/mova.log"`
- **Example**: `"file": {"path": "/var/log/mova/mova.log"}`

### `file.auto_create`
- **Description**: whether Mova creates the log file (and its parent
  directories) automatically the first time something is logged.
- **Allowed values**: `true` | `false`
- **Default**: `true`
- **Recommendation**: leave this `true` unless your deployment
  pre-creates `logs/` itself with specific permissions.

### `rotation.mode`
- **Description**: how often the active log file is rolled over into a
  dated backup (`mova-2026-07-30.log`).
- **Allowed values**: `"daily"` | `"weekly"` | `"monthly"` | `"yearly"` |
  `"custom"`
- **Default**: `"daily"`
- **Example**: `"rotation": {"mode": "weekly"}`

### `rotation.custom_days`
- **Description**: rotation interval, in days, used only when
  `rotation.mode` is `"custom"`.
- **Allowed values**: any positive integer
- **Default**: `1` if omitted while `mode` is `"custom"`
- **Example**: `"rotation": {"mode": "custom", "custom_days": 3}` —
  rotate every 3 days.

### `retention.policy`
- **Description**: how long rotated log files are kept before being
  deleted automatically.
- **Allowed values**: `"daily"` (1 day) | `"weekly"` (7 days) |
  `"monthly"` (30 days) | `"yearly"` (365 days) | `"custom"`
- **Default**: `"daily"`
- **Recommendation**: `"monthly"` is a reasonable default for
  production servers; `"daily"` is fine for local development.

### `retention.custom_days`
- **Description**: retention window, in days, used only when
  `retention.policy` is `"custom"`.
- **Allowed values**: any positive integer
- **Default**: `30` if omitted while `policy` is `"custom"`
- **Example**: `"retention": {"policy": "custom", "custom_days": 90}` —
  keep 90 days of rotated logs, delete anything older automatically.

## Notes

- Old rotated files are deleted automatically according to
  `retention` — there is no manual cleanup step.
- Logging failures (disk full, permission denied...) never interrupt
  the operation being logged; Mova simply skips that log line.
- See `docs/SOURCE.md` § Logging for the internal package
  (`mova.local/logging`) this file configures.
