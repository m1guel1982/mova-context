# mova + Claude Code (MCP)

Claude Code es el modelo; mova es un **servidor local de herramientas MCP** al que puede llamar (`mova mcp start --stdio`). mova no reemplaza a Claude Code: prepara, recorta, enmascara, cotiza y registra el contexto que Claude recibe *a través de mova*.

## 1. Conectar (2 minutos)
Requisito: `mova` en el PATH y `MOVA_PROJECT_ROOT` = carpeta de mova (el instalador deja ambos).

```bash
# desde tu repo de TRABAJO; --scope project escribe .mcp.json (compartible con el equipo)
claude mcp add --transport stdio --scope project \
  --env MOVA_PROJECT_ROOT=/ruta/abs/a/mova-context \
  mova-context -- mova mcp start --stdio
claude mcp list            # debe aparecer mova-context
```
Las opciones van **antes** del nombre del servidor (docs de Claude Code). Equivalente en archivo: copia `config/mcp_clients/claude_code.json` a `.mcp.json` y edita la ruta (Windows: `"C:\\appMovaContext"`). La primera vez Claude Code pide aprobar un servidor de scope proyecto. Las herramientas aparecen como `mcp__mova-context__<tool>`. Quitar: `claude mcp remove mova-context -s project`.

## 2. Qué usar desde Claude Code
| Objetivo | Tool | Nota |
|---|---|---|
| Ver proyectos | `list_projects` | |
| Cotizar el contexto antes | `estimate_budget` | local, sin llamar a un modelo |
| Obtener contexto gobernado | `get_full_context` | Focus/AST + sanitizer + PII + presupuesto |
| Auditar un repo sin proyecto | `context_trace` (`repo` o `project`) | informe + diagrama + `pii-audit-log.json` |
| Imagen de evidencia | `generate_diagram` | |
| Recordar entre sesiones | `save_memory` / `get_memory` | compartida por CLI/chat/MCP |
| Varios roles | `list_agents` / `run_agent` | ejemplo 08 |
**No** uses `chat_completion` aquí: llama a *otro* modelo vía mova. Claude Code ya es el modelo.

## 3. Verificado vs no (honesto)
Verificado (Linux, binario real, cliente stdio JSON-RPC como Claude Code): `initialize` → servidor `mova-context`, protocolo `2024-11-05`; `tools/list` → 26 tools; `estimate_budget` en el ejemplo 02 → 7153 tokens.
Comportamiento verificado de `get_full_context` en el ejemplo 02:
- `dry_run: true` → Claude recibe solo un aviso de auditoría (555 bytes, `tokens_sent: 0`). Sirve para **demostrar** que no se entrega nada.
- `dry_run: false` → Claude recibe el contexto gobernado: 23.463 bytes, **171 pseudónimos `[PII_xxxxxxxx]`, 0 emails crudos**.
**No verificado:** una sesión real de Claude Code (el autor no tenía Claude Code disponible), ejecución MCP en Windows/macOS.

## 4. Límites que debes conocer (leer antes de usarlo en el trabajo)
- mova gobierna solo lo que pasa **a través de mova**. Claude Code puede leer archivos por sí mismo (`Read`, `Bash`) y enviarlos al proveedor. mova no lo ve ni lo impide; con `dry_run` un host decidido puede intentar reconstruir el contexto desde archivos locales (documentado en el README).
- El enmascarado PII es heurístico; recall/precisión sin medir. Es evidencia y mitigación, no certificación de cumplimiento.
- Consulta a seguridad/legal los términos de datos de Claude Code en tu organización; mova no los cambia.

## 5. Hacer que Claude Code pase realmente por mova
1. **CLAUDE.md** en el repo de trabajo:
   ```
   Antes de leer código para una tarea: llama a mcp__mova-context__estimate_budget y luego get_full_context del proyecto <nombre>.
   No abras archivos listados en "exclude" de project.json. Al terminar, llama a save_memory con los hallazgos.
   ```
2. **`.claude/settings.json`** — permite las tools de mova y niega la lectura directa de lo que no debe salir (sintaxis según la doc de permisos de Claude Code; las reglas `Read` no cubren comandos de shell, así que restringe también `Bash` o usa su sandbox):
   ```json
   { "permissions": {
       "allow": ["mcp__mova-context__estimate_budget","mcp__mova-context__get_full_context","mcp__mova-context__context_trace"],
       "deny":  ["Read(./.env)","Read(./secrets/**)","Read(./data/customers*.json)"] } }
   ```
3. **project.json** del repo de trabajo: `mova init <nombre>` (o `mova context-trace --repo <ruta> < /dev/null` y responder `Y` para generar uno), `"repo"` con ruta absoluta, `focus` / `exclude`, `budget.pii_masking.enabled: true`, `budget.max_tokens`, `"memory": true` y `egress_audit.dry_run: false` (para que Claude lea el contexto *gobernado*).
4. Revisa la evidencia: `egress_sanitized.md`, `context-report.md`, `pii-audit-log.json` en la carpeta del proyecto.

## 6. Problemas frecuentes
`claude mcp list` falla → ejecuta `mova mcp start --stdio` a mano; si no halla la raíz, falta `MOVA_PROJECT_ROOT` en `env`. Una tool responde «bloqueado» → `dry_run: true`. En `claude -p` (no interactivo) los servidores de proyecto se cargan sin pedir aprobación.
