# Example 06 — dependency closure (project `04-nebula-delivery`)

[Español](README.md)

A 182-line fictional repo (Node.js). It is small on purpose: token savings do not matter here. What matters is whether each task's spec is **closed** with respect to what it excludes.

```bash
mova run 04-nebula-delivery analizar-trazabilidad   # released
mova run 04-nebula-delivery agregar-columnas        # released
mova run 04-nebula-delivery recalcular-tarifas      # BLOCKED
```

`recalcular-tarifas` selects `calcularTarifa` but excludes `src/legacy/tarifasV1.js`. `calcularTarifa` calls `tarifaPlanaV1` when `ruta.catalogo === 'v1'`, so Mova does not release the context:

```
[recalcular-tarifas] src/facturacion/tarifas.js::calcularTarifa -> src/legacy/tarifasV1.js::tarifaPlanaV1 (call, excluido)
```

There are three ways to resolve it, each recorded in `manifest.json`:
1. Add `src/legacy/tarifasV1.js::func=tarifaPlanaV1` to `focus`.
2. Accept it with a reason: `"accept_missing": [{"symbol": "tarifaPlanaV1", "reason": "catalog v1 is out of scope"}]`.
3. Use `"dependency_policy": "warn"`: it is released, and the conflict stays in the evidence.

In released tasks, `manifest.json → dependency_closure.outside` lists what stays outside the context without being excluded. It is information for the spec author to decide on, not a block.

**What it does NOT show:** that the focus is the right one for the task, or that a model solves the task better with this context. That requires real tasks with a control group. Dynamic calls are not detected either.
