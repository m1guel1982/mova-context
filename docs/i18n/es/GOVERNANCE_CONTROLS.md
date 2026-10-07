# Perímetro y controles

[README](README.md) · [English](../en/GOVERNANCE_CONTROLS.md)

## Orden fijo dentro de Mova

```
especificación (project.json) → cierre de dependencias → selección (focus/exclude, AST)
  → sanitizador (dedup, comentarios) → secretos y PII por bloque → circuit breaker → max_tokens
  → evidencia (runs/<run_id>/) → dry_run → liberación (stdout, host MCP o proveedor)
```

Toda puerta (`mova run`, `mova chat`, `get_full_context`, `chat_completion`, `run_agent`) usa la misma función (`budget.BuildGatedContext`) y escribe un run **antes** de liberar. Si la evidencia no se puede escribir, no se libera nada.

## Qué corta el proceso

| Control | Configuración | Efecto | Evidencia |
|---|---|---|---|
| Cierre de dependencias | `dependency_policy`: `block` (por defecto), `warn`, `off`; `accept_missing: [{symbol, reason}]` | `block` + conflicto → no se libera contexto | `manifest.json → dependency_closure`, `decision.gate = dependency_closure` |
| Secretos | `config/policy/security.json`: `block_on_private_key`, `block_on_api_key` (true por defecto) | El bloque afectado se reemplaza por `[MOVA: contenido omitido por política …]`. Sin bloqueo, se redacta solo el literal (`[REDACTED_SECRET]`) y la sintaxis del código queda intacta | `governance.changed_blocks[]` |
| Presupuesto | `budget.max_tokens`, `on_exceed`, circuit breaker | Se rechaza el contexto. En los loops que controla Mova, también los tool results que superen el tope acumulado | `decision.gate = budget` / evento `tool_result.budget_exceeded` |
| `dry_run` | `egress_audit.dry_run: true` | Nada sale de Mova: ni stdout, ni host, ni proveedor; tampoco memoria ni lecturas | `decision.outcome = dry_run` / evento `dry_run_block` |
| Política de lectura | `read_scope`: `focus` (por defecto si hay focus) o `repo`; `exclude`; límite del repo | Deniega `read_file`/`read_document_layer`, lecturas del loop y `check_read` | evento `read` / `tool_result` / `hook_check_read` |

## Qué solo transforma

- **PII** (`budget.pii_masking.enabled: true`):
  - `field_keys`: los valores de esas claves en datos estructurados se seudonimizan, y el mismo valor se enmascara en cualquier otro bloque del contexto;
  - detectores tipados (email, RUT, teléfono);
  - un puntaje de forma/entropía, **solo en bloques de datos**.
  
  **No detecta** nombres ni direcciones en texto libre que no aparezcan como valor de `field_keys`. No se ha medido precisión ni recall. Es mitigación, no cumplimiento.
- **Sanitizador:** colapsa líneas repetidas y quita comentarios o líneas en blanco. El reporte de `mova budget --focus` separa el ahorro por **selección** del ahorro por **sanitización**.

## Qué queda fuera del perímetro

- Lo que un agente o IDE lee o envía por su cuenta (sus tools, @-menciones, otros servidores MCP). Sin hooks, Mova no lo ve. Con hooks, solo ve lo que dispara un hook.
- El modelo que usa un host MCP: el manifest lo marca como `not_observable`.
- El autor de la política y el agente declarado por el cliente MCP: se registran como `declared` / `observed: mcp-initialize`. No hay verificación de identidad.
