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
17. [Jobs — ejecución programada en segundo plano](#17-jobs--ejecución-programada-en-segundo-plano)
18. [Multiagente — grupos de agentes](#18-multiagente--grupos-de-agentes)
19. [Interfaz visual — `mova ui`](#19-interfaz-visual--mova-ui)
20. [Token Firewall](#20-token-firewall)

---

## 1. Referencia rápida

```text
mova run           [project] [task]         genera el contexto para el LLM
  --count                                   no arma/imprime el contexto, solo estima su costo en tokens/USD
                                             (igual que mova budget, sin escribir un archivo de reporte) —
                                             también acepta el nombre de un grupo multiagente, sumando una
                                             estimación por cada agente en vez de fallar. La misma estimación
                                             es alcanzable de forma idéntica desde "/budget" en el chat, la
                                             tool MCP "estimate_budget", y la ruta /mcp de HTTP — una sola
                                             implementación, todas las puertas de entrada.
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

mova budget        [project] [task]         estima tokens y costo, 100% local, escribe un reporte
  --focus                                   compara repo completo vs. solo lo que focus selecciona
  (también funciona con el nombre de un grupo multiagente — ver `mova run --count` más arriba; un
  grupo no tiene un único archivo de reporte, así que imprime el detalle por agente en su lugar)

mova mcp start                              inicia el servidor MCP
  --port 3000                               como servidor HTTP (default)
  --stdio                                   como servidor Stdio (para Claude/Cursor)

mova jobs list      [project]               lista los jobs programados de un proyecto — también funciona
                                             con el nombre de un grupo multiagente (lista los jobs de cada
                                             agente, en secciones separadas)
mova jobs run       [project] [index|--all] corre un job (o todos) ahora, ignorando el cron — con un grupo,
                                             [index] se interpreta en cambio como el nombre de un agente (u
                                             omitido/--all para todos); para correr un job por índice dentro
                                             de un solo agente, dirígete a él directamente:
                                             mova jobs run grupo/agente 0
mova jobs start                             inicia el daemon del scheduler (revisa cron cada minuto)

mova agents list    [group]                 lista los agentes de un grupo multiagente
mova agents run     [group] [agent|--all]   corre uno o todos los agentes de un grupo
  (para contar tokens, usa: mova run --count [group] — ver más arriba)

mova ui             [project]               abre la interfaz visual (chat, configs, jobs, agentes, logs...)
  En el visor/editor de archivos: ctrl+f abre una barra de búsqueda para buscar dentro del documento.
  En el chat: los comandos también funcionan aquí, los mismos que reconoce `mova chat` — set -model
  <nombre>, /memory, /budget, /tools, /clear, /save, /delete, exit|quit — nunca se envían al modelo.
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

### `--count` — estimar en vez de armar

`mova run --count <project>` se salta el armado/impresión del contexto y solo estima cuántos tokens/cuánto costaría — la misma estimación 100% local (tiktoken-go) que calcula `mova budget`, sin escribir un archivo de reporte. `<project>` también puede ser el nombre de un grupo multiagente (tiene su propio `config.json`, sin `project.json` propio — ver [§18](#18-multiagente--grupos-de-agentes)): en ese caso suma una estimación por cada agente en vez de fallar.

```bash
mova run --count mi-proyecto
mova run --count mi-grupo          # suma todos los agentes del grupo
mova run --count mi-proyecto revisar-auth --focus
```

Es exactamente la misma estimación alcanzable desde el comando `/budget` del chat, la tool MCP `estimate_budget`, y la ruta `/mcp` de HTTP — una sola implementación detrás de las cuatro puertas, así que un nombre de grupo que funciona en una funciona en todas las demás.

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

### Trabajar entre distintas unidades/ubicaciones (Windows/Linux/macOS)

Hay dos cosas distintas que pueden vivir en dos lugares distintos, y
Mova Context está diseñado justo para eso:

- **La raíz de Mova** — la carpeta con `workflow.md`, `projects/`,
  `agents/`, `config/`... es la configuración propia de Mova Context,
  donde sea que la hayas clonado/instalado (por ejemplo, `C:\mova` en
  Windows).
- **El `"repo"` de cada proyecto** (en su `project.json`) — el código
  real sobre el que ese proyecto trabaja, que puede estar **en
  cualquier lado**, incluida una unidad o volumen totalmente distinto
  (por ejemplo, `D:\mi-app` en Windows, `/mnt/data/mi-app` en Linux,
  `/Volumes/Data/mi-app` en macOS).

`"repo"` acepta una ruta absoluta exactamente así, y cada parte de Mova
que toca archivos del proyecto (`focus`, `save`, `delete`, las
acciones `save`/`delete` de los jobs, las ediciones en lenguaje natural
del chat) ya la resuelve correctamente — no hace falta configurar nada
extra por función. Por ejemplo:

```json
{
  "project": "auditoria-mi-app",
  "repo": "D:\\mi-app",
  "tasks": { "revisar": { "focus": ["src/checkout.js"] } }
}
```

Lo único que necesita un empujón es **encontrar la raíz de Mova**
cuando estás parado en `D:\mi-app` (no dentro de la instalación de
Mova) y simplemente escribís `mova`. Por defecto, Mova busca
`workflow.md` subiendo desde tu directorio actual — lo cual no
encuentra nada en una unidad completamente separada. Para eso está
`MOVA_PROJECT_ROOT`:

```bash
# Windows (PowerShell), una sola vez, permanente para tu usuario:
[Environment]::SetEnvironmentVariable("MOVA_PROJECT_ROOT", "C:\mova", "User")

# Linux/macOS, una sola vez, permanente (agregarlo a ~/.bashrc o ~/.zshrc):
export MOVA_PROJECT_ROOT="/home/vos/mova"
```

**Los instaladores de doble clic ya hacen esto por vos** (ver
`installers/README.md` § Consola lista para usar) — después de
instalar, podés hacer `cd` a `D:\mi-app` (o `/mnt/data/mi-app`, o
cualquier otra carpeta en cualquier unidad) y correr `mova run
auditoria-mi-app` de inmediato, sin ninguna configuración adicional.
Esto se verificó de punta a punta: un proyecto cuyo `"repo"` apunta a
una carpeta totalmente fuera de la instalación de Mova arma su
contexto de focus, guarda reportes de jobs y borra archivos en esa
carpeta externa correctamente, desde un directorio de trabajo que no
tiene relación con ninguna de las dos ubicaciones.

### Recursos de red — rutas UNC (`\\servidor\carpeta`)

Un recurso compartido de red de Windows funciona exactamente igual que
una unidad local: `"repo"` acepta una ruta UNC directamente —

```json
{ "repo": "\\\\servidor\\carpeta\\mi-app" }
```

Cada lugar donde Mova resuelve un `"repo"` absoluto (focus, save,
delete, jobs, budget) usa `filepath.IsAbs`/`filepath.Join` de Go, cuya
implementación para Windows reconoce explícitamente un prefijo UNC
como nombre de volumen y lo preserva en cada operación de join/clean
(verificado leyendo `volumeNameLen`/`Clean` de `internal/filepathlite`
en la librería estándar de Go — el prefijo `\\servidor\carpeta` nunca
se recorta ni se confunde con una ruta relativa). No hace falta
ninguna configuración extra más allá de lo que ya pide "otra unidad"
en la sección de arriba.

Si escribís una ruta directamente en el chat/MCP/HTTP en vez de
ponerla en `project.json`, la misma sintaxis UNC también se reconoce
(ver `documents.IsAbsCrossPlatform`) — con una limitación honesta: una
ruta UNC o de letra de unidad solo se resuelve cuando Mova corre como
binario de **Windows**. Una instancia de Mova corriendo en Linux/macOS
no tiene forma nativa de llegar a `\\servidor\carpeta` (no existe
soporte UNC a nivel de sistema operativo fuera de Windows) — montá el
recurso compartido primero (por ejemplo, `mount -t cifs
//servidor/carpeta /mnt/carpeta` en Linux, o conectate desde Finder en
macOS, que lo monta bajo `/Volumes/`), y usá esa ruta nativa resultante
como `"repo"` en su lugar.

### WSL (Windows Subsystem for Linux)

Dos direcciones, ambas ya cubiertas por lo anterior:

- **Corriendo el binario Linux de `mova` dentro de WSL2, llegando a una
  unidad de Windows**: WSL2 ya monta automáticamente las unidades de
  Windows en `/mnt/c`, `/mnt/d`, etc. — simplemente usá esa ruta como
  `"repo"` (`"repo": "/mnt/d/mi-app"`), sin ninguna diferencia respecto
  a cualquier otra ruta absoluta de Linux.
- **Corriendo `mova.exe` de Windows, llegando al sistema de archivos
  Linux de una distro WSL**: Windows expone esto como una ruta UNC —
  `"repo": "\\\\wsl$\\Ubuntu\\home\\vos\\mi-app"` (o
  `\\wsl.localhost\Ubuntu\...` en versiones más nuevas de WSL) —
  cubierto por el mismo soporte UNC descrito arriba.

### Docker / contenedores

Correr `mova` dentro de un contenedor tampoco necesita ningún manejo
especial de rutas — un directorio del host montado como volumen
simplemente *es* una ruta absoluta nativa desde el punto de vista del
contenedor. Montá tanto el repositorio de Mova como el proyecto
externo dentro del contenedor, y apuntá `MOVA_PROJECT_ROOT`/`"repo"` a
sus ubicaciones dentro del contenedor:

```bash
docker run \
  -v /ruta/host/a/mova:/mova \
  -v /ruta/host/a/mi-app:/workspace/mi-app \
  -e MOVA_PROJECT_ROOT=/mova \
  tu-imagen-de-mova mova run auditoria-mi-app
```

```json
{ "repo": "/workspace/mi-app" }
```

Es exactamente el mismo comportamiento de "`repo` absoluto externo"
verificado arriba — el sistema de archivos de un contenedor es
simplemente otro espacio de rutas absolutas nativo desde la
perspectiva de Mova, ya sean rutas Linux en un contenedor Linux o
(menos común) rutas de Windows en contenedores de Windows con Docker
Desktop.


---

## 15. Tokenomics — `mova budget`

Estima cuántos tokens usaría el contexto real de un proyecto (agents + skills + prompt + focus + memory — lo mismo que arma `mova run`) y cuánto costaría en cada proveedor de `config/prices.json`. **Todo el cálculo es local**: no llama a ningún LLM ni API externa, no manda una sola línea del proyecto fuera de tu máquina, no usa base de datos, y no guarda prompts ni contenido en ningún lado.

```bash
mova budget mi-proyecto
mova budget mi-proyecto mi-task --focus
```

Genera `mova-budget-report.md` (ruta configurable vía `"budget_path"` en `project.json` — ver [PROJECT_JSON.md](PROJECT_JSON.md); por default `projects/<project>/mova-budget-report.md`) — siempre en inglés simple, para que quien paga la factura lo entienda sin depender del idioma del resto de Mova Context. Alcanzable idéntico desde el CLI, `mova run --count` (sin archivo de reporte, ver [§2](#2-ensamblar-el-contexto--mova-run)), MCP (`estimate_budget`), y el chat REPL/chat de `mova ui` (`/budget`) — una sola implementación, todas las puertas de entrada:

```json
{"name":"estimate_budget","arguments":{"project":"mi-proyecto","task":"mi-task","focus":"true"}}
```

`mi-proyecto` arriba también puede ser el nombre de un grupo multiagente (ver [§18](#18-multiagente--grupos-de-agentes)) en vez de un proyecto normal — todas estas puertas suman una estimación por cada agente en ese caso en vez de fallar; solo se omite el archivo de reporte, ya que un grupo no tiene un único archivo donde escribirlo (el reporte de cada agente está disponible individualmente: `mova budget <grupo>/<agente>`).

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

### Instaladores de doble clic (Windows/Linux/macOS)

Para instalar con interfaz gráfica, sin escribir nada en la terminal,
ver [`installers/README.md`](../../installers/README.md) —
`installers/windows/install.bat`, `installers/macos/install.command`,
`installers/linux/install.sh`. Usan la misma ubicación de instalación
que `make install` de arriba (`$(go env GOPATH)/bin`), así que ambos
métodos son compatibles entre sí.

Cada instalador termina abriendo una consola **ya lista para usar
`mova`** — PowerShell o CMD en Windows, la misma ventana de Terminal
(o una nueva) en macOS, la misma terminal (o una nueva, detectada
automáticamente) en Linux — sin dejar un paso separado de "ahora abrí
una terminal vos mismo". Ver [`installers/README.md` § Consola lista para usar](../../installers/README.md#ready-to-use-console).

---

## 17. Jobs — ejecución programada en segundo plano

> **Ejemplo funcionando, con output real capturado:**
> [`examples/EJEMPLO-jobs-multiagente-WALKTHROUGH.md`](../../examples/EJEMPLO-jobs-multiagente-WALKTHROUGH.md)
> — `mova jobs list/run/start`, logging y multiagente, todo en un
> proyecto que podés correr tal cual (`projects/ejemplo-jobs-multiagente/`).

El Job Engine lee el array `jobs` del `project.json` de un proyecto
(ver [PROJECT_JSON.md § Jobs](PROJECT_JSON.md#jobs)) y lo ejecuta — el
mismo motor sin importar si se dispara desde CLI, chat, HTTP o MCP.

**Listar los jobs de un proyecto:**

```bash
mova jobs list ejemplo-ley21719
```
```text
  [0] schedule="0 2 * * *"  Auditoría nocturna de checkout y cookies
  [1] schedule="0 3 1 * *"  Archivado mensual de memoria, sin tasks
```

**Ejecutar todos los jobs de un proyecto ahora mismo** (ignora `schedule`):

```bash
mova jobs run ejemplo-ley21719
```

**Ejecutar solo un job, por su índice en el array `jobs`:**

```bash
mova jobs run ejemplo-ley21719 0
```

**Iniciar el daemon del scheduler** — revisa cada proyecto una vez por
minuto y dispara cualquier job cuyo `schedule` coincida:

```bash
mova jobs start
```
```text
[Jobs] scheduler started — checking every project once a minute (Ctrl+C to stop)
[ejemplo-ley21719] 2026-07-30 02:00:00
  ✓ task "auditar-checkout" executed (1,842 tokens)
  ✓ reports/auditoria_2026-07-30.pdf saved
  ✓ memory updated: projects/ejemplo-ley21719/memory.md
```

**Desde el chat (español/inglés), MCP y HTTP** — el mismo flujo, solo
cambia la puerta de entrada:

```text
> ejecuta los jobs de ejemplo-ley21719
> run the jobs for ejemplo-ley21719
```

```bash
curl -X POST http://localhost:3000/jobs/run \
  -d '{"project": "ejemplo-ley21719"}'
```

Tools MCP: `"list_jobs"` y `"run_job"` (argumentos: `project`, `index`
opcional) — accesibles desde Claude Desktop, Cursor o cualquier cliente
MCP, igual que cualquier otra tool de Mova (ver § 13).

---

## 18. Multiagente — grupos de agentes

Un grupo de agentes relacionados vive bajo un directorio de
`projects/`, cada agente es un proyecto normal, orquestado por un
`config.json` padre — ver [PROJECT_JSON.md § Multiagente](PROJECT_JSON.md#multiagente-grupos-de-agentes).

**Listar los agentes de un grupo:**

```bash
mova agents list ventas_online
```
```text
Group: ventas_online
Agentes de ventas, soporte y atención al cliente
Agents:
  - ventas_online/vendedor
  - ventas_online/atencionCliente
  - ventas_online/soporte
```

**Ejecutar todos los agentes, en secuencia:**

```bash
mova agents run ventas_online
```

**Ejecutar solo un agente** (también se puede direccionar como un proyecto normal):

```bash
mova agents run ventas_online vendedor
# equivale a:
mova run ventas_online/vendedor
```

**Desde el chat (español/inglés), MCP y HTTP:**

```text
> ejecuta todos los agentes de ventas_online
> run every agent in ventas_online
```

```bash
curl -X POST http://localhost:3000/agents/run \
  -d '{"group": "ventas_online"}'
```

Tools MCP: `"list_agents"` y `"run_agent"` (argumentos: `group`,
`agent` opcional, `task` opcional).

---

## 19. Interfaz visual — `mova ui`

Una interfaz de terminal (TUI), construida con [Bubble Tea](https://github.com/charmbracelet/bubbletea)
y [Lip Gloss](https://github.com/charmbracelet/lipgloss), que agrupa **todo**
lo que ya hacen los comandos de este documento detrás de un solo comando
y una navegación por menús. No reemplaza ningún comando existente —
`mova run`, `mova chat`, `mova jobs`, `mova agents`, etc. siguen
funcionando exactamente igual que antes. La interfaz visual solo llama a
los mismos componentes internos (`core`, `budget`, `jobs`,
`orchestrator`, `documents`, `models`, `logging`) desde una capa de
presentación distinta.

### Abrir la interfaz

```bash
mova ui                          # abre el menú principal
mova ui ejemplo-jobs-multiagente/auditor-checkout   # entra directo al panel de ese proyecto
```

Un único comando, con a lo sumo un argumento opcional (el proyecto) —
todo lo demás se navega desde adentro con el teclado:

```text
↑ / ↓        moverse entre opciones
enter        abrir / confirmar / correr
/            buscar dentro de una lista
esc          volver a la pantalla anterior
ctrl+s       guardar (en pantallas de edición de archivos)
ctrl+c       salir de la interfaz en cualquier momento
```

### Menú principal

```text
Chat            Multiagentes     Logs
Proyectos       Modelos          Salir
Workflow.md     Logging
```

### Chat

Reutiliza exactamente la misma sesión, el mismo motor de herramientas y
el mismo gate de Budget que `mova chat` — la única diferencia es que la
respuesta se muestra completa al terminar (sin streaming token por
token, para no romper el renderizado de la pantalla). Si necesitás
streaming en vivo o comandos como `/save`, `/delete` o `/budget` dentro
del chat, `mova chat` en una terminal común sigue siendo la vía
recomendada — la TUI no le quita nada.

### Proyectos

Elegís un proyecto de la lista (la misma que devuelve `mova list`,
incluyendo los agentes anidados de un grupo multiagente) y entrás a su
panel:

```text
project.json              ver y editar (Ctrl+S guarda, valida JSON antes de escribir)
memory.md                 ver y editar la memoria activa
Jobs                      listar y correr los jobs programados del proyecto
Reportes                  ver los archivos generados por la acción "save" de un job
Memoria archivada         (si existe) entradas archivadas por memory_archive
Historial de ejecuciones  mova-budget-report.md y mova-token-history.json
```

### Workflow.md

Editor directo del `workflow.md` de la raíz del repositorio — el mismo
archivo que interpretan `mova chat`, el comando `mova mcp`, y el Job
Engine.

### Modelos

Lista todos los `.json` bajo `config/models/` (proveedores, `active.json`)
y los abre para editar — el mismo directorio que ya lee
`mova.local/models` para resolver el proveedor activo. No hay un
formato nuevo: es el mismo `config/models/*.json` de siempre, con una
forma más cómoda de navegarlo y editarlo.

### Logging

Abre `config/log/logging.json` directamente — activar el logging,
cambiar el nivel, las categorías, la rotación o la retención es editar
este archivo y guardar con Ctrl+S. Ver `config/log/README.es.md` para
el detalle de cada parámetro.

### Multiagentes

```text
Multiagentes → elegí un grupo → elegí "▶ Correr todos" o un agente puntual
```

Corre exactamente `orchestrator.RunGroup` — lo mismo que `mova agents run`.

### Logs

Muestra el archivo de log activo (la misma ruta que usa
`mova.local/logging`, definida en `config/log/logging.json` →
`"file"."path"`), en solo lectura, actualizándose solo cada segundo. Si
el logging está deshabilitado, lo indica y explica cómo activarlo.

### Integración con el resto del ecosistema

| Función | Motor real detrás (sin duplicar lógica) |
|---|---|
| Chat | `mova.local/models.Session` + `sendWithTools` (el mismo de `mova chat`) |
| Jobs | `mova.local/jobs.RunJobByIndex` (el mismo de `mova jobs run`) |
| Multiagentes | `mova.local/orchestrator.RunGroup` (el mismo de `mova agents run`) |
| Guardar archivos editados | `mova.local/documents.ValidateTextFormat` + escritura directa |
| Listar proyectos | `mova.local/core.Adapter.ListProjects` (el mismo de `mova list`) |
| Logs | `mova.local/logging.LoadConfig` (misma ruta que usa el logger real) |

### Instalación de las dependencias de la interfaz

`mova ui` depende de tres librerías nuevas (Bubble Tea, Lip Gloss y
Bubbles). Se agregaron a `go.mod` igual que cualquier otra dependencia
del proyecto (por ejemplo `glamour`, que ya se usaba para el chat) —
`go build`/`go install` las descarga automáticamente la primera vez,
sin ningún paso manual. Esto es válido tanto compilando directo
(`go build -o mova ./src/cli`) como usando cualquiera de los
instaladores de doble clic (`installers/`) — no cambia nada en ellos,
porque ya invocan `go build` internamente.

---

## 20. Token Firewall

El Token Firewall es un conjunto de etapas determinísticas, sin IA, que
corren automáticamente, en este orden fijo, cada vez que Mova ensambla
el contexto de un proyecto — `mova run`, `mova chat`, la pantalla de
chat del TUI, `mova jobs run`, `mova agents run`, y `chat_completion`
de MCP pasan todos por acá, ya que todos comparten la misma función
subyacente (`budget.BuildGatedContext`):

```
ensamblar contexto
      │
      ▼
[1] Sanitizer           — elimina ruido repetitivo (logs, líneas en blanco, encabezados duplicados)
      │
      ▼
[2] Cache Layout Guard   — organiza el prompt para el prompt-caching del proveedor
      │
      ▼
[3] Circuit Breaker      — detiene ANTES de enviar nada, si se supera un límite de gasto
      │
      ▼
Gate de Budget (max_tokens) — el límite duro original de tamaño de contenido, sin cambios
      │
      ▼
se envía al modelo (Claude, GPT, Gemini, Ollama, o cualquier otro proveedor configurado)
```

**Cada etapa está habilitada por defecto.** Cada una se puede
desactivar de forma independiente en `"budget"` de `project.json` —
ver [PROJECT_JSON.md § Budget (y el Token Firewall)](PROJECT_JSON.md#budget-y-el-token-firewall)
para cada campo.

### Qué hace cada etapa, y qué problema resuelve

**[1] Sanitizer** (`mova.local/sanitize`) — el Focus de un proyecto
suele incluir logs y archivos fuente reales con ruido repetitivo que
suma tokens sin sumar información: 50 líneas de log casi idénticas que
solo difieren por el timestamp, corridas de líneas en blanco,
encabezados de licencia duplicados entre varios archivos. El Sanitizer
colapsa todo eso, de forma determinística, en microsegundos, sin
ningún llamado a un modelo — nada se resume ni se reescribe, solo se
eliminan repeticiones exactas y ruido de formato. **Problema que
resuelve:** pagar (en tokens y en dinero) por una repetición que el
modelo no necesita ver más de una vez.

**[2] Cache Layout Guard** (`budget.LayoutForCache`) — reordena el
system prompt en un prefijo estable (agents + skills + prompt —
archivos curados del proyecto que no cambian entre corridas) seguido
de todo lo que cambia cada vez (el encabezado con timestamp, focus,
memoria). Los proveedores Cloud cachean en base a un PREFIJO
exactamente igual; un solo byte distinto al principio (como un
timestamp) lo rompe en cada llamada. **Problema que resuelve:** un
proyecto que ya tiene disponible el prompt caching de un proveedor
Cloud, pero nunca lo activa porque el inicio del prompt cambia en cada
corrida.

**[3] Circuit Breaker** (`budget.CheckCircuitBreaker`, respaldado por
`mova-spend.json`) — dos límites independientes y opcionales:
`max_tokens_per_run` (esta llamada) y `max_monthly_usd` (el gasto
acumulado de este proyecto en el mes calendario actual). Con
`"on_exceed": "abort"`, detiene la ejecución ANTES de enviar nada a un
modelo — no es una advertencia después del hecho. **Problema que
resuelve:** un job programado o un loop automatizado acumulando una
factura en silencio, sin que nadie lo esté mirando.

**Context Cache** (`budget.SanitizeCached`, respaldado por
`mova-context-cache.json`) — un cuarto mecanismo, opcional, distinto de
las tres etapas de arriba: memoriza el resultado del Sanitizer por hash
de contenido, así una corrida repetida sobre archivos SIN CAMBIOS se
salta ese trabajo. Ahorra tiempo real, nunca tokens ni dinero por sí
mismo. **Problema que resuelve:** un daemon (`mova jobs start`) o un
loop de CI volviendo a sanitizar los mismos archivos exactos en cada
revisión.

### Cómo el reporte une todo

Cada una de estas etapas escribe en el mismo `mova-budget-report.md`
que `mova budget` ya generaba antes de que existiera el Token Firewall
— ningún archivo de reporte nuevo, ningún comando nuevo. Con
`"detailed_reports": true` (por defecto), incluye:

- Tokens por archivo individual de Focus
- Qué eliminó el Sanitizer, y el ahorro resultante
- El tamaño del prefijo estático del Cache Layout Guard, su huella
  (fingerprint), y los tokens estimados que se reutilizarían en un
  acierto de caché
- El estado actual del Circuit Breaker contra ambos límites
- **Una comparación antes/después**: tokens totales y costo estimado
  con la limpieza del Token Firewall aplicada vs. sin ella, en
  porcentajes

### Cache Layout Guard, por proveedor

El reordenamiento del prefijo del Cache Layout Guard es universal —
siempre corre igual sin importar el proveedor. Lo que cambia es si (y
cómo) ese prefijo estable realmente obtiene un descuento:

| Proveedor | Prompt caching nativo | Cómo lo usa Mova | Cuándo no aplica |
|---|---|---|---|
| **Anthropic (Claude)** | Sí — breakpoints explícitos `cache_control` en la API de Messages | Mova marca el prefijo estático con `"cache_control": {"type": "ephemeral"}` automáticamente (ver `models/provider_anthropic.go`) — un descuento real, a nivel de proveedor, sobre esa porción de llamadas futuras dentro de la ventana de caché | Prefijo bajo ~1.024 tokens (mínimo aproximado de Anthropic, varía por modelo) — el layout igual se aplica, solo que por debajo del tamaño que califica |
| **OpenAI (GPT)** | Sí — automático, basado en el prefijo, sin marcador explícito necesario | El layout de prefijo-estable-primero es exactamente lo que busca el caching propio de OpenAI; Mova no necesita enviar nada extra — el reordenamiento en sí es lo que ayuda | Prompts muy cortos, o un prefijo que cambia en cada llamada sin importar el layout de Mova (por ejemplo, archivos de focus que genuinamente son distintos cada vez) |
| **Google (Gemini)** | Sí, en la API Cloud (caching implícito, y context caching explícito para prefijos más grandes) | Mismo beneficio que OpenAI: el layout de prefijo-estable-primero es lo que activa el caching implícito | El context caching explícito de Gemini (para contextos muy grandes y de larga duración) es una función separada, opt-in, de la API de Google, que Mova no configura automáticamente |
| **Ollama (local) / otros proveedores locales** | No — no hay facturación Cloud ni caché del lado del servidor que aprovechar | El Cache Layout Guard igual corre (sin desventaja, tampoco costo) y el reporte igual muestra la división estático/dinámico — útil para entender la estructura del prompt aunque no haya impacto en el costo | El caching no aplica en el sentido de "ahorrar dinero en la próxima llamada", ya que un modelo local no tiene precio por token ni una caché persistente del lado del servidor como las APIs Cloud |
| **Cualquier otro proveedor** | Desconocido / varía | El layout igual se aplica — un prefijo estable nunca es una desventaja, incluso donde no hace nada | Si más adelante se agrega un proveedor con su propio mecanismo de caching, solo hace falta un equivalente a `cache_control` en `models/provider_<nombre>.go` — la etapa de layout en sí nunca cambia |

**En todos los casos, esto aumenta la PROBABILIDAD de un acierto de
caché — nunca lo garantiza.** El caching real depende del proveedor, el
modelo específico, y de si otra llamada reutiliza el mismo prefijo
mientras la ventana de caché del proveedor sigue abierta (típicamente
minutos, no horas). Mova no tiene forma de verificar si un acierto de
caché realmente ocurrió después del hecho — los proveedores hoy no
exponen eso en sus respuestas de API de una forma que Mova pueda
reportar de vuelta.

### Una corrida real completa, de principio a fin

Esta es una ejecución real del proyecto de ejemplo incluido en este
repositorio (`projects/ejemplo-token-firewall/`) — cada número de abajo
se midió de verdad, no se estimó para la documentación.

**La entrada:** un proyecto que audita un módulo de checkout. Su Focus
incluye `checkout.js` (un encabezado de comentarios grande, varias
líneas en blanco) y `server.log` (53 líneas, 48 de ellas entradas
`INFO 200 OK` casi idénticas que solo difieren por el timestamp).

```bash
$ mova budget ejemplo-token-firewall
✓ mova-budget-report.md generated
Total tokens: 1764 (cl100k_base)
  anthropic/claude: $0.0053 USD
  google/gemini:    $0.0022 USD
  openai/gpt-5:     $0.0088 USD
```

**Adentro de `mova-budget-report.md`, lo que realmente pasó:**

```text
## Sanitizer
- 47 línea(s)/encabezado(s) repetidos colapsados
- 1 corrida(s) de líneas en blanco excesivas colapsadas
Ahorro aproximado: ~518 tokens (~2075 caracteres)

## Token Firewall — Summary
|        | Antes | Después | Ahorro |
|--------|-------|---------|--------|
| Tokens | 2737  | 1764    | 35.6%  |
| Costo (claude) | $0.0082 | $0.0053 | 35.6% |

## Cache Layout Guard
Prefijo estático: 1167 tokens
Huella del prefijo: d58f4ec76275f850
Tokens estimados reutilizados en un acierto de caché: ~1050

## Circuit Breaker
Límite por corrida: 1764 / 5000 tokens (OK)
Gasto mensual: $0.00 / $5.00 (OK)
Estado: dentro del presupuesto.
```

**Tiempo** (medido en la misma máquina, corridas consecutivas): la
primera corrida tomó ~69ms de punta a punta (arranque del proceso,
lectura de archivos, Sanitizer, tokenización, escritura del reporte);
la segunda corrida, con el Context Cache ya "caliente", tomó ~57ms —
la mayor parte de ese tiempo restante es arranque del proceso y E/S de
archivos, no el Sanitizer en sí, que corre en microsegundos con
contenido de este tamaño.

**Qué significa esto en términos simples:** sin el Token Firewall, esta
tarea habría enviado 2.737 tokens al modelo, cada vez. Con él, la misma
tarea envía 1.764 — **una reducción real y medida del 35,6%**, incluso
antes de contar el descuento que el prefijo estático de 1.167 tokens
del Cache Layout Guard gane en un proveedor que realmente lo cachea.
Multiplicá eso en un job que corre todas las noches, o en una sesión de
chat con muchos turnos, y el ahorro se acumula.

Ver [`examples/EJEMPLO-token-firewall-WALKTHROUGH.md`](../../examples/EJEMPLO-token-firewall-WALKTHROUGH.md)
para la guía completa paso a paso, incluyendo el Circuit Breaker
deteniendo una corrida de verdad en modo `"abort"`.

### Una analogía simple

Pensá en el Token Firewall como **hacer la valija antes de un vuelo que
cobra por kilo.**

- El **Sanitizer** es doblar bien la ropa en vez de meterla a presión —
  la misma ropa, el mismo viaje, notablemente menos espacio, y no se
  queda nada afuera.
- El **Cache Layout Guard** es llevar lo que vas a mostrar en cada
  control (pasaporte, tarjeta de embarque) siempre en el mismo bolsillo
  delantero — el personal del control reconoce más rápido tu valija
  cuando siempre se ve igual de un vistazo, y te hacen pasar más
  rápido.
- El **Circuit Breaker** es la balanza de la aerolínea en el check-in —
  te frena antes de embarcar con una valija con sobrepeso, en vez de
  sorprenderte con una factura después de que el vuelo ya salió.

**Beneficios:** cada viaje cuesta menos, los descuentos propios de la
aerolínea (el caching) realmente se aplican más seguido, y nunca
embarcás en un vuelo que no podés pagar. **Desventaja posible:** doblar
la ropa toma un momento, y si de verdad necesitás mostrarle al oficial
de aduana cada arruga de una camisa (por ejemplo, una tarea que
genuinamente trata de analizar ruido de logs crudo, sin modificar),
doblar de más podría esconder algo que necesitabas. **Mitigación:**
cada etapa se puede desactivar de forma independiente, `strip_comments`
está desactivado por defecto específicamente para no eliminar intención
de documentación, y nunca se resume ni se reescribe nada — solo se
tocan repeticiones exactas y ruido de formato, así que lo que queda
siempre es el contenido real, solo que sin el relleno.
