# mova — per-task context specification, validated and evidenced

[Español](../es/README.md) · [English](README.md) · [Back to root](../../../README.md)

## What problem it solves (and which it doesn't)

When you **restrict** what a model may see for a task — for cost, privacy, or because excluded code must not be touched — two questions appear that neither `AGENTS.md`, nor the agent itself, nor a gateway answers:

1. **Does the restriction break the task?** If selected code calls code the same spec excludes, the context is incomplete by construction. Mova detects it via AST **before** releasing anything (`dependency_policy`).
2. **What exactly was released, under which rules, from which code?** Mova writes one immutable record per execution, with the hash of the released bytes.

If your agent may read the whole repo and you do not want to restrict it, Mova adds little: the agent resolves dependencies by reading, and a gateway logs what was sent.

## The model in 4 pieces

| Piece | What it does | Where |
|---|---|---|
| **Specification** | `project.json`: per task, `focus` (file or symbol: `file::func=a,b`), `exclude`, `policies`, `budget`, `read_scope` | [PROJECT_JSON](PROJECT_JSON.md) |
| **Closure validation** | A focused symbol that calls or references something **excluded** → conflict. `block` (default) releases nothing; `warn` releases and records; `accept_missing` accepts with a written reason. It also lists what stays **outside** the context | [AST_FILTER](AST_FILTER.md) |
| **Deterministic, sanitized context** | Same inputs → same bytes (no timestamps). Secrets: block, or redact only the literal. PII (opt-in): `field_keys`, typed detectors (email, Chilean RUT, phone), and a heuristic score on data blocks only | [GOVERNANCE_CONTROLS](GOVERNANCE_CONTROLS.md) |
| **Immutable evidence** | `projects/<p>/runs/<run_id>/`: `context.txt` and `manifest.json` (write-once) plus `events.jsonl` (append-only) | [ARTIFACTS](ARTIFACTS.md) |

## Perimeter: what it controls and what it doesn't

| Integration | Initial context | Later turns / tool results |
|---|---|---|
| `mova run` (stdout) | Controlled and recorded | Do not exist for Mova |
| `mova chat` / `chat_completion` with `llm_profile` (Mova calls the model) | Controlled and recorded | **Controlled**: each tool result goes through the same read policy, sanitization and budget, and is recorded in `events.jsonl` |
| MCP / HTTP with a host (Claude Code, Cursor, Codex) | Controls what the host asks Mova for | **Not visible**: the host's own tools (Read, Bash, Grep, @-mentions), unless hooks are installed |
| Claude Code **with hooks** (`check_read`, `sanitize_tool_output`) | — | Denies reads outside the spec and replaces Read output with the governed view. Does not cover @-mentions or anything that fires no hook |
| Codex with hooks | — | Only a coarse block applies today: Codex reads via shell and cannot replace MCP output. See [MCP_INTEGRATION](MCP_INTEGRATION.md) |

Mova's own tools **cannot** bypass its rules: every read requires `project`, stays inside the repo (absolute paths, `../` and symlinks are rejected), honors `exclude` and `read_scope`, and is sanitized.

## Try it (no API key, no model call)

```bash
mova run 04-nebula-delivery agregar-columnas     # released; [Evidence] run <id> → projects/04-nebula-delivery/runs/<id>
mova run 04-nebula-delivery recalcular-tarifas   # BLOCKED: calcularTarifa -> tarifaPlanaV1 (src/legacy/tarifasV1.js is excluded)
mova run 02-pii-compliance-governance            # dry_run: nothing is released; the manifest shows what was masked per block
```

Install: `make install` (Go ≥ 1.24). Cost figures are input-token estimates with `cl100k_base` (other providers tokenize differently); when Mova calls the model, the real token counts the provider reports go to `events.jsonl`.

## What differentiates it (and what doesn't)

- **Commodity — not a reason to pick Mova:** token counting, deduplication, PII masking, dry-run, per-request logs, cross-session memory, repo packing and AST compression. Gateways (LiteLLM, Portkey), packers (Repomix) and the agents themselves already do these, often better.
- **What Mova adds:** a **per-task**, versionable, symbol-level context spec; verification that the spec is **closed** with respect to what it excludes, before release; and a manifest stating **why** each context was released or blocked (rule, conflict, human decision with a reason). A gateway knows which bytes were sent; it does not know which task, which symbols, or which dependencies were missing.

## Documentation

- [GOVERNANCE_CONTROLS](GOVERNANCE_CONTROLS.md) — perimeter, controls and what stops the process
- [MCP_INTEGRATION](MCP_INTEGRATION.md) — Claude Code (MCP + hooks), Codex, secure HTTP
- [PROJECT_JSON](PROJECT_JSON.md) · [AST_FILTER](AST_FILTER.md) · [COMMANDS](COMMANDS.md) · [FUNCTIONS](FUNCTIONS.md)
- [ARTIFACTS](ARTIFACTS.md) — format of `runs/<run_id>/`
- [CONTEXT-TRACE](CONTEXT-TRACE.md) · [SOURCE](SOURCE.md) · [FAQ](FAQ.md) · [VERIFICATION](../../VERIFICATION.md)
