# FAQ

**What is Mova Context, exactly?** A Go binary (CLI + MCP server) that sits between "context is ready" and
"send it to the LLM". It's not an agent framework or a code editor.

**What does "Context Governance" actually do?** Decides and leaves evidence of: what context was selected,
what was sanitized/masked (PII, secrets), what it would cost, and who/what authorized it — before the LLM
call. It decides nothing during or after inference.

**How does it protect PII before it reaches a model?** Heuristic structural masking (shape + entropy,
`budget.pii_masking.enabled`), not a name dictionary. Not 100% foolproof and doesn't replace legal review —
see `docs/i18n/en/ARTIFACTS.md`.

**How does it help with cost?** `mova budget`/`context-trace` estimate tokens (real `cl100k_base` tokenizer)
and cost against `config/prices.json` per provider, before spending anything.

**What happens if I go over budget?** `budget.on_exceed` in `project.json` (`warn` or `block`) — the circuit
breaker stops or warns before sending, not after.

**Does my data leave my machine?** Only if your `llm_profile` points to a cloud provider. With `ollama`/
`lm-studio` (local), nothing leaves — check the `TargetModel` field in any report to confirm.

**Why does `pii-audit-log.json` sometimes show zeros under `security`?** That fine-grained detail only
populates in *discovery* mode (remote repo, no `project.json`). In project mode, the real PII Masking
result is in `mova-budget-report.md`.
