# mova — Commands

Manual style: `NAME`, `SYNOPSIS`, `EXAMPLES`. Each command reads in 15 seconds.

## run

**NAME** — assembles a project's context (Agents+Skills+Prompt+Focus+Memory), optionally generates visual evidence.

**SYNOPSIS** — `mova run <project> [task] [--diagram --export svg,png,pdf --path <dir|file.ext>]`

**EXAMPLES**
```bash
mova run 02-pii-compliance-governance
mova run 02-pii-compliance-governance --diagram --export png --path ./evidence.png
```

## context-trace (alias: trace)

**NAME** — audits context before inference: what was selected, what was sanitized, tokens, cost, audit identity.

**SYNOPSIS** — `mova context-trace <project> [--export md,pdf] [--diagram] [--policies_include <list>] [--policies_exclude <list>]` · `mova context-trace --repo <url> [--task <text>] [--ignore <patterns>]`

**EXAMPLES**
```bash
mova context-trace 02-pii-compliance-governance --export md
mova trace --repo https://github.com/user/repo --task "review login" --ignore "docs/**,*.lock"
```

**POLICIES** — `--policies_include` / `--policies_exclude` take a comma-separated list mixing bare names (recursive lookup under `config/policy/`) and full cross-platform paths. They take precedence over `project.json` and `config/policy.json` — see `PROJECT_JSON.md § policies`.

```bash
mova context-trace 02-pii-compliance-governance \
  --policies_include "security.json,C:\custom\pii_strict_sales.json" \
  --policies_exclude "pii_permissive.json"
```

## budget

**NAME** — token/cost breakdown per provider for a project's active task; writes `mova-budget-report.md`.

**SYNOPSIS** — `mova budget <project> [task] [--focus]`

## mcp start

**NAME** — starts the MCP server so an agent (Claude Code, Cursor, Windsurf) can connect and call its tools (`context_trace`, `get_full_context`, `chat_completion`, etc.) — full list below.

**SYNOPSIS** — `mova mcp start [--stdio] [--port 3000]`

`--stdio` starts the server over standard input/output (what Claude Code/Cursor/Windsurf use to connect). Without `--stdio`, it starts an HTTP server on the given port (default `3000`) — see **HTTP** below.

**EXAMPLES**
```bash
mova mcp start --stdio
mova mcp start --port 3000
```

```json
{ "mcpServers": { "mova": { "command": "mova", "args": ["mcp", "start", "--stdio"] } } }
```

### Available MCP functions (`tools/call`)

| Tool | What it's for |
|---|---|
| `list_projects` | Lists Mova's registered projects. |
| `get_full_context` | The full assembled context (= `mova run`): agents+skills+prompt+memory+focus. |
| `get_knowledge` | A single agent, skill, or prompt. |
| `get_memory` / `get_memory_all` | A project's active / active+archived memory. |
| `save_memory` | Appends an entry to `memory.md`. |
| `get_workflow` | Reads `workflow.md`, but only after resolving the project and validating its budget. |
| `search_context` | Searches all knowledge (agents/skills/prompts). |
| `chat_completion` | Sends a message to a local/cloud model, with Mova's context as the system prompt. **This is the tool `egress_audit` affects** — see `PROJECT_JSON.md § egress_audit`: it can audit the outgoing context and/or stop before reaching the provider (dry run). |
| `estimate_budget` | Estimates token/USD cost of a project's real context; writes `mova-budget-report.md`. |
| `generate_diagram` | Visual diagram of the real pipeline (SVG/PNG/PDF). |
| `context_trace` | Pre-inference audit — see `context-trace.md`. |
| `list_agents` / `run_agent` | List/run a multi-agent group. |
| `save` / `delete_path` / `create_directory` / `read_file` / `read_document_layer` | Unified file/directory management. |

## HTTP

**NAME** — the same MCP server exposed over HTTP — for Postman/curl/integrations without stdio.

**SYNOPSIS** — `mova mcp start --port <port>` (without `--stdio`)

### Available HTTP endpoints

| Endpoint | Method | What it does |
|---|---|---|
| `/mcp` | `POST` | Generic JSON-RPC 2.0 — `{"method":"tools/call","params":{"name":"<tool>","arguments":{...}}}`. Every tool in the table above, including `chat_completion` (and therefore `egress_audit`), goes through here. |
| `/save` | `POST` | Direct shortcut to the `save` tool. |
| `/delete` | `POST` | Direct shortcut to `delete_path`. |
| `/workflow` | `POST` | Direct shortcut to `get_workflow`. |
| `/agents/run` | `POST` | Direct shortcut to `run_agent`. |
| `/diagram` | `POST` | Direct shortcut to `generate_diagram`. |
| `/api/v1/context-trace` | `POST` | Direct shortcut to `context_trace`. |
| `/health` | `GET` | Server health check. |

**EXAMPLES**
```bash
curl -X POST http://localhost:3000/mcp \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"chat_completion","arguments":{"project":"02-pii-compliance-governance","message":"summarize the context"}}}'
```

## list / init / search

**NAME** — `list`: available projects · `init <name>`: scaffolds a minimal `projects/<name>/project.json` · `search <query>`: searches agents/skills/prompts.

## config / show / install / model-list / remove

**NAME** — model management: `config` (active profile) · `show <provider/model>` · `install <provider/model>` · `model-list` · `remove <provider/model>`.

## chat

**NAME** — interactive REPL. Inside: `/context-trace`, `/memory`, `/save`, `/delete`.

**SYNOPSIS** — `mova chat <project>`

## memory / memory-read / memory-archive / memory-clear / memory-config

**NAME** — manage a project's `memory.md` (see `docs/i18n/en/ARTIFACTS.md`).

## agents list / agents run

**NAME** — multi-agent orchestrator: `list` (defined groups) · `run <group>` (runs each member project).

---
See `docs/i18n/en/COMMANDS_ADVANCED.md` for environment variables and advanced flags.
