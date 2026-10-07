# mova(1) — Commands (merged: COMMANDS + COMMANDS_ADVANCED)

MAN-page style. MCP tools / HTTP routes / chat slash commands: see `FUNCTIONS.md` (single matrix).

## NAME
`mova` — pre-inference governance for LLM context: select, sanitize, mask PII, budget, audit.

## SYNOPSIS
`mova <command> [project] [task] [flags]` — `[project]` = folder under `projects/`.

## COMMANDS

### run
Assemble a project's context (Agents+Skills+Prompt+Focus+Memory); optionally draw evidence.
`mova run <project> [task] [--count] [--diagram --export svg,png,pdf --path <dir|file.ext>]`
- `--count`: estimate tokens/USD only, no context printed, no report file; accepts a multiagent group.
- `--export`: comma list. `--path`: a directory (file named `<project>.<fmt>`) or, with one format, a full file path.
```bash
mova run 02-pii-compliance-governance --diagram --export png --path ./evidence.png
```

### context-trace (alias `trace`)
Audit before inference: what was selected, sanitized, tokens, cost, who/what authorized it.
`mova context-trace <project> [--export md|pdf] [--diagram] [--policies_include <l>] [--policies_exclude <l>]`
`mova context-trace --repo <url|path> [--task <text>] [--ignore <globs>] [--prune-docstrings]` (discovery, no `project.json`)
`--prune-docstrings` strips comments/docstrings from selected code.
- `--repo` ends with `Generate a 'project.json'? [Y/n]` on stdin: answer, or redirect `< /dev/null` in scripts/CI (otherwise it waits).
- Policies: comma list of bare names (searched recursively in `config/policy/`) or full paths; they override `project.json` and `config/policy.json`.
```bash
mova trace --repo https://github.com/user/repo --task "review login" --ignore "docs/**,*.lock"
```

### budget
Per-provider token/USD breakdown for the active task; writes `mova-budget-report.md`. 100 % local (tiktoken-go).
`mova budget <project> [task] [--focus]` — `--focus` also compares full-repo vs focus-only cost.

### chat
Interactive REPL with a local/cloud model; Mova context is the system prompt.
`mova chat <project> [task|all]` · `mova chat <group> <agent>` · `mova chat <group>` (lists agents)
- No task + several tasks in `project.json` → loads all. `all` forces it.
- Inside: `/tasks` `/task <n|all>` `/run <n>` `/budget` `/diagram` `/context-trace` `/memory` `/save` `/delete` `/tools` `/clear`, `set -model <n>`, `exit`.
- `"memory"` on → each substantial answer leaves a synthesis block in `memory.md`.
- `"apply"` on → proposed ```` ```lang:path::func() ```` blocks are listed; you choose `[s]` all · `[1..N]` · `[n]` none. Existing files get `<file>.mova-<date>.bak` (`"backup": false` disables). `exclude`d paths are never touched.
- If the model spends its output quota without text, an error with tokens consumed is shown (see `MODEL_CONFIG.md`).

### mcp start
Start the MCP server (Claude Code, Cursor, Windsurf) or the HTTP server for curl/Postman.
`mova mcp start` — MCP over stdio (default). `mova mcp start --http [--port 3000] [--bind 127.0.0.1]` — HTTP on loopback; a non-loopback `--bind` requires `MOVA_HTTP_TOKEN` (see [MCP_INTEGRATION](MCP_INTEGRATION.md)).
```json
{ "mcpServers": { "mova": { "command": "mova", "args": ["mcp", "start", "--stdio"] } } }
```
```bash
curl -s localhost:3000/health     # liveness
```

### agents list | agents run
Multiagent orchestrator. `mova agents list <group>` · `mova agents run <group> [agent|--all]`
Builds each agent's context and graph **without calling the model** (parallel, bounded workers).
A **group** is `projects/<group>/config.json` = `{"group","description","agents":[...]}`; each agent is an
ordinary project at `projects/<group>/<agent>/project.json`, addressed `<group>/<agent>`.
If `agents` is omitted, subfolders are auto-discovered. (There is no `is_group`/`members` field.)
```bash
mova agents run 05-nebula-flota facturador
```

### memory family
`mova memory <p> "text"` save · `memory-read <p> [--all] [--month YYYY-MM]` read · `memory-archive <p> [--days N]` (default 30)
· `memory-clear <p> [--archived|--keep-active|--date D|--from D --to D] [--yes]` · `memory-config <p> enable|disable|days N|confirm true|false`.

### list · init · search
`mova list` projects · `mova init <name>` minimal `projects/<name>/project.json` · `mova search "query" [domain]`.

### config · show · install · model-list · remove
Local models: `mova config <provider>` · `mova show config [model]` · `mova install a,b` · `mova model-list` · `mova remove a,b`.
Profiles live in `config/models/<provider>/*.json`; selected per project via `llm_profile.config`.

## ENVIRONMENT
| Variable | Effect |
|---|---|
| `MOVA_PROJECT_ROOT` | Force the root instead of searching upward for `workflow.md` (needed when an MCP client launches `mova` elsewhere). |
| `MOVA_PROJECT_PATH` | Same, and skips the `workflow.md` search entirely. |
| `MOVA_POLICY_AUTHOR` | Policy authority (`project.json` → `config/policy.json` → this → `system:default`). |
| `MOVA_ADAPTER=db` `MOVA_DSN=postgres://…` | PostgreSQL adapter instead of files. |

## POLICY CASCADE
`config/policy.json` lists files from `config/policy/` (`security`, `review`, `compliance`, `pii_*`) loaded in order;
adding/removing an entry needs no recompilation.

## INSTALL
From the repo root (needs Go ≥ 1.24): `make install` — builds the host binary with `CGO_ENABLED=0`, copies it to
`$(go env GOPATH)/bin/mova` and adds that folder to your shell profile. `make build-all` cross-builds
`dist/` for Windows, Linux (amd64, arm64) and macOS (amd64, arm64). Installers: `installers/{linux,macos,windows}`.

## EXAMPLES
```bash
mova list
mova budget 03-tokenomics-context-trace --focus
mova run --count 05-nebula-flota
mova mcp start --http --port 3000 &  curl -s localhost:3000/health
```

## SEE ALSO
`FUNCTIONS.md` · `PROJECT_JSON.md` · `GOVERNANCE_CONTROLS.md` · `ARTIFACTS.md` · `AST_FILTER.md` · `CONTEXT-TRACE.md`
