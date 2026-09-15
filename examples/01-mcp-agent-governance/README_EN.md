# 01 — Context Governance for MCP Agents

**What it is:** an MCP agent (Claude Code, Cursor, Windsurf) requests context from this repo.

Mova decides what gets included, detects a secret, and leaves evidence — before anything reaches the LLM.

## Run (1 command)

```bash
mova run 01-mcp-agent-governance --diagram --export png --path ./evidence.png
```

## What you'll see

| Step             | Result                                                                                                                          |
| ---------------- | ------------------------------------------------------------------------------------------------------------------------------- |
| Context selected | `repo/api.js` + `repo/config.txt` (`focus` in `project.json`)                                                                   |
| Control applied  | Sanitizer ON · `config.txt` contains an example secret (`STRIPE_SECRET_KEY=...`) that Mova recognizes as a credential indicator |
| Decision         | Context allowed, with the policy and agent that authorized it                                                                   |
| Evidence         | `evidence.png` (diagram) + `context-report.md` + `pii-audit-log.json`                                                           |

`evidence-example.png` in this folder is an already-generated sample — run the command above to regenerate it yourself.

## Connect a real MCP agent

```json
{ "mcpServers": { "mova": { "command": "mova", "args": ["mcp"] } } }
```

With this configuration, Claude Code/Cursor call the same `context_trace` tool used by this example — same engine; the only difference is who is asking (`agent_client` is recorded automatically; see `pii-audit-log.json`).

## View the trace as text (without a diagram)

```bash
mova context-trace 01-mcp-agent-governance --export md
```

`project/project.json` in this folder is a read-only copy — the actual file used by the engine lives at `projects/01-mcp-agent-governance/project.json` (the fixed path Mova always looks for).
