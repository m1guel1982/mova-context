# mova + Claude Code (MCP)

Claude Code is the model; mova is a **local MCP tool server** it can call (`mova mcp start --stdio`). mova does not replace Claude Code: it prepares, trims, masks, prices and records the context Claude receives *through mova*.

## 1. Connect (2 minutes)
Prerequisite: `mova` on PATH and `MOVA_PROJECT_ROOT` = the mova folder (the installer sets both).

```bash
# from your WORK repo; --scope project writes .mcp.json (shareable with the team)
claude mcp add --transport stdio --scope project \
  --env MOVA_PROJECT_ROOT=/abs/path/to/mova-context \
  mova-context -- mova mcp start --stdio
claude mcp list            # mova-context should be listed
```
Options go **before** the server name (Claude Code docs). Equivalent file: copy `config/mcp_clients/claude_code.json` to `.mcp.json` and edit the path (Windows: `"C:\\appMovaContext"`). On first use Claude Code asks you to approve a project-scoped server. Tools appear as `mcp__mova-context__<tool>`. Remove: `claude mcp remove mova-context -s project`.

## 2. What to use from Claude Code
| Goal | Tool | Note |
|---|---|---|
| See projects | `list_projects` | |
| Price the context first | `estimate_budget` | local, no model call |
| Get governed context | `get_full_context` | Focus/AST + sanitizer + PII masking + budget |
| Audit a repo without a project | `context_trace` (`repo` or `project`) | report + diagram + `pii-audit-log.json` |
| Evidence image | `generate_diagram` | |
| Remember across sessions | `save_memory` / `get_memory` | shared by CLI/chat/MCP |
| Several roles | `list_agents` / `run_agent` | example 08 |
Do **not** use `chat_completion` here: it calls *another* model through mova. Claude Code already is the model.

## 3. Verified vs not (honest)
Verified (Linux, real binary, a stdio client speaking JSON-RPC like Claude Code): `initialize` → server `mova-context`, protocol `2024-11-05`; `tools/list` → 26 tools; `estimate_budget` on example 02 → 7153 tokens.
Verified behaviour of `get_full_context` on example 02:
- `dry_run: true` → Claude receives only an audit notice (555 bytes, `tokens_sent: 0`). Use it to **prove** nothing is handed over.
- `dry_run: false` → Claude receives the governed context: 23,463 bytes, **171 `[PII_xxxxxxxx]` pseudonyms, 0 raw e-mails**.
**Not verified:** a live Claude Code session (no Claude Code available to the author), Windows/macOS MCP runs.

## 4. Limits you must know (read before using at work)
- mova governs only what passes **through mova**. Claude Code can still read files itself (`Read`, `Bash`) and send them to the provider. mova cannot see or stop that; with `dry_run` a determined host may try to reconstruct context from local files (documented in the README).
- PII masking is heuristic; recall/precision not measured. It is evidence and mitigation, not compliance certification.
- Ask your security/legal team about Claude Code's data terms for your organisation; mova does not change them.

## 5. Make Claude Code actually go through mova
1. **CLAUDE.md** in the work repo:
   ```
   Before reading source files for a task: call mcp__mova-context__estimate_budget, then get_full_context for project <name>.
   Do not open files listed in project.json "exclude". After finishing, call save_memory with findings.
   ```
2. **`.claude/settings.json`** — allow the mova tools, deny direct reads of what must not leave (syntax per Claude Code permissions docs; `Read` rules do not cover shell commands, so also restrict `Bash` or use its sandbox):
   ```json
   { "permissions": {
       "allow": ["mcp__mova-context__estimate_budget","mcp__mova-context__get_full_context","mcp__mova-context__context_trace"],
       "deny":  ["Read(./.env)","Read(./secrets/**)","Read(./data/customers*.json)"] } }
   ```
3. **project.json** for the work repo: `mova init <name>` (or `mova context-trace --repo <path> < /dev/null`, then answer `Y` to generate one), set `"repo"` to the absolute path, `focus` / `exclude`, `budget.pii_masking.enabled: true`, `budget.max_tokens`, `"memory": true`, and `egress_audit.dry_run: false` (so Claude can read the *governed* context).
4. Check the evidence: `egress_sanitized.md`, `context-report.md`, `pii-audit-log.json` in the project folder.

## 6. Troubleshooting
`claude mcp list` shows it failed → run `mova mcp start --stdio` by hand; if root is not found, `MOVA_PROJECT_ROOT` is missing in `env`. Tool answers a "blocked" notice → `dry_run: true`. Server not asked for approval in `claude -p` (non-interactive loads project servers without asking).
