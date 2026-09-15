# mova — Pre-Inference Context Governance for LLMs
 
**Category: Pre-Inference Context Governance and Audit for LLMs.**

`mova` sits between "context is ready" and "send it to the LLM". For developers and agents (Claude Code,
Cursor, Windsurf), it answers **11 audit questions** about selection, governance, security, traceability,
and cost of the context — with evidence, before performing inference and consuming provider tokens.

**Scope:** `mova` is not a full AI Security platform. Its scope is the context an application or agent
intends to hand to an LLM, and the evidence of the decisions made about that context before inference.

## Get started in 1 command

```bash
mova run 02-pii-compliance-governance --diagram --export png --path ./evidence.png
```

See `examples/` — 3 examples, each 1 command .

## Pre-Inference Audit Matrix

| # | Governance / Audit Question | How `mova` answers | Evidence / Artifact |
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
| 11 | Which model was intended to receive it? | `TargetModel` — `<provider>/<config>` from `llm_profile` | Reports / diagram |

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

One governance policy, four integration methods: CLI, mova chat, MCP (stdio/HTTP), and HTTP REST.

## Examples (1 click, 15 seconds)

| Example | What it demonstrates |
|---|---|
| `examples/01-mcp-agent-governance` | An MCP agent requests context; Mova decides and leaves evidence |
| `examples/02-pii-compliance-governance` | PII/secrets, masking, applied policy (Chile's Law 21.719 scenario) |
| `examples/03-tokenomics-context-trace` | `focus`/`exclude` (AST)/`task`, budget, Context Trace |

## Install

```bash
git clone <this-repo> && cd mova/src && make install
```

## Connect an MCP agent

```json
{ "mcpServers": { "mova": { "command": "mova", "args": ["mcp"] } } }
```

## Documentation

- [`docs/i18n/en/COMMANDS.md`](../en/COMMANDS.md) — commands (MAN page style)
- [`docs/i18n/en/PROJECT_JSON.md`](../en/PROJECT_JSON.md) — `project.json` reference
- [`docs/i18n/en/AST_FILTER.md`](../en/AST_FILTER.md) — `file::kind=name` syntax
- [`docs/i18n/en/context-trace.md`](../en/context-trace.md) — how the context decision is made
- [`docs/i18n/en/ARTIFACTS.md`](../en/ARTIFACTS.md) — what every file Mova generates is for
- [`docs/i18n/en/source.md`](../en/source.md) — technical architecture reference
- [`docs/i18n/en/FAQ.md`](../en/FAQ.md)

**Technical note:** PII masking is a heuristic mitigation, not a legal compliance guarantee.
