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
  "skills": { "domain": "base", "use": ["lazy-minimalism"], "custom": [] },
  "tasks": {
    "revisar": { "prompt": "review-project", "variables": {}, "focus": ["archivo.js"] }
  },
  "llm_profile": { "type": "local", "provider": "ollama", "config": "llama3.2.3b" },
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
| `budget.on_exceed` | `"warn"` o `"block"` al exceder el presupuesto. |
| `budget.sanitize.{enabled,dedupe_logs,strip_blank,strip_comments}` | Controles del Sanitizer. |
| `budget.pii_masking.enabled` | Enmascarado estructural de PII (ver `ARTIFACTS.md`). |
| `debug` | `true` imprime, en cada puerta (chat, CLI, HTTP, MCP), qué se resolvió antes de correr una tarea: ruta del repo, cada agente/skill/prompt con su ruta resuelta (o `"inline"`), y las entradas de `focus`/`exclude` con su ruta absoluta. Por defecto `false`. Nunca se agrega solo por generar un `project.json` con `mova init` — es opt-in manual. |
| `egress_audit` | Auditoría del contexto **ya sanitizado** justo antes de salir hacia el LLM, y/o un dry-run que corta la llamada — ver sección dedicada abajo. |

Ver `docs/i18n/es/AST_FILTER.md` para la sintaxis exacta de `archivo::kind=nombre`.

## `egress_audit` — auditoría y dry-run antes del proveedor LLM

```json
"egress_audit": {
  "dry_run": true,
  "output_file": ".mova/egress_sanitized.md"
}
```

| Clave | Tipo | Default | Qué hace |
|---|---|---|---|
| `dry_run` | bool | `false` | Si es `true`: se completa la gobernanza/sanitización (y el log, si `output_file` está configurado), pero **nunca se llama al proveedor LLM**. El cliente recibe una respuesta exitosa indicando que el dry-run terminó y que no hubo inferencia. |
| `output_file` | string | `""` (deshabilitado) | Dónde se agrega el log del contexto sanitizado. Independiente de `dry_run` — se puede auditar sin dry-run, o hacer dry-run sin auditar. |

**Con el bloque ausente:** `dry_run=false`, `output_file=""` — comportamiento idéntico al actual, sin cambios.

### Resolución de `output_file` (siempre relativa al `project.json`, nunca al directorio de trabajo)

- Ruta relativa → se resuelve contra `projects/<proyecto>/`, **no** contra el directorio desde donde se ejecutó `mova`.
  Ejemplo: en `projects/02-pii-compliance-governance/project.json`, `"output_file": ".mova/egress_sanitized.md"` escribe en `projects/02-pii-compliance-governance/.mova/egress_sanitized.md`.
- Ruta absoluta — Unix (`/var/log/...`), Windows (`C:\...`, `D:\...`, `E:\...`) o UNC (`\\servidor\recurso\...`) — se usa tal cual, reconocida de forma multiplataforma sin importar en qué SO corre el binario de Mova (misma utilidad que ya usan `write_file`/`create_directory`).
- Si `output_file` termina en `/` o `\` (nombra un directorio, no un archivo), se usa el nombre por defecto **`egress_sanitized.md`** dentro de ese directorio. Un nombre sin barra final (aunque no tenga extensión, ej. `.mova`) se respeta tal cual — no se le fuerza extensión.
- Los directorios necesarios se crean automáticamente (`os.MkdirAll`, permisos estándar). El archivo se abre siempre en modo **append**: nunca se sobrescribe una ejecución anterior, y cada ejecución queda delimitada con su propio `execution_id` y `timestamp`.
- Si el archivo configurado no puede crearse o escribirse, `mova` **devuelve error y no llama al proveedor LLM** — el mismo comportamiento en CLI/Chat, MCP y HTTP, porque las tres puertas comparten una única implementación (`models.Session.Send`/`SendStream`).

### Qué se escribe

Únicamente el contexto **ya gobernado y sanitizado** (lo mismo que de verdad se enviaría al modelo) — nunca el contenido crudo pre-sanitización. Cada bloque incluye como mínimo `execution_id` y `timestamp` (UTC).
