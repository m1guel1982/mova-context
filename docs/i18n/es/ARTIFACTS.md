# Artefactos generados por Mova Context

Tabla de referencia rápida: qué archivo, cuándo se genera, para qué sirve. Cada fila se entiende en 15 segundos.

| Archivo | Se genera con | Para qué sirve | ¿Se mantiene? |
|---|---|---|---|
| `context-report.md` | `mova context-trace` (o `--export md`) | Reporte legible: qué contexto se ensambló, presupuesto, costo estimado, agente/modelo/autor de política. La evidencia principal para un humano. | Sí — evidencia primaria |
| `context-report.pdf` | `mova context-trace --export pdf` | Mismo contenido que el `.md`, formato imprimible/compartible (auditoría, compliance). | Sí — evidencia primaria |
| `context-diagram.png` / `evidencia.png` | `mova context-trace --diagram` o `mova run --diagram --export png` | Evidencia **visual**: fuentes → gobernanza (Sanitizer/PII/Circuit Breaker) → agente → tokens → costo, en un solo vistazo. | Sí — evidencia primaria |
| `pii-audit-log.json` | Cualquier corrida de `context-trace` | JSON máquina-legible con la decisión (`ALLOWED`/`SANITIZED`/`BLOCKED`/`EXCLUDED`) y la identidad de auditoría (`agent_client`, `target_model`, `policy_author`). El detalle fino de hallazgos de seguridad solo se llena en modo *discovery* (repo remoto sin `project.json`); en modo proyecto, el resultado real de PII Masking está en `mova-budget-report.md`. | Sí — evidencia primaria, formato de integración (CI/CD, SIEM) |
| `mova-budget-report.md` | `mova budget <proyecto>` | Desglose de tokens y costo por proveedor para la tarea activa, y el resultado real de PII Masking (cuántos tokens se pseudonimizaron). | Sí — se regenera en cada corrida, no se versiona |
| `mova-context-cache.json` | Automático, primera vez que se lee un archivo pesado (focus/exclude) | Caché de rendimiento: evita re-sanitizar el mismo archivo si no cambió (hash de contenido). Los campos de `stats` (`LinesRemoved`, `CommentsRemoved`, etc.) solo tienen valores > 0 cuando `sanitize` realmente quitó algo de ESE bloque; si están en cero, el archivo se sirvió tal cual. | Sí, pero es 100% desechable — bórralo cuando quieras, se reconstruye solo. **No se versiona en git.** |
| `mova-token-history.json` | Automático, cada corrida con `project.json` | Serie histórica de tokens/costo por ejecución — para ver tendencia en el tiempo. | Opcional — útil si te importa la tendencia, prescindible si no |
| `memory.md` | Se crea vacío al iniciar el proyecto; se escribe con `mova memory`, `/memory` en `mova chat`, o la herramienta MCP `save_memory` | Notas persistentes entre ejecuciones para un proyecto (decisiones, contexto de negocio). No es evidencia de auditoría — es contexto para el agente. | Opcional — específico de memoria de proyecto, no de gobernanza |

## Regla general

Los archivos marcados **"evidencia primaria"** son los que responden la Matriz de Auditoría de 11 preguntas (ver `README.md`) y son los que vale la pena guardar o adjuntar a un ticket/auditoría.
Los demás (`mova-context-cache.json`, `mova-token-history.json`, `memory.md`) son soporte operativo: se regeneran solos y no hace falta versionarlos ni revisarlos manualmente.

## Recomendación de `.gitignore`

```
mova-context-cache.json
mova-token-history.json
mova-budget-report.md
```
