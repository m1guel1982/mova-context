# Example 07 — Multi-agent: the Nebula Delivery fleet

> Same fictional universe as [example 06](../06-nebula-delivery-graph-memory/README_EN.md), now with **three specialized agents** reviewing the same repo, each with its own `focus`/`exclude`, graph and egress, and **passing the baton through a shared memory**.

## What a multi-agent setup is in Mova

A **group** is a folder under `projects/` with a `config.json` listing agents; **each agent is an ordinary project** (`projects/<group>/<agent>/project.json`) with its own budget, tasks, focus, memory, graph and egress. There is no special format: an agent is a project addressed as `<group>/<agent>`.

```
projects/05-nebula-flota/
├── config.json                  ← the orchestrator: {"group", "description", "agents": [...]}
├── memory.md                    ← the group's SHARED memory
├── planificador/  project.json · grafo.png · egress_sanitized.md   → reviews routing
├── facturador/    project.json · grafo.png · egress_sanitized.md   → improves billing and fares
└── auditor/       project.json · grafo.png · egress_sanitized.md   → audits PII (pii_masking on)
```

| Agent | Task | Focus | Excludes |
|---|---|---|---|
| `planificador` | `revisar-ruteo` | `planificador.js`, `rutas.js` | `depurarTrazaPlanificador`, `telemetria/sensores.js` |
| `facturador` | `mejorar-cobros` | `cobros.js`, `tarifas.js` | `tarifasV1.js`, `recargoRadiacionLegacy` |
| `auditor` | `auditar-datos` | `notificaciones.js`, `cobros.js`, `pedidos-demo.json` | `tarifasV1.js` |

**Shared memory:** all three agents declare `"memory": "../memory.md"` (relative to each agent's folder), so they write to and read the **same** `projects/05-nebula-flota/memory.md`. Each entry carries the task that produced it.

## How to run it

```bash
mova agents list 05-nebula-flota              # the group's 3 agents
mova agents run  05-nebula-flota              # builds all 3 contexts (NO LLM call) and generates their graphs
mova agents run  05-nebula-flota facturador   # just one

# Chat with one agent after another: each reads what the previous left behind
mova chat 05-nebula-flota planificador        # leaves its finding in the shared memory
mova chat 05-nebula-flota facturador          # receives that finding in MEMORY and starts from it
mova chat 05-nebula-flota auditor             # receives both; its context leaves with PII masked
mova chat 05-nebula-flota                     # no agent: lists the available agents
```

**Over MCP / HTTP** (same orchestrator):

```bash
mova mcp start --port 3000

# list / run agents (MCP tools list_agents · run_agent)
curl -X POST localhost:3000/agents/run -H 'Content-Type: application/json' \
     -d '{"group":"05-nebula-flota","agent":"facturador"}'

# chat with an agent over JSON-RPC: the project is "<group>/<agent>"
curl -X POST localhost:3000/mcp -H 'Content-Type: application/json' -d '{
  "jsonrpc":"2.0","id":1,"method":"tools/call",
  "params":{"name":"chat_completion","arguments":{"project":"05-nebula-flota/planificador","message":"review the routing"}}}'
```

## What it demonstrates

- **Orchestration without magic:** `config.json` + ordinary projects; the same commands (`run`, `chat`, MCP, HTTP) work for one agent and for a group.
- **Specialization:** each agent sees only its slice of the code (its own `focus`/`exclude`), so its context and cost stay small.
- **Memory hand-off:** `facturador` receives in `MEMORY` the `planificador`'s finding (`etaReal` lost in `normalizarPedido`) and `auditor` receives both. Open `memory.md` and you will see three entries, one per task.
- **Per-agent governance:** `auditor` turns on `budget.pii_masking`; in its `egress_sanitized.md` the customer's email and document appear as `[PII_xxxxxxxx]`.
- **One graph per agent** (`grafo.png`), spending no tokens.

## Honest notes

- `mova agents run` **does not call the model**: it assembles and returns each agent's context (in parallel, with a worker cap). So it writes no memory; memory is recorded when you chat (`mova chat`) or via `chat_completion`.
- The hand-off order is **the order in which you talk to the agents**; the group does not impose a chain.
- This example reuses the code in `examples/06-nebula-delivery-graph-memory/repo` and the `nebula-demo` model.
- The sample `memory.md`, `egress_sanitized.md` and graphs came from the real binary with **simulated model replies** (consistent with the code). The subtitle path in each PNG is that of the machine where it was generated.
