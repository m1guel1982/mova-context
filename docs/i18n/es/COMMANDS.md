# COMMANDS — guía de comandos

> Docs: [Español](COMMANDS.md) · [English](../en/COMMANDS.md)

El CLI (`mova`) es un complemento — todo lo que hace también se puede hacer pidiéndole a un modelo que lea `workflow.md` directamente. Ver [README.md](README.md#lo-esencial-antes-que-nada).

`mova` sube directorios automáticamente hasta encontrar `workflow.md`, así que funciona desde cualquier subcarpeta del repo. Si hay un único proyecto en `projects/`, `[project]` es opcional — se detecta solo.

## Todos los comandos

```text
mova run           [project] [task]        genera el contexto para el LLM
mova memory        [project] "respuesta"    guarda la sesión en memory.md
mova memory-read   [project]                imprime la memoria activa
  --all                                     incluye archivos históricos
  --month 2024-01                           un mes archivado específico
mova memory-archive [project]               archiva entradas antiguas
  --days N                                  días a mantener activos (default 30)
mova list                                   lista todos los proyectos
mova init          [name]                   crea un proyecto
mova search        "consulta" [dominio]     busca en el conocimiento
mova mcp start                              inicia el servidor MCP
  --port 3000                               como servidor HTTP (default)
  --stdio                                   como servidor Stdio (para Claude/Cursor)
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
```

## `mova run [project] [task]`

Ensambla agents + skills + prompt + memoria + focus, y lo imprime por stdout — listo para pegar en un chat o enviar a una API.

```bash
mova run mi-proyecto revisar-auth
```

Si la task tiene `focus` (en `project.json` o global), esa sección se agrega automáticamente al final del contexto — ver más abajo.

## `mova memory [project] "respuesta del LLM"`

Extrae el bloque ` ```memory ` de la respuesta de un modelo y lo agrega a `memory.md`.

```bash
mova memory mi-proyecto "$(cat respuesta.txt)"
```

La próxima vez que ejecutes `mova run mi-proyecto`, esa memoria aparece en el contexto automáticamente.

## `mova memory-read [project] [--all] [--month YYYY-MM]`

```bash
mova memory-read mi-proyecto --all
mova memory-read mi-proyecto --month 2024-01
```

## `mova memory-archive [project] [--days N]`

Mueve entradas más antiguas que `N` días (default 30) fuera de `memory.md`, agrupadas por mes.

```bash
mova memory-archive mi-proyecto --days 15
```

## `mova memory-clear [project] [flags]`

Pide confirmación salvo que uses `--yes`.

```bash
mova memory-clear mi-proyecto --archived --yes
```

## `mova memory-config [project] [action] [value]`

```bash
mova memory-config mi-proyecto days 45
```

## `mova list` / `mova init [name]`

```bash
mova list
mova init mi-proyecto
```

`init` crea `projects/mi-proyecto/project.json` (plantilla mínima) y un `memory.md` vacío.

## `mova search "consulta" [dominio]`

Busca en agents, skills y prompts — por palabra clave, sin necesidad de un modelo.

```bash
mova search "autenticación" software
```

## FOCUS — trabajar sobre una parte específica del proyecto

`focus` (definido en `project.json`, global o por task) le dice al motor que trabaje solo sobre ciertos archivos, carpetas o símbolos — en vez de todo el repo. Funciona igual con o sin CLI: si un modelo lee `workflow.md` directamente, la sección `## FOCUS` de la especificación explica exactamente cómo resolverlo.

**Importante:** `focus` es relativo al campo `"repo"` de `project.json`, no a la raíz de `mova-context`. Si `"repo": "examples/mi-repo"`, un item `"manual.md"` busca dentro de `examples/mi-repo/`, no en la raíz del proyecto Mova Context.

Si `task.focus` está definido, **reemplaza** por completo el `focus` global del proyecto (no se suman ambas listas).

### Cómo matchea cada item — igual que SQL LIKE

Cada item de `focus` se resuelve con una cascada de resolvers (archivo → símbolo de código → sección Markdown → artículo legal → memoria → fallback). Todos usan el mismo criterio de dos pasadas, equivalente a `LIKE` de SQL:

| Pasada | Equivalente SQL | Cuándo se usa |
|---|---|---|
| 1 — Exacta | `WHERE nombre = 'CreateOrder'` (por límite de palabra) | Siempre se intenta primero — prioridad más alta |
| 2 — LIKE / contiene | `WHERE nombre ILIKE '%CreateOrder%'` | Solo si la pasada 1 no encontró nada — tolerante a mayúsculas y acentos |

No hace falta declarar cuál pasada usar — el motor prueba la 1 y, si no hay resultado, cae automáticamente a la 2. Insensible a mayúsculas/acentos en ambas pasadas (`articulo 6` encuentra `Artículo 6`). Nunca usa un LLM ni heurísticas de significado — es búsqueda de texto, determinista: mismo input, mismo resultado siempre.

### Tipos de item soportados

| Item en `focus` | Qué resuelve |
|---|---|
| `"manual.md"` | el archivo completo, buscado por nombre en todo el repo |
| `"src/auth"` | índice del directorio (no el contenido de cada archivo) |
| `"CreateOrder()"` | la función/método/clase — la sintaxis `()` le indica al motor que es un símbolo de código, no un archivo |
| `"Artículo 6"` | la sección de un documento legal/estructurado (Título, Capítulo, Sección, Artículo, Inciso) |
| `"## Alguna sección"` o `"Alguna sección"` | un heading de Markdown |
| `"nombre_tabla"` | la definición `CREATE TABLE ...;` en un `.sql` |

### Ejemplo real

```json
"tasks": {
  "revisar-orden": {
    "prompt": "review-project",
    "focus": [
      "CreateOrder()",
      "manual.md",
      "Artículo 6"
    ]
  }
}
```

```bash
mova run mi-proyecto revisar-orden
```

Contexto resultante (fragmento):

```text
---
## FOCUS
FOCUS:CreateOrder()
  (src/orders.go)
func CreateOrder(clientID string, amount float64) (string, error) {
    ...
}

FOCUS:manual.md
  (manual.md)
# Manual de Operaciones
...

FOCUS:Artículo 6
  (manual.md)
### Artículo 6 — Cancelación de órdenes
Una orden puede cancelarse solo si no ha sido despachada.
```

Si un item no se encuentra en ninguna pasada, aparece como `not found: [item]` en vez de omitirse en silencio — para que sepas de inmediato si un nombre está mal escrito o el archivo no existe en el `repo` configurado.

### Si tu `focus.txt` sale vacío — checklist

1. ¿`"repo"` en `project.json` apunta a una carpeta que **existe** y contiene los archivos que buscas? (`focus` nunca busca fuera de `repo`)
2. ¿El símbolo de código lleva `()` al final (`"CreateOrder()"`) para que el motor sepa que es una función y no un archivo?
3. ¿El `task` que ejecutaste tiene su propio `focus`? Si sí, ese reemplaza al global — revisa cuál se está usando realmente.
4. ¿Estás usando un binario `mova` compilado **antes** de este fix? Si el contexto no muestra ninguna sección `## FOCUS` (ni siquiera un `not found:`), recompílalo: `go build -o mova ./src/cli`.

## `mova mcp start` — exponer Mova Context como servidor

Mismo motor que `mova run`, expuesto por el protocolo MCP (JSON-RPC 2.0) — para que un cliente (Claude Desktop, Cursor) pida el contexto solo, sin que copies y pegues nada.

**Modo stdio** (el que usan Claude Desktop / Cursor):

```bash
mova mcp start --stdio
```

Configuración típica del cliente MCP:

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

**Modo HTTP** (para probar con curl/Postman o integrarlo a tu propio backend):

```bash
mova mcp start --port 3000
```

```bash
curl -X POST http://localhost:3000/rpc \
  -H "content-type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"get_full_context","arguments":{"project":"mi-proyecto","task":"revisar-auth"}}}'
```

### Tools disponibles vía MCP

| Tool | Equivale a |
|---|---|
| `get_full_context` | `mova run [project] [task]` |
| `get_knowledge` | leer un agent/skill/prompt puntual |
| `get_memory` | `mova memory-read [project]` |
| `get_memory_all` | `mova memory-read [project] --all` |
| `get_workflow` | leer `workflow.md` |
| `search_context` | `mova search "consulta" [dominio]` |

## Variables de entorno

```bash
MOVA_ADAPTER=db MOVA_DSN=postgres://user:pass@host/db mova run mi-proyecto
```

| Variable | Efecto |
|---|---|
| `MOVA_ADAPTER` | Sobreescribe `project.json.adapter` (`file` / `db`) |
| `MOVA_DSN` | Sobreescribe `project.json.dsn` |
| `MOVA_PROJECT_ROOT` | Punto de partida extra para la búsqueda de `workflow.md` hacia arriba |
| `MOVA_PROJECT_PATH` | Usa esta ruta como raíz directamente, sin búsqueda |

### Resolución de raíz y clientes MCP

Los clientes MCP (Claude Desktop, Cursor) lanzan `mova` desde un directorio que normalmente no es tu proyecto — por eso el ejemplo de configuración de arriba fija `MOVA_PROJECT_ROOT`. Orden de resolución: `MOVA_PROJECT_PATH` (directo) → `MOVA_PROJECT_ROOT` (búsqueda hacia arriba desde ahí) → directorio de trabajo actual → directorio del binario.

## `llm_profile` — a qué modelo se le entrega el contexto

`llm_profile` (en `project.json`) es lo único que cambia cuando pasas de un modelo/proveedor a otro. Agents, skills, prompts, memoria y `focus` nunca cambian — el mismo `mova run` genera el mismo contexto sin importar qué modelo lo va a leer.

```json
"llm_profile": {
  "type": "local",
  "provider": "ollama",
  "model": "llama3.2:3b",
  "base_url": "http://localhost:11434"
}
```

| Campo | Valores | Para qué sirve |
|---|---|---|
| `type` | `"powerful"` (default) \| `"local"` | Con `"local"` el motor adapta el formato: listas con guión pasan a numeradas, se antepone `INSTRUCTIONS:` — modelos locales pequeños siguen mejor instrucciones secuenciales explícitas. Con `"powerful"` el contenido se entrega sin tocar. |
| `provider` | `"claude"` \| `"gpt"` \| `"gemini"` \| `"ollama"` \| cualquier string | Informativo — aparece en el encabezado del contexto (`Profile: local/ollama:llama3.2:3b`). No cambia qué se genera, salvo a través de `type`. |
| `model` | nombre exacto del modelo | Igual que `provider`: informativo, útil para saber con qué modelo se generó un `contexto.txt` dado. |
| `base_url` | URL del servidor | Necesario para `ollama` u otro servidor compatible con OpenAI corriendo localmente — no lo usa el motor de ensamblado, es para que tu propio script sepa dónde mandar el contexto. |

### Forma simple (legacy)

Si no necesitas `base_url` ni ser explícito con `model`, el campo `llm` (string) sigue funcionando y se traduce automáticamente a un `llm_profile`:

```json
"llm": "ollama"
```

equivale a:

```json
"llm_profile": { "type": "local", "provider": "ollama" }
```

Reconocidos como locales automáticamente: `ollama`, `llama`, `mistral`, `deepseek`, `qwen`, `gemma`, `phi`. Cualquier otro valor (`claude`, `gpt`, `gemini`, o algo custom) se trata como `"powerful"`.

### Cambiar de proveedor sin tocar nada más

```json
// Claude / GPT / Gemini — vía API o pegado en un chat web
"llm_profile": { "type": "powerful", "provider": "claude", "model": "claude-sonnet-4-6" }

// Ollama local
"llm_profile": { "type": "local", "provider": "ollama", "model": "llama3.2:3b", "base_url": "http://localhost:11434" }
```

```bash
mova run mi-proyecto mi-task > contexto.txt
ollama run llama3.2:3b < contexto.txt
```

## Compilar el CLI

```bash
go build -o mova ./src/cli
```

No hay ediciones ni flags de build especiales — un solo binario, todos los comandos de arriba.
