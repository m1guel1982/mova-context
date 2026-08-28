# mova — Guía de comandos

> Docs: [Español](COMMANDS.md) · [English](COMMANDS.en.md)

`mova` busca `workflow.md` subiendo desde el directorio actual, así que funciona desde cualquier subcarpeta del repo. Si `projects/` tiene un único proyecto, `[proyecto]` es opcional. Convención: `[opcional]` · `<obligatorio>`.

## 1. Referencia rápida

| Comando | Propósito | Ejemplo |
|---|---|---|
| `run` | Ensambla el contexto (agents+skills+prompt+focus+memoria) | `mova run mi-proyecto revisar-auth` |
| `run --count` | Estima tokens/costo sin ejecutar | `mova run --count mi-proyecto` |
| `run --diagram` | Genera un diagrama visual del pipeline | `mova run mi-proyecto --diagram` |
| `budget` | Estima tokens/costo y escribe un reporte | `mova budget mi-proyecto --focus` |
| `chat` | Chat interactivo con un modelo local o Cloud | `mova chat mi-proyecto revisar-auth` |
| `memory` | Guarda la sesión en `memory.md` | `mova memory mi-proyecto "..."` |
| `memory-read` | Imprime la memoria activa | `mova memory-read mi-proyecto --all` |
| `memory-archive` | Archiva entradas antiguas | `mova memory-archive mi-proyecto --days 15` |
| `memory-clear` | Borra memoria (total o parcial) | `mova memory-clear mi-proyecto --yes` |
| `memory-config` | Configura retención/confirmación | `mova memory-config mi-proyecto days 45` |
| `list` | Lista todos los proyectos | `mova list` |
| `init` | Crea un proyecto nuevo | `mova init mi-proyecto` |
| `search` | Busca en agents/skills/prompts, sin modelo | `mova search "autenticación" software` |
| `config` | Fija el proveedor activo | `mova config ollama` |
| `show config` | Muestra proveedor/modelo activo | `mova show config llama3.1` |
| `install` | Descarga modelos de Ollama | `mova install llama3.1,mistral` |
| `model-list` / `remove` | Lista / elimina modelos instalados | `mova remove mistral` |
| `/save` | Crea/edita cualquier archivo o carpeta (chat) | `/save "informe.pdf"` |
| `/delete` | Elimina archivos/carpetas, con confirmación | `/delete "a.txt" "logs/"` |
| `jobs list` / `run` / `start` | Jobs programados por proyecto | `mova jobs run mi-proyecto 0` |
| `agents list` / `run` | Agentes de un grupo multiagente | `mova agents run ventas vendedor` |
| `mcp start` | Servidor MCP (stdio o HTTP) | `mova mcp start --stdio` |
| `ui` | Interfaz visual de terminal | `mova ui mi-proyecto` |

## 2. `mova run` — ensamblar el contexto

```
mova run [proyecto] [tarea] [--count] [--focus] [--diagram]
```
Une agents, skills, prompt, memoria y focus, e imprime el resultado por stdout — listo para pegar en un chat o mandar a una API.

`tarea` es opcional: usa `default_task` si existe, o la única task del proyecto si solo hay una. La validación de **Budget** corre siempre primero; si el contexto supera el límite configurado, se imprime solo el error, nada más:

```text
ERROR
Current context (128,400 tokens) exceeds the configured limit (100,000).
Suggestion: Use --focus to reduce the included files.
```

| Flag | Qué hace |
|---|---|
| `--count` | no imprime el contexto: solo estima tokens/costo (acepta también un grupo multiagente) |
| `--focus` | compara repo completo vs. solo lo que selecciona `focus` |
| `--diagram` | genera un diagrama visual en vez de ejecutar — [§18](#18-diagramas-visuales) |

```bash
mova run mi-proyecto revisar-auth
mova run --count mi-grupo              # suma todos los agentes del grupo
```

## 3. Focus — trabajar sobre una parte del proyecto

`focus` (en `project.json`, global o por task) limita el contexto a ciertos archivos, carpetas o símbolos. Es relativo al campo `"repo"`. Si `task.focus` está definido, reemplaza por completo el `focus` global.

Matchea primero por nombre exacto y, solo si eso no encuentra nada, por coincidencia parcial — insensible a mayúsculas/acentos, búsqueda de texto determinista, nunca un modelo.

| Item | Qué resuelve |
|---|---|
| `"manual.md"` | el archivo completo, por nombre |
| `"src/auth"` | índice del directorio |
| `"CreateOrder()"` | una función/método/clase (`()` = código) |
| `"Artículo 6"` | una sección de un documento legal |
| `"## Sección"` | un heading de Markdown |
| `"nombre_tabla"` | un `CREATE TABLE` en `.sql` |
| `"**/*"` | todo el repositorio |
| `"."` | solo la raíz del proyecto |
| `"src/", "pkg/"` | directorios específicos — forma habitual de excluir `node_modules`, `vendor`, `.git` |

Un item no encontrado aparece como `not found: [item]` — nunca se omite en silencio.

```json
"tasks": { "revisar-orden": { "focus": ["CreateOrder()", "manual.md", "Artículo 6"] } }
```

## 4. Memoria

```
mova memory         <proyecto> "<respuesta>"
mova memory-read    [proyecto] [--all | --month AAAA-MM]
mova memory-archive [proyecto] [--days N]
mova memory-clear   [proyecto] [--archived] [--keep-active] [--date F | --from F --to F] [--yes]
mova memory-config  [proyecto] <enable|disable|days N|confirm true|false>
```

`memory` extrae el bloque de memoria de la respuesta de un modelo y lo agrega a `memory.md`; la próxima corrida de `mova run` lo incluye solo.

| Comando | Flag | Qué hace |
|---|---|---|
| `memory-read` | `--all` / `--month` | histórico completo / un mes archivado |
| `memory-archive` | `--days N` | días a mantener activos (default 30) |
| `memory-clear` | `--archived` | borra solo lo archivado |
| `memory-clear` | `--keep-active` | borra archivos, conserva `memory.md` |
| `memory-clear` | `--date` / `--from`/`--to` | un día o rango de fechas |
| `memory-config` | `enable`/`disable` | archivado automático |
| `memory-config` | `days N` / `confirm` | retención / confirmación al borrar |

```bash
mova memory mi-proyecto "$(cat respuesta.txt)"
mova memory-clear mi-proyecto --archived --yes
```

## 5. Proyectos — list, init, search

```
mova list
mova init   [nombre]
mova search "<consulta>" [dominio]
```
`init` crea un `project.json` mínimo + `memory.md` vacío. `search` busca por palabra clave en agents/skills/prompts, sin usar ningún modelo.

```bash
mova init mi-proyecto
mova search "autenticación" software
```

## 6. Modelos y proveedores

```
mova config      <proveedor>
mova show        config [modelo]
mova install     modelo1,modelo2
mova model-list
mova remove      modelo1,modelo2
```

`llm_profile` (en `project.json`) es lo único que cambia entre modelos — agents, skills, prompts, memoria y focus nunca cambian.

```json
"llm_profile": {"config": "llama3.2.3b" }
```

| Campo | Valores | Para qué |
|---|---|---|
| `type` | `powerful` (default) \| `local` | `local` adapta el formato para que modelos chicos sigan mejor instrucciones secuenciales |
| `provider` | `ollama`, `google`, `anthropic`, `openai`, `lmstudio`... | subdirectorio de `config/models/` |
| `config` | nombre de archivo, sin `.json` | conexión + parámetros de inferencia juntos |

```json
"llm_profile": { "config": "claude-sonnet-4-6" }
"llm_profile": { "config": "gemini-2.5-flash" }
```

Editar un `.json` de modelo se recarga en caliente. `install`/`model-list`/`remove` usan la API de Ollama; con Cloud o LM Studio/vLLM, el modelo se instala aparte y solo se crea su `.json`.

```bash
mova install llama3.1,mistral,phi3
mova show config
```

**Endpoint remoto (Oracle Cloud, AWS, cualquier red privada):** `base_url`, dentro del `.json` del modelo, no tiene que ser `localhost` — puede apuntar a un servidor centralizado (ver [DEPLOY.md](../es/DEPLOY.md)) sobre una red privada (Tailscale/WireGuard):

```json
{ "base_url": "http://100.x.y.z:11434", "model": "llama3.2:3b" }
```

Nada más cambia: `project.json` sigue apuntando al mismo `llm_profile.config`, y el pipeline de sanitización/PII Masking sigue corriendo siempre en la máquina local, nunca en el servidor remoto — ver [PROJECT_JSON.md § Arquitectura distribuida](PROJECT_JSON.md#arquitectura-distribuida-endpoints-remotos). Ejemplo listo para probar: `config/models/ollama/llama3.2.3b-remote.json`.

## 7. Chat — `mova chat`

```
mova chat [proyecto] [tarea]
```
Chat interactivo con un modelo local o Cloud. Con `[proyecto]`, carga el mismo contexto completo que arma `mova run` como mensaje de sistema.

| Comando en chat | Qué hace |
|---|---|
| `set -model <nombre>` | cambia de modelo sin perder el historial |
| `/memory` | guarda el último intercambio en `memory.md` |
| `/budget` | genera `mova-budget-report.md` |
| `/diagram [export] [ruta]` | genera un diagrama visual |
| `/save "ruta"` / `-d "carpeta"` / `-c "ruta"` | guarda respuesta / crea carpeta / solo código |
| `/tools` | lista comandos y tools disponibles |
| `exit` / `quit` / `salir` | termina la sesión |

```bash
mova chat
> set -model llama3.1
> hola, revisa el módulo de auth
```

También puedes pedir cosas con tus propias palabras, sin comandos: *"Genera el informe de auditoría en docs/auditoria.pdf"* crea el archivo directo — funciona en español e inglés, en la misma sesión.

## 8. `save` / `/save` — crear o editar archivos

```
/save ["ruta/archivo.ext"]
/save -d "ruta/carpeta"
/save [-all | -range N-M] [-c | -text] ["ruta"]
/save [-overwrite | -no-overwrite | -append] "ruta"
```
Un único punto de entrada: se indica `path` (o `directory`) + contenido, y `mova` elige el generador correcto según la extensión.

| Flag | Guarda... |
|---|---|
| *(ninguno)* | la última respuesta del modelo |
| `-all` | toda la conversación, como transcripción |
| `-range N-M` | los intercambios N a M (1-indexado, inclusive) |
| `-c` / `-text` | solo bloques de código / solo texto |

| Flag | Archivo existente |
|---|---|
| *(ninguno)* / `-overwrite` | sobreescribe |
| `-no-overwrite` | falla en vez de sobreescribir |
| `-append` | agrega al final |

Se combinan libremente: `/save -all -c "snippets.go"`. Lenguaje natural equivalente: *"Sobreescribe reporte.pdf"* = `-overwrite`; *"No sobreescribas reporte.pdf"* = `-no-overwrite`.

**Vía MCP/HTTP** — misma tool `save`, con `path`/`directory` + `content`, o `history` + `mode` (`all`/`range`/último intercambio):
```bash
curl -X POST http://localhost:3000/mcp -d '{"jsonrpc":"2.0","id":1,"method":"tools/call",
  "params":{"name":"save","arguments":{"project":"mi-proyecto","path":"informes/checkout.pdf","content":"Hallazgo 1: ..."}}}'
```

| Extensión | Cómo se interpreta `content` |
|---|---|
| `.md`, `.txt`, `.json`, `.yml`, `.csv`, código | tal cual |
| `.docx` | Markdown (`#`, `##`, negrita, párrafos) |
| `.pdf` | HTML tal cual, o texto envuelto en `<p>` |
| `.xlsx` | `sheets_data` JSON tipado, o CSV/TSV plano |
| `.svg` | código SVG válido |

## 9. `delete` / `/delete`

```
/delete "ruta1" ["ruta2" ...]
```
Elimina archivos y directorios, siempre con confirmación previa. Una `/` final (`"logs/"`) indica directorio; sin ella, `mova` consulta el sistema de archivos.

```text
> /delete "a.txt" "logs/"
Delete "a.txt"? (Y/N)
```

Vía MCP/HTTP: se llama una vez sin `confirm` para recibir el prompt, y de nuevo con `confirm:true` para ejecutar.

## 10. `workflow.md`

```
lee workflow.md
workflow.md <proyecto> [tarea]
```
Nunca se abre directamente: primero resuelve el proyecto, arma su contexto y lo valida contra el Budget — solo si pasa, se carga.

| Dónde | Qué decir/enviar |
|---|---|
| `mova chat` | `workflow.md mi-proyecto` |
| Cliente MCP | pedirle que lea el workflow — llama solo a `get_workflow` |
| HTTP | `curl -X POST localhost:3000/workflow -d '{"project":"mi-proyecto"}'` |

Si el contexto supera el Budget, el archivo no se carga.

## 11. Tool-calling autónomo desde el chat

```json
{ "tools": { "enabled": true, "allow": ["save", "read_file"] } }
```
Con `tools.enabled`, el propio modelo puede pedirle a `mova` que ejecute una acción real (guardar, leer, editar) durante la conversación, en cualquier proveedor. `allow` restringe a un subconjunto (`save`, `read_file`, `patch_file`, `read_document_layer`); sin `allow`, las cuatro quedan habilitadas. Es adicional a `/save`, que siempre funciona sin depender de esto.

El protocolo es texto plano, no la API nativa de function-calling de cada proveedor — un mismo formato (`<<<MOVA_TOOL_CALL>>>{"name":...,"arguments":...}<<<END_MOVA_TOOL_CALL>>>`) para Ollama, Claude, GPT o Gemini por igual. Modelos locales chicos (por ejemplo `lfm2.5-1.2b`) a veces "ecoan" ese mismo formato como si fuera su respuesta en vez de un llamado real; `mova` detecta y descarta ese residuo automáticamente antes de mostrar la respuesta — en CLI, Chat, MCP y HTTP por igual — así que nunca aparece un encabezado JSON crudo donde debería ir la respuesta del modelo.

## 12. Documentos, medios, texto y código

`save` elige el generador correcto según la extensión — no hace falta recordar qué tool usa cada formato. `.docx`, `.xlsx`, `.pdf` no requieren paquetes adicionales.

```text
> Genera un contrato simple de arriendo en salida/contrato.docx
> Arma una tabla con los gastos del mes en salida/gastos.xlsx
```

Imágenes (`trigger_diffusion_image`): enruta el prompt a un servidor de difusión local compatible con AUTOMATIC1111 — requiere ese servidor corriendo aparte, con un modelo instalado.

Texto y código (`.js`, `.py`, `.go`, `.sql`, `.yaml`... ver [SUPPORTED_FORMATS.md](SUPPORTED_FORMATS.md)):

| Tool | Para qué |
|---|---|
| `read_file` / `write_file` | leer / escribir un archivo |
| `patch_file` | reemplazar una única aparición exacta de `search` por `replace` |

`write_file`/`save` validan `.json`/`.xml` bien formados, `.go` con sintaxis real y `.csv` con columnas consistentes antes de escribir. `patch_file` rechaza el cambio si `search` no aparece, o aparece más de una vez.

## 13. Servidor MCP — `mova mcp start`

```
mova mcp start --stdio
mova mcp start --port 3000
```
Expone el motor de `mova run` por MCP (JSON-RPC 2.0), para que un cliente (Claude Desktop, Cursor) pida el contexto sin copiar y pegar.

```json
{ "mcpServers": { "mova-context": {
  "command": "/ruta/a/mova", "args": ["mcp", "start", "--stdio"],
  "env": { "MOVA_PROJECT_ROOT": "/ruta/a/tu/mova-context" }
}}}
```

| Tool | Equivale a |
|---|---|
| `get_full_context` | `mova run [proyecto] [tarea]` |
| `get_knowledge` | leer un agent/skill/prompt puntual |
| `get_memory` / `get_memory_all` | `mova memory-read` |
| `get_workflow` | leer `workflow.md`, validando Budget primero |
| `search_context` | `mova search` |
| `chat_completion` | `mova chat` |
| `save` / `delete_path` | crear/editar/eliminar archivos |
| `estimate_budget` | `mova budget` |
| `generate_diagram` | `mova run --diagram` |
| `list_jobs` / `run_job` | `mova jobs list` / `run` |
| `list_agents` / `run_agent` | `mova agents list` / `run` |

Orden de resolución de raíz: `MOVA_PROJECT_PATH` → `MOVA_PROJECT_ROOT` → directorio de trabajo actual → directorio del binario.

## 14. Variables de entorno

| Variable | Efecto |
|---|---|
| `MOVA_DSN` | sobreescribe `project.json.dsn` |
| `MOVA_PROJECT_ROOT` | punto de partida extra para buscar `workflow.md` hacia arriba |
| `MOVA_PROJECT_PATH` | usa esta ruta como raíz, sin búsqueda |

`"repo"` en `project.json` acepta cualquier ruta absoluta — otra unidad de Windows, un punto de montaje de Linux, una ruta UNC, WSL, o un volumen de Docker. `MOVA_PROJECT_ROOT` es lo único que hace falta fijar aparte, para que `mova` encuentre su propia raíz al pararte fuera de ella.

## 15. Tokenomics — `mova budget`

```
mova budget [proyecto] [tarea] [--focus]
```
Estima tokens y costo del contexto real de un proyecto por proveedor. Todo el cálculo es local — nunca llama a un LLM ni envía contenido fuera de tu máquina. Genera `mova-budget-report.md`.

El reporte incluye: tokenización y deduplicación automática, desglose por Agents/Skills/Prompt/Focus/Memory, comparación repo completo vs. focus (con `--focus`), el límite configurado, y precisión histórica frente a proveedores Cloud reales (`mova-token-history.json`).

| | `budget.max_tokens` | `num_predict` |
|---|---|---|
| Limita | el contexto enviado al modelo | la respuesta del modelo |
| Lo aplica | `mova`, antes de enviar | el proveedor |
| Al superarse | corta la ejecución, cero tokens gastados | el proveedor corta la respuesta |

```json
{ "budget": { "max_tokens": 6000 } }
```

Precios en `config/prices.json` (global, recarga en caliente):
```json
{ "providers": { "google": { "models": { "gemini-2.5-flash": { "input": 0.0003, "output": 0.0025 } } } } }
```

```bash
mova budget mi-proyecto mi-task --focus
```

## 16. Token Firewall

Etapas determinísticas y sin IA que corren automáticamente en cada ensamblado de contexto — `run`, `chat`, `jobs`, `agents`, MCP:

| Etapa | Qué hace | Default |
|---|---|---|
| **Sanitizer** | colapsa ruido repetitivo (logs casi idénticos, líneas en blanco, headers duplicados) | activado |
| **PII Masking** | pseudonimiza tokens con forma de dato personal antes de enviarlos | **desactivado** |
| **Cache Layout Guard** | ordena el prompt en prefijo estable + contenido variable, para el prompt caching del proveedor | activado |
| **Circuit Breaker** | detiene antes de enviar si se supera `max_tokens_per_run` o `max_monthly_usd` | activado |

Cada etapa se activa/desactiva por separado en `"budget"` de `project.json`.

```json
{ "budget": { "pii_masking": { "enabled": true },
  "circuit_breaker": { "max_tokens_per_run": 5000, "max_monthly_usd": 5.0, "on_exceed": "abort" } } }
```

El reporte de `mova budget` siempre compara tokens/costo con el Token Firewall aplicado vs. sin él.

## 17. Diagramas visuales — `mova run --diagram`

```
mova run <proyecto> --diagram [--export svg,png,pdf] [--path ./carpeta]
```
Genera una imagen real del pipeline de un proyecto o grupo multiagente: fuentes → Context Compiler → Token Firewall → agentes → jobs → resumen final con tokens/costo reales.

| Flag | Default | Qué hace |
|---|---|---|
| `--export` | `svg` | lista separada por comas: `svg`, `png`, `pdf` |
| `--path` | directorio actual | carpeta de salida, se crea si no existe |

`<proyecto>` puede ser un proyecto normal, un grupo multiagente completo, o un agente puntual (`<grupo>/<agente>`).

```bash
mova run mi-grupo --diagram --export svg,png,pdf --path ./diagramas
```

| Canal | Cómo |
|---|---|
| Chat | `/diagram`, `/diagram png`, `/diagram svg,png ./salida` |
| MCP | tool `generate_diagram` con `{"project":"..."}` |
| HTTP | `curl -X POST localhost:3000/diagram -d '{"project":"..."}'` |

## 18. Jobs — ejecución programada

```
mova jobs list  [proyecto]
mova jobs run   [proyecto] [índice|--all]
mova jobs start
```
Lee el array `jobs` de `project.json` y lo ejecuta — mismo motor sin importar el canal. `jobs start` inicia el daemon del scheduler, revisando cada proyecto una vez por minuto.

```bash
mova jobs run mi-proyecto            # todos, ignora el cron
mova jobs run mi-proyecto 0          # solo el job en el índice 0
```

Con un grupo multiagente, `[índice]` se interpreta como nombre de agente; para un job por índice dentro de un solo agente: `mova jobs run grupo/agente 0`.

## 19. Multiagente — grupos de agentes

```
mova agents list <grupo>
mova agents run  <grupo> [agente|--all]
```
Un grupo vive bajo `projects/`, cada agente es un proyecto normal, orquestado por un `config.json` padre.

```bash
mova agents run ventas_online              # todos, en secuencia
mova agents run ventas_online vendedor     # = mova run ventas_online/vendedor
mova chat ventas_online vendedor           # = mova chat ventas_online/vendedor
```

`mova chat <grupo>` (sin agente) lista los agentes disponibles en vez de abrir un chat.

## 20. Interfaz visual — `mova ui`

```
mova ui [proyecto]
```
Interfaz de terminal que agrupa todo lo que hacen los comandos de esta guía detrás de una navegación por menús — no reemplaza ningún comando, llama a los mismos componentes internos.

`↑`/`↓` mover · `enter` abrir/confirmar/correr · `/` buscar en lista · `ctrl+f` buscar en el documento abierto · `esc` volver · `ctrl+s` guardar · `ctrl+c` salir.

**Menú principal:** Chat · Proyectos · Workflow.md · Modelos · Logging · Multiagentes · Search · Logs. Dentro del chat de la TUI funcionan los mismos comandos que en `mova chat`.

```bash
mova ui mi-grupo/mi-agente
```

## 21. Instalación

```
make install
go build -o mova ./src/cli
```
`make install` compila y copia el binario a `$(go env GOPATH)/bin/mova`. Con esa carpeta en el `PATH`, `mova` corre desde cualquier directorio — siempre busca `workflow.md` hacia arriba (o usa `MOVA_PROJECT_ROOT`/`MOVA_PROJECT_PATH`).

Para instalar con interfaz gráfica, sin usar la terminal: [`installers/README.md`](../../installers/README.md) — doble clic en Windows, macOS y Linux, compatible con `make install`.