# project.json — referencia mínima

Vive siempre en `projects/<nombre>/project.json` (ruta fija, el motor no busca en otro lado).

## Ejemplo mínimo real (ver `examples/01-mcp-agent-governance/project/project.json`)

```json
{
  "project": "mi-proyecto",
  "author": "equipo-plataforma",
  "description": "una línea",
  "repo": "ruta/al/codigo",
  "lang": "es",
  "adapter": "file",
  "default_task": "revisar",
  "agents": { "domain": "base", "use": ["backend-dev"], "custom": [] },
  "skills": { "domain": "base", "use": ["api-security"], "custom": [] },
  "tasks": {
    "revisar": { "prompt": "review-project", "variables": {}, "focus": ["archivo.js"] }
  },
  "llm_profile": { "config": "llama3.2.3b" },
  "budget": { "max_tokens": 20000, "sanitize": { "enabled": true } }
}
```

## Campos clave

| Campo | Qué hace |
|---|---|
| `author` | **Obligatorio.** Autor de la política — Matriz de Auditoría #3. Vacío → `system:default`. |
| `repo` | Directorio real analizado (relativo a la raíz de Mova). |
| `llm_profile.{provider,config}` | Qué modelo recibiría el contexto — Matriz de Auditoría #11. `config` referencia un archivo en `config/models/<provider>/`. |
| `tasks.<t>.focus` | Qué entra: archivos, directorios, globs, o `archivo::kind=símbolo` (AST). |
| `tasks.<t>.exclude` | Qué se excluye del `focus` — soporta la misma sintaxis AST (nuevo: analiza un archivo completo dejando solo una función fuera). |
| `budget.max_tokens` / `max_tokens_per_run` / `max_monthly_usd` | Techos del circuit breaker. |
| `budget.on_exceed` | `"warn"` (por defecto) informa y continúa. `"abort"` (alias: `"block"`) corta **antes de cualquier llamada al proveedor LLM** — mismo corte duro en CLI/Chat, MCP y HTTP. |
| `budget.sanitize.{enabled,dedupe_logs,strip_blank,strip_comments}` | Controles del Sanitizer. |
| `budget.pii_masking.enabled` | Enmascarado estructural de PII (ver `ARTIFACTS.md`). |
| `debug` | `true` imprime, en cada puerta (chat, CLI, HTTP, MCP), qué se resolvió antes de correr una tarea: ruta del repo, cada agente/skill/prompt con su ruta resuelta (o `"inline"`), las entradas de `focus`/`exclude` con su ruta absoluta, **y — en `context-trace` — la ruta exacta de cada política incluida/excluida** (ver `policies` abajo). Por defecto `false`. Nunca se agrega solo por generar un `project.json` con `mova init` — es opt-in manual. |
| `egress_audit` | Auditoría del contexto **ya sanitizado** justo antes de salir hacia el LLM, y/o un dry-run que corta la llamada — ver sección dedicada abajo. |
| `apply` / `tasks.<t>.apply` | Permite que la respuesta del modelo **modifique archivos** previa confirmación (`true`, `false` o `{enabled, backup}`; el respaldo `.bak` junto al archivo es `true` por defecto) — ver «Modificar archivos (`apply`)». |

Ver `docs/i18n/es/AST_FILTER.md` para la sintaxis exacta de `archivo::kind=nombre`.

## Variables dinámicas (agents, skills y prompts)

Cualquier clave de un bloque `"variables"` reemplaza `${CLAVE}` o `{{CLAVE}}` en los `.md` — sin importar el nombre ni la tecnología. Mayúsculas/minúsculas no importan (`query` = `QUERY`); una marca sin variable se deja tal cual.

```json
{
  "variables": { "STACK": "Node.js / JavaScript" },
  "agents": { "domain": "base", "use": ["frontend-dev"], "custom": [],
              "variables": { "QUERY": "Optimizar la carga del Gantt" } },
  "skills": { "domain": "base", "use": ["ui-accessibility"], "custom": [],
              "variables": { "UPGRADE_TRIGGER": "> 5.000 elementos en Gantt.js" } },
  "tasks": { "optimizar": {
      "prompt": "fix-or-improve",
      "variables": { "QUERY": "Optimizar la carga de programación", "SKIPPED_ABSTRACTIONS": "sin capas de API nuevas" },
      "focus": ["Gantt.js", "MenuOrders.js::func=show()"] } }
}
```

Precedencia (gana el último): `PROJECT`/`REPO`/`TASK`/`LANG` (automáticas) < `variables` raíz < `agents.variables` / `skills.variables` (solo su bloque) < `tasks.<t>.variables` (todos los bloques).

`focus` y `exclude` aceptan un nombre suelto (`Gantt.js`), una ruta parcial (`schedule/Gantt.js`) o una ruta completa, con o sin `::func=...`. Si el nombre existe en varias carpetas, entran **todas** las coincidencias y `debug` lo avisa: para elegir una, usa una ruta más larga.

## Grafo de dependencias (`graph`)

Dentro de una tarea, `graph` genera — **sin LLM y sin gastar tokens** — un diagrama de las llamadas, referencias a variables e importaciones entre los archivos y símbolos de `focus` y `exclude`. Usa el mismo Tree-Sitter y los mismos `kind` que el [filtro AST](AST_FILTER.md) (`func`/`method`, `class`, `var`, `const`, `struct`, `interface`…).

```json
"tasks": { "analizar": {
  "focus":   ["api-core\\lib\\schedule\\planner.js::func=getOrders,saveOrder", "Admin.js::func=getModelo"],
  "exclude": ["planner.js::func=splitOrder"],
  "graph":   "graph.png"
} }
```

| Valor de `graph` | Resultado |
|---|---|
| ausente, `""`, `false`, `null` | no se genera nada |
| `true` | `graph.png` en la carpeta de `project.json` |
| `"grafo.png"` / `"grafo.svg"` / `"grafo.pdf"` | ese formato, relativo a la carpeta de `project.json` (`projects/<proyecto>/`) |
| `"grafo"` (sin extensión) o una carpeta (`"salida/"`) | `grafo.png` / `salida/graph.png` |
| ruta absoluta: `C:\graficos\g.png`, `\\servidor\compartido\g.pdf`, `/opt/g.svg` | esa ruta; se crean las carpetas intermedias |
| otra extensión (`.jpg`…) | no se genera y se avisa (solo `.png`, `.svg`, `.pdf`) |

**Multiplataforma:** la ruta se resuelve con el mismo criterio del resto de Mova (`\` y `/` valen igual; Windows `C:\…`, UNC y Unix se reconocen en cualquier SO). Una ruta de Windows en un equipo Linux/macOS se rechaza con un mensaje claro en vez de escribirse en otro lugar.

**Qué dibuja.** Cada archivo es un subgrafo; cada símbolo un nodo — función/método (azul), clase/tipo (violeta), variable/constante (ámbar). Con `::kind=nombres` solo entran esos símbolos; un archivo sin `::` aporta sus funciones y clases. Los símbolos de `exclude` salen en rojo punteado; los símbolos que los de `focus` usan pero no pediste salen punteados grises («fuera de foco»). Flechas: llamada (azul), referencia a variable (ámbar), importación archivo→archivo (violeta punteada, por carriles sobre los archivos) y llamada *inferida por nombre único* (azul punteada: no hay `require` que la resuelva, pero solo un símbolo de `focus`/`exclude` se llama así). Archivos con muchos símbolos pasan a 2–3 columnas internas y el espacio entre archivos crece con la cantidad de relaciones. La paleta y tipografía son las de `context-diagram.png`.

**Una tarea o todas.** Con una tarea nombrada (`mova chat <proyecto> analizar`) se genera **solo** su grafo; sin tarea y con varias declaradas (modo «todas»), el de cada tarea que declare `graph`, cada una con su propio `focus`/`exclude` (o el heredado del proyecto). Usa un archivo distinto por tarea (`"analizar.png"`, `"agregar-columnas.png"`…): si dos tareas apuntan al mismo archivo, la segunda se omite con un aviso `[Graph]`.

**Cuándo y cómo se genera.** Cada vez que se arma el contexto (`mova run`, `mova chat`, MCP, HTTP, presupuesto) y hay tareas con `graph`.
- **`mova chat`, `mova mcp start` (stdio y HTTP):** en **segundo plano**. El arranque, cada turno y `exit` no esperan; al terminar cada grafo el chat avisa `[Graph] <tarea>: grafo generado → ruta` y el servidor MCP/HTTP lo deja en su log.
- **`mova run`, `trace`, `budget`:** en línea; el comando termina con los archivos ya escritos (las tareas se generan en paralelo).
- **Caché:** la huella de `project.json` (focus, exclude, `graph`, `lang`) y el tamaño+fecha de cada archivo analizado se guardan en `graph-cache.json`, junto a `project.json`. Si nada cambió, ni un arranque nuevo de `mova chat` vuelve a renderizar (un turno sin cambios solo hace `stat` de los archivos). Se puede borrar sin riesgo.
- **En caliente:** si editas `project.json` o un archivo analizado, el siguiente turno regenera lo afectado. El archivo se escribe de forma atómica (temporal + renombrado): cerrar el chat o abrir el grafo en un visor nunca deja un archivo a medias.
- **Idioma:** los textos del diagrama y los mensajes `[Graph]` salen de `config/lang/{es,en}.json` (sección `graph`) en el `lang` del proyecto.

**Rendimiento.** Cada forma se rasteriza solo dentro de su caja y en paralelo: un grafo típico (~35 símbolos) tarda ~0,1–0,4 s y uno de 130 símbolos / 300 relaciones unos 2 s en un núcleo; los grafos muy densos bajan la resolución automáticamente. `.svg` es instantáneo.

**Alcance y límites.** Los `require`/`import` se resuelven hacia archivos de `focus`/`exclude` para JavaScript/TypeScript (rutas relativas o sufijo único) y Python (`from x import y`); en los demás lenguajes se resuelven las llamadas del mismo archivo y las inferidas por nombre único. No se siguen llamadas dinámicas (`obj[nombre]()`), funciones pasadas como callback sin invocar, ni archivos fuera de `focus`/`exclude`. Si un mismo archivo repite el nombre de un método en dos clases, se dibuja el primero. Si `focus` pide símbolos que no existen, el aviso `[Graph]` los lista.

## Memoria automática (`memory`, `memory_max_chars`)

`memory` activa el registro **automático** de memoria. Cada respuesta del modelo con trabajo real (≥200 caracteres) deja su **bloque de síntesis** `memory` en `memory.md`, y **todas** las tareas lo leen en la sección `MEMORY` de su contexto. Así `analizar` alimenta a `agregar-columnas` aunque las ejecutes por separado (`mova chat <proyecto> analizar`, luego `mova chat <proyecto> agregar-columnas`), cambies de tarea en el chat (`/task`) o uses MCP/HTTP.

| Valor | Efecto |
|---|---|
| ausente / `false` | **No** se registra memoria automática. `memory.md` existente se sigue leyendo, y `/memory` manual sigue funcionando. |
| `true` | `memory.md` junto a `project.json` (`projects/<proyecto>/memory.md`). |
| `"<ruta>"` | Esa ubicación. Archivo (`…/memoria.md`) o carpeta (`…/memoria/` → `memoria/memory.md`). |

Rutas multiplataforma, con las mismas reglas que `memory_path` y `egress_audit.output_file`: `C:\…`, `D:\…`, `E:\…` (Windows), `/mnt/…`, `/home/…` (Linux), `/Volumes/…` (macOS), `\\servidor\recurso\…` (red, en Windows), `~/…` (home) o relativa (a la carpeta del proyecto). Una ruta de otro sistema (p. ej. `C:\` en un servidor Linux) da un error explícito, no un archivo fantasma. Precedencia: `memory` (ruta) > `memory_path` > por defecto.

```json
"memory": true
"memory": "D:\\mova\\memoria\\mi-proyecto\\"
"memory": "/mnt/compartido/mova/mi-proyecto/memory.md"
```

**Qué se guarda.** Solo el bloque `memory` que el modelo entrega al final (Tarea, Realizado, Hallazgos `archivo::función`, Datos clave, Resuelto, Decisiones, Pendiente), no la respuesta completa: preciso y barato en tokens. Si el modelo no lo entrega, un resumen automático con las líneas técnicas. Cada entrada lleva fecha, tarea y una huella; una síntesis idéntica no se vuelve a escribir. Si el modelo entrega un bloque por tarea, se guardan por separado.

**Qué se lee.** Con `memory.md` pequeño, entero. Con tope (`memory_max_chars`, por defecto 20000 caracteres ≈ 5k tokens) se conserva **siempre la entrada más reciente de cada tarea** y, con el espacio restante, las más nuevas; las demás quedan abreviadas.

**Escritura segura.** Bloqueo de archivo y escritura atómica: chat, MCP y HTTP pueden escribir a la vez sin perder entradas.

**Modo delegado** (sin `llm_profile`): Mova no ve la respuesta del anfitrión; `chat_completion` le pide llamar a `save_memory` con su bloque, que respeta este mismo campo.

**Caché.** `mova-context-cache.json` solo acelera el saneado de `focus`/`memory` (clave: hash del texto); un cambio de `memory.md` se ve siempre. Con `pii_masking` activo el caché se desactiva (guardaba texto antes del enmascarado).

## Modificar archivos desde la respuesta del modelo (`apply`)

Con `apply` activo, las respuestas del modelo pueden **modificar archivos del repo** — **siempre después de preguntarte**. Funciona igual en Chat, MCP y HTTP (mismo código: `src/applyflow`).

```json
"apply": true                                   // aplicar, con respaldo (valor por defecto)
"apply": { "enabled": true, "backup": true }    // respaldo al lado del archivo
"apply": { "enabled": true, "backup": false }   // sin respaldo
"apply": false                                  // (o ausente) solo lectura, como siempre
```

Se declara en el proyecto y/o **por tarea** (`tasks.<t>.apply`); el de la tarea gana. Con `apply` ausente o `false` no cambia nada de lo que ya hacía Mova.

| Clave | Default | Qué hace |
|---|---|---|
| `apply` / `tasks.<t>.apply` | ausente = apagado | `true`, `false` o `{ "enabled", "backup" }`. Con el objeto, `enabled` vale `true` si se omite. |
| `apply.backup` | `true` | **Antes de modificar un archivo existente, se copia al lado, en el mismo directorio**, como `<archivo>.mova-<AAAAMMDD-HHMMSS>.bak` (ej.: `cobros.js.mova-20261003-153012.bak`). Todos los respaldos de una confirmación comparten sello. Un archivo nuevo no tiene respaldo. `false` = se sobrescribe sin copia. |

### Cómo funciona

1. **Mova le enseña al modelo el formato** (se agrega a la sección `INSTRUCTION` del contexto): cada cambio es un bloque ` ```javascript:ruta/archivo.js::nombreFuncion() ` con la función **completa**, o ` ```javascript:ruta/archivo.js ` con el archivo completo. Un bloque sin ese encabezado es explicación y no se aplica.
2. **Mova detecta los bloques, los valida y pregunta** (nada se escribe todavía):

```
Mova Context detectó 3 cambio(s) propuesto(s) en 2 archivo(s):
  1. [MODIFICAR] src/despacho/planificador.js::normalizarPedido()  (+10 −9 líneas)
  2. [MODIFICAR] src/facturacion/cobros.js::generarCobro()  (+9 −9 líneas)
  3. [MODIFICAR] src/legacy/tarifasV1.js::tarifaPlanaV1()
       ⚠ no se aplicará: el archivo está en exclude
Respaldo activado: cada archivo existente se copia al lado (<archivo>.mova-<fecha>.bak) antes de modificarlo.
¿Quieres modificar los archivos propuestos?
  [s] Sí, TODOS los archivos   [1..N] Solo esos números (ej: 1,3)   [n] No, ninguno
```

3. **Tu respuesta decide**: `s`/`sí`/`todos` aplica **todos** los cambios aplicables; `1,3` (o `#1,#3`) solo esos; `n`/`no` ninguno. Cualquier otra cosa **no modifica nada** y se vuelve a preguntar. Los omitidos se informan con su motivo; el resto igual se aplica.

### Dónde se responde

| Puerta | Cómo se pregunta y se responde |
|---|---|
| **Chat** | La pregunta aparece en la terminal justo tras la respuesta; contestas `s`, `1,3` o `n`. |
| **MCP / HTTP** | La respuesta de `chat_completion` termina con la lista y la pregunta («Todavía NO se ha cambiado nada»); la propuesta queda **pendiente** en `projects/<proyecto>/pending-changes.json` (vence a los 60 min). Respondes en la **siguiente llamada**: `message: "sí"` / `"1,3"` / `"no"`, o el argumento `apply_changes: "all" \| "1,3" \| "none"`. Esa llamada aplica **sin volver a llamar al modelo**. Un mensaje que no parece una respuesta se trata como consulta nueva y la propuesta sigue pendiente. |

### Garantías (todas con tests)

- **Nada se escribe sin una respuesta afirmativa explícita.**
- **Nada fuera del repo**: rutas absolutas o con `..` se rechazan.
- **`exclude` se respeta**: un archivo o función excluidos nunca se modifican (`archivo`, `archivo::func=a,b`). Un archivo completo no se reescribe si alguna de sus funciones está excluida: se pide solo las funciones a cambiar.
- **Un fragmento nunca pisa un archivo**: si la función no se encuentra con certeza en un archivo existente, el bloque se omite y el archivo queda intacto (antes se reescribía entero con el fragmento). Un reemplazo que desbalancea llaves (fragmento truncado) también se rechaza. Reconoce funciones `function`, métodos de clase (`async _processOrder(...) {`) y funciones flecha con llaves de JS/TS, además de Go/Java/C/Python.
- **Cada archivo es independiente**: el fallo de uno no detiene a los demás.

### Prompts que ya preguntan «(Sí/No)» al modelo

Un prompt clásico (p. ej. «¿Deseas que aplique estas modificaciones directamente en los archivos fuentes? (Sí/No)») no podía funcionar: **el modelo no puede escribir archivos** y su propuesta, sin destino, no tenía nada que Mova pudiera aplicar — por eso responder «Sí» «no hacía nada». Ahora, con `apply` activo, si respondes «Sí» a esa pregunta y la propuesta no trae bloques con destino, Mova pide al modelo convertirla en bloques aplicables y **te muestra la lista real con la pregunta final**. Lo recomendado es quitar la pregunta del prompt y dejar que la haga Mova (ver `capabilities/<tu-proyecto>/prompts/<tu-prompt>.md`, fases 1-3).

### Archivos que Mova crea

`*.mova-*.bak` (respaldos, junto al código) y `projects/<proyecto>/pending-changes.json` (propuesta MCP/HTTP pendiente). Agrega `*.mova-*.bak` a tu `.gitignore`. Para deshacer: copia el `.bak` sobre el archivo.

## Qué tarea carga chat / MCP / HTTP

Con una tarea nombrada se carga **solo** su prompt, focus y grafo (y solo se genera su `graph`). Sin tarea y con varias declaradas se cargan **todas**. `mova run` conserva su comportamiento (`default_task`).

## `egress_audit` — auditoría y dry-run antes del proveedor LLM

```json
"egress_audit": {
  "dry_run": true,
  "output_file": ".mova/egress_sanitized.log"
}
```

| Clave | Tipo | Default | Qué hace |
|---|---|---|---|
| `dry_run` | bool | `false` | Si es `true`: se completa la gobernanza/sanitización (y el log, si `output_file` está configurado), pero **nunca se llama al proveedor LLM**. El cliente recibe una respuesta exitosa indicando que el dry-run terminó y que no hubo inferencia. |
| `output_file` | string | `""` (deshabilitado) | Dónde se escribe el almacén de bloques del contexto sanitizado (sin duplicados). Independiente de `dry_run` — se puede auditar sin dry-run, o hacer dry-run sin auditar. |

**Con el bloque ausente:** `dry_run=false`, `output_file=""` — comportamiento idéntico al actual, sin cambios.

### Resolución de `output_file` (siempre relativa al `project.json`, nunca al directorio de trabajo)

- Ruta relativa → se resuelve contra `projects/<proyecto>/`, **no** contra el directorio desde donde se ejecutó `mova`.
  Ejemplo: en `projects/02-pii-compliance-governance/project.json`, `"output_file": ".mova/egress_sanitized.log"` escribe en `projects/02-pii-compliance-governance/.mova/egress_sanitized.log`.
- Ruta absoluta — Unix (`/var/log/...`), Windows (`C:\...`, `D:\...`, `E:\...`) o UNC (`\\servidor\recurso\...`) — se usa tal cual, reconocida de forma multiplataforma sin importar en qué SO corre el binario de Mova (misma utilidad que ya usan `write_file`/`create_directory`).
- Si `output_file` termina en `/` o `\` (nombra un directorio, no un archivo), se usa el nombre por defecto **`egress_sanitized.md`** dentro de ese directorio. Un nombre sin barra final (aunque no tenga extensión, ej. `.mova`) se respeta tal cual — no se le fuerza extensión.
- Los directorios necesarios se crean automáticamente (`os.MkdirAll`, permisos estándar). El archivo es un almacén de **bloques sin duplicados** (ver «Qué se escribe»): ya no crece con cada mensaje.
- Si el archivo configurado no puede crearse o escribirse, `mova` **devuelve error y no llama al proveedor LLM** — el mismo comportamiento en CLI/Chat, MCP y HTTP, porque las tres puertas comparten una única implementación (`models.Session.Send`/`SendStream`).

### Qué se escribe

Únicamente el contexto **ya gobernado y sanitizado** (lo mismo que de verdad se enviaría al modelo) — nunca el contenido crudo pre-sanitización.

**Sin duplicados.** El archivo guarda un bloque por pieza del contexto (cabecera, cada agent/skill/prompt, cada `FOCUS`, memoria), cada uno con `key`, `sha` y `updated` (UTC):
- misma clave y mismo contenido → no se toca nada (ni se reescribe el archivo);
- misma clave y contenido distinto → el bloque se **reemplaza** completo;
- clave nueva → se **anexa** al final.

La escritura es atómica y serializada por archivo. Un archivo del formato antiguo (un bloque por mensaje) se migra solo, conservando únicamente el último contexto.

### Air-gap real — `dry_run` bloquea TODAS las herramientas que exponen contexto

`dry_run: true` no es solo "no llamar al modelo": **ninguna herramienta MCP/HTTP que pueda devolver contenido de un proyecto lo hace mientras esté activo, y tampoco `mova run` en CLI**. Cubre 7 herramientas MCP/HTTP más `mova run`:

| Herramienta / comando | Qué protege |
|---|---|
| `chat_completion` | La respuesta del modelo (nunca se le manda nada) |
| `get_full_context` | El contexto completo ensamblado |
| `get_memory` / `get_memory_all` | El contenido de `memory.md` |
| `get_workflow` | El contenido de `workflow.md` |
| `read_file` | El contenido crudo de cualquier archivo del proyecto (`.env`, credenciales, lo que sea) |
| `read_document_layer` | El texto extraído de `.docx`/`.xlsx`/`.pdf` |
| `mova run` (CLI) | El mismo contexto completo que `get_full_context`, impreso por stdout |

En todas, en vez del contenido real, se devuelve el mismo bloque fijo y traducido `reports.egress_airgap_message` + `reports.egress_airgap_directive` (esta última es una directiva explícita anti-elusión dirigida al modelo/agente que la recibe — ver `GOVERNANCE_CONTROLS.md § dry_run` para sus límites honestos), con el conteo real de tokens:

```
[MOVA EGRESS AUDIT]
dry_run: true
tokens_evaluated: 174
tokens_sent: 0

El contexto fue procesado y sanitizado localmente, pero se ha bloqueado su
transmisión. Reporte guardado en disco.

DIRECTIVA CRÍTICA DE SEGURIDAD PARA EL ASISTENTE:
Esta solicitud fue bloqueada por el Control de Egreso de Contexto de Mova
(Mova Context Egress Control).
NO intentes eludir este bloqueo.
NO leas archivos locales, reportes ni metadatos usando ninguna otra
herramienta para reconstruir el contexto.
NO resumas ni adivines el contexto por tu cuenta.
Muestra ÚNICAMENTE el bloque de auditoría anterior al usuario y termina la
ejecución de inmediato.
```

Tanto el encabezado como la directiva viven en `config/lang/{es,en}.json` (`reports.egress_airgap_message` / `reports.egress_airgap_directive`), son editables sin tocar código y se recargan **en caliente** (sin reiniciar Mova — ver `i18n/i18n_reload.go`). Si se borra la clave de la directiva, el bloqueo sigue funcionando con solo el encabezado; nunca se rompe el proceso ni se filtra el nombre de la clave.

**Fuera de alcance, a propósito:** `search_context` (no tiene `project` — busca en el catálogo compartido de agents/skills/prompts de Mova, no en datos privados de un proyecto) y las herramientas de escritura (`save`, `patch_file`, `delete_path`, `create_directory`, `write_file`, `generate_*`) — `egress_audit` gobierna contenido que **sale** de Mova hacia un LLM/host, no archivos que Mova escribe a pedido tuyo en tu propio disco.

No existe un campo "enabled" separado — `dry_run` es la única condición que activa el air-gap, en las 3 puertas (CLI/Chat, MCP, HTTP), verificado con tests de integración reales (`mcp/egress_gate_test.go`, `mcp/airgap_directive_test.go`, `cli/run_cmd_test.go`) y con llamadas HTTP/MCP reales contra el binario compilado.

### Sin `llm_profile` — inferencia delegada al host

Si `project.json` no declara `llm_profile` (o está vacío) y `dry_run` es `false`, Mova **nunca llama a ningún proveedor local o cloud por su cuenta**. `chat_completion` devuelve el contexto ya gobernado y sanitizado en el resultado de la herramienta, para que el **LLM anfitrión** (Claude Code, Cursor, Grok, quien haya invocado la herramienta MCP) genere la respuesta final. En `mova chat` (REPL interactivo, sin host al que delegar) se imprime un aviso explícito indicando qué modelo global se está usando en su lugar — nunca en silencio.

## `policies` — selección de políticas de gobernanza

```json
"policies": {
  "include": ["security.json", "review.json", "compliance.json", "pii_strict.json"],
  "exclude": ["pii_permissive.json"]
}
```

También se acepta la forma corta `"policies": ["security.json", "review.json"]` (equivale a `include` sin `exclude`).

| Clave | Qué hace |
|---|---|
| `include` | Archivos de política a cargar, en orden. |
| `exclude` | Nombres a omitir, aunque `include` o la búsqueda recursiva los hubiera encontrado. Se compara por **nombre de archivo**, sin importar el directorio. |

### Precedencia (gana el primero que declare algo)

1. `--policies_include` / `--policies_exclude` en la CLI.
2. `policies` en `project.json` — **sobrescribe por completo** a `config/policy.json`.
3. `policies` en `config/policy.json` — solo se usa en modo *discovery* (`--repo`, sin `project.json`).

**Opt-in:** si `project.json` no declara `policies`, no se carga **ningún archivo** de política. Los valores internos seguros de Mova (bloqueo de llaves privadas, enmascarado PII por defecto) siguen aplicando; nada más estricto que eso se activa sin declararlo.

### Resolución de rutas (multiplataforma)

| Forma en `include` | Cómo se resuelve |
|---|---|
| `pii_strict.json` (nombre simple) | Búsqueda **recursiva** bajo `config/policy/`. Gana la coincidencia menos profunda; a igual profundidad, orden alfabético (resultado determinista en todo SO). |
| `config/custom/ventas.json` (relativa) | Contra la raíz de Mova. |
| `/etc/mova/p.json`, `C:\pol\p.json`, `\\servidor\recurso\p.json` | Absoluta Unix / Windows (C:, D:, E:) / UNC de red. |

### Prioridad por nombre personalizado

Un archivo con nombre propio (ej. `pii_strict_ventas.json`) se integra en la dimensión correcta porque la dimensión se detecta por el **contenido** del archivo (qué claves declara), no por su nombre. Si además excluyes `pii_strict.json`, el archivo del directorio por defecto queda fuera y el personalizado lo reemplaza.

El reporte indica siempre qué se aplicó: `Origen de política: CLI -> {security.json}`.

## `paths` — jerarquía de rutas por proyecto (agents/skills/prompts/cache_dir/temp_dir/output_dir)

```json
"paths": {
  "agents": "projects/mi-proyecto/local-agents",
  "skills": "projects/mi-proyecto/local-skills",
  "prompts": "projects/mi-proyecto/local-prompts",
  "output_dir": "projects/mi-proyecto/reportes"
}
```

Mismos 6 campos que `config/general/config.json` (nunca `projects` — ver por qué en `PATHS.md`).
Prioridad: `project.json` del proyecto actual → `config/general/config.json` → ruta por defecto
histórica de Mova. Los 3 niveles usan la misma regla multiplataforma que `repo` (ver
`docs/i18n/es/PATHS.md § Jerarquía por proyecto` para la referencia completa, incluido un ejemplo
funcionando de punta a punta en `projects/02-pii-compliance-governance/project.json`). Ni este
archivo ni `config/general/config.json` se cachean — cualquier edición se aplica en la siguiente
solicitud, sin reiniciar Mova.
