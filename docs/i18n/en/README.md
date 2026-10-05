![mova in action: real outputs of 3 commands](../../assets/mova-demo.gif)

# mova — context sovereignty before inference

> **You decide what context may reach the AI. mova leaves evidence of that decision.**
> *Tú decides qué contexto puede llegar a la IA. mova deja evidencia de esa decisión.*

[Español](../es/README.md) · [English](README.md) · [Back to root](../../../README.md)

| You decide | You block | You verify |
|---|---|---|
| **What goes in:** `focus`/`exclude` (AST: functions, not whole files), task | **What must not leave:** PII masking (opt-in), token cap (`max_tokens`), `dry_run` | **What the model received:** `context-report.md`, `pii-audit-log.json`, `egress_sanitized.md`, diagram |

`mova` is a local binary that runs **before** the model call (CLI · `mova chat` · MCP · HTTP). It works with Claude Code, Cursor, Windsurf, Ollama and any MCP client.

**Honest scope:** "you decide" applies to the context that passes **through mova**. It cannot see what an IDE or agent sends on its own, and PII masking is heuristic — evidence and mitigation, not a compliance guarantee. It is not a gateway, an IDE, a RAG or a platform.

## Try it (what the GIF above shows — all real outputs)

```bash
# 1) Governed context, token savings, PII and an evidence image (no API key, no model call)
mova run 02-pii-compliance-governance --diagram --export png
#    → 02-pii-compliance-governance.png   (20,014 tok before → 7,153 tok after; 171 of 1,694 tokens pseudonymized)

# 2) Audit ANY repo before sending it to an AI — nothing is sent (add "< /dev/null" in scripts: it asks [Y/n] at the end)
mova context-trace --repo https://github.com/fastapi/fastapi --export pdf \
  --task "solve_dependencies get_dependant in fastapi/dependencies/utils.py" --prune-docstrings \
  --ignore "docs/**, tests/**, *.lock, docs_src/**, .github/**, docs/en/**"
#    → context-report.pdf · context-diagram.png · pii-audit-log.json

# 3) Dependency graphs + sanitized egress record, no LLM call (example 04 has "memory": true and "dry_run": false)
mova run 04-nebula-delivery analizar-trazabilidad
mova run 04-nebula-delivery agregar-columnas
#    → analizar.png · agregar-columnas.png · egress_sanitized.md   (memory.md is written by `mova chat` / `save_memory`)
```

Install: `make install` (needs Go ≥ 1.24) or double-click `installers/<your OS>/…`. Uninstall: [`uninstallers/`](../../../uninstallers/README.md). Costs shown are theoretical input-token estimates (`config/prices.json`).

## Use it with Claude Code
`claude mcp add --transport stdio --scope project --env MOVA_PROJECT_ROOT=<path to mova> mova-context -- mova mcp start --stdio` → [full guide, limits and permissions](../en/MCP_INTEGRATION.md). Claude Code stays the model; mova governs the context it asks for.

## Where it has been verified
Windows: the author's sessions (steps 1–2 in the GIF). Linux amd64: build, 21 test packages, MCP stdio and example 08 (see [VERIFICATION](../../VERIFICATION.md)). macOS and Linux arm64: cross-compiled, **not executed**.

## Pre-Inference Audit Matrix

| # | Security / CISO question | How `mova` answers | Evidence / Artifact |
|---|---|---|---|
| 1 | What information reached the model? | Exact inventory of files and AST symbols selected | `context-report.md`/`.pdf` |
| 2 | Why did it reach it? | Task relevance + `focus`/`exclude` rules (incl. AST) | `context-trace` |
| 3 | What policy/authority allowed this? | `PolicyAuthor` — hierarchy `project.json` → `config/policy.json` → `MOVA_POLICY_AUTHOR` → `system:default` | `project.json` / reports |
| 4 | What policy applied? | Scan inclusion/exclusion rules | `config/policy.json` |
| 5 | Did it have PII or secrets? | Heuristic detection of sensitive patterns | `pii-audit-log.json` |
| 6 | Was it sanitized? | Sanitizer + PII Masking (opt-in) vs. unchanged pass-through | Diagram · `mova-budget-report.md` |
| 7 | How many tokens? | Measured with a real tokenizer (`cl100k_base`) | `context-report.md` |
| 8 | How much did it cost? | Per-provider estimate before sending | `mova budget` |
| 9 | Which commit was analyzed? | Exact repo state at audit time | `context-report.md` (Execution ID + commit) |
| 10 | Which agent requested it? | `AgentClient` — `mova-cli`, or the real MCP client captured at `initialize` | Reports / diagram |
| 11 | Which model received it? | `TargetModel` — `<provider>/<config>` from `llm_profile` | Reports / diagram |

## Architecture

```
[ Agent / IDE (Claude Code, Cursor, MCP) ]
                   │
                   ▼  Context request
┌───────────────────────────────────────────────────┐
│      MOVA — PRE-INFERENCE GOVERNANCE ENGINE        │
│  Focus/Exclude (AST) · Sanitizer · PII Masking     │
│  Budget circuit breaker · PNG/PDF diagram export   │
└───────────────────────────────────────────────────┘
                   │
                   ▼  Auditable context, with evidence
        [ Target LLM (Anthropic, Ollama, etc.) ]
```

One engine, four doors — CLI, `mova chat`, MCP (stdio/HTTP), HTTP REST — all calling the same function.

## Examples (1 click, 15 seconds)

| Example | What it demonstrates |
|---|---|
| `examples/01-mcp-agent-governance` | An MCP agent requests context; Mova decides and leaves evidence |
| `examples/02-pii-compliance-governance` | PII/secrets, masking, applied policy (Chile's Law 21.719 scenario) |
| `examples/03-tokenomics-context-trace` | `focus`/`exclude` (AST)/`task`, budget, Context Trace |
| `examples/04-output-mova-trace-fastApi` | Validation of `mova context-trace` on the remote FastAPI repository. Demonstrates precise `focus`/`exclude` filtering via AST parsing, token budget control, and context traceability on real-world codebases. |
| `examples/05-test-mcp-cursor` | Validation of Air-Gap isolation and egress governance on the `02-pii-compliance-governance` project. Demonstrates `chat_completion` interception under `dry_run: true`, PII sanitization, and audit evidence generated via an MCP client. |
| `examples/06-nebula-delivery-graph-memory` | Two chained tasks (analyze → add columns) with memory, AST graph and audited egress (project `04-nebula-delivery`) |
| `examples/07-nebula-multiagente-flota` | 3-agent group with shared memory (project `05-nebula-flota`) |
| `examples/08-nebula-release-gate` | Multiagent without its own LLM: a host agent orchestrates via MCP/HTTP with `run_agent` + `save_memory` |

## Honest limits of the "Air-Gap" (`dry_run`) — 15 seconds

With `egress_audit.dry_run: true`, Mova guarantees ITS side: it never sends the real context to any
provider, and it always leaves evidence on disk. What Mova **cannot guarantee** is that the **host
model** (Cursor, Claude Code, Grok, etc. — whoever calls the MCP tool) obeys the security directive
that comes with the block: a determined host can still try to read other local files
(`context-report.md`, `project.json`, memory, etc.) to "reconstruct" the context on its own — this
already happened in real testing. Mova hardens the blocked message with an explicit anti-bypass
directive (see `GOVERNANCE_CONTROLS.md § dry_run`), but whether that directive is actually followed
depends on the host, not on Mova — Mova does not control, and cannot control, what an external
process does with the text it receives. I don't present this as an absolute guarantee, because it isn't one.

## Documentation

- [`docs/i18n/en/COMMANDS.md`](../en/COMMANDS.md) — commands (MAN page style)
- [`docs/i18n/en/PROJECT_JSON.md`](../en/PROJECT_JSON.md) — `project.json` reference
- [`docs/i18n/en/AST_FILTER.md`](../en/AST_FILTER.md) — `file::kind=name` syntax
- [`docs/i18n/en/CONTEXT-TRACE.md`](../en/CONTEXT-TRACE.md) — how the context decision is made
- [`docs/i18n/en/ARTIFACTS.md`](../en/ARTIFACTS.md) — what every file Mova generates is for
- [`docs/i18n/en/GOVERNANCE_CONTROLS.md`](../en/GOVERNANCE_CONTROLS.md) — `debug`, `policies`, `on_exceed`, `dry_run`: what stops the process vs. what just explains it
- [`docs/i18n/en/MCP_INTEGRATION.md`](../en/MCP_INTEGRATION.md) — **use mova with Claude Code** (MCP, permissions, limits)
- [`docs/i18n/en/POSITIONING.md`](../en/POSITIONING.md) — what mova is, what it is not
- [`docs/VERIFICATION.md`](../../VERIFICATION.md) — what was actually run, and where
- [`uninstallers/`](../../../uninstallers/README.md) — remove mova (binary, PATH, `MOVA_PROJECT_ROOT`) without leftovers
- [`docs/i18n/en/FUNCTIONS.md`](../en/FUNCTIONS.md) — every function and argument, by channel (MCP · HTTP · Chat/CLI)
- [`docs/i18n/en/SOURCE.md`](../en/SOURCE.md) — technical architecture reference
- [`docs/i18n/en/FAQ.md`](../en/FAQ.md)

**Technical note:** PII masking is a heuristic mitigation, not a legal compliance guarantee.
