# Mova con Claude Code, Codex y HTTP

[README](README.md) · [English](../en/MCP_INTEGRATION.md)

## 1. Registrar Mova (stdio)

```bash
claude mcp add --transport stdio --scope project --env MOVA_PROJECT_ROOT=<ruta de mova> mova-context -- mova mcp start
```

`mova mcp start` usa stdio por defecto. Con esto, el agente **puede** pedir `get_full_context`, `read_file`, `estimate_budget`… pero nada lo obliga: sus propias tools (Read, Bash, Grep) siguen fuera de Mova.

## 2. Extender el perímetro con hooks (Claude Code)

Claude Code permite hooks de tipo `mcp_tool` en `PreToolUse` y `PostToolUse`. Mova expone dos tools pensadas para eso, que reutilizan exactamente la misma política que `read_file`:

- `check_read` (PreToolUse): devuelve `permissionDecision: "deny"`, con la razón, si la ruta está fuera del repo, excluida o fuera del focus (`read_scope: focus`). Si la lectura está permitida devuelve `{}`: Mova nunca auto-aprueba, así que el flujo normal de permisos se mantiene.
- `sanitize_tool_output` (PostToolUse): para `Read`, reemplaza la salida por la vista gobernada (solo los símbolos en focus, sin los símbolos excluidos, con secretos y PII sanitizados) mediante `updatedToolOutput`. Para otras tools (Bash, Grep), sanitiza el texto.

Ejemplo para `.claude/settings.json`. Ajusta los nombres de campo a la sintaxis de hooks `mcp_tool` de tu versión de Claude Code:

```json
{
  "hooks": {
    "PreToolUse": [{ "matcher": "Read|Grep|Glob", "hooks": [{
      "type": "mcp_tool", "server": "mova-context", "tool": "check_read",
      "input": { "project": "04-nebula-delivery", "tool_name": "${tool_name}", "tool_input": "${tool_input}" } }] }],
    "PostToolUse": [{ "matcher": "Read|Bash", "hooks": [{
      "type": "mcp_tool", "server": "mova-context", "tool": "sanitize_tool_output",
      "input": { "project": "04-nebula-delivery", "tool_name": "${tool_name}", "tool_input": "${tool_input}", "tool_response": "${tool_response}" } }] }]
  }
}
```

Cada decisión queda en `projects/<p>/runs/<run de sesión>/events.jsonl` (`hook_check_read`, `hook_sanitize_output`).

**Límites:**
- Las @-menciones insertan contenido sin pasar por una tool: para eso usa también reglas `deny` de permisos de Claude Code.
- Grep y Glob solo se evalúan a nivel de directorio.
- Un host que no ejecuta hooks no queda gobernado.

**Estado de verificación:** las respuestas de ambas tools están cubiertas por tests de Mova. La integración de punta a punta dentro de Claude Code **no** está verificada todavía.

## 3. Codex

Los hooks de Codex permiten denegar en PreToolUse y, en PostToolUse, reemplazar el resultado por un mensaje de bloqueo. Todavía no permiten reemplazar la salida de una tool MCP por una versión sanitizada. Además, Codex lee archivos mediante comandos de shell, así que aplicar `read_scope` exigiría interpretar comandos, algo que Mova no hace.

En Codex, Mova sirve para generar y validar la especificación (`mova run`, cierre de dependencias, evidencia) y para el contexto que el agente pide por MCP. No gobierna las lecturas propias de Codex.

## 4. HTTP

```bash
mova mcp start --http --port 3000                                 # escucha en 127.0.0.1
MOVA_HTTP_TOKEN=… mova mcp start --http --bind 0.0.0.0            # fuera de loopback: token obligatorio
```

- Sin `MOVA_HTTP_TOKEN`, un bind fuera de loopback se rehúsa a arrancar.
- Con token, cada request debe enviar `Authorization: Bearer <token>`.
- Se rechazan los `Origin` que no sean localhost.

El servidor expone lecturas **y escrituras** del repo del proyecto: no lo publiques en una red sin token.
