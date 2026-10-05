# Example 06 — Nebula Delivery: AST graph + memory across tasks + audited egress

> A **100 % fictional** project (delivery capsules to the lunar station Selene) written in JavaScript, to see three things Mova does together in 2 minutes: it **draws the function graph without spending tokens**, it lets **one task start from what the previous one concluded** (`memory`), and it **audits what leaves for the LLM** (`egress_audit`).

| What it demonstrates | Where to see it |
|---|---|
| Per-task dependency graph, no LLM | `projects/04-nebula-delivery/analizar.png` and `agregar-columnas.png` |
| `exclude` by file **and** by function inside a file that is still used | red dotted in the graphs (`tarifasV1.js`, `recargoRadiacionLegacy`, `depurarTrazaPlanificador`) |
| `memory: true`: task 2 reads task 1's synthesis | `projects/04-nebula-delivery/memory.md` |
| `egress_audit`: sanitized evidence of what was sent, duplicate-free | `projects/04-nebula-delivery/egress_sanitized.md` |
| Own agents/skills/prompts (`paths`) matching the JSON variables | `capabilities/` |

## The story (fictional)

Nebula Delivery's billing report shows the **scheduled** arrival time (`etaProgramada`) even though telemetry already measured the **real** one (`etaReal`). Business rule: the final report value uses `etaReal` if present, otherwise `etaProgramada`. Two tasks, in order:

1. **`analizar-trazabilidad`** — in which function is `etaReal` lost? (changes nothing)
2. **`agregar-columnas`** — add *Llegada real* / *Llegada programada* to the CSV, **starting from task 1's finding**.

## Layout

```
examples/06-nebula-delivery-graph-memory/
├── repo/                          ← the fictional code Mova analyzes
│   ├── src/despacho/{planificador,rutas}.js
│   ├── src/facturacion/{tarifas,cobros,notificaciones}.js
│   ├── src/telemetria/sensores.js
│   ├── src/legacy/tarifasV1.js    ← excluded
│   └── datos/pedidos-demo.json
└── capabilities/{agents,skills,prompts}/   ← this example's agent, skill and 2 prompts
projects/04-nebula-delivery/
├── project.json                   ← 2 tasks, memory: true, egress_audit, graph per task
├── analizar.png · agregar-columnas.png
├── memory.md · egress_sanitized.md
config/models/ollama/nebula-demo.json   ← num_ctx 16384 and max_tokens 2048
```

## How to run it (from Mova's root)

```bash
ollama pull llama3.2:3b                       # or point llm_profile to another model

# 1) No LLM: builds the context, generates both graphs and the egress file
mova run 04-nebula-delivery analizar-trazabilidad
mova run 04-nebula-delivery agregar-columnas

# 2) Task 1 in a chat → on finishing it leaves its synthesis in memory.md
mova chat 04-nebula-delivery analizar-trazabilidad
> revisar            # you will see: [Memory] 1 entrada(s) guardada(s) ... (tarea: analizar-trazabilidad)
> exit

# 3) Task 2 in ANOTHER chat (another process): it receives that synthesis in the MEMORY section
mova chat 04-nebula-delivery agregar-columnas
```

Also without leaving the chat: `mova chat 04-nebula-delivery` (no task = **all**) then `/tasks`, `/task agregar-columnas`, `/run agregar-columnas`. Over MCP/HTTP the same memory applies to `chat_completion` with `project: "04-nebula-delivery"` and `task`.

## What you will see

**Task 2's graph** — blue = in focus, gray dotted = out of focus, **red dotted = excluded**; `calcularTarifa → tarifaPlanaV1` shows up even though `tarifasV1.js` is excluded:

![agregar-columnas graph](../../projects/04-nebula-delivery/agregar-columnas.png)

**`memory.md`** — per entry: date, task and the synthesis block (`file::function` findings, key data, decisions, pending). An identical synthesis is never written twice.

**`egress_sanitized.md`** — one block per context piece (each agent/skill/prompt, each `FOCUS`, the `MEMORY`). Same → skipped; changed → replaced; new → appended. In chat 2 you can see here that the `MEMORY` with task 1's finding **did** go to the model.

## Why the graph helps

- **Zero tokens:** drawn with the same Tree-Sitter as the AST filter, never calling the model.
- **Check your `focus`/`exclude` before spending:** see what is in, what is out of focus and what you excluded (and which excluded symbol is still being called).
- **One image per task**, regenerated only when code or JSON changes (`graph-cache.json`).

## Honest notes

- The sample `memory.md`, `egress_sanitized.md` and PNG files were produced by running the real Mova binary; the **model replies were simulated** (consistent with the code) so the example is reproducible without a GPU. With your local model the wording will differ.
- `mova chat` generates the graph in the background and **does not wait on exit**: run `mova run` first (as above) or wait for the `[Graph] … grafo generado` notice.
- With `egress_audit.dry_run: true` Mova audits and measures but **never calls the model** (so no memory is recorded).
- This example targets models with a window ≥ 8k: adjust `num_ctx` in `nebula-demo.json` if you use another.
