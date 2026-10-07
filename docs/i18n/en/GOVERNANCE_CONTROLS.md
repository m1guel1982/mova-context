# Perimeter and controls

[README](README.md) · [Español](../es/GOVERNANCE_CONTROLS.md)

## Fixed order inside Mova

```
spec (project.json) → dependency closure → selection (focus/exclude, AST)
  → sanitizer (dedup, comments) → per-block secrets & PII → circuit breaker → max_tokens
  → evidence (runs/<run_id>/) → dry_run → release (stdout, MCP host or provider)
```

Every door (`mova run`, `mova chat`, `get_full_context`, `chat_completion`, `run_agent`) uses the same function (`budget.BuildGatedContext`) and writes a run **before** releasing. If the evidence cannot be written, nothing is released.

## What stops the process

| Control | Config | Effect | Evidence |
|---|---|---|---|
| Dependency closure | `dependency_policy`: `block` (default), `warn`, `off`; `accept_missing: [{symbol, reason}]` | `block` + conflict → no context released | `manifest.json → dependency_closure`, `decision.gate = dependency_closure` |
| Secrets | `config/policy/security.json`: `block_on_private_key`, `block_on_api_key` (true by default) | The affected block is replaced by `[MOVA: contenido omitido por política …]`. Otherwise only the literal is redacted (`[REDACTED_SECRET]`) and code syntax stays intact | `governance.changed_blocks[]` |
| Budget | `budget.max_tokens`, `on_exceed`, circuit breaker | Context rejected. In loops Mova controls, tool results that exceed the cumulative cap are rejected too | `decision.gate = budget` / `tool_result.budget_exceeded` event |
| `dry_run` | `egress_audit.dry_run: true` | Nothing leaves Mova: no stdout, host or provider; no memory or reads either | `decision.outcome = dry_run` / `dry_run_block` event |
| Read policy | `read_scope`: `focus` (default when focus exists) or `repo`; `exclude`; repo boundary | Denies `read_file`/`read_document_layer`, loop reads and `check_read` | `read` / `tool_result` / `hook_check_read` events |

## What only transforms

- **PII** (`budget.pii_masking.enabled: true`):
  - `field_keys`: values of those keys in structured data are pseudonymized, and the same value is masked in every other block of the context;
  - typed detectors (email, RUT, phone);
  - a shape/entropy score, **on data blocks only**.
  
  It does **not** detect names or addresses in free text that never appear as a `field_keys` value. Precision and recall have not been measured. It is mitigation, not compliance.
- **Sanitizer:** collapses repeated lines and strips comments or blank lines. The `mova budget --focus` report separates **selection** savings from **sanitization** savings.

## Outside the perimeter

- Whatever an agent or IDE reads or sends on its own (its tools, @-mentions, other MCP servers). Without hooks, Mova does not see it. With hooks, it sees only what fires a hook.
- The model an MCP host uses: the manifest marks it `not_observable`.
- The policy author and the agent declared by the MCP client: recorded as `declared` / `observed: mcp-initialize`. There is no identity verification.
