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

Ver `docs/i18n/es/AST_FILTER.md` para la sintaxis exacta de `archivo::kind=nombre`.

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
| `output_file` | string | `""` (deshabilitado) | Dónde se agrega el log del contexto sanitizado. Independiente de `dry_run` — se puede auditar sin dry-run, o hacer dry-run sin auditar. |

**Con el bloque ausente:** `dry_run=false`, `output_file=""` — comportamiento idéntico al actual, sin cambios.

### Resolución de `output_file` (siempre relativa al `project.json`, nunca al directorio de trabajo)

- Ruta relativa → se resuelve contra `projects/<proyecto>/`, **no** contra el directorio desde donde se ejecutó `mova`.
  Ejemplo: en `projects/02-pii-compliance-governance/project.json`, `"output_file": ".mova/egress_sanitized.log"` escribe en `projects/02-pii-compliance-governance/.mova/egress_sanitized.log`.
- Ruta absoluta — Unix (`/var/log/...`), Windows (`C:\...`, `D:\...`, `E:\...`) o UNC (`\\servidor\recurso\...`) — se usa tal cual, reconocida de forma multiplataforma sin importar en qué SO corre el binario de Mova (misma utilidad que ya usan `write_file`/`create_directory`).
- Si `output_file` termina en `/` o `\` (nombra un directorio, no un archivo), se usa el nombre por defecto **`egress_sanitized.md`** dentro de ese directorio. Un nombre sin barra final (aunque no tenga extensión, ej. `.mova`) se respeta tal cual — no se le fuerza extensión.
- Los directorios necesarios se crean automáticamente (`os.MkdirAll`, permisos estándar). El archivo se abre siempre en modo **append**: nunca se sobrescribe una ejecución anterior, y cada ejecución queda delimitada con su propio `execution_id` y `timestamp`.
- Si el archivo configurado no puede crearse o escribirse, `mova` **devuelve error y no llama al proveedor LLM** — el mismo comportamiento en CLI/Chat, MCP y HTTP, porque las tres puertas comparten una única implementación (`models.Session.Send`/`SendStream`).

### Qué se escribe

Únicamente el contexto **ya gobernado y sanitizado** (lo mismo que de verdad se enviaría al modelo) — nunca el contenido crudo pre-sanitización. Cada bloque incluye como mínimo `execution_id` y `timestamp` (UTC).

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

En todas, en vez del contenido real, se devuelve el mismo bloque fijo y traducido `reports.egress_airgap_message` + `reports.egress_airgap_directive` (esta última es una directiva explícita anti-elusión dirigida al modelo/agente que la recibe — ver `GOVERNANCE_CONTROLS.md § dry_run` para sus límites), con el conteo real de tokens:

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
