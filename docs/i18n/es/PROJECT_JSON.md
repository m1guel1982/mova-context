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

`repo`, `workflow_path`, `budget_path` y `token_history_path` son todos strings únicos — a propósito. Un solo valor por campo mantiene un project.json fácil de leer y de razonar; ninguno de ellos tiene forma de array.

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
