# 01 — Gobernanza de Contexto para agentes MCP

**Qué es:** un agente MCP (Claude Code, Cursor, Windsurf) pide contexto de este repo.
Mova decide qué entra, detecta un secreto, y deja evidencia — antes de que nada llegue al LLM.

## Ejecutar (1 comando)

```bash
mova run 01-mcp-agent-governance --diagram --export png --path ./evidencia.png
```

## Qué vas a ver 

| Paso | Resultado |
|---|---|
| Contexto seleccionado | `repo/api.js` + `repo/config.txt` (`focus` en `project.json`) |
| Control aplicado | Sanitizer ON · `config.txt` contiene un secreto de ejemplo (`STRIPE_SECRET_KEY=...`) que Mova reconoce como indicador de credencial |
| Decisión | Contexto permitido, con la política y el agente que lo autorizaron |
| Evidencia | `evidencia.png` (diagrama) + `context-report.md` + `pii-audit-log.json` |

`evidencia-ejemplo.png` en esta carpeta es una muestra ya generada — corré el comando de arriba para regenerarla tú mismo.

## Conectar un agente MCP real

```json
{ "mcpServers": { "mova": { "command": "mova", "args": ["mcp"] } } }
```

Con eso, Claude Code/Cursor llaman la misma herramienta `context_trace` que corrió este ejemplo — mismo motor, la única diferencia es quién pregunta (`agent_client` queda registrado automáticamente, ver `pii-audit-log.json`).

## Ver la traza en texto (sin diagrama)

```bash
mova context-trace 01-mcp-agent-governance --export md
```

`project/project.json` en esta carpeta es una copia de lectura — el archivo real que usa el motor vive en `projects/01-mcp-agent-governance/project.json` (ruta fija que Mova siempre busca).
