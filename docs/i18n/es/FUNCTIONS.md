# mova — Matriz de Funciones y Argumentos (MCP · HTTP · Chat/CLI)

Referencia única. Fuente de verdad: `src/mcp/tool_registry.go` (MCP), `src/http/server.go` (HTTP),
`src/cli/dispatch.go` + `src/cli/chat_cmd.go` (CLI/Chat). `R` = requerido, `o` = opcional.
Todos los argumentos son strings (booleanos como `"true"`). Un motor, cuatro puertas: cada ruta HTTP es un
envoltorio delgado que reemite un `tools/call` MCP, por lo que HTTP ≡ MCP por construcción.

**Plantilla de llamada (MCP stdio o HTTP `POST /mcp`):**
```json
{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"<tool>","arguments":{...}}}
```
```bash
echo '<json>' | mova mcp start --stdio                      # stdio
curl -s localhost:3000/mcp -H 'Content-Type: application/json' -d '<json>'   # HTTP (mova mcp start --port 3000)
```

## 1. Núcleo — gobernanza pre-inferencia (el producto)

| Función | Tool MCP · argumentos | HTTP | CLI | Chat |
|---|---|---|---|---|
| Ensamblar contexto (agents+skills+prompt+memory+focus) | `get_full_context` · `project`R `task`o | `/mcp` | `mova run <proyecto> [tarea]` | `/run <tarea>` |
| Estimar tokens/USD (sin llamar al LLM) | `estimate_budget` · `project`R `task`o `focus`o | `/mcp` | `mova budget <proyecto> [tarea] [--focus]` · `mova run --count` | `/budget` |
| Context trace (auditoría previa) | `context_trace` · `project`o `task`o `repo`o `branch`o `export`o(md\|pdf) `output`o `generate_project_json`o | `POST /api/v1/context-trace` (mismos args) | `mova context-trace <proyecto>` / `--repo <url>` · alias `trace` | `/context-trace [--repo <url>]` |
| Diagrama de evidencia (SVG/PNG/PDF) | `generate_diagram` · `project`R `task`o `export`o `path`o `detail`o(simple\|verbose) | `POST /diagram` | `mova run <proyecto> --diagram --export png --path <p>` | `/diagram` |
| Llamada al LLM con contexto gobernado | `chat_completion` · `message`R `model`o `project`o `task`o `apply_edits`o `apply_changes`o | `/mcp` | `mova chat <proyecto> [tarea]` | (el propio chat) |
| Egress audit / air-gap | sin tool propia: `egress_audit` en `project.json` protege `chat_completion`, `get_full_context`, `get_memory*`, `get_workflow`, `read_file`, `read_document_layer` | igual | igual | igual |
| Enmascarado PII, sanitizer, tope de presupuesto | `project.json` → `budget.pii_masking` / `budget.sanitize` / `budget.max_tokens` + `on_exceed` | igual | igual | igual |
| Políticas | `project.json` → `policies`; CLI `--policies_include/--policies_exclude <lista>` | — | `mova context-trace <p> --policies_include a.json,b.json` | — |

**Comportamiento verificado:** con `egress_audit.dry_run: true`, las llamadas MCP/HTTP a `get_full_context`, `get_memory*` y `chat_completion` devuelven un aviso de auditoría (`tokens_sent: 0`) en vez del contenido. Un flujo *orquestado por un anfitrión* (ejemplo 08) usa por eso `dry_run: false`; `dry_run: true` sirve para demostrar que nada sale.

## 2. Memoria (compartida por CLI, chat, MCP, HTTP)

| Función | Tool MCP · argumentos | HTTP | CLI | Chat |
|---|---|---|---|---|
| Leer memoria activa | `get_memory` · `project`R | `/mcp` | `mova memory-read <p>` | — |
| Leer activa + archivada | `get_memory_all` · `project`R | `/mcp` | `mova memory-read <p> --all [--month AAAA-MM]` | — |
| Guardar una síntesis | `save_memory` · `project`R `entry`R `task`o | `/mcp` | `mova memory <p> "texto"` | `/memory` (última respuesta) |
| Archivar / borrar / configurar | **no expuesto** | — | `memory-archive [--days N]` · `memory-clear [--archived\|--keep-active\|--date\|--from --to\|--yes]` · `memory-config enable\|disable\|days N\|confirm true\|false` | — |

## 3. Multiagente (grupo = `projects/<grupo>/config.json`, agente = proyecto normal)

| Función | Tool MCP · argumentos | HTTP | CLI | Chat |
|---|---|---|---|---|
| Listar agentes de un grupo | `list_agents` · `group`R | `/mcp` | `mova agents list <grupo>` | `mova chat <grupo>` (lista) |
| Ejecutar uno / todos (arma contexto, no llama al LLM) | `run_agent` · `group`R `agent`o `task`o | `POST /agents/run` | `mova agents run <grupo> [agente\|--all]` | — |
| Conversar con un agente | `chat_completion` · `project`=`<grupo>/<agente>` | `/mcp` | `mova chat <grupo> <agente>` | — |
| Estimación de tokens del grupo | `estimate_budget` · `project`=`<grupo>` (suma agentes) | `/mcp` | `mova run --count <grupo>` | — |

## 4. Conocimiento y descubrimiento

| Función | Tool MCP · argumentos | HTTP | CLI | Chat |
|---|---|---|---|---|
| Listar proyectos | `list_projects` | `/mcp` | `mova list` | — |
| Un agent/skill/prompt | `get_knowledge` · `kind`R `domain`R `name`R `lang`o | `/mcp` | — | — |
| Buscar conocimiento | `search_context` · `query`R `domain`o | `/mcp` | `mova search "q" [dominio]` | — |
| Leer `workflow.md` (con presupuesto validado) | `get_workflow` · `project`o `task`o `workflow`o `lang`o | `POST /workflow` | — | lenguaje natural |
| Crear proyecto | — | — | `mova init <nombre>` | — |
| Salud | — | `GET /health` | — | — |

## 5. Operaciones de archivos (secundarias; mismo escritor tras cada puerta)

| Función | Tool MCP · argumentos | HTTP | Chat |
|---|---|---|---|
| Crear/editar archivo o carpeta (formato por extensión) | `save` · `path`o `directory`o `content`o `overwrite`o `append`o `project`o `history`o `mode`o `range`o `code_only`o `text_only`o | `POST /save` | `/save [-c\|-d\|-append\|-overwrite\|-no-overwrite] "ruta"` |
| Borrar (exige `confirm:"true"`) | `delete_path` · `path`o `paths`o `project`o `confirm`o | `POST /delete` | `/delete "ruta" ["ruta2"…]` (pregunta Y/N c/u) |
| Leer texto / capa de documento | `read_file` · `filename`R `project`o — `read_document_layer` · `filename`R `project`o | `/mcp` | lenguaje natural |
| Edición quirúrgica | `patch_file` · `filename`R `search`R `replace`R `project`o | `/mcp` | lenguaje natural («corrige X en archivo») |
| Crear directorio | `create_directory` · `path`o `project`o | `/mcp` | lenguaje natural |
| Aplicar cambios propuestos por el modelo | `chat_completion` · `apply_edits:"true"` | `/mcp` | prompt interactivo `[s]/[1..N]/[n]` (`"apply"` en `project.json`) |
| Legado (preferir `save`) | `write_file` · `generate_word_contract` · `generate_pdf_document` · `generate_vector_graphic` · `generate_excel_report` · `trigger_diffusion_image` | `/mcp` | — |

Formatos de escritura: texto/config (`.txt .md .json .yml .xml .csv .toml .ini .env .log`), código
(`.js .ts .py .go .cs .java .php .rb .rs .c .cpp .h .kt .swift .sh`), web (`.html .css .sql`),
office (`.docx .xlsx .pdf`), media (`.svg .png`). Otro formato devuelve `Unsupported file type`.

## 6. Controles solo-chat

| Comando | Efecto |
|---|---|
| `/tasks` · `/task <nombre\|all>` | lista tareas · cambia de tarea y recarga contexto (conserva historial) |
| `/run <nombre>` | cambia de tarea y envía su variable `QUERY` |
| `/tools` · `/clear` · `exit`/`quit` | lista capacidades de archivos · limpia pantalla · salir |
| `set -model <nombre>` | cambia de modelo a mitad del chat, conserva historial |

## 7. Administración de modelos locales (solo CLI, por diseño)

`mova config <proveedor>` · `mova show config [modelo]` · `mova install a,b` · `mova model-list` · `mova remove a,b`.

## 8. Brechas de paridad (conocidas, verificadas en código)

- Archivar/borrar/configurar memoria: solo CLI. Administración de modelos e `init`: solo CLI (administrativo).
- `get_knowledge`, `get_memory*`, `list_projects`: sin comando slash en chat (usar CLI o lenguaje natural).
- Tools MCP heredadas (`write_file`, `generate_*`, `trigger_diffusion_image`) duplican `save` y quedan fuera del
  propósito de gobernanza; candidatas a eliminación (ver informe estratégico).
