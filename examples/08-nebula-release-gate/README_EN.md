# Example 08 — Multiagent release gate (no LLM of its own)

Three agents (`mapeador` → `guardian-pii` → `revisor`) over the fictional Nebula Delivery repo. **No new API or functions**: only `config.json` + ordinary projects + existing MCP tools.

| Agent | Focus | Control |
|---|---|---|
| `mapeador` | `normalizarPedido`, `planificarEntrega` | where `etaReal` is lost |
| `guardian-pii` | `notificarCliente`, `generarCobro`, `pedidos-demo.json` | `pii_masking` on |
| `revisor` | `generarCobro`, `construirFilaReporte` | reads both previous findings |

All three share `projects/08-nebula-release-gate/memory.md` (`"memory": "../memory.md"`) and use `egress_audit.dry_run: false` **on purpose**: verified, with `dry_run: true` the host cannot read context or memory over MCP (it gets `tokens_sent: 0`). With no `llm_profile` there is no provider to send anything to.

## Run
```bash
mova agents run 08-nebula-release-gate      # context + graph for all 3, no LLM
bash examples/08-nebula-release-gate/run-demo.sh   # MCP/HTTP orchestration
```
`run-demo.sh` plays the host agent: `list_agents` → `estimate_budget` → `run_agent` → `save_memory` (one per agent). In practice Claude Code or Cursor does this.

## Honest notes
- With no `llm_profile`, Mova never calls a model; the host reasons and records via `save_memory`. The script's findings are **hand-written** (consistent with the code), not model output.
- `mova agents run` writes no memory. Order is imposed by whoever orchestrates.
- **Executed** with the real Linux amd64 binary; output and resulting memory are in `evidence/`. `evidence/guardian-pii.egress_sanitized.md` shows 8 `[PII_…]` occurrences (pseudonyms of fictional data).
- Verified on Linux amd64 only.
