# workflow.md — Guía Operacional

> Este documento explica en detalle cómo funciona el workflow de Mova Context.
> El archivo `workflow.md` en la raíz del proyecto es el punto de entrada. Este documento es su documentación completa.

---

## Filosofía

Mova Context se basa en un principio simple:

```text
project.json  →  workflow.md  →  contexto para el LLM
```

El workflow no es un motor. Es un orquestador.

Lee `project.json`, resuelve idioma, perfil del LLM y adaptador, carga el contexto correspondiente y continúa el flujo recursivo. Nada más.

---

## Activación

```text
Lee workflow.md
Lee workflow.md → [PROJECT]
Lee workflow.md → [PROJECT] → [TASK]

carga [PROJECT]
ejecuta [TASK] de [PROJECT]
```

---

## project.json — fuente de verdad

Todas las decisiones del workflow parten desde `project.json`.

```json
{
  "project": "mi-proyecto",
  "description": "Descripción del proyecto",
  "repo": ".",
  "lang": "es",
  "adapter": "file",
  "llm": "claude",
  "default_task": "mi-tarea",

  "variables": {
    "empresa": "Acme Corp",
    "stack": "Node.js + PostgreSQL"
  },

  "agents": {
    "domain": "software",
    "use": ["backend-dev", "security-architect"]
  },

  "skills": {
    "domain": "software",
    "use": ["api-security", "jwt-security"]
  },

  "tasks": {
    "mi-tarea": {
      "prompt": "review-project",
      "variables": { "modulo": "auth" }
    },
    "nuevo-modulo": {
      "prompt": "create-module",
      "agents": { "domain": "software", "use": ["architect"] },
      "skills": { "domain": "software", "use": ["sql-optimization"] },
      "variables": { "modulo": "pagos" }
    }
  }
}
```

### Campos principales

| Campo | Descripción |
|-------|-------------|
| `lang` | Idioma del contexto: `es`, `en`, `fr`, etc. |
| `adapter` | Almacenamiento: `file`, `postgresql`, `mongodb` |
| `llm` | Modelo: `claude`, `gpt`, `gemini`, `ollama`, `openrouter` |
| `llm_profile` | Perfil del modelo: `powerful`, `local`, `fast` |
| `default_task` | Task que se ejecuta si no se especifica ninguna |

---

## Resolución de archivos

Para `domain: software`, `lang: es`, archivo `backend-dev`:

```text
agents/software/i18n/es/backend-dev.md   ← usar
agents/software/i18n/en/backend-dev.md   ← fallback si es/ no existe
agents/base/i18n/es/backend-dev.md       ← fallback legacy
```

El sistema encuentra el archivo correcto automáticamente. Sin configuración adicional.

---

## Secuencia de ejecución

```text
1.  Leer projects/[PROJECT]/project.json
2.  Resolver lang → directorio de idioma
3.  Resolver llm_profile → ajustar comportamiento del modelo
4.  Resolver adapter → filesystem o base de datos
5.  Resolver task (indicada o default_task)
6.  Cargar agents (globales + task)
7.  Cargar skills (globales + task)
8.  Cargar prompt (base + custom)
9.  Leer memory.md
10. Fusionar variables (task sobreescribe globales)
11. Inyectar {{VARIABLES}} en agents, skills y prompt
12. Ejecutar
13. Actualizar memory.md
```

---

## Variables

Toda clave en `variables` se normaliza automáticamente:

```text
empresa        → {{EMPRESA}}
api_prefix     → {{API_PREFIX}}
stack          → {{STACK}}
cualquier_clave → {{CUALQUIER_CLAVE}}
```

Prioridad: `task > global`

Variables reservadas (siempre disponibles, sin declararlas):

```text
{{PROJECT}}    → valor de "project"
{{REPO}}       → valor de "repo"
{{TASK}}       → nombre de la task activa
{{LANG}}       → idioma configurado
```

Si una variable referenciada en un archivo no existe en `project.json`, se deja el placeholder sin reemplazar y se informa al usuario qué variable falta.

---

## Focus

Restringe el trabajo a archivos, directorios o símbolos específicos del código:

```json
"focus": ["src/auth", "userService()", "UserController"]
```

Cuando está presente, el modelo trabaja solo sobre esos elementos.

```text
SI el elemento es una ruta → usar directamente
SI es un nombre sin ruta  → buscar recursivamente en "repo"
SI termina en ()          → tratarlo como función o método
```

`{{FOCUS}}` se inyecta como lista legible en agents, skills y prompt.

---

## Orden de carga

```text
1.  agents globales base
2.  agents globales custom
3.  agents task base
4.  agents task custom

5.  skills globales base
6.  skills globales custom
7.  skills task base
8.  skills task custom

9.  prompt base
10. prompt custom
```

`base → custom`. Custom complementa o sobreescribe base.

Si un archivo no existe → ignorar y continuar.

---

## Memoria

```text
Ruta: projects/[PROJECT]/memory.md

Leer antes de ejecutar.
Actualizar al finalizar.
Crear si no existe.
```

Formato:

```md
## YYYY-MM-DD — título de la sesión

**Hecho:**
**Resuelto:**
**Pendiente:**
**Decisiones:**
**Errores LLM:**
```

---

## Workspace (directorio de trabajo)

```json
"repo": "."                           → dentro de mova-context
"repo": "../mi-proyecto"              → carpeta hermana externa
"repo": "/ruta/absoluta/mi-proyecto"  → ruta absoluta
```

La búsqueda de agents/skills/prompts siempre ocurre dentro de `mova-context`.
La generación de archivos siempre ocurre en la ruta de `repo`.

---

## Acceso a archivos

```text
SI el entorno permite filesystem → leer automáticamente
SI NO                           → solicitar solo los archivos faltantes
                                → continuar cuando estén disponibles
```

---

## Adaptadores

Solo cambia un campo en `project.json`. El workflow es idéntico.

```json
"adapter": "file"
```

```json
"adapter": "postgresql",
"dsn": "postgres://user:pass@host/db"
```

```json
"adapter": "mongodb",
"dsn": "mongodb://user:pass@host/db"
```

---

## Perfiles de LLM

```json
"llm_profile": {
  "type": "powerful",
  "provider": "claude",
  "model": "claude-sonnet-4-6"
}
```

```json
"llm_profile": {
  "type": "local",
  "provider": "ollama",
  "model": "llama3.2"
}
```

El tipo informa al orquestador cómo ajustar el contexto para el modelo. Modelos locales pueden requerir prompts más directos. Modelos potentes pueden manejar contextos más ricos.

---

## Agregar un nuevo idioma

```bash
mkdir -p agents/software/i18n/fr
mkdir -p skills/software/i18n/fr
mkdir -p prompts/software/i18n/fr
mkdir -p docs/i18n/fr
```

Establecer `"lang": "fr"` en `project.json`. Listo.

---

## Agregar un nuevo dominio

```bash
mkdir -p agents/healthcare/i18n/es
mkdir -p skills/healthcare/i18n/es
mkdir -p prompts/healthcare/i18n/es
```

Establecer `"domain": "healthcare"` en los agentes, skills o prompts del proyecto.
