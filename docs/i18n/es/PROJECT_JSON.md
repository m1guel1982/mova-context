# PROJECT.JSON — referencia de campos

Cada proyecto vive en `projects/<nombre>/project.json`. Esta es la única fuente de configuración de ese proyecto — los precios de modelos son lo único que vive en otro lado (`config/prices.json`; ver [COMMANDS.md §15](COMMANDS.md#15-tokenomics--mova-budget)).

## Ejemplo mínimo

```json
{
  "project": "mi-proyecto",
  "repo": "examples/mi-proyecto-repo",
  "lang": "es",
  "adapter": "file",
  "default_task": "revisar",

  "agents": { "domain": "base", "use": ["backend-dev"], "custom": [] },
  "skills": { "domain": "base", "use": ["lazy-minimalism"], "custom": [] },

  "tasks": {
    "revisar": { "prompt": "review-project" }
  },

  "llm_profile": { "provider": "ollama", "config": "llama3.2.3b" }
}
```

## Todos los campos

| Campo | Tipo | Para qué |
|---|---|---|
| `project` | string | nombre del proyecto — coincide con la carpeta bajo `projects/` |
| `description` | string | texto libre, se muestra en `mova projects` |
| `repo` | string | el único repositorio del proyecto. Un repositorio por proyecto, una sola ruta — ver **¿Más de un directorio?** abajo si la tentación es agregar un segundo |
| `lang` | string | `"es"`, `"en"`, ... — qué variante de idioma de prompts/agents/skills cargar |
| `adapter` | string | `"file"` (default) o `"db"` |
| `dsn` | string | connection string, solo con `adapter: "db"` |
| `llm` | string | legacy: `"claude"` \| `"gpt"` \| `"ollama"` — sigue funcionando |
| `llm_profile` | object | `{ "provider", "config" }` — la forma moderna de elegir modelo (ver [COMMANDS.md §6](COMMANDS.md#6-modelos-y-proveedores)) |
| `default_task` | string | task usada cuando no se indica ninguna en la línea de comandos |
| `variables` | object | `{nombre: valor}` inyectadas en prompts/agents/skills |
| `agents` / `skills` | object | `{ "domain", "use": [...], "custom": [...] }` |
| `tasks` | object | tasks nombradas — cada una puede sobreescribir `prompt`/`agents`/`skills`/`variables`/`focus`/`budget` |
| `archive` | object | configuración de gestión de memoria (ver [COMMANDS.md §4](COMMANDS.md#4-memoria)) |
| `focus` | array | archivos/directorios/símbolos sobre los que trabajar en vez de todo el repo — ver **Trabajar sobre parte de `repo`** abajo |
| `budget` | object | `{ "max_tokens": N }` — el techo de contexto (ver [COMMANDS.md §15](COMMANDS.md#15-tokenomics--mova-budget)) |
| `workflow_path` | string | dónde vive `workflow.md` para este proyecto — ver **Workflow** abajo |
| `budget_path` | string | dónde se escribe `mova-budget-report.md` — ver **Reporte de Budget** abajo |
| `token_history_path` | string | dónde se escribe `mova-token-history.json` — ver **Historial de tokens** abajo |
| `tools` | object | `{ "enabled": true/false }` — permite que el chat llame tools de archivos/documentos durante la conversación |
| `jobs` | array | jobs programados en segundo plano (cron `schedule` + acciones) — ver **Jobs** más abajo |

`repo`, `workflow_path`, `budget_path` y `token_history_path` son todos strings únicos — a propósito. Un solo valor por campo mantiene un project.json fácil de leer y de razonar; ninguno de ellos tiene forma de array.

`repo` no tiene que vivir cerca de la instalación de Mova — acepta una ruta absoluta que apunte a cualquier lado, incluida una unidad o volumen totalmente distinto (`"repo": "D:\\mi-app"` en Windows, `"repo": "/mnt/data/mi-app"` en Linux, `"repo": "/Volumes/Data/mi-app"` en macOS), y cada acción de focus/save/delete/jobs ya la resuelve correctamente. Ver [COMMANDS.md § Trabajar entre distintas unidades/ubicaciones](COMMANDS.md#trabajar-entre-distintas-unidadesubicaciones-windowslinuxmacos) para la explicación completa y cómo correr `mova` desde adentro de esa carpeta externa.

## Trabajar sobre parte de `repo`, en vez de un segundo repositorio

Si parte del trabajo toca solo una carpeta dentro de `repo` — un servicio en un monorepo, un módulo, un directorio — para eso está `focus`, no un segundo `repo` (no existe tal campo):

```json
{
  "project": "mi-proyecto-monorepo",
  "repo": "examples/mi-monorepo",
  "tasks": {
    "revisar-api": { "prompt": "review-api", "focus": ["services/api/"] },
    "revisar-web": { "prompt": "review-ui", "focus": ["services/web/"] }
  }
}
```

Cada task se acota a su propio directorio con `focus` — ver la [sección de Focus en COMMANDS.md](COMMANDS.md#3-focus--acotar-el-contexto-a-parte-del-repo) para la sintaxis completa de glob/directorio (`"**/*"`, `"."`, `"src/"`, ...). Esto mantiene `project.json` simple (un `repo`, una ruta) y a la vez permite que cada task trabaje solo sobre su propia porción del repositorio.

Si el trabajo realmente cruza dos repositorios NO relacionados — dueños distintos, ciclos de release distintos, nunca tocados por la misma task — la configuración más simple y honesta es dos proyectos separados (`projects/servicio-a/project.json`, `projects/servicio-b/project.json`), cada uno con su propio `repo`, `workflow_path`, `budget_path` y `token_history_path`. Ver **Más de un proyecto** abajo.

## Ejemplo de workflow

```json
{
  "project": "mi-proyecto",
  "repo": "examples/mi-proyecto-repo",
  "workflow_path": "workflow.md"
}
```

Decir "lee workflow.md", "ejecuta workflow.md", "workflow.md mi-proyecto", etc. resuelve este proyecto, valida su Budget, y solo entonces carga el archivo — ver [COMMANDS.md §9](COMMANDS.md#workflowmd--ejecución-con-validación-de-budget). Sin `workflow_path`, Mova busca un `workflow.md` simple en la raíz de Mova, igual que antes de que existiera este campo.

## Ejemplo de Budget

```json
{
  "budget": { "max_tokens": 8000 },
  "budget_path": "mova-budget-report.md"
}
```

Sin `budget_path`, el reporte se escribe en `projects/<proyecto>/mova-budget-report.md`.

## Ejemplo de historial de tokens

```json
{
  "token_history_path": "mova-token-history.json"
}
```

Sin este campo, el archivo se escribe en `projects/<proyecto>/mova-token-history.json`.

## Ejemplo de tasks

```json
{
  "tasks": {
    "revisar": { "prompt": "review-project" },
    "auditar": {
      "prompt": "audit-consent-flow",
      "agents": ["security-architect"],
      "variables": { "module": "checkout" },
      "focus": ["checkout.html"],
      "budget": { "max_tokens": 4000 }
    }
  }
}
```

## Ejemplo de Agents / Skills / Prompts

```json
{
  "agents": { "domain": "base", "use": ["backend-dev", "qa-engineer"], "custom": [] },
  "skills": { "domain": "base", "use": ["lazy-minimalism"], "custom": ["mi-skill-propio"] }
}
```

Los prompts se referencian por nombre desde el campo `"prompt"` de cada task (ver el ejemplo de **Tasks** arriba) — no existe un campo `prompts` separado a nivel raíz; un prompt se elige por task.

## Más de un proyecto — agents/skills/prompts compartidos, nada duplicado

No existe un único `project.json` con varios proyectos adentro — cada proyecto es su propia carpeta con su propio archivo:

```text
projects/
├── proyecto-a/
│   └── project.json
└── proyecto-b/
    └── project.json
```

**`projects/proyecto-a/project.json`**
```json
{
  "project": "proyecto-a",
  "repo": "examples/proyecto-a-repo",
  "lang": "es",
  "agents": { "domain": "base", "use": ["backend-dev"], "custom": [] },
  "skills": { "domain": "base", "use": ["lazy-minimalism"], "custom": [] },
  "tasks": { "revisar": { "prompt": "review-project" } },
  "budget_path": "mova-budget-report.md",
  "token_history_path": "mova-token-history.json"
}
```

**`projects/proyecto-b/project.json`**
```json
{
  "project": "proyecto-b",
  "repo": "examples/proyecto-b-repo",
  "lang": "es",
  "agents": { "domain": "base", "use": ["security-architect"], "custom": [] },
  "skills": { "domain": "base", "use": ["lazy-minimalism"], "custom": ["mi-skill-propio"] },
  "tasks": { "auditar": { "prompt": "audit-consent-flow" } },
  "budget_path": "mova-budget-report.md",
  "token_history_path": "mova-token-history.json"
}
```

Agents/skills/prompts son recursos globales (viven una sola vez, bajo `agents/`, `skills/`, `prompts/` en la raíz de Mova); cada proyecto solo los **referencia por nombre** en `use`/`prompt`. Si `proyecto-a` y `proyecto-b` usan `"lazy-minimalism"`, es literalmente el mismo archivo de skill para los dos — nada se copia por proyecto. Cada proyecto mantiene su propio `budget_path`/`token_history_path`/`workflow_path`, así que sus reportes de Budget e historial de tokens nunca se mezclan.

## Budget (y el Token Firewall)

`"budget"` (a nivel de proyecto o de task — el `budget` propio de una
task reemplaza el del proyecto, misma regla que `focus`) siempre aceptó
`max_tokens`, un techo duro de tamaño de contenido. Desde el Token
Firewall, también acepta cada campo de abajo — un conjunto de etapas
determinísticas, sin IA, que reducen lo que se envía a un modelo y
gobiernan cuánto cuesta, corriendo automáticamente antes de ese gate.
**Cada etapa está habilitada por defecto** — poné el campo
correspondiente en `false` para desactivar solo esa:

```json
{
  "budget": {
    "max_tokens": 20000,
    "max_tokens_per_run": 8000,
    "max_monthly_usd": 15.00,
    "on_exceed": "warn",
    "sanitize": { "enabled": true, "dedupe_logs": true, "strip_blank": true, "strip_comments": false },
    "cache_hint": true,
    "circuit_breaker": true,
    "token_estimation": true,
    "detailed_reports": true,
    "context_cache": true
  }
}
```

| Campo | Tipo | Por defecto | Qué hace |
|---|---|---|---|
| `max_tokens` | number | ninguno | Techo duro sobre el contexto ensamblado — sin cambios desde antes del Token Firewall |
| `max_tokens_per_run` | number | ninguno (0 = sin techo) | Circuit Breaker: aborta/avisa si una corrida supera esta cantidad de tokens |
| `max_monthly_usd` | number | ninguno (0 = sin techo) | Circuit Breaker: aborta/avisa si el gasto registrado de este proyecto en el mes calendario actual llega a este monto |
| `on_exceed` | `"warn"` \| `"abort"` | `"warn"` | Qué hace el Circuit Breaker cuando se supera un límite de arriba |
| `sanitize` | object | habilitado, conservador | Configuración propia del Sanitizer — ver abajo |
| `cache_hint` | boolean | `true` | Habilita el Cache Layout Guard (reordena el system prompt para el prompt-caching del proveedor) — poné `false` para desactivarlo |
| `circuit_breaker` | boolean | `true` | Habilita el mecanismo del Circuit Breaker en sí, independiente de si hay un límite configurado — poné `false` para desactivarlo aunque los límites sigan configurados |
| `token_estimation` | boolean | `true` | Usa el tokenizador real (tiktoken) — poné `false` para una aproximación rápida de caracteres/4 (es una decisión de rendimiento, no de ahorro) |
| `detailed_reports` | boolean | `true` | Incluye el desglose completo (tokens por archivo, comparación antes/después) en mova-budget-report.md — poné `false` para solo los totales |
| `context_cache` | boolean | `true` | Habilita la caché local propia de Mova de los resultados del Sanitizer (mova-context-cache.json) — ahorra tiempo real en corridas repetidas sobre archivos sin cambios |

### `sanitize` — configuración propia del Sanitizer

| Campo | Tipo | Por defecto | Qué hace |
|---|---|---|---|
| `enabled` | boolean | `true` | Interruptor general de toda la etapa Sanitizer |
| `dedupe_logs` | boolean | `true` | Colapsa 3+ líneas consecutivas casi idénticas (ignorando un timestamp inicial) en la primera ocurrencia + un contador — el caso de "50 líneas de INFO 200 OK" |
| `strip_blank` | boolean | `true` | Colapsa corridas de 3+ líneas en blanco a 1 |
| `strip_comments` | boolean | `false` | Elimina bloques de solo comentarios de 5+ líneas — desactivado por defecto, ya que una tarea sobre documentación los necesita intactos |

Ver [COMMANDS.md § Token Firewall](COMMANDS.md#token-firewall) para la
explicación completa de cada etapa, cómo se comporta el Cache Layout
Guard según el proveedor (Claude, GPT, Gemini, Ollama...), y un ejemplo
completo con ahorros reales medidos.

## Jobs

`jobs` es un array de tareas programadas en segundo plano, ejecutadas
por el Job Engine (`mova.local/jobs`) — el mismo motor que comparten
`mova jobs run`, la tool MCP "run_job", `POST /jobs/run` y el daemon
`mova jobs start`. Cada entrada combina un `schedule` (cron) con una o
más acciones independientes:

```json
{
  "jobs": [
    {
      "comment": "Auditoría nocturna de checkout y cookies",
      "schedule": "0 2 * * *",
      "tasks": ["auditar-checkout", "auditar-cookies"],
      "save": "reports/auditoria_{date}.pdf",
      "budget": { "focus": true },
      "memory": "Auditoría de checkout y cookies realizada ({date})"
    },
    {
      "comment": "Archivado mensual de memoria, sin tasks",
      "schedule": "0 3 1 * *",
      "memory_archive": { "days": 30 }
    },
    {
      "comment": "Ejecutar todas las tareas del proyecto",
      "schedule": "0 4 * * *",
      "tasks": ["*"],
      "save": "reports/auditoria_completa_{date}.pdf"
    },
    {
      "comment": "Eliminar archivos temporales",
      "schedule": "0 5 * * *",
      "delete": ["reports/temp_*.csv", "logs/draft.md"]
    }
  ]
}
```

| Campo | Tipo | Qué hace |
|---|---|---|
| `comment` | string | texto libre, nunca se interpreta — para humanos que leen project.json |
| `schedule` | string | cron de 5 campos (`min hour dom month dow`) — ver README.md § Cron para ejemplos |
| `tasks` | array de strings | nombres de tasks del `tasks` del proyecto, o `["*"]` para ejecutarlas todas |
| `save` | string | ruta de salida; `{date}` se expande a `YYYY-MM-DD`. El formato se elige por la extensión (`.md`, `.pdf`, `.docx`...), igual que `/save` en el chat |
| `memory` | string | texto agregado a `memory.md` vía `AppendMemory`; soporta `{date}` y `{time}` |
| `memory_archive` | object | `{ "days": N }` — archiva entradas de memoria más viejas que N días (usa la retención propia del `archive` del proyecto si se omite) |
| `delete` | array de strings | patrones glob (relativos a `repo`), ej. `"reports/temp_*.csv"` |
| `budget` | object | `{ "focus": true }` — también escribe un `mova-budget-report.md`, igual que `mova budget --focus`. Es distinto del campo `budget` de nivel superior (un techo de tokens): este es una ACCIÓN, no un gate |

Cada campo es independiente — un job puede declarar cualquier
subconjunto de `tasks`/`save`/`memory`/`memory_archive`/`delete`/`budget`.
Todas las acciones declaradas de un job se ejecutan en este orden fijo:
tasks → save → memory → memory_archive → delete → budget, sin importar
el orden en que aparezcan en el JSON.

Para ejecutar jobs bajo demanda, ignorando `schedule`, usar `mova jobs
run <project>` (o `mova jobs run <project> <index>` para un solo job)
— ver COMMANDS.md § Jobs.

## Multiagente (grupos de agentes)

Un **grupo** es un directorio bajo `projects/` que contiene varios
agentes independientes, cada uno un proyecto normal con su propio
`project.json`:

```text
projects/
    ventas_online/
        config.json
        vendedor/
            project.json
        atencionCliente/
            project.json
        soporte/
            project.json
```

`projects/ventas_online/config.json` es el archivo padre/orquestador:

```json
{
  "group": "ventas_online",
  "description": "Agentes de ventas, soporte y atención al cliente",
  "agents": ["vendedor", "atencionCliente", "soporte"]
}
```

| Campo | Tipo | Para qué sirve |
|---|---|---|
| `group` | string | nombre visible (usa el nombre del directorio si se omite) |
| `description` | string | texto libre |
| `agents` | array de strings | nombres de subdirectorios, cada uno con su propio `project.json`. Si se omite, se auto-descubre cada subdirectorio que contenga un `project.json` |

Cada agente se referencia como `<group>/<agent>` — un nombre de
proyecto normal que cualquier comando existente ya entiende (`mova run
ventas_online/vendedor`, `mova budget ventas_online/vendedor`, `mova
jobs run ventas_online/vendedor`, ...). `mova agents run
ventas_online` ejecuta todos los agentes secuencialmente, a través del
mismo pipeline de ensamblado+Budget-gate que usa `mova run` para
cualquier proyecto — ver COMMANDS.md § Multiagente.
