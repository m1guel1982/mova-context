# mova — Comandos

Estilo manual: `NAME`, `SYNOPSIS`, `EXAMPLES`. Cada comando se entiende en 15 segundos.

## run

**NAME** — ensambla el contexto de un proyecto (Agents+Skills+Prompt+Focus+Memory) y opcionalmente genera evidencia visual.

**SYNOPSIS** — `mova run <proyecto> [tarea] [--diagram --export svg,png,pdf --path <dir|archivo.ext>]`

**EXAMPLES**
```bash
mova run 02-pii-compliance-governance
mova run 02-pii-compliance-governance --diagram --export png --path ./evidencia.png
```

## context-trace (alias: trace)

**NAME** — audita el contexto antes de la inferencia: qué se seleccionó, qué se sanitizó, tokens, costo, identidad de auditoría.

**SYNOPSIS** — `mova context-trace <proyecto> [--export md,pdf] [--diagram] [--policies_include <lista>] [--policies_exclude <lista>]` · `mova context-trace --repo <url> [--task <texto>] [--ignore <patrones>]`

**EXAMPLES**
```bash
mova context-trace 02-pii-compliance-governance --export md
mova trace --repo https://github.com/usuario/repo --task "revisar login" --ignore "docs/**,*.lock"
```

**POLICIES** — `--policies_include` / `--policies_exclude` aceptan una lista separada por comas, mezclando nombres simples (búsqueda recursiva en `config/policy/`) y rutas completas multiplataforma. Tienen precedencia sobre `project.json` y `config/policy.json` — ver `PROJECT_JSON.md § policies`.

```bash
mova context-trace 02-pii-compliance-governance \
  --policies_include "security.json,C:\custom\pii_strict_ventas.json" \
  --policies_exclude "pii_permissive.json"
```

## budget

**NAME** — desglose de tokens/costo por proveedor para la tarea activa de un proyecto; genera `mova-budget-report.md`.

**SYNOPSIS** — `mova budget <proyecto> [tarea] [--focus]`

## mcp start

**NAME** — levanta el servidor MCP para que un agente (Claude Code, Cursor, Windsurf) se conecte y llame sus tools (`context_trace`, `get_full_context`, `chat_completion`, etc.) — ver más abajo la lista completa.

**SYNOPSIS** — `mova mcp start [--stdio] [--port 3000]`

`--stdio` levanta el servidor por entrada/salida estándar (lo que usan Claude Code/Cursor/Windsurf al conectarse). Sin `--stdio`, levanta HTTP en el puerto indicado (por defecto `3000`) — ver sección **HTTP** más abajo.

**EXAMPLES**
```bash
mova mcp start --stdio
mova mcp start --port 3000
```

```json
{ "mcpServers": { "mova": { "command": "mova", "args": ["mcp", "start", "--stdio"] } } }
```

### Funciones MCP disponibles (`tools/call`)

| Tool | Para qué |
|---|---|
| `list_projects` | Lista los proyectos del registro de Mova. |
| `get_full_context` | Contexto completo ensamblado (= `mova run`): agents+skills+prompt+memory+focus. |
| `get_knowledge` | Un agente, skill o prompt puntual. |
| `get_memory` / `get_memory_all` | Memoria activa / activa+archivada de un proyecto. |
| `save_memory` | Agrega una entrada a `memory.md`. |
| `get_workflow` | Lee `workflow.md`, pero solo después de resolver el proyecto y validar su presupuesto. |
| `search_context` | Busca en todo el conocimiento (agents/skills/prompts). |
| `chat_completion` | Envía un mensaje a un modelo local/cloud, con el contexto de Mova como system prompt. **Si `project.json` declara `egress_audit`, esta es la tool afectada** — ver `PROJECT_JSON.md § egress_audit`: puede auditar el contexto saliente y/o cortar antes de llegar al proveedor (dry-run). |
| `estimate_budget` | Estima tokens/costo USD del contexto real de un proyecto; escribe `mova-budget-report.md`. |
| `generate_diagram` | Diagrama visual del pipeline real (SVG/PNG/PDF). |
| `context_trace` | Auditoría pre-inferencia — ver `context-trace.md`. |
| `list_agents` / `run_agent` | Listar/ejecutar un grupo multiagente. |
| `save` / `delete_path` / `create_directory` / `read_file` / `read_document_layer` | Gestión unificada de archivos/directorios. |

## HTTP

**NAME** — el mismo servidor MCP expuesto sobre HTTP — pensado para Postman/curl/integraciones sin stdio.

**SYNOPSIS** — `mova mcp start --port <puerto>` (sin `--stdio`)

### Endpoints HTTP disponibles

| Endpoint | Método | Qué hace |
|---|---|---|
| `/mcp` | `POST` | JSON-RPC 2.0 genérico — `{"method":"tools/call","params":{"name":"<tool>","arguments":{...}}}`. Todas las tools de la tabla de arriba, incluida `chat_completion` (y por lo tanto `egress_audit`), pasan por acá. |
| `/save` | `POST` | Atajo directo a la tool `save`. |
| `/delete` | `POST` | Atajo directo a `delete_path`. |
| `/workflow` | `POST` | Atajo directo a `get_workflow`. |
| `/agents/run` | `POST` | Atajo directo a `run_agent`. |
| `/diagram` | `POST` | Atajo directo a `generate_diagram`. |
| `/api/v1/context-trace` | `POST` | Atajo directo a `context_trace`. |
| `/health` | `GET` | Chequeo de vida del servidor. |

**EXAMPLES**
```bash
curl -X POST http://localhost:3000/mcp \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"chat_completion","arguments":{"project":"02-pii-compliance-governance","message":"resume el contexto"}}}'
```

## list / init / search

**NAME** — `list`: proyectos disponibles · `init <nombre>`: crea `projects/<nombre>/project.json` mínimo · `search <query>`: busca en agentes/skills/prompts.

## config / show / install / model-list / remove

**NAME** — gestión de modelos: `config` (perfil activo) · `show <proveedor/modelo>` · `install <proveedor/modelo>` · `model-list` · `remove <proveedor/modelo>`.

## chat

**NAME** — REPL interactivo. Dentro: `/context-trace`, `/memory`, `/save`, `/delete`.

**SYNOPSIS** — `mova chat <proyecto>`

## memory / memory-read / memory-archive / memory-clear / memory-config

**NAME** — gestión de `memory.md` de un proyecto (ver `docs/i18n/es/ARTIFACTS.md`).

## agents list / agents run

**NAME** — orquestador multiagente: `list` (grupos definidos) · `run <grupo>` (ejecuta cada proyecto miembro).

---
Ver `docs/i18n/es/COMMANDS_ADVANCED.md` para variables de entorno y flags avanzados.
