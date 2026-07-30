# COMMANDS — guía de comandos

> Docs: [Español](COMMANDS.md) · [English](COMMANDS.en.md)

El CLI (`mova`) es un complemento — todo lo que hace también se le puede pedir a un modelo leyendo `workflow.md` directamente. Ver [README.md](README.md#1-la-convención).

`mova` sube directorios automáticamente hasta encontrar `workflow.md`, así que funciona desde cualquier subcarpeta del repo. Si hay un único proyecto en `projects/`, `[project]` es opcional — se detecta solo.

## Índice

1. [Referencia rápida](#1-referencia-rápida) — todos los comandos en una tabla
2. [Ensamblar el contexto — `mova run`](#2-ensamblar-el-contexto--mova-run)
3. [Focus — trabajar sobre una parte del proyecto](#3-focus--trabajar-sobre-una-parte-del-proyecto)
4. [Memoria](#4-memoria)
5. [Proyectos — list, init, search](#5-proyectos--list-init-search)
6. [Modelos y proveedores](#6-modelos-y-proveedores)
7. [Chat con un modelo local o Cloud](#7-chat-con-un-modelo-local-o-cloud)
8. [Crear archivos hablando: lenguaje natural en el chat](#8-crear-archivos-hablando-lenguaje-natural-en-el-chat)
9. [`save` — crear o editar cualquier archivo o directorio](#9-save--crear-o-editar-cualquier-archivo-o-directorio)
10. [Tool-calling autónomo desde el chat](#10-tool-calling-autónomo-desde-el-chat)
11. [Documentos de oficina y medios (PDF, Word, Excel, SVG, imágenes)](#11-documentos-de-oficina-y-medios-pdf-word-excel-svg-imágenes)
12. [Archivos de texto y código](#12-archivos-de-texto-y-código)
13. [Servidor MCP — `mova mcp start`](#13-servidor-mcp--mova-mcp-start)
14. [Variables de entorno](#14-variables-de-entorno)
15. [Tokenomics — `mova budget`](#15-tokenomics--mova-budget)
16. [Instalación global del CLI](#16-instalación-global-del-cli)

---

## 1. Referencia rápida

```text
mova run           [project] [task]         genera el contexto para el LLM
mova memory        [project] "respuesta"    guarda la sesión en memory.md
mova memory-read   [project]                imprime la memoria activa
  --all                                     incluye archivos históricos
  --month 2024-01                           un mes archivado específico
mova memory-archive [project]               archiva entradas antiguas
  --days N                                  días a mantener activos (default 30)
mova memory-clear  [project]                borra TODA la memoria
  --archived                                borra solo los meses archivados
  --keep-active                             borra archivos, conserva memory.md
  --date 2024-06-15                         borra un día específico
  --from 2024-06-01 --to 2024-06-30         borra un rango de fechas
  --yes                                     omite la confirmación
mova memory-config [project] [action] [value]
  enable | disable                          activa/desactiva el archivado automático
  days N                                    días de retención (1, 10, 30, 90...)
  confirm true|false                        activa/desactiva confirmación al borrar

mova list                                   lista todos los proyectos
mova init          [name]                   crea un proyecto
mova search        "consulta" [dominio]     busca en el conocimiento, sin usar un modelo

mova config        <provider>               fija el proveedor activo (ollama, lmstudio...)
mova show          config [modelo]          muestra el proveedor activo, o la config de un modelo
mova install       llama3.1,mistral         instala modelos (con barra de progreso)
mova model-list                             lista los modelos instalados
mova remove        llama3.1,mistral         elimina modelos instalados

mova chat          [project] [task]         chat interactivo con un modelo local o Cloud
  set -model <nombre>                       cambia de modelo sin perder el historial
  /memory                                   guarda el último intercambio en memory.md
  /budget                                   genera mova-budget-report.md del proyecto activo
  /save "ruta/archivo.ext"                  guarda la última respuesta ahí (formato por extensión)
  /save -d "ruta/carpeta"                   crea solo un directorio
  /save -c "src/index.js"                   guarda ÚNICAMENTE los bloques de código de la última respuesta
  /tools                                    lista comandos y tools disponibles
  exit | quit | salir                       termina la sesión

mova budget        [project] [task]         estima tokens y costo, 100% local
  --focus                                   compara repo completo vs. solo lo que focus selecciona

mova mcp start                              inicia el servidor MCP
  --port 3000                               como servidor HTTP (default)
  --stdio                                   como servidor Stdio (para Claude/Cursor)
```

---

## 2. Ensamblar el contexto — `mova run`

Junta agents + skills + prompt + memoria + focus y lo imprime por stdout — listo para pegar en un chat o mandar a una API.

```bash
mova run mi-proyecto revisar-auth
```

`task` es opcional: si el proyecto define `"default_task"`, se usa esa; si el proyecto declara exactamente UNA task y no se indica ninguna, se usa automáticamente esa única task — sin fricción para el caso común de un proyecto, una task. Con dos o más tasks y ni una explícita ni un `default_task`, Mova sigue sin poder adivinar cuál y pregunta, listando las tasks disponibles.

**La validación de Budget siempre corre primero, sin importar cuál de los casos anteriores aplique** — incluso "solo el nombre del proyecto, sin ninguna task". Si el contexto ensamblado (agents+skills+prompt+focus+memoria) supera el `budget` configurado para el proyecto/task, no se imprime nada más que el mismo bloque de ERROR/Sugerencia que muestra `mova budget`:

```text
$ mova run mi-proyecto
ERROR

Current context (128,400 tokens) exceeds the configured limit (100,000).

Suggestion:
Use --focus to reduce the included files.
```

Esto es exactamente lo que hacen también el tool `get_full_context` de MCP y `chat_completion` — ver [§10](#10-mcp-model-context-protocol) — así que ya sea que el contexto lo lea `mova run`, `mova chat`, o un modelo externo por MCP (Claude Console, Codex, Gemini, o cualquier cliente MCP), la misma validación de Budget corre antes, siempre, sin ningún paso extra que recordar.

Si la task tiene `focus` (en `project.json` o global), esa sección se agrega automáticamente al final — ver la sección siguiente.

---

## 3. Focus — trabajar sobre una parte del proyecto

`focus` (en `project.json`, global o por task) le dice al motor que trabaje solo sobre ciertos archivos, carpetas o símbolos, en vez de todo el repo. Funciona igual con o sin CLI: si un modelo lee `workflow.md` directo, la sección `## FOCUS` de la especificación explica cómo resolverlo.

**Importante:** `focus` es relativo al campo `"repo"` de `project.json`, nunca a la raíz de `mova-context`. Si `task.focus` está definido, **reemplaza por completo** el `focus` global (no se suman ambas listas).

### Cómo matchea cada item — igual que SQL `LIKE`

| Pasada | Equivalente SQL | Cuándo se usa |
|---|---|---|
| 1 — Exacta | `WHERE nombre = 'CreateOrder'` (por límite de palabra) | Siempre se intenta primero |
| 2 — Contiene | `WHERE nombre ILIKE '%CreateOrder%'` | Solo si la pasada 1 no encontró nada |

Insensible a mayúsculas/acentos en ambas pasadas (`articulo 6` encuentra `Artículo 6`). Nunca usa un LLM — es búsqueda de texto determinista: mismo input, mismo resultado, siempre.

### Tipos de item soportados

| Item en `focus` | Qué resuelve |
|---|---|
| `"manual.md"` | el archivo completo, por nombre en todo el repo |
| `"src/auth"` | índice del directorio |
| `"CreateOrder()"` | la función/método/clase (`()` le dice al motor que es código, no un archivo) |
| `"Artículo 6"` | la sección de un documento legal (Título, Capítulo, Sección, Artículo, Inciso) |
| `"## Alguna sección"` | un heading de Markdown |
| `"nombre_tabla"` | la definición `CREATE TABLE ...;` en un `.sql` |

### Ejemplo

```json
"tasks": {
  "revisar-orden": {
    "prompt": "review-project",
    "focus": ["CreateOrder()", "manual.md", "Artículo 6"]
  }
}
```

```bash
mova run mi-proyecto revisar-orden
```

Si un item no se encuentra en ninguna pasada, aparece como `not found: [item]` — nunca se omite en silencio.

### Focus sobre todo el repositorio o directorios completos

Además de archivos/símbolos/secciones individuales, `focus` también acepta patrones glob y rutas de directorio simples:

**Todo el repositorio:**

```json
"focus": ["**/*"]
```

Activa el `GlobResolver`: recorre de forma recursiva todos los archivos y directorios bajo `repo`. Útil cuando una tarea realmente necesita todo el código y se prefiere ser explícito en vez de dejar `focus` sin definir:

```json
"tasks": {
  "revisar-backend": {
    "prompt": "review-project",
    "focus": ["**/*"]
  }
}
```

**Solo la raíz del proyecto:**

```json
"focus": ["."]
```
o
```json
"focus": ["./"]
```

Activa el `DirectoryResolver` sobre la raíz de `repo` — un índice de directorio de lo que hay directamente ahí, con el mismo resolver que se usa para cualquier otro directorio.

**Directorios específicos:**

```json
"focus": ["src/", "pkg/", "cmd/"]
```

Cada ruta se resuelve mediante el `DirectoryResolver`. Esta es la forma habitual de mantener fuera del contexto ensamblado directorios pesados — `node_modules`, `vendor`, `.git`, `dist`, `build` — listando solo los directorios que realmente importan en vez de `"**/*"`: lo que no se lista simplemente nunca se recorre.

### Checklist si `focus` sale vacío

1. ¿`"repo"` apunta a una carpeta que existe y contiene lo que se busca? (`focus` nunca busca fuera de `repo`)
2. ¿El símbolo de código lleva `()` al final?
3. ¿La `task` tiene su propio `focus`? Si sí, reemplaza al global.
4. ¿Tu binario es anterior a este fix? Recompilá: `go build -o mova ./src/cli`.

---

## 4. Memoria

```bash
mova memory mi-proyecto "$(cat respuesta.txt)"
```

Extrae el bloque ` ```memory ` de la respuesta de un modelo y lo agrega a `memory.md`. La próxima vez que corras `mova run mi-proyecto`, esa memoria aparece sola en el contexto.

```bash
mova memory-read mi-proyecto --all
mova memory-read mi-proyecto --month 2024-01

mova memory-archive mi-proyecto --days 15     # mueve entradas viejas fuera de memory.md, por mes

mova memory-clear mi-proyecto --archived --yes   # pide confirmación salvo --yes

mova memory-config mi-proyecto days 45
```

---

## 5. Proyectos — list, init, search

```bash
mova list                       # lista todos los proyectos
mova init mi-proyecto           # crea project.json (plantilla mínima) + memory.md vacío
mova search "autenticación" software   # busca en agents/skills/prompts por palabra clave
```

`search` no usa ningún modelo — es búsqueda por palabra clave sobre el conocimiento del repositorio.

---

## 6. Modelos y proveedores

`llm_profile` (en `project.json`) es lo único que cambia al pasar de un modelo/proveedor a otro. Agents, skills, prompts, memoria y focus **nunca** cambian — el mismo `mova run` genera el mismo contexto sin importar qué modelo lo va a leer.

```json
"llm_profile": { "type": "local", "provider": "ollama", "config": "llama3.2.3b" }
```

| Campo | Valores | Para qué sirve |
|---|---|---|
| `type` | `"powerful"` (default) \| `"local"` | Con `"local"` el motor adapta el formato (listas numeradas, `INSTRUCTIONS:`) para que modelos chicos sigan mejor instrucciones secuenciales. Con `"powerful"` el contenido se entrega sin tocar. |
| `provider` | `"ollama"` \| `"google"` \| `"anthropic"` \| `"openai"` \| `"lmstudio"` \| lo que quieras | Un subdirectorio de `config/models/` — de ahí sale toda la configuración de este modelo. |
| `config` | nombre de archivo, sin `.json` | Apunta a `config/models/<provider>/<config>.json` — el ÚNICO archivo con conexión (`base_url`, `api_key`, `timeout_seconds`) **y** parámetros de inferencia (`temperature`, `num_predict`, el tag real del modelo) juntos. |

**Una sola fuente de verdad.** `llm_profile.config` y el nombre del archivo son el mismo string, por construcción — no hay dos identificadores que puedan desincronizarse (el bug clásico era nombrar el archivo distinto al tag real de Ollama, por ejemplo por la restricción de `:` en nombres de archivo de Windows).

```json
// config/models/ollama/llama3.2.3b.json — TODO en un solo archivo
{
  "provider": "ollama", "type": "ollama",
  "base_url": "http://localhost:11434", "timeout_seconds": 300,
  "model": "llama3.2:3b",
  "top_k": 40, "top_p": 0.9, "num_ctx": 4096,
  "temperature": 0, "num_predict": 512,
  "context_window": 131072, "repeat_penalty": 1.1
}
```

### Cambiar de proveedor sin tocar nada más

```json
// Claude — Cloud, vía API
"llm_profile": { "type": "powerful", "provider": "anthropic", "config": "claude-sonnet-4-6" }

// Gemini Flash — Cloud, vía API
"llm_profile": { "type": "powerful", "provider": "google", "config": "gemini-2.5-flash" }

// Ollama local
"llm_profile": { "type": "local", "provider": "ollama", "config": "llama3.2.3b" }
```

```bash
mova run mi-proyecto mi-task > contexto.txt
ollama run llama3.2:3b < contexto.txt
```

### Estructura de `config/models/`

```text
config/models/
├── active.json              ← puntero: {"provider", "config"} elegidos ahora (nunca copia datos)
├── ollama/
│   ├── llama3.2.3b.json     ← conexión + parámetros de ESTE modelo, todo junto
│   └── mistral.json
├── google/gemini-2.5-flash.json
├── anthropic/claude-sonnet-4-6.json
└── lmstudio/                ← proveedor elegido, todavía sin modelo
```

Editar cualquier `<modelo>.json` a mano se recarga **en caliente**: el próximo mensaje de chat (o la próxima llamada MCP/HTTP) ya usa los valores nuevos, sin reiniciar nada.

```bash
mova config ollama                    # elige el proveedor activo
mova show config                      # ¿qué proveedor/modelo estoy usando?
mova show config llama3.1             # config completa de un modelo puntual
mova install llama3.1,mistral,phi3    # descarga modelos de Ollama (con progreso)
mova model-list                       # modelos instalados en el proveedor activo
mova remove mistral                   # elimina un modelo instalado
```

`install`/`model-list`/`remove` usan la API nativa de Ollama. Con LM Studio/vLLM/Cloud, se instala o contrata el modelo desde su propia herramienta y solo se crea su `.json`. `mova install` de un modelo nuevo copia la conexión de cualquier modelo hermano existente para ese proveedor.

### Proveedores Cloud reales (OpenAI, Anthropic, Google)

`config/models/<provider>/<config>.json` es el único lugar que hay que tocar. **OpenAI y Google Gemini** exponen un endpoint compatible con `"openai-compatible"` (el mismo formato que ya usaba Mova para LM Studio/vLLM):

```json
// config/models/openai/gpt-5.json
{
  "provider": "openai", "type": "openai-compatible",
  "base_url": "https://api.openai.com", "api_key": "sk-...",
  "model": "gpt-5", "temperature": 0.2, "num_predict": 1024
}
```

```json
// config/models/google/gemini-2.5-flash.json
{
  "provider": "google", "type": "openai-compatible",
  "base_url": "https://generativelanguage.googleapis.com/v1beta/openai",
  "api_key": "AIza...", "model": "gemini-2.5-flash",
  "temperature": 0.2, "num_predict": 1024
}
```

**Anthropic (Claude)** tiene headers y forma de respuesta distintos, así que usa su propio tipo nativo:

```json
// config/models/anthropic/claude-sonnet-4-6.json
{
  "provider": "anthropic", "type": "anthropic",
  "base_url": "https://api.anthropic.com", "api_key": "sk-ant-...",
  "model": "claude-sonnet-4-6", "temperature": 0.2, "num_predict": 1024
}
```

Agregar un proveedor Cloud nuevo en el futuro es exactamente esto: un `.json` nuevo, y si su API no es compatible con el formato OpenAI, una implementación nueva de la interfaz `Provider` (`models/provider.go`) — nunca tocar `core`, `budget`, `cli` ni `mcp`. Mova llama a estos proveedores directo por HTTP, sin depender de Claude Desktop, Claude Console, Codex ni Gemini CLI.

---

## 7. Chat con un modelo local o Cloud

```bash
mova chat
> set -model llama3.1
✓ modelo cambiado a: llama3.1 (proveedor: ollama)
> hola, revisa el módulo de auth
[llama3.1] ...
> set -model mistral
✓ modelo cambiado a: mistral (proveedor: ollama)
> sigue con lo mismo
[mistral] ...          # el historial se conserva al cambiar de modelo
> exit
```

Si se pasa `[project]` (y opcionalmente `[task]`), el chat carga el **mismo contexto completo** que arma `mova run` (agents+skills+prompt+memoria+focus) como mensaje de sistema. Comandos disponibles dentro del chat, siempre:

| Comando | Qué hace |
|---|---|
| `/memory` | guarda el último intercambio en `memory.md` |
| `/budget` | genera `mova-budget-report.md` para el proyecto activo |
| `/save "ruta/archivo.ext"` | guarda la última respuesta ahí — formato por extensión (ver [§9](#9-save--crear-o-editar-cualquier-archivo-o-directorio)) |
| `/save -d "ruta/carpeta"` | crea solo un directorio |
| `/tools` | lista comandos + las tools que el modelo puede invocar (si el proyecto las habilitó) |

### Vía MCP / HTTP — tool `chat_completion`

La misma sesión de chat es una tool MCP, y por lo tanto también HTTP:

```bash
mova mcp start --port 3000
```

```bash
curl -X POST http://localhost:3000/mcp -H "content-type: application/json" -d '{
  "jsonrpc":"2.0","id":1,"method":"tools/call",
  "params":{"name":"chat_completion","arguments":{
    "model":"llama3.1","project":"mi-proyecto","task":"revisar-auth",
    "message":"¿qué revisarías primero?"
  }}
}'
```

`model`, `project` y `task` son opcionales — sin `project` es un chat "vacío" con ese modelo; sin `model` usa el activo en `active.json`.

### Cómo encaja todo

```text
config/models/<provider>/<config>.json ──► ConfigCache (recarga en caliente)
      (conexión + parámetros, un solo archivo)
        ┌───────────────────────────────────┬───────────────────────────────┐
        ▼                                   ▼                               ▼
   mova chat (REPL)               tool MCP chat_completion            (mismo tool)
        │                                   │                          vía HTTP /mcp
        └──────────────────────► models.Session.Send() ◄────────────────────┘
                                            │
                                  models.Provider (interfaz)
                                    ├─ ollamaProvider   → POST /api/chat
                                    ├─ openAIProvider   → POST /v1/chat/completions (LM Studio, vLLM, OpenAI, Gemini)
                                    └─ anthropicProvider → POST /v1/messages (Claude)
```

Un único cliente HTTP compartido (keep-alive + pool de conexiones) atiende todas las llamadas, pensado para alto volumen.

---

## 8. Crear archivos hablando: lenguaje natural en el chat

Además de `/save`, es posible simplemente **pedirlo con las propias palabras**, en el mismo mensaje del chat — sin memorizar ningún comando.

```text
> Genera el informe de auditoría en docs/auditoria.pdf
[el modelo escribe el informe, y la respuesta se guarda sola en docs/auditoria.pdf]

> Crea el directorio informes/2026 y genera el resumen en informes/2026/resumen.docx
[se crea la carpeta, y después se guarda el .docx con la respuesta]

> Guarda el análisis en salida/analisis.txt
[se guarda tal cual, sin pasar por el modelo si ya se tiene el texto — ver detalle abajo]
```

### Cómo funciona por dentro

Es un detector **heurístico por regex**, no un modelo de lenguaje ni una IA aparte — vive en `documents/nl_intent.go` y lo comparten `mova chat` y `chat_completion` (MCP/HTTP). No hay nada misterioso: el mensaje se separa en cláusulas por la palabra **y** / **and**, y en cada cláusula se busca un verbo de creación:

| Español | Inglés |
|---|---|
| genera, generar, generame/generáme | generate |
| crea, crear, creame/creáme | create |
| guarda, guardar | save |
| — | make |

Se pueden usar tantas veces como se quiera en un mensaje, mezclando español e inglés en la misma sesión.

Si encuentra el verbo, busca **o bien** (a) una palabra clave de directorio + una ruta, **o bien** (b) un token con forma de ruta terminado en extensión (`.pdf`, `.md`, `.docx`, etc.):

- **Solo directorio** ("Crea el directorio X") → se crea al instante, sin llamar al modelo.
- **Solo archivo** ("Genera reporte.pdf") → el mensaje sí va al modelo (necesita generar el contenido), y la respuesta se guarda sola en esa ruta.
- **Ambos en un mensaje** ("Crea el directorio X y genera Y") → primero el directorio, después el archivo.

### Límites reales (para no esperar más de lo que hay)

- Necesita el verbo **y** (palabra clave de carpeta o una ruta con extensión) en la **misma cláusula** — *"Genera algo interesante sobre X"* no dispara nada, porque no hay ruta ni extensión reconocible.
- Solo entiende **y**/**and** como conector entre dos pedidos — no maneja construcciones más complejas ("primero... después...", comas, etc.).
- Es determinístico: mismo texto, mismo resultado, siempre — no interpreta intención más allá del patrón.

`/save` sigue funcionando exactamente igual que siempre, para cuando esta heurística no alcance o quieras control exacto sobre la ruta.

---

## 9. `save` — crear o editar cualquier archivo o directorio

Antes existían tools distintas para cada formato (`generate_word_contract`, `generate_pdf_document`, `generate_excel_report`, `write_file`...), cada una con su propio nombre de argumento para el contenido — fácil de confundir, y causa real de errores como un `.docx` vacío. `save` reemplaza todo eso con un único punto de entrada: se indica `path` (o `directory`) + `content`, y Mova elige internamente qué generador usar según la extensión.

**Desde el chat:**

```text
> Audita el checkout y arma el informe de correcciones
[llama3.1] (la respuesta con el informe completo)

> /save "informes/checkout-corregido.md"
[Save] ✓ archivo guardado: examples/ejemplo-ley21719-repo/informes/checkout-corregido.md

> /save -d "carpeta  dos"
[Save] ✓ directorio creado: examples/ejemplo-ley21719-repo/carpeta  dos
```

`/save -d "carpeta  dos"` crea exactamente ese nombre de directorio, incluidos los espacios dobles, igual en Windows, Linux y macOS — el nombre nunca se normaliza ni se recorta.

Por default, `/save` usa la ÚLTIMA respuesta del modelo como contenido. Varios flags cambian QUÉ texto se guarda y cómo:

| Flag | Guarda... |
|---|---|
| *(ninguno)* | la última respuesta del modelo (comportamiento por default, sin cambios) |
| `-all` | toda la conversación hasta el momento, como transcripción `### You` / `### Assistant` |
| `-range N-M` | los intercambios N a M, 1-indexado e inclusive, con el mismo formato de transcripción |
| `-c` | únicamente los bloques de código (` ``` `) de lo seleccionado arriba — cualquier lenguaje: Go, Python, Java, C#, Rust, JavaScript, TypeScript, Kotlin, SQL, YAML, JSON, XML, Bash, y más. No se guarda texto adicional. |
| `-text` | lo opuesto a `-c` — únicamente el texto, sin los bloques de código |

Y, por separado, cómo se maneja un archivo existente:

| Flag | Efecto |
|---|---|
| *(ninguno)* | sobreescribe un archivo existente (comportamiento por default, sin cambios) |
| `-overwrite "notes.txt"` | sobreescritura explícita — igual que el default, útil para scripts que siempre quieren declarar su intención |
| `-no-overwrite "notes.txt"` | falla con un mensaje claro en vez de sobreescribir si el archivo ya existe |
| `-append "notes.txt"` | agrega el contenido seleccionado al final del archivo existente |

Estos se combinan libremente: `/save -all -c "src/todos_los_snippets.go"` guarda todos los bloques de código de toda la conversación; `/save -range 2-4 -text "resumen.md"` guarda únicamente el texto de los intercambios 2 a 4.

**El lenguaje natural también funciona**, para overwrite/no-overwrite, en cualquiera de los dos idiomas:

```text
> Sobreescribe reporte.pdf
```

es exactamente `/save -overwrite "reporte.pdf"`, y

```text
> No sobreescribas reporte.pdf
```

es exactamente `/save -no-overwrite "reporte.pdf"`. Mismo detector, mismo resultado, ya sea que se escriba el flag o se diga de forma natural — ver `documents/save_modifiers.go`.

**Vía MCP / HTTP** — la misma tool, alcanzable por stdio, HTTP, o `chat_completion`:

```bash
mova mcp start --port 3000
```

```bash
curl -X POST http://localhost:3000/mcp -H "content-type: application/json" -d '{
  "jsonrpc":"2.0","id":1,"method":"tools/call",
  "params":{"name":"save","arguments":{
    "project":"mi-proyecto",
    "path":"informes/checkout-corregido.pdf",
    "content":"Hallazgo 1: ...\nHallazgo 2: ..."
  }}
}'
```

```json
{"name":"save","arguments":{"project":"mi-proyecto","directory":"informes/2026"}}
```

Un caller de MCP/HTTP que mantiene su propia conversación (una interfaz de chat, un script) puede usar exactamente las mismas selecciones de actual/rango/conversación completa/solo-código/solo-texto que `/save` soporta en el chat, pasando `history` en vez de `content`:

```json
{"name":"save","arguments":{
  "history":[{"role":"user","content":"..."},{"role":"assistant","content":"..."}],
  "mode":"range",
  "range":"2-4",
  "code_only":true,
  "path":"src/snippets.go"
}}
```

`mode` es `"all"`, `"range"` (con `range: "N-M"`), o se omite para tomar solo el último intercambio — misma lógica de selección que `-all`/`-range` en el chat (ver `documents/save_selection.go`; hay una única implementación, compartida por todas las puertas de entrada).

**Vía HTTP directo** — un endpoint `POST /save` dedicado, mismo cuerpo JSON, misma lógica interna sin duplicación:

```bash
curl -X POST http://localhost:3000/save -H "content-type: application/json" -d '{
  "path":"informes/checkout-corregido.xlsx",
  "content":"item,riesgo\nconsentimiento agrupado,alto"
}'
```

### Argumentos de `save`

| Argumento | Para qué |
|---|---|
| `path` | archivo a crear/editar — su extensión elige el formato. Se resuelve con la misma lógica de siempre: ruta absoluta tal cual, ruta relativa bajo el `repo` del proyecto, o nombre suelto (busca coincidencias, pregunta si hay más de una) |
| `directory` | en vez de `path` — solo crea la carpeta (y padres faltantes); no interviene ningún Writer |
| `content` | texto/Markdown/HTML/CSV — el Writer interno decide cómo convertirlo (ver tabla abajo) |
| `history` | en vez de `content` — un array JSON de mensajes `{"role","content"}` para seleccionar (ver arriba) |
| `mode` | `"all"` \| `"range"` \| se omite (último intercambio) — solo se usa junto con `history` |
| `range` | `"N-M"`, 1-indexado e inclusive — solo se usa con `mode:"range"` |
| `code_only` / `text_only` | booleanos — igual que `-c`/`-text` en el chat |
| `overwrite` | `false` explícito → `save` rechaza el pedido en vez de sobreescribir. Sin este argumento, sigue sobreescribiendo como siempre |
| `append` | `true` → suma el contenido al final del archivo existente (formatos de texto) |
| `project` | resuelve rutas relativas contra el `repo` de ese proyecto |

### Cómo `save` interpreta `content` según la extensión

| Extensión | Qué hace `save` |
|---|---|
| `.md`, `.txt`, `.json`, `.yml`/`.yaml`, `.xml`, `.csv`, y ~20 lenguajes de código más | se escribe tal cual |
| `.docx` | se interpreta como Markdown (`#`/`##`/`###`, **negrita**, párrafos) |
| `.pdf` | si ya parece HTML se usa tal cual; si es texto plano o Markdown, se envuelve en tags `<p>` antes de generar el PDF |
| `.xlsx` | acepta `sheets_data` JSON tipado, **o** texto CSV/TSV plano (una única hoja "Sheet1", cada celda tipada automáticamente) |
| `.svg` | espera código SVG válido |

### Resaltado de sintaxis

Cada vez que el chat, MCP o HTTP generan código fuente, cualquier bloque de código sin tag de lenguaje recibe uno automáticamente — el mismo detector (`documents.DetectLanguage`/`AutoTagCodeFences`) corre en todos lados, así que un bloque Go, una consulta SQL o un archivo YAML se resaltan correctamente sin importar qué puerta de entrada los generó. En la terminal, `mova chat` renderiza ese Markdown (incluido el resaltado) con `glamour`; por MCP/HTTP, la respuesta vuelve como Markdown correctamente etiquetado para que lo renderice el cliente que sea.

### `delete` — eliminar archivos y directorios

Un único comando unificado elimina archivos y directorios, de forma idéntica desde el chat, MCP y HTTP — nunca se elimina nada sin confirmación.

```text
> /delete "informes/borrador-viejo.md"
Delete "borrador-viejo.md"? (Y/N)
y
✓ deleted: examples/ejemplo-ley21719-repo/informes/borrador-viejo.md
```

```text
> /delete "a.txt" "b.txt" "logs/"
Delete "a.txt"?
(Y/N)
y
Delete "b.txt"?
(Y/N)
n
Delete "logs/"?
(Y/N)
y
✓ deleted: .../a.txt
⚠ not found, skipped: b.txt
✓ deleted: .../logs
```

Una `/` al final (`"logs/"`) es una pista de que el destino es un directorio; sin ella, `/delete` le pregunta al sistema de archivos qué es realmente en vez de adivinar.

**Vía MCP / HTTP** — como no hay una terminal donde escribir Y/N, la confirmación es explícita: se llama una vez sin `confirm` para recibir de vuelta el texto exacto del prompt, y se llama de nuevo con `confirm:true` una vez que la persona está de acuerdo — la misma convención que ya usa `apply_edits` de `chat_completion` para ediciones en lenguaje natural.

```json
{"name":"delete_path","arguments":{"paths":"a.txt,b.txt,logs/","project":"mi-proyecto"}}
```
```json
{"name":"delete_path","arguments":{"paths":"a.txt,b.txt,logs/","project":"mi-proyecto","confirm":true}}
```

```bash
curl -X POST http://localhost:3000/delete -H "content-type: application/json" -d '{
  "path":"informes/borrador-viejo.md","confirm":true
}'
```

### `workflow.md` — ejecución con validación de Budget

`workflow.md` nunca se abre directamente. Decir cualquiera de las siguientes frases resuelve primero el proyecto, construye su contexto, y valida el resultado contra su Budget configurado — solo si esa validación pasa, `workflow.md` se carga de verdad:

```text
lee workflow.md
leer workflow.md
ejecuta workflow.md
run workflow.md
execute workflow.md
workflow.md <project>
workflow.md <project> <task>
```

**La forma más simple posible, una línea, igual en todos lados:**

| Dónde | Qué decir/enviar |
|---|---|
| `mova chat` (este CLI) | escribir `workflow.md mi-proyecto` |
| Claude Console, Codex, Gemini, o cualquier cliente MCP | pedirle que lea el workflow — el modelo llama solo al tool `get_workflow` con `{"project":"mi-proyecto"}`; no hace falta ninguna frase especial de tu lado |
| HTTP | `curl -X POST http://localhost:3000/workflow -d '{"project":"mi-proyecto"}'` |

Un solo llamado, una sola puerta de entrada, siempre validado contra el Budget primero — no hay ningún paso separado de "primero valido el budget, después cargo el archivo" que recordar en ninguno de los casos.


```text
> workflow.md mi-proyecto
[Project] Loading project configuration...
[Project] Using configured provider...
[Context] Building context...
[Dedup] No duplicates found.
[Focus] No focus configured — using the full agents/skills/prompt context.
[Workflow] Loaded /path/to/workflow.md (4,102 tokens).

(el contenido de workflow.md, renderizado)
```

Si el contexto estimado (agents + skills + prompt + focus + el propio `workflow.md`) supera el Budget configurado para el proyecto/tarea, `workflow.md` **no** se carga — se imprime el mismo bloque de ERROR/Sugerencia que ya muestra `mova budget`:

```text
[Project] Loading project configuration...
[Project] Using configured provider...
[Context] Building context...
[Dedup] Removed 3 duplicated paragraph(s) (512 chars).
[Focus] No focus configured — using the full agents/skills/prompt context.

ERROR

Current context (128,400 tokens) exceeds the configured limit (100,000).

Suggestion:
Use --focus to reduce the included files.
```

El mismo pipeline corre desde MCP (tool `get_workflow`) y HTTP (`POST /workflow`):

```json
{"name":"get_workflow","arguments":{"project":"mi-proyecto","task":"revisar-backend"}}
```
```bash
curl -X POST http://localhost:3000/workflow -H "content-type: application/json" -d '{
  "project":"mi-proyecto"
}'
```

Qué archivo `workflow.md` se usa depende de `workflow_path` en `project.json` (ver [PROJECT_JSON.md](PROJECT_JSON.md)); una vez configurado, Mova siempre usa exactamente ese archivo y nunca busca otro.

| Extensión | Qué hace `save` |
|---|---|
| `.md`, `.txt`, `.json`, `.yml`/`.yaml`, `.xml`, `.csv`, y ~20 lenguajes de código más | se escribe tal cual |
| `.docx` | se interpreta como Markdown (`#`/`##`/`###`, **negrita**, párrafos) |
| `.pdf` | si ya parece HTML se manda tal cual; si es texto plano o Markdown, se envuelve en `<p>` antes de generar el PDF |
| `.xlsx` | acepta el JSON tipado de `sheets_data`, **o** texto CSV/TSV plano (una hoja "Sheet1", cada celda se tipa sola) |
| `.svg` | se espera código SVG válido |


---

## 10. Tool-calling autónomo desde el chat

Agrega esto a `project.json` para que el propio modelo (Ollama local, Gemini, Claude, GPT — cualquiera) pueda pedirle a Mova que ejecute una acción real durante la conversación, en vez de solo describir en texto lo que haría:

```json
{ "tools": { "enabled": true } }
```

Con eso, `mova chat` y `chat_completion` agregan al mensaje de sistema un protocolo simple en texto plano, que funciona igual para cualquier proveedor (no depende de "function calling" nativo, que un modelo chico como `llama3.2:3b` suele manejar mal): el modelo responde con `<<<MOVA_TOOL_CALL>>> {"name":"save", "arguments":{...}} <<<END_MOVA_TOOL_CALL>>>`, Mova lo ejecuta de verdad, y le devuelve el resultado real para que termine su respuesta.

```json
{ "tools": { "enabled": true, "allow": ["save", "read_file"] } }
```

`allow` es opcional — restringe a un subconjunto (`save`, `read_file`, `patch_file`, `read_document_layer`); si se omite, las cuatro están habilitadas. Esto es **además** de `/save`: `/save` siempre funciona sin depender de `tools.enabled` ni de que el modelo coopere (es la vía determinística); `tools.enabled` es para cuando se quiere que el modelo mismo decida cuándo crear o editar algo, de forma autónoma.

---

## 11. Documentos de oficina y medios (PDF, Word, Excel, SVG, imágenes)

Con `save` no hace falta acordarse de qué tool genera qué formato — alcanza con la extensión de `path`. No requieren ningún paquete adicional: `.docx`, `.xlsx` y `.pdf` se escriben a mano con la librería estándar de Go. Solo `trigger_diffusion_image` necesita un servidor de difusión aparte.

### Ejemplos simples, en lenguaje natural

```text
> Genera un contrato simple de arriendo en salida/contrato.docx
[Save] ✓ documento Word generado: salida/contrato.docx

> Arma una tabla con los gastos del mes en salida/gastos.xlsx
[Save] ✓ hoja de cálculo generada: salida/gastos.xlsx

> Escribe un resumen ejecutivo de una página en informes/resumen.pdf
[Save] ✓ PDF generado: informes/resumen.pdf

> Guarda las notas de la reunión en notas/reunion-23-julio.txt
[Save] ✓ archivo guardado: notas/reunion-23-julio.txt
```

### Ejemplos vía MCP/HTTP

Generar un Word con `save` y volver a leerlo:

```bash
mova mcp start --port 3000
```

```bash
curl -X POST http://localhost:3000/mcp -H "content-type: application/json" -d '{
  "jsonrpc":"2.0","id":1,"method":"tools/call",
  "params":{"name":"save","arguments":{
    "path":"salida/contrato.docx",
    "content":"# Contrato\n\nEste es un **párrafo** de prueba."
  }}
}'
```

```bash
curl -X POST http://localhost:3000/mcp -H "content-type: application/json" -d '{
  "jsonrpc":"2.0","id":2,"method":"tools/call",
  "params":{"name":"read_document_layer","arguments":{"filename":"salida/contrato.docx"}}
}'
```

Una hoja de Excel desde CSV plano (no hace falta JSON tipado si no se necesita):

```json
{"name":"save","arguments":{"path":"salida/reporte.xlsx","content":"Item,Monto\nCafé,4.5\nTe,3.2"}}
```

O con el `sheets_data` tipado de siempre (evita ambigüedades de tipo):

```json
{"name":"generate_excel_report","arguments":{
  "filename":"salida/reporte.xlsx",
  "sheets_data":{"Gastos":[
    [{"type":"string","value":"Item"},{"type":"string","value":"Monto"}],
    [{"type":"string","value":"Café"},{"type":"number","value":4.5}]
  ]}
}}
```

Todas las tools resuelven `path`/`filename` relativo al `repo` del proyecto (si se pasa `project`).

### Instalación de paquetes necesarios

```bash
# .docx, .xlsx, .pdf y .svg: ninguna dependencia adicional —
# se generan con la librería estándar de Go.
go build -o mova ./src/cli
```

`read_document_layer` sobre `.pdf` es de mejor esfuerzo (extrae texto de streams FlateDecode); PDFs escaneados o con codificaciones exóticas pueden no devolver texto.

### Modelo local para imágenes (`trigger_diffusion_image`)

Esta tool no genera imágenes por sí misma: enruta el prompt a un servidor de difusión local compatible con la API de **AUTOMATIC1111** (`/sdapi/v1/txt2img`), configurado en `config/models/diffusion/config.json` — un archivo aparte porque un servidor de difusión no tiene parámetros de inferencia tipo `temperature`/`num_predict` que fusionar. Se necesita ese servidor corriendo aparte, con un modelo de difusión instalado (Stable Diffusion 1.5, SDXL, vía AUTOMATIC1111 o ComfyUI en modo API). Mova solo hace la llamada HTTP y guarda el PNG resultante.

---

## 12. Archivos de texto y código

Tres tools cubren texto plano, config y código fuente — `.js`, `.ts`, `.py`, `.go`, `.cs`, `.java`, `.php`, `.rb`, `.rs`, `.c`, `.cpp`, `.h`, `.kt`, `.swift`, `.sh`, `.html`, `.css`, `.sql`, `.csv`, `.toml`, `.ini`, `.env`, `.log`, y más (lista completa en [SUPPORTED_FORMATS.md](SUPPORTED_FORMATS.md)):

| Tool | Para qué 
|---|---|---|
| `read_file` | leer un archivo 
| `write_file` | escribir un archivo 
| `patch_file` | reemplazar una única aparición exacta de `search` por `replace` 

Pedir una extensión no soportada (`.exe`, `.bin`) devuelve:

```
Unsupported file type: .exe. Supported extensions: .c, .cpp, .cs, .css, .csv, ...
```

`write_file`/`save` validan `.json`/`.xml` (bien formados), `.go` (sintaxis real vía `go/parser`) y `.csv` (columnas consistentes) antes de escribir. `patch_file` rechaza el cambio si `search` no aparece, o aparece más de una vez — nunca arriesga una edición ambigua.

### Ejemplo — "crea un archivo con tal cosa" desde el chat

Si se pide *"crea un NOTAS.md en mi proyecto con el estado actual"*, el asistente resuelve la ruta contra el `repo` del proyecto y llama a `save` — el archivo aparece en el directorio real, no en un chat efímero:

```bash
curl -X POST http://localhost:3000/mcp -H "content-type: application/json" -d '{
  "jsonrpc":"2.0","id":1,"method":"tools/call",
  "params":{"name":"save","arguments":{
    "project":"mi-proyecto","path":"NOTAS.md",
    "content":"# Notas del proyecto\n\nEstado: en progreso"
  }}
}'
```

```json
{"name":"patch_file","arguments":{
  "project":"mi-proyecto","filename":"NOTAS.md",
  "search":"Estado: en progreso","replace":"Estado: completado"
}}
```

```json
{"name":"read_file","arguments":{"project":"mi-proyecto","filename":"NOTAS.md"}}
```

Mismo patrón para `.json`, `.yml`, `.xml`, `.docx`, `.pdf` o `.xlsx` — solo cambia `path`/`filename` y `content`. Si el chat te da una ruta absoluta (ej. `E:/otros-proyectos/README.md`), todas estas tools la usan directo, sin pasar por `repo`.

---

## 13. Servidor MCP — `mova mcp start`

Mismo motor que `mova run`, expuesto por el protocolo MCP (JSON-RPC 2.0) — para que un cliente (Claude Desktop, Cursor) pida el contexto solo, sin que copies y pegues nada.

**Modo stdio** (el que usan Claude Desktop / Cursor):

```bash
mova mcp start --stdio
```

```json
{
  "mcpServers": {
    "mova-context": {
      "command": "/ruta/a/mova",
      "args": ["mcp", "start", "--stdio"],
      "env": { "MOVA_PROJECT_ROOT": "/ruta/a/tu/mova-context" }
    }
  }
}
```

**Modo HTTP** (para curl/Postman o tu propio backend):

```bash
mova mcp start --port 3000
```

```bash
curl -X POST http://localhost:3000/rpc -H "content-type: application/json" -d '{
  "jsonrpc":"2.0","id":1,"method":"tools/call",
  "params":{"name":"get_full_context","arguments":{"project":"mi-proyecto","task":"revisar-auth"}}
}'
```

### Tools disponibles vía MCP

| Tool | Equivale a |
|---|---|
| `get_full_context` | `mova run [project] [task]` |
| `get_knowledge` | leer un agent/skill/prompt puntual |
| `get_memory` | `mova memory-read [project]` |
| `get_memory_all` | `mova memory-read [project] --all` |
| `get_workflow` | leer `workflow.md`, pero solo después de resolver el proyecto y validar su Budget (ver [§9](#9-save--crear-o-editar-cualquier-archivo-o-directorio)) |
| `search_context` | `mova search "consulta" [dominio]` |
| `chat_completion` | `mova chat [project] [task]` |
| `save` | crear/editar cualquier archivo o directorio (ver [§9](#9-save--crear-o-editar-cualquier-archivo-o-directorio)) |
| `delete_path` | eliminar uno o más archivos/directorios, con confirmación (ver [§9](#9-save--crear-o-editar-cualquier-archivo-o-directorio)) |
| `estimate_budget` | `mova budget [project] [task]` |

### Resolución de raíz para clientes MCP

Los clientes MCP (Claude Desktop, Cursor) lanzan `mova` desde un directorio que normalmente no es tu proyecto — por eso el ejemplo de arriba fija `MOVA_PROJECT_ROOT`. Orden: `MOVA_PROJECT_PATH` (directo) → `MOVA_PROJECT_ROOT` (búsqueda hacia arriba desde ahí) → directorio de trabajo actual → directorio del binario.

---

## 14. Variables de entorno

```bash
MOVA_ADAPTER=db MOVA_DSN=postgres://user:pass@host/db mova run mi-proyecto
```

| Variable | Efecto |
|---|---|
| `MOVA_ADAPTER` | Sobreescribe `project.json.adapter` (`file` / `db`) |
| `MOVA_DSN` | Sobreescribe `project.json.dsn` |
| `MOVA_PROJECT_ROOT` | Punto de partida extra para la búsqueda de `workflow.md` hacia arriba |
| `MOVA_PROJECT_PATH` | Usa esta ruta como raíz directamente, sin búsqueda |

---

## 15. Tokenomics — `mova budget`

Estima cuántos tokens usaría el contexto real de un proyecto (agents + skills + prompt + focus + memory — lo mismo que arma `mova run`) y cuánto costaría en cada proveedor de `config/prices.json`. **Todo el cálculo es local**: no llama a ningún LLM ni API externa, no manda una sola línea del proyecto fuera de tu máquina, no usa base de datos, y no guarda prompts ni contenido en ningún lado.

```bash
mova budget mi-proyecto
mova budget mi-proyecto mi-task --focus
```

Genera `mova-budget-report.md` (ruta configurable vía `"budget_path"` en `project.json` — ver [PROJECT_JSON.md](PROJECT_JSON.md); por default `projects/<project>/mova-budget-report.md`) — siempre en inglés simple, para que quien paga la factura lo entienda sin depender del idioma del resto de Mova Context. Alcanzable idéntico desde CLI, MCP (`estimate_budget`) y el chat REPL (`/budget`):

```json
{"name":"estimate_budget","arguments":{"project":"mi-proyecto","task":"mi-task","focus":"true"}}
```

El reporte incluye, en este orden:

- **Tokenization** — qué encoding de tiktoken-go se usó.
- **Deduplication** — cuántos párrafos idénticos se quitaron y cuántos tokens ahorraron.
- **Token & Cost Breakdown** — una fila por Agents/Skills/Prompt/Focus/Memory/Overhead, en tokens y USD por proveedor, más el total.
- **Context Optimization** — solo con `--focus`: repo completo sin filtrar vs. solo lo que `focus` selecciona.
- **Budget Limit** — solo si `project.json` define `"budget"`.
- **Historical Token Accuracy** — el feedback loop (ver abajo).
- **Important** — el recordatorio de que esto es una estimación, no una factura.

### Deduplicación automática

Mova deduplica párrafos idénticos en **todo** el contexto ensamblado — no solo dentro de `focus`, sino cruzando Agents+Skills+Prompt+Focus+Memory entre sí. Nunca es una reformulación ni un resumen — solo texto idéntico (normalizado por espacios), nunca código/SQL/JSON:

```text
[Dedup] Removed 3 duplicated paragraphs (~450 tokens saved).
```

Aparece igual en `mova chat`, en `chat_completion`, y en la sección "Deduplication" del reporte — corre en cada ensamblado, sin configurar nada.

### Dos límites que suenan parecido pero no son lo mismo

| | `budget.max_tokens` (`project.json`) | `num_predict` (`config/models/<provider>/<config>.json`) |
|---|---|---|
| Limita | el **contexto ensamblado** — lo que se manda AL modelo | la **respuesta** del modelo — lo que genera de vuelta |
| Quién lo aplica | Mova, ANTES de mandar nada | el proveedor mismo, como parámetro de la request |
| Si se pasa | Mova corta la ejecución con un error, cero tokens gastados | el proveedor simplemente corta la respuesta ahí |

```json
// project.json
{ "budget": { "max_tokens": 6000 } }
```

```json
// config/models/google/gemini-2.5-flash.json
{ "num_predict": 1024 }
```

### Presupuesto como regla, no como sugerencia

Un límite en `project.json` (o por task, que sobreescribe al del proyecto) se valida **antes de enviar cualquier contexto** a un modelo — desde `mova chat`, `chat_completion`, y por lo tanto HTTP, siempre igual:

```json
{ "budget": { "max_tokens": 8000 } }
```

```text
ERROR
Current context (14,250 tokens) exceeds the configured limit (8,000).
Suggestion: Use --focus to reduce the included files.
```

`mova budget` (el reporte, que nunca envía nada a ningún modelo) solo lo muestra como informativo en "Budget Limit" — la ejecución real se corta en `mova chat`/`chat_completion`, antes de gastar un solo token de verdad.

### Feedback loop — cerrar el ciclo con la realidad

Cada vez que `mova chat` o `chat_completion` mandan el contexto a un proveedor **Cloud real** (OpenAI, Anthropic, Google) y ese proveedor devuelve cuántos tokens contó, Mova acumula esa diferencia en `mova-token-history.json` (junto al `project.json` por default, o en la ruta de `"token_history_path"` en `project.json`; ver [PROJECT_JSON.md](PROJECT_JSON.md)). El archivo **solo** guarda dos números por proveedor — nunca prompts, respuestas ni contenido:

```json
{
  "anthropic": { "total_local_tokens": 120000, "total_api_tokens": 122760 },
  "google": { "total_local_tokens": 85000, "total_api_tokens": 85150 }
}
```

`mova budget` lee este archivo y calcula `(API - Local) / Local * 100` por proveedor, mostrando la desviación promedio en "Historical Token Accuracy" — un proveedor sin datos muestra `No historical data`, nunca un error. Cuantas más llamadas reales, más preciso el número para **ese proyecto específico** — una calibración propia, no un benchmark genérico.

### Herramienta de conteo

Todo el conteo se hace con **[tiktoken-go](https://github.com/tiktoken-go/tokenizer)**, embebida (sin llamadas de red), el mismo tokenizador que OpenAI publica y usa en su API. Para modelos OpenAI el conteo suele ser exacto. Para Claude y Gemini no existe un tokenizador local oficial público, así que se reutiliza el mismo encoding como aproximación — la diferencia real suele ser chica, y el feedback loop la va acotando con el tiempo por proyecto.

### Configurar precios (`config/prices.json`)

`config/prices.json` guarda **únicamente** precios de modelos — configuración global, compartida por todos los proyectos. Dónde escribe cada proyecto su propio `mova-budget-report.md` o `mova-token-history.json` es configuración particular de ese proyecto, y vive exclusivamente en su `project.json` (`budget_path`, `token_history_path` — ver [PROJECT_JSON.md](PROJECT_JSON.md)), nunca acá.

```json
{
  "currency": "USD",
  "exchange_rate_clp": 950,
  "unit": "per_1k_tokens",
  "providers": {
    "google": { "models": { "gemini-2.5-flash": { "input": 0.0003, "output": 0.0025 } } }
  }
}
```

Se recarga en caliente (mismo mecanismo que `config/models/`). Agregar un proveedor o modelo nuevo es solo JSON, nunca código. Los valores de ejemplo hay que actualizarlos con los precios reales y vigentes de cada proveedor.

### Ejemplo completo — Gemini Flash + budget + tools + archivos reales

`projects/ejemplo-gemini-flash/` junta todo esto en un único `project.json`:

```json
{
  "project": "ejemplo-gemini-flash",
  "repo": "examples/ejemplo-gemini-flash-repo",
  "default_task": "revisar-backend",
  "agents": { "domain": "base", "use": ["backend-dev", "security-architect"] },
  "skills": { "domain": "base", "use": ["api-security"] },
  "tasks": {
    "revisar-backend": {
      "prompt": "review-project",
      "variables": { "PROJECT": "backend-api", "REVIEW_TYPE": "completa" },
      "focus": ["server.js"]
    }
  },
  "llm_profile": { "type": "powerful", "provider": "google", "config": "gemini-2.5-flash" },
  "budget": { "max_tokens": 6000 },
  "tools": { "enabled": true }
}
```

```bash
mova config google
mova chat ejemplo-gemini-flash revisar-backend
> Audita server.js y prioriza los hallazgos
[gemini-2.5-flash] (encuentra el secreto hardcodeado, la falta de validación, el endpoint sin auth...)
> /save -d "informes"
[Save] ✓ directorio creado: examples/ejemplo-gemini-flash-repo/informes
> /save "informes/auditoria-backend.md"
[Save] ✓ archivo guardado: examples/ejemplo-gemini-flash-repo/informes/auditoria-backend.md
> exit
```

Antes de mandar nada a Gemini, `budget.max_tokens: 6000` ya se validó contra el contexto real (`mova budget ejemplo-gemini-flash revisar-backend` lo muestra sin gastar un token: en este ejemplo da ~1.600, bien debajo del techo). Después de la primera llamada real, `mova-token-history.json` empieza a acumular la desviación de Gemini contra la estimación local, visible en la próxima corrida de `mova budget`.

---

## 16. Instalación global del CLI

```bash
make install
```

Compila y copia el binario a `$(go env GOPATH)/bin/mova` — la misma carpeta que usa `go install` en Linux, macOS y Windows. Con esa carpeta en el `PATH`, `mova` corre desde cualquier directorio. Nunca depende de dónde está el binario: siempre busca `workflow.md` subiendo desde el directorio actual (o desde `MOVA_PROJECT_ROOT`/`MOVA_PROJECT_PATH`), nunca guarda rutas absolutas.

```text
ERROR
No Mova project was found (looking for workflow.md, the project root marker).
Suggestion: Run "mova init" or move to a Mova project directory.
```

```bash
go build -o mova ./src/cli
```

No hay ediciones ni flags de build especiales — un solo binario, todos los comandos de esta guía.
