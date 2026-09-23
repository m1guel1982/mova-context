# Herramientas MCP y HTTP — referencia copiar-pegar

Todas las herramientas MCP se llaman **igual** por stdio o por HTTP — mismo `name`, mismos `arguments`. Plantilla única:

```json
{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"<tool>","arguments":{...}}}
```

**Por MCP (stdio):**
```bash
echo '<JSON de arriba>' | mova mcp start --stdio
```

**Por HTTP:**
```bash
curl -X POST http://localhost:3000/mcp -H "Content-Type: application/json" -d '<JSON de arriba>'
```

De acá en adelante, cada fila muestra solo el `arguments` — pegalo en la plantilla de arriba, cambiando `name`.

**🔒 = respeta `egress_audit.dry_run`** (ver `PROJECT_JSON.md § Air-gap real`) — con `dry_run: true` en el `project.json` de ese proyecto, estas 7 herramientas nunca devuelven el contenido real, solo el mensaje de auditoría. `search_context` no tiene `project` y busca en el catálogo compartido de Mova, no en datos privados — por diseño, no está gateada.

## Contexto y gobernanza (el núcleo de Mova)

| Tool | Qué hace | `arguments` de ejemplo |
|---|---|---|
| `context_trace` | Auditoría pre-inferencia — tokens, costo, políticas, decisión. Ver `GOVERNANCE_CONTROLS.md`. | `{"project": "02-pii-compliance-governance"}` |
| `get_full_context` 🔒 | Contexto completo ensamblado (= `mova run`). Respeta `egress_audit.dry_run`. | `{"project": "02-pii-compliance-governance"}` |
| `chat_completion` 🔒 | Envía un mensaje a un modelo local, con el contexto como system prompt. Respeta `egress_audit.dry_run` y delega al LLM anfitrión si no hay `llm_profile`. | `{"project": "02-pii-compliance-governance", "message": "resume el contexto"}` |
| `estimate_budget` | Estima tokens/costo USD sin llamar a ningún modelo — escribe `mova-budget-report.md`. | `{"project": "02-pii-compliance-governance"}` |
| `generate_diagram` | Diagrama visual del pipeline real (SVG/PNG/PDF). | `{"project": "02-pii-compliance-governance", "export": "png"}` |

## Proyectos y conocimiento

| Tool | Qué hace | `arguments` de ejemplo |
|---|---|---|
| `list_projects` | Lista los proyectos registrados. | `{}` |
| `get_knowledge` | Un agente/skill/prompt puntual. | `{"kind": "agent", "domain": "base", "name": "backend-dev"}` |
| `search_context` | Busca en todo el conocimiento. | `{"query": "PII"}` |
| `get_workflow` 🔒 | Lee `workflow.md` tras validar presupuesto. | `{"project": "02-pii-compliance-governance"}` |

## Memoria

| Tool | Qué hace | `arguments` de ejemplo |
|---|---|---|
| `get_memory` 🔒 | Memoria activa de un proyecto. | `{"project": "02-pii-compliance-governance"}` |
| `get_memory_all` 🔒 | Memoria activa + archivada. | `{"project": "02-pii-compliance-governance"}` |
| `save_memory` | Agrega una entrada. | `{"project": "02-pii-compliance-governance", "entry": "decisión: usar Ley 21.719 como referencia"}` |

## Multiagente

| Tool | Qué hace | `arguments` de ejemplo |
|---|---|---|
| `list_agents` | Lista agentes de un grupo. | `{"group": "mi-grupo"}` |
| `run_agent` | Corre uno o todos los agentes del grupo. | `{"group": "mi-grupo"}` |

## Archivos (unificadas — usa `save` para todo lo nuevo)

| Tool | Qué hace | `arguments` de ejemplo |
|---|---|---|
| `save` | Crea/edita cualquier archivo (.md, .docx, .pdf, .xlsx, .svg, código) — el formato lo decide la extensión. | `{"path": "notas.md", "content": "# Hola"}` |
| `delete_path` | Elimina (pide `confirm: true`). | `{"path": "notas.md", "confirm": true}` |
| `create_directory` | Crea un directorio. | `{"path": "nueva-carpeta"}` |
| `read_document_layer` 🔒 | Extrae texto de .docx/.xlsx/.pdf. | `{"project": "02-pii-compliance-governance", "filename": "informe.pdf"}` |
| `read_file` 🔒 | Lee un archivo de texto. | `{"project": "02-pii-compliance-governance", "filename": "notas.md"}` |
| `patch_file` | Reemplaza un fragmento exacto. | `{"filename": "notas.md", "search": "Hola", "replace": "Hola mundo"}` |
| `write_file` *(legacy, usa `save`)* | Crea/sobrescribe texto plano. | `{"filename": "notas.txt", "content": "hola"}` |
| `generate_word_contract` *(legacy, usa `save`)* | Markdown → .docx. | `{"filename": "contrato.docx", "markdown_content": "# Contrato"}` |
| `generate_pdf_document` *(legacy, usa `save`)* | HTML/CSS → .pdf. | `{"filename": "reporte.pdf", "layout_html_css": "<h1>Reporte</h1>"}` |
| `generate_vector_graphic` *(legacy, usa `save`)* | SVG → .svg. | `{"filename": "diagrama.svg", "svg_code": "<svg></svg>"}` |
| `generate_excel_report` *(legacy, usa `save`)* | JSON tabular → .xlsx. | `{"filename": "datos.xlsx", "sheets_data": {"Hoja1": [["A","B"]]}}` |
| `trigger_diffusion_image` | Genera una imagen (si hay proveedor configurado). | `{"prompt": "diagrama de arquitectura minimalista"}` |

## HTTP — endpoints directos (atajos, sin envolver en `tools/call`)

| Endpoint | Método | Atajo de |
|---|---|---|
| `/mcp` | `POST` | Genérico — cualquier tool de arriba |
| `/save` | `POST` | `save` |
| `/delete` | `POST` | `delete_path` |
| `/workflow` | `POST` | `get_workflow` |
| `/agents/run` | `POST` | `run_agent` |
| `/diagram` | `POST` | `generate_diagram` |
| `/api/v1/context-trace` | `POST` | `context_trace` |
| `/health` | `GET` | chequeo de vida (sin body) |

```bash
curl -X POST http://localhost:3000/save -H "Content-Type: application/json" \
  -d '{"path":"notas.md","content":"# Hola"}'
```

Guía paso a paso para probar todo esto gratis, sin GPU: `MCP_HTTP_TESTING.md`.
