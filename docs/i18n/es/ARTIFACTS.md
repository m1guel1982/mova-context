# Qué genera Mova

[README](README.md) · [English](../en/ARTIFACTS.md)

## Evidencia por ejecución — `projects/<proyecto>/runs/<run_id>/`

Un directorio por cada intento de liberar contexto (`mova run`, `mova chat`, `get_full_context`, `chat_completion`, `run_agent`), más un run de sesión por proceso MCP/HTTP para lecturas y hooks.

| Archivo | Escritura | Contenido |
|---|---|---|
| `context.txt` | única | Los bytes exactos que Mova liberó, o habría liberado con `dry_run`. Vacío si un gate bloqueó |
| `manifest.json` | única | `door`, `project`, `task`, `repo` (commit y estado dirty, o una nota si no hay git), `config` (sha256 de `project.json` y de cada política), `spec` (focus, exclude, read_scope, dependency_policy, accept_missing, max_tokens, pii_masking), `selection`, `dependency_closure`, `governance` (bloques cambiados o bloqueados), `context` (sha256, bytes, tokens estimados, tokenizer), `agent` / `model` / `policy_author` con su fuente (`declared`, `observed`, `not_observable`), `decision` (`released`, `blocked` + gate, `dry_run`), `perimeter` |
| `events.jsonl` | solo anexado | `provider_call` (tokens reales reportados por el proveedor), `dry_run_block`, `tool_result`, `read`, `hook_check_read`, `hook_sanitize_output` |

Responde a la pregunta: **¿qué contexto produjo Mova, bajo qué reglas, desde qué código y para qué tarea?**

No responde qué hizo el agente fuera de Mova. El contexto no incluye timestamps, así que dos runs con las mismas entradas tienen el mismo `context.sha256`.

Los `runs/` no se versionan (están en `.gitignore`).

## Otros archivos

- `memory.md`: síntesis entre tareas. Cada entrada lleva `source=observed run=<id>` (Mova llamó al modelo), `source=host` (la registró un host con `save_memory`; Mova no vio el modelo) o `source=user`.
- `mova-budget-report.md` (`mova budget`), grafos `*.png` (`"graph"` en la tarea), `context-report.*` (`mova context-trace`): reportes para leer. **No** son evidencia de egreso.
