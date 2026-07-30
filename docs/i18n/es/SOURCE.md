# SOURCE — estructura del código (referencia en español)

Un solo binario (`mova`), sin build tags, sin ediciones. Este documento es la contraparte en español de [`docs/SOURCE.md`](../../SOURCE.md) (la referencia completa vive ahí, en inglés); acá se repite la estructura general y se documenta en detalle, en español, el conjunto de cambios más reciente: eliminación unificada, `/save` extendido, resaltado de sintaxis, `workflow.md` con validación de Budget, y la configuración de rutas en `project.json`.

## Layout (resumen)

```text
src/
├── core/                  el motor — sin dependencias externas
│   ├── types.go            Project, Task — structs compartidas (reflejan project.json); repo/workflow_path/budget_path/token_history_path son strings únicos
│   ├── engine.go           BuildContext()/BuildContextSections() — ensambla agents+skills+prompt+memory+focus; ResolveTaskName() elige la task (explícita → default_task → la única task del proyecto si solo declara una) — el único lugar donde se toma esa decisión
│   └── focus/              motor de resolución de Focus (GlobResolver, DirectoryResolver, FileResolver, ...)
│
├── dedup/                 deduplicación exacta de párrafos — compartida por core y core/focus/render
│
├── adapters/              backends de storage alternativos (Postgres/MongoDB)
│
├── documents/              formatos de oficina y media — leer/generar PDF, DOCX, XLSX, SVG, imágenes por difusión
│   ├── save_service.go      Save()/SaveRequest — ÚNICO punto de entrada detrás de `/save`, el chat, y la tool `save` de MCP
│   ├── save_selection.go    SelectContent()/GroupExchanges()/TranscriptText()/ExtractCodeBlocks()/StripCodeBlocks()/ParseRangeToken() — ÚNICA implementación detrás de los modos actual/`-all`/`-range N-M`/`-c`/`-text` de `/save`
│   ├── delete_service.go    Delete()/DeleteRequest/DeleteResult/FormatDeletePrompt — ÚNICO punto de entrada detrás de `/delete`, la tool `delete_path` de MCP, y `POST /delete`
│   ├── highlight.go         DetectLanguage()/AutoTagCodeFences() — ÚNICA implementación de detección de lenguaje/auto-etiquetado, compartida por el chat y por `chat_completion` (y por lo tanto HTTP)
│   ├── pathresolve.go       resolución de rutas cross-platform, con desambiguación
│   └── (resto: nl_intent.go, edit_intent.go, docx.go, xlsx.go, pdf.go, svg.go, ... sin cambios)
│
├── budget/                estimación de tokens/costo — `mova budget`, 100% local, sin llamar a ningún modelo
│   ├── limit.go             CheckLimit()/EnforceLimit() — el gate duro de "budget": {"max_tokens": N}; ResolveTask() siempre devuelve un *Task no nulo (uno real, o un Task{} vacío) para que el gate nunca se salte cuando no hay task nombrada
│   ├── prices.go            PricesConfig — lee config/prices.json (solo precios de modelos, ver "Cambios recientes")
│   ├── history.go           mova-token-history.json — HistoryPath() resuelve la ruta desde "token_history_path" (un único string)
│   ├── report.go            WriteReport()/BudgetReportPath() — mova-budget-report.md, ruta resuelta desde "budget_path" en project.json (un único string)
│   └── workflow.go          LoadWorkflow() — el pipeline con validación de Budget detrás de "lee/ejecuta workflow.md"
│
├── models/                proveedores de LLM locales + Cloud (Ollama, LM Studio, vLLM, OpenAI, Anthropic, Google...)
├── cli/                   `mova` — comandos + REPL de chat (incluye delete_cmd.go, workflow_cmd.go, chat_save.go)
├── mcp/                   servidor MCP (stdio y HTTP) — mismo motor, mismas tools, sin importar el transporte
├── http/                  wrapper HTTP fino sobre MCP — sin lógica propia
└── runtime/               detección de raíz del proyecto (busca workflow.md como marcador), config global
```

Para la explicación completa del motor de Focus, los Adapters, y cómo extenderlos, ver el documento en inglés — esa parte de la arquitectura no cambió con este conjunto de trabajo y mantenerla en un solo idioma evita que las dos versiones se desincronicen con el tiempo.

## Cambios recientes — eliminación unificada, `/save` extendido, resaltado de sintaxis, `workflow.md` con Budget, configuración de rutas en project.json

### Motivo

Cinco requerimientos llegaron juntos, todos con la misma restricción: **Chat, MCP y HTTP deben comportarse de forma idéntica**, sin lógica particular por puerta de entrada y sin implementaciones duplicadas. Esa restricción es exactamente la arquitectura que este proyecto ya tenía para `save`/`chat_completion`/`estimate_budget` — el trabajo fue extender ese mismo patrón, no inventar uno nuevo.

### Archivos nuevos

| Archivo | Qué es |
|---|---|
| `documents/delete_service.go` | `Delete()`/`DeleteRequest`/`DeleteResult`/`FormatDeletePrompt` — la única implementación detrás de `/delete`, `delete_path` de MCP, y `POST /delete` |
| `documents/save_selection.go` | `SelectContent()`/`GroupExchanges()`/`TranscriptText()`/`ExtractCodeBlocks()`/`StripCodeBlocks()`/`ParseRangeToken()` — la única implementación detrás de los modos actual/`-all`/`-range`/`-c`/`-text` de `/save` |
| `documents/highlight.go` | `DetectLanguage()`/`AutoTagCodeFences()` — la única implementación de detección de lenguaje detrás de "el código siempre vuelve resaltado" |
| `documents/highlight_test.go`, agregados en `budget/*_test.go` | cobertura de tests para lo anterior |
| `budget/workflow.go` | `LoadWorkflow()` — el pipeline con validación de Budget detrás de "lee/ejecuta workflow.md" |
| `cli/delete_cmd.go` | el loop interactivo Y/N de `/delete` — el único código específico de esta puerta de entrada que hizo falta (MCP/HTTP usan `confirm:true`, misma convención que ya establecía `apply_edits`) |
| `cli/workflow_cmd.go` | reconoce cada frase de workflow.md que pide el spec y llama a `budget.LoadWorkflow` |
| `docs/i18n/{en,es}/PROJECT_JSON.md` | referencia nueva de campos de project.json, en ambos idiomas |

### Archivos modificados (no documentación)

- **`core/types.go`** — se agregaron `Project.WorkflowPath`/`Project.BudgetPath`/`Project.TokenHistoryPath` (todos `string` simples, claves JSON nuevas `workflow_path`/`budget_path`/`token_history_path`) y `ResolveWorkflowPath`. `Project.Repo` no cambió — sigue siendo el único repositorio del proyecto; no existe un array `repos` (ver **Simplificación posterior** abajo para el motivo).
- **`budget/history.go`** — se agregó `HistoryPath` (resuelve `token_history_path`, o `projects/<proyecto>/mova-token-history.json` por default); los call sites de `RecordUsage` no cambian.
- **`budget/prices.go`** — se eliminó el campo y la función `ReportPath`. `config/prices.json` ahora es solo precios de modelos.
- **`budget/report.go`** — la firma de `WriteReport` cambió de `(root string, prices *PricesConfig, report *Report)` a `(root, project string, proj *core.Project, report *Report)`; se agregó `BudgetReportPath` (resuelve `budget_path`, o `projects/<proyecto>/mova-budget-report.md` por default). Se actualizaron todos los callers existentes (`cli/budget_cmd.go`, `mcp/budget_tool.go`, `cli/chat_save.go`).
- **`mcp/server.go`** — el schema y el case de `get_workflow` ahora requieren `project` y pasan por `budget.LoadWorkflow`; se registró `delete_path` (schema + dispatch).
- **`mcp/documents_tool.go`**, **`mcp/documents_tool_helpers.go`** — se agregó el case `delete_path`; el case `save` acepta `history`/`mode`/`range`/`code_only`/`text_only`; se agregaron `pathsArg` (propio de `delete_path`, "uno o más archivos a borrar en la misma llamada" — no relacionado con la configuración de project.json) y `chatTurnsArg`.
- **`mcp/chat_tool.go`** — la respuesta de `chat_completion` ahora pasa por `documents.AutoTagCodeFences` antes de devolverse.
- **`http/server.go`** — se agregaron `/delete` y `/workflow`, ambos wrappers finos sobre la tool MCP correspondiente (mismo patrón que ya usaba `/save`) — sin lógica propia de HTTP.
- **`cli/chat_cmd.go`** — despacha `/delete` y, antes de la detección de edición/guardado en lenguaje natural, `handleWorkflowCommand`.
- **`cli/chat_save.go`** — se agregaron los flags `-all`/`-range N-M`/`-text`; la lógica propia de código/texto/rango del archivo se eliminó y se reemplazó por llamadas a `documents/save_selection.go` (ver "Eliminado" abajo); `renderMarkdown` ahora llama primero a `documents.AutoTagCodeFences`.
- **`config/prices.json`** — se eliminó la clave `"report_path"`.

### Simplificación posterior: rutas únicas, sin multi-repositorio

Una iteración anterior de este trabajo hacía que `repo` (mediante un array nuevo `repos` + `RepoConfig`), `workflow_path`, `budget_path` y `token_history_path` aceptaran un string único O un array (un tipo `core.StringList` que decodificaba ambas formas), para poder configurar más de un repositorio o más de un destino de reporte por proyecto. Eso se revirtió deliberadamente, a pedido, en favor de lo que finalmente quedó: cada uno de esos campos es un **string único**, sin excepción.

Por qué ganó la versión más simple: un proyecto realmente solo necesita UN `workflow.md`, UN destino de reporte de Budget, y UN archivo de historial de tokens — los casos de "más de uno" que parecían justificar un array resultaron, en todos los ejemplos reales, ser "más de un DIRECTORIO dentro del mismo repo", que `focus` ya resuelve (ver la sección "Trabajar sobre parte de `repo`" en [PROJECT_JSON.md](PROJECT_JSON.md)) sin agregar una segunda forma de configuración a cada campo `*_path`. `StringList`/`RepoConfig`/`Project.Repos`/`Project.Repositories()`/`HistoryPaths`/`RecordUsageAll`/`BudgetReportPaths` (las formas plurales que devolvían arrays) existieron brevemente y se eliminaron — `core/types.go`, `budget/history.go` y `budget/report.go` ahora solo exponen las funciones de valor único `HistoryPath`/`BudgetReportPath` descritas arriba. Si más adelante aparece una necesidad real de más de un repositorio por proyecto, reintroducir `RepoConfig` es directo (queda preservado en el historial de este documento), pero debería esperar a esa necesidad real en vez de anticiparse a ella sin un caso concreto.

### Eliminado

- `budget.ReportPath` (función) y `PricesConfig.ReportPath` (campo) — reemplazados por `budget.BudgetReportPath`, que toma la configuración de `budget_path` en `project.json` en vez de `config/prices.json`.
- Las funciones privadas `exchangePairs`/`transcriptText`/`stripCodeBlocks`/`extractCodeBlocks`/`buildSaveContent`/`parseRangeToken`/`splitFirstToken` (la parte de parseo de rango) de `cli/chat_save.go` — se movieron a `documents/save_selection.go` como implementación compartida; `cli/chat_save.go` ahora solo adapta `models.ChatMessage` a `documents.ChatTurn` y llama a las funciones compartidas.
- `core.StringList`/`core.RepoConfig`/`Project.Repos`/`Project.Repositories()`, y `budget.HistoryPaths`/`RecordUsageAll`/`BudgetReportPaths` — el soporte multi-repositorio/multi-ruta basado en arrays descrito en **Simplificación posterior** arriba, se lanzó brevemente y luego se eliminó en favor de campos de valor único.

Nada más se eliminó. Las tools legacy por formato de `mcp/server.go` (`write_file`, `generate_word_contract`, `generate_pdf_document`, `generate_excel_report`, `generate_vector_graphic`, `trigger_diffusion_image`) quedan intactas — ya estaban reemplazadas por `save` antes de este conjunto de cambios, y no se pidió eliminarlas ni lo requiere nada de lo anterior.

### Compatibilidad

Todo cambio acá es aditivo a nivel `project.json`: un proyecto que nunca definió `workflow_path`/`budget_path`/`token_history_path` se comporta exactamente igual que antes — las rutas de salida por default (`projects/<proyecto>/mova-budget-report.md`, `projects/<proyecto>/mova-token-history.json`) coinciden con lo que `HistoryPath`/el viejo `ReportPath` ya usaban por default para el único proyecto de ejemplo que tenía un `report_path` configurado. `/save` sin flags sigue guardando solo la última respuesta, sin cambios; `/save -c`/`-d`/`-append`/`-overwrite`/`-no-overwrite` no cambiaron de comportamiento, solo se refactorizaron para compartir su lógica de filtrado con los modos nuevos.

### Flujo de ejecución (workflow.md)

```
"lee workflow.md" / "workflow.md <proyecto> [tarea]"
        │
        ▼
budget.LoadWorkflow(adapter, root, project, task, explicitPath, modelHint)
        │
        ├─ 1. adapter.GetProject(project)              — resuelve project.json
        ├─ 2. resolveTask(proj, task)                   — resuelve la task, si se nombró una
        ├─ 3. core.ResolveWorkflowPath(root, proj, …)   — qué archivo workflow.md usar
        ├─ 4. core.BuildContextSections(...)            — agents+skills+prompt+focus+memory
        │      (Dedup y Focus ya corren DENTRO de esta llamada — nada extra que invocar)
        ├─ 5. os.ReadFile(path)                          — lee los bytes de workflow.md
        ├─ 6. CountTokens(context + workflow.md)         — estimación
        ├─ 7. CheckLimit / EnforceLimit                  — la validación de Budget
        │      ├─ excede el límite  → devuelve {Log, sin Content}, err = el bloque ERROR/Sugerencia
        │      └─ dentro del límite → devuelve {Log, Content = el texto de workflow.md}
        ▼
El chat imprime Log + Content y agrega Content a sess.System (disponible para los siguientes turnos)
MCP/HTTP devuelven Log + Content (o Log + el error) como el texto de resultado de la tool
```

### Bug corregido en esta sesión: el gate de Budget se saltaba en silencio sin task

`cli/run_cmd.go`, `cli/chat_cmd.go`, `mcp/context_tool.go` (`get_full_context`) y `mcp/chat_tool.go` (`chat_completion`) tenían el mismo tipo de bug: `if t, ok := proj.Tasks[resolvedTask]; ok { EnforceLimit(...) }` — la validación de Budget solo corría cuando `resolvedTask` coincidía con una entrada real en `proj.Tasks`. Un proyecto sin `default_task` (o con un nombre de task que no coincidía), al pedir simplemente "leer todo el proyecto" sin nombrar ninguna task, se saltaba el gate de Budget por completo — exactamente el escenario de "lee `<proyecto>`" (sin task) llegando a Claude Console, Codex, Gemini, o cualquier otro cliente MCP.

Se corrigió centralizando la resolución de la task en dos funciones pequeñas y exportadas, en vez de cuatro copias ad-hoc:

- **`core.ResolveTaskName(proj, taskName)`** — la `taskName` explícita, si no `proj.DefaultTask`, si no la única task del proyecto si solo declara una, si no `""`. La usa la propia `core.BuildContextSections` (así el comportamiento de auto-selección vive en un solo lugar) y también cada punto donde corre el gate de Budget, así que la task contra la que valida el Budget siempre es exactamente la misma con la que se construyó el contexto.
- **`budget.ResolveTask(proj, taskName)`** — busca ese nombre en `proj.Tasks`, o devuelve un puntero a un `Task{}` vacío (nunca `nil`) para que `EnforceLimit` siempre tenga algo que llamar, incluso para "ninguna task" (que correctamente cae al `budget` de nivel proyecto).

Los cuatro puntos ahora hacen `t := budget.ResolveTask(proj, core.ResolveTaskName(proj, taskName)); if err := budget.EnforceLimit(proj, t, tokens); err != nil { ... }` sin condición — el gate ya no se puede saltear. Se verificó de punta a punta con un binario compilado real y un proyecto con `budget.max_tokens: 5` y sin `default_task`: `mova run`, el "workflow.md `<proyecto>`" de `mova chat`, los tools MCP `get_workflow`/`get_full_context` (sobre stdio JSON-RPC real), y `POST /workflow` (sobre un servidor HTTP real) devolvieron correctamente el bloque de ERROR/Sugerencia en vez del contenido.

También se corrigió en el mismo paso: el wrapper genérico de errores de `mcp/server.go` agregaba "Please use 'list_projects' to see valid projects." a TODOS los errores de tools, incluidos los de Budget — engañoso, porque un error de Budget excedido no tiene nada que ver con que el nombre del proyecto esté mal. Ahora solo agrega esa sugerencia a errores que no traen ya su propia explicación (es decir, cualquiera que no tenga un bloque `"\nSuggestion:"`).

### Cómo extender esto más adelante

- **Un nuevo modo de selección para `/save`**: agregar un case a `documents.SelectionMode`/`SelectContent` en `save_selection.go` — todas las puertas de entrada lo reciben automáticamente vía `flags.mode()` (CLI) o el argumento JSON `mode` (MCP/HTTP); no hace falta ningún cambio por puerta de entrada.
- **Una nueva frase disparadora de workflow.md**: agregarla a las expresiones regulares de `cli/workflow_cmd.go` — MCP/HTTP ya aceptan cualquier combinación de argumentos `project`/`task`/`workflow`, así que no hace falta ningún cambio ahí.
- **Una necesidad real de más de un repositorio por proyecto**: no conviene volver a un campo array sobre `Project` directamente — conviene modelarlo explícitamente (un tipo con forma de `RepoConfig`, con tasks/focus acotados por repo) una vez que haya un caso de uso multi-repositorio real que justifique el diseño, en vez de soportarlo especulativamente ahora (ver **Simplificación posterior** arriba).
