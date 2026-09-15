# Mova Context — Referencia técnica de arquitectura

**Un binario. Un comportamiento.** `mova` es un único ejecutable Go. Toda capacidad se alcanza igual desde
cuatro puertas — **CLI**, **Chat (REPL)**, **MCP** (stdio/HTTP) y **HTTP REST** — porque las cuatro llaman
exactamente a la misma función interna. Nunca hay una segunda implementación de la lógica de negocio por puerta.

## 1. Mapa del sistema

```text
src/
├── core/          Motor — sin dependencias externas (solo stdlib)
│   └── focus/       Motor de resolución de Focus/Exclude (incl. AST, §3)
├── dedup/         Deduplicación exacta de párrafos
├── adapters/      Backends de almacenamiento alternativos (Postgres/MongoDB)
├── documents/     Servicio de guardado · formatos Office (docx/xlsx/pdf/svg)
├── sanitize/      Gobernanza de Contexto: Sanitizer, detección de secretos, PII Masking
├── budget/        Estimación de tokens/costo, circuit breaker, caché de contexto
├── models/        Proveedores de LLM — locales y cloud, una sola fuente de verdad por modelo
├── orchestrator/  Orquestador multiagente — grupos de proyectos como agentes
├── diagram/       Renderiza el pipeline real de un proyecto como SVG/PNG/PDF
├── trace/         Motor de `context-trace` — auditoría de contexto/presupuesto/gobernanza (§4)
├── cli/           Comando `mova` — dispatcher delgado (CLI + REPL `mova chat`)
├── mcp/           Capa JSON-RPC de MCP — tools/list, tools/call
├── http/          Transporte HTTP — envoltorio delgado sobre mcp.Process()
└── runtime/       FindRoot()/AutoDetect() — bootstrapping compartido
```

**Fuera de alcance, ya retirado:** `jobs/` (scheduler cron) y la TUI de terminal (`mova ui`) — recortados para
especializarse en gobernanza pre-inferencia en vez de ser una herramienta "hace de todo". Si ves notas viejas
que las mencionan, ya no existen en este código.

## 2. Pipeline de ejecución del núcleo

```
core.BuildContext(adapter, root, proyecto, tarea)
    1. Adapter.GetProject(name)        — lee project.json, siempre fresco
    2. ResolveTaskName(...)            — explícita → default_task → única tarea → ""
    3. resuelve agents/skills/prompt   — dominio + i18n/[lang] + fallback "en"
    4. inyecta variables               — nivel proyecto, luego overrides de tarea
    5. agrega memory.md (si existe)
    6. resuelve `focus`/`exclude`      — incl. sintaxis AST (`archivo::kind=nombre`)
    7. dedup.Paragraphs (Agents→Skills→Prompt→Focus→Memory)
    ▼
contexto final (string)
```

`mova run`, `get_full_context` (MCP/HTTP) y `mova chat` llaman **exactamente a esta función** — nunca hay
un segundo camino de código.

## 3. Focus / Exclude — con AST en ambos (novedad)

`project.json`'s `"focus"` y `"exclude"` son listas de objetivos (archivos, directorios, globs, o
`archivo::kind=símbolo` vía Tree-Sitter). Antes solo `focus` soportaba AST; ahora `exclude` también
(`ApplyAstSymbolExcludes`, `core/focus/resolvers/ast_symbol.go`) — permite analizar un archivo completo
dejando fuera solo una función/símbolo puntual (ver ejemplo `03-tokenomics-context-trace`).

## 4. `context-trace` — motor de auditoría pre-inferencia

`src/trace/` es el único motor detrás de `mova context-trace`/`mova trace` (CLI), `/context-trace` (Chat),
la tool MCP `context_trace`, y `POST /api/v1/context-trace` (HTTP).

**Dos modos, dos caminos de conteo de tokens:**
- **Modo proyecto** (`AnalyzeLocal`) — nunca cuenta tokens él mismo; llama `budget.BuildReport`, la misma
  función que usa `mova budget`.
- **Modo discovery** (`AnalyzeRemote`, repo remoto sin `project.json`) — cuenta tokens archivo por archivo
  para poder armar el desglose "tokens por directorio".

**Identidad de auditoría** (nuevo — responde las preguntas #3, #10, #11 de la Matriz, ver `README.md`):
`Data.AgentClient` / `Data.TargetModel` / `Data.PolicyAuthor`, calculados en `trace.applyAuditIdentity` y
`core.ResolvePolicyAuthor`/`core.TargetModelFor`. Nunca quedan vacíos — ver `docs/i18n/es/ARTIFACTS.md`.

## 5. Transportes — mismo motor, distinta puerta

CLI, Chat, MCP (stdio/HTTP) y HTTP REST llaman todos a `mcp.Process()` o a las mismas funciones de `core`/
`trace`/`budget`. La única diferencia entre puertas es **quién** llama — ver `AgentClient` arriba.

## 6. Diagramas

`mova run <proyecto> --diagram [--export svg,png,pdf]` (`diagram/build.go` + `svg.go`) renderiza el pipeline
real de un proyecto a partir de su `project.json` — nada inventado. Acepta `--path <archivo>.<formato>`
como destino completo (no solo un directorio) cuando se pide un único formato.

## 7. Extensibilidad

Cuatro puntos de extensión Open/Closed: **Adapters** (`core.Adapter`, en `adapters/`), **Focus Resolvers**
(`core/focus/resolvers/`), **Model Providers** (`models/`, interfaz `Provider`), **Save Writers**
(`documents/`, `RegisterWriter(".ext", ...)`). Cada uno: implementar una interfaz, registrar, nada más
cambia.

## 8. Deliberadamente fuera de alcance

Sin compilador propio, sin nivel de licenciamiento, sin build tags `premium`. `focus`/`exclude` es la
implementación completa y permanente. Cada capacidad es aditiva a nivel `project.json`.
