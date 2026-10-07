![mova en acción / mova in action](docs/assets/mova-demo.gif)

# mova — especificación de contexto por tarea, validada y con evidencia

[Español](README.md) · [English](../en/README.md) · [Volver a la raíz](../../../README.md)

## Qué problema resuelve (y cuál no)

Cuando **restringes** lo que un modelo puede ver para una tarea — por costo, por privacidad o porque el código excluido no debe tocarse — aparecen dos preguntas que ni `AGENTS.md`, ni el propio agente, ni un gateway responden:

1. **¿La restricción rompe la tarea?** Si el código seleccionado llama a código que la misma especificación excluye, el contexto está incompleto por construcción. Mova lo detecta por AST **antes** de liberar nada (`dependency_policy`).
2. **¿Qué se liberó exactamente, bajo qué reglas y desde qué código?** Mova deja un registro inmutable por ejecución, con el hash de los bytes liberados.

Si tu agente puede leer todo el repo y no quieres restringirlo, Mova aporta poco: el agente resuelve las dependencias leyendo, y un gateway registra lo que se envió.

## El modelo en 4 piezas

| Pieza | Qué hace | Dónde |
|---|---|---|
| **Especificación** | `project.json`: por tarea, `focus` (archivo o símbolo: `archivo::func=a,b`), `exclude`, `policies`, `budget`, `read_scope` | [PROJECT_JSON](PROJECT_JSON.md) |
| **Validación de cierre** | Si un símbolo en focus llama o referencia algo **excluido** → conflicto. `block` (por defecto) no libera nada; `warn` libera y registra; `accept_missing` acepta con una razón escrita. También lista lo que queda **fuera** del contexto | [AST_FILTER](AST_FILTER.md) |
| **Contexto determinista y sanitizado** | Mismas entradas → mismos bytes (sin timestamps). Secretos: bloqueo o redacción solo del literal. PII (opcional): `field_keys`, detectores tipados (email, RUT, teléfono) y un puntaje heurístico solo sobre bloques de datos | [GOVERNANCE_CONTROLS](GOVERNANCE_CONTROLS.md) |
| **Evidencia inmutable** | `projects/<p>/runs/<run_id>/`: `context.txt`, `manifest.json` (de escritura única) y `events.jsonl` (solo anexado) | [ARTIFACTS](ARTIFACTS.md) |

## Perímetro: qué controla y qué no

| Integración | Contexto inicial | Turnos posteriores / tool results |
|---|---|---|
| `mova run` (stdout) | Controla y registra | No existen para Mova |
| `mova chat` / `chat_completion` con `llm_profile` (Mova llama al modelo) | Controla y registra | **Controla**: cada tool result pasa por la misma política de lectura, sanitización y presupuesto, y queda en `events.jsonl` |
| MCP / HTTP con un host (Claude Code, Cursor, Codex) | Controla lo que el host pide a Mova | **No ve** las tools propias del host (Read, Bash, Grep, @-menciones), salvo con hooks |
| Claude Code **con hooks** (`check_read`, `sanitize_tool_output`) | — | Deniega lecturas fuera de la especificación y reemplaza la salida de Read por la vista gobernada. No cubre @-menciones ni lo que no dispara hooks |
| Codex con hooks | — | Hoy solo es aplicable un bloqueo grueso: Codex lee por shell y no permite reemplazar la salida de MCP. Ver [MCP_INTEGRATION](MCP_INTEGRATION.md) |

Las tools de Mova **no** pueden saltarse sus propias reglas: toda lectura exige `project`, queda confinada al repo (rutas absolutas, `../` y symlinks se rechazan), respeta `exclude` y `read_scope`, y se sanitiza.

## Pruébalo (sin API key, sin llamar a un modelo)

```bash
mova run 04-nebula-delivery agregar-columnas     # se libera; [Evidence] run <id> → projects/04-nebula-delivery/runs/<id>
mova run 04-nebula-delivery recalcular-tarifas   # BLOQUEADO: calcularTarifa -> tarifaPlanaV1 (src/legacy/tarifasV1.js está excluido)
mova run 02-pii-compliance-governance            # dry_run: nada se libera; el manifest muestra qué se enmascaró por bloque
```

Instalar: `make install` (Go ≥ 1.24). Los costos que muestra Mova son estimaciones de tokens de entrada con `cl100k_base` (otros proveedores tokenizan distinto); cuando Mova llama al modelo, los tokens reales reportados por el proveedor quedan en `events.jsonl`.

## En qué se diferencia (y en qué no)

- **Commodity — no es argumento para elegir Mova:** conteo de tokens, deduplicación, enmascarado de PII, dry-run, logs por request, memoria entre sesiones, empaquetado del repo y compresión por AST. Gateways (LiteLLM, Portkey), empaquetadores (Repomix) y los propios agentes ya lo hacen, y en varios casos mejor.
- **Lo que Mova agrega:** una especificación de contexto **por tarea**, versionable y a nivel de símbolo; la verificación de que esa especificación es **cerrada** respecto de lo que excluye, antes de liberar; y un manifiesto que dice **por qué** se liberó o bloqueó cada contexto (regla, conflicto, decisión humana con razón). Un gateway sabe qué bytes se enviaron; no sabe qué tarea, qué símbolos ni qué dependencias faltaban.

## Documentación

- [GOVERNANCE_CONTROLS](GOVERNANCE_CONTROLS.md) — perímetro, controles y qué corta el proceso
- [MCP_INTEGRATION](MCP_INTEGRATION.md) — Claude Code (MCP + hooks), Codex, HTTP seguro
- [PROJECT_JSON](PROJECT_JSON.md) · [AST_FILTER](AST_FILTER.md) · [COMMANDS](COMMANDS.md) · [FUNCTIONS](FUNCTIONS.md)
- [ARTIFACTS](ARTIFACTS.md) — formato de `runs/<run_id>/`
- [CONTEXT-TRACE](CONTEXT-TRACE.md) · [SOURCE](SOURCE.md) · [FAQ](FAQ.md) · [VERIFICATION](../../VERIFICATION.md)
