# mova — Functions & Arguments Matrix (MCP · HTTP · Chat/CLI)

Single reference. Source of truth: `src/mcp/tool_registry.go` (MCP), `src/http/server.go` (HTTP),
`src/cli/dispatch.go` + `src/cli/chat_cmd.go` (CLI/Chat). `R` = required, `o` = optional.
All arguments are strings (booleans as `"true"`). One engine, four doors: every HTTP route is a thin
wrapper that re-issues an MCP `tools/call`, so HTTP ≡ MCP by construction.

**Call template (MCP stdio or HTTP `POST /mcp`):**
```json
{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"<tool>","arguments":{...}}}
```
```bash
echo '<json>' | mova mcp start --stdio                      # stdio
curl -s localhost:3000/mcp -H 'Content-Type: application/json' -d '<json>'   # HTTP (mova mcp start --http --port 3000)
```

## 1. Core — pre-inference governance (the product)

| Function | MCP tool · arguments | HTTP | CLI | Chat |
|---|---|---|---|---|
| Assemble context (agents+skills+prompt+memory+focus) | `get_full_context` · `project`R `task`o | `/mcp` | `mova run <project> [task]` | `/run <task>` |
| Token/USD estimate (no LLM call) | `estimate_budget` · `project`R `task`o `focus`o | `/mcp` | `mova budget <project> [task] [--focus]` · `mova run --count` | `/budget` |
| Context trace (audit before inference) | `context_trace` · `project`o `task`o `repo`o `branch`o `export`o(md\|pdf) `output`o `generate_project_json`o | `POST /api/v1/context-trace` (same args) | `mova context-trace <project>` / `--repo <url>` · alias `trace` | `/context-trace [--repo <url>]` |
| Evidence diagram (SVG/PNG/PDF) | `generate_diagram` · `project`R `task`o `export`o `path`o `detail`o(simple\|verbose) | `POST /diagram` | `mova run <project> --diagram --export png --path <p>` | `/diagram` |
| LLM call with governed context | `chat_completion` · `message`R `model`o `project`o `task`o `apply_edits`o `apply_changes`o | `/mcp` | `mova chat <project> [task]` | (the chat itself) |
| Egress audit / air-gap | no own tool: `egress_audit` in `project.json` gates `chat_completion`, `get_full_context`, `get_memory*`, `get_workflow`, `read_file`, `read_document_layer` | same | same | same |
| PII masking, sanitizer, budget gate | `project.json` → `budget.pii_masking` / `budget.sanitize` / `budget.max_tokens` + `on_exceed` | same | same | same |
| Policies | `project.json` → `policies`; CLI `--policies_include/--policies_exclude <list>` | — | `mova context-trace <p> --policies_include a.json,b.json` | — |

**Verified behaviour:** with `egress_audit.dry_run: true`, MCP/HTTP calls to `get_full_context`, `get_memory*` and `chat_completion` return an audit notice (`tokens_sent: 0`) instead of content. A *host-orchestrated* flow (example 08) therefore uses `dry_run: false`; `dry_run: true` is for proving that nothing leaves.

## 2. Memory (shared by CLI, chat, MCP, HTTP)

| Function | MCP tool · arguments | HTTP | CLI | Chat |
|---|---|---|---|---|
| Read active memory | `get_memory` · `project`R | `/mcp` | `mova memory-read <p>` | — |
| Read active + archived | `get_memory_all` · `project`R | `/mcp` | `mova memory-read <p> --all [--month YYYY-MM]` | — |
| Save a synthesis | `save_memory` · `project`R `entry`R `task`o | `/mcp` | `mova memory <p> "text"` | `/memory` (last answer) |
| Archive / clear / configure | **not exposed** | — | `memory-archive [--days N]` · `memory-clear [--archived\|--keep-active\|--date\|--from --to\|--yes]` · `memory-config enable\|disable\|days N\|confirm true\|false` | — |

## 3. Multiagent (group = `projects/<group>/config.json`, agent = ordinary project)

| Function | MCP tool · arguments | HTTP | CLI | Chat |
|---|---|---|---|---|
| List agents of a group | `list_agents` · `group`R | `/mcp` | `mova agents list <group>` | `mova chat <group>` (lists) |
| Run one / all agents (assembles context, no LLM call) | `run_agent` · `group`R `agent`o `task`o | `POST /agents/run` | `mova agents run <group> [agent\|--all]` | — |
| Talk to one agent | `chat_completion` · `project`=`<group>/<agent>` | `/mcp` | `mova chat <group> <agent>` | — |
| Group token estimate | `estimate_budget` · `project`=`<group>` (sums agents) | `/mcp` | `mova run --count <group>` | — |

## 4. Knowledge & discovery

| Function | MCP tool · arguments | HTTP | CLI | Chat |
|---|---|---|---|---|
| List projects | `list_projects` | `/mcp` | `mova list` | — |
| One agent/skill/prompt | `get_knowledge` · `kind`R `domain`R `name`R `lang`o | `/mcp` | — | — |
| Search knowledge | `search_context` · `query`R `domain`o | `/mcp` | `mova search "q" [domain]` | — |
| Read `workflow.md` (budget-gated) | `get_workflow` · `project`o `task`o `workflow`o `lang`o | `POST /workflow` | — | natural language |
| Create project | — | — | `mova init <name>` | — |
| Health | — | `GET /health` | — | — |

## 5. File operations (secondary; same writer behind every door)

| Function | MCP tool · arguments | HTTP | Chat |
|---|---|---|---|
| Create/edit any file or folder (format by extension) | `save` · `path`o `directory`o `content`o `overwrite`o `append`o `project`o `history`o `mode`o `range`o `code_only`o `text_only`o | `POST /save` | `/save [-c\|-d\|-append\|-overwrite\|-no-overwrite] "path"` |
| Delete (needs `confirm:"true"`) | `delete_path` · `path`o `paths`o `project`o `confirm`o | `POST /delete` | `/delete "path" ["path2"…]` (asks Y/N each) |
| Read text / document layer | `read_file` · `filename`R `project`o — `read_document_layer` · `filename`R `project`o | `/mcp` | natural language |
| Surgical edit | `patch_file` · `filename`R `search`R `replace`R `project`o | `/mcp` | natural language ("fix X in file") |
| Create directory | `create_directory` · `path`o `project`o | `/mcp` | natural language |
| Apply model-proposed changes | `chat_completion` · `apply_edits:"true"` | `/mcp` | interactive `[s]/[1..N]/[n]` prompt (`"apply"` in `project.json`) |
| Hooks (Claude Code `mcp_tool`) | `check_read` (PreToolUse: deny/no-decision) · `sanitize_tool_output` (PostToolUse: `updatedToolOutput`) — see [MCP_INTEGRATION](MCP_INTEGRATION.md) |

Supported write formats: text/config (`.txt .md .json .yml .xml .csv .toml .ini .env .log`), code
(`.js .ts .py .go .cs .java .php .rb .rs .c .cpp .h .kt .swift .sh`), web (`.html .css .sql`),
office (`.docx .xlsx .pdf`), media (`.svg .png`). Anything else returns `Unsupported file type`.

## 6. Chat-only controls

| Command | Effect |
|---|---|
| `/tasks` · `/task <name\|all>` | list tasks · switch task and reload context (history kept) |
| `/run <name>` | switch task and send its `QUERY` variable |
| `/tools` · `/clear` · `exit`/`quit` | list file capabilities · clear screen · leave |
| `set -model <name>` | switch model mid-chat, keeps history |

## 7. Local model administration (CLI only, by design)

`mova config <provider>` · `mova show config [model]` · `mova install a,b` · `mova model-list` · `mova remove a,b`.

## 8. Parity gaps (known, verified in code)

- Memory archive/clear/config: CLI only. Models admin and `init`: CLI only (administrative).
- `get_knowledge`, `get_memory*`, `list_projects`: no chat slash command (use CLI or natural language).
- Legacy tools (`write_file`, `generate_*`, `trigger_diffusion_image`) were removed: `save` covers those formats. Every file tool requires `project` and is confined to the repo; read tools apply `exclude`, `read_scope` and sanitization.
  governance purpose; candidates for removal (see strategic report).
