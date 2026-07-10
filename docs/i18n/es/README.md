# Mova Context

> **El conocimiento operativo pertenece al proyecto. El razonamiento pertenece al modelo.**

Docs: **[Español](README.md)** · **[English](../en/README.md)**

---

# Filosofía

Todo el conocimiento de un proyecto vive en archivos de texto versionables.

El CLI (`mova`) únicamente automatiza tareas repetitivas.

Si mañana desaparece el ejecutable `mova`, tu proyecto sigue funcionando porque el conocimiento continúa estando en el repositorio, no dentro de una herramienta.

---

## El problema

Cuando usas IA en un proyecto, gran parte del conocimiento operativo —convenciones, reglas de negocio, decisiones tomadas y memoria del trabajo realizado— termina **atrapado dentro del chat**.

Con el tiempo aparecen siempre los mismos problemas:

- Debes volver a explicar el proyecto en cada conversación.
- Cambias de modelo o de proveedor y pierdes el contexto.
- Cada integrante del equipo explica el proyecto de forma distinta.
- Nadie recuerda con precisión qué se decidió semanas atrás.

Mova Context convierte ese conocimiento operativo en una parte más del repositorio.

En lugar de vivir dentro de conversaciones con un LLM, pasa a vivir en archivos versionables que cualquier modelo puede utilizar sin que tengas que volver a explicárselo a Claude, GPT, Gemini, Ollama o cualquier otro modelo.

---

## Lo esencial, antes que nada

**Mova Context es una convención de archivos, no una herramienta.**

Todo lo necesario para utilizarlo es esta estructura:

```text
workflow.md                       ← especificación que describe cómo construir el contexto

agents/[dominio]/                 ← quién razona (rol, experiencia)
skills/[dominio]/                 ← qué sabe (conocimiento técnico o de negocio)
prompts/[dominio]/                ← qué debe hacer (la tarea)

projects/[proyecto]/
├── project.json                  ← qué agents, skills y prompts utilizar
└── memory.md                     ← historial de sesiones del proyecto
```

Puedes utilizar todo esto **sin instalar absolutamente nada**.

Solo necesitas un agente capaz de acceder al repositorio (Claude Code, Cursor, Claude Desktop, Gemini CLI, etc.) o incluso copiar los archivos manualmente dentro de un chat.

Por ejemplo:

```text
Lee workflow.md, resuelve el proyecto [nombre], ejecuta la task [task] y construye el contexto.
```

Un agente capaz de acceder al repositorio puede seguir `workflow.md` para ensamblar el contexto automáticamente.

Siguiendo esa especificación:

- resuelve el proyecto definido en `project.json`
- carga los `agents`, `skills` y `prompts` correspondientes
- inyecta las variables necesarias
- incorpora la memoria del proyecto
- construye el contexto final

Si trabajas desde un chat web (ChatGPT, Claude.ai o Gemini), donde el modelo no puede acceder directamente al repositorio, **`mova run`** genera exactamente ese mismo contexto listo para copiar y pegar.

**El CLI (`mova`) no es necesario para que Mova Context funcione. Simplemente automatiza el ensamblado del contexto y otras tareas como la gestión de memoria, HTTP y MCP.**

---

## Cómo funciona

```text
                     Mova Context

          agents/
          skills/
          prompts/
          project.json
          memory.md
                 │
                 ▼
           workflow.md
        (especificación)
                 │
      ┌──────────┴──────────┐
      │                     │
      ▼                     ▼
Agente que lee         mova run (CLI)
el repositorio         (opcional)
      │                     │
      └──────────┬──────────┘
                 ▼
        Contexto ensamblado
                 │
                 ▼
 Claude • GPT • Gemini • Ollama
        o cualquier LLM
```

---

## ¿Cuándo conviene usar el CLI?

| Situación | ¿Necesitas el CLI? |
|---|---|
| Ya usas Claude Code, Cursor o un agente que puede leer el repositorio | **No.** El agente sigue `workflow.md` directamente. |
| Quieres pegar el contexto en un chat web (Claude.ai, ChatGPT o Gemini) | **Sí.** `mova run` genera el contexto listo para copiar y pegar. |
| Quieres llamar la API de un modelo desde un script o automatización | **Sí.** Es más rápido que hacer que el modelo lea todos los archivos. |
| Quieres ejecutar un modelo local (Ollama) | **Sí.** `mova run ... \| ollama run modelo` en una sola línea. |
| Quieres guardar la memoria de una sesión sin editar `memory.md` manualmente | **Sí.** `mova memory` actualiza el archivo automáticamente. |
| Quieres exponer el contexto mediante HTTP o como servidor MCP | **Sí.** `mova http` o `mova mcp start`. |

**En resumen:**

Sin el CLI pierdes comodidad.

Con el CLI ganas velocidad, automatización e integración.

La fuente de verdad sigue siendo siempre:

- `workflow.md`
- `agents/`
- `skills/`
- `prompts/`
- `project.json`
- `memory.md`

Nunca el ejecutable.

---

## Antes vs Mova Context

```text
ANTES                               MOVA CONTEXT

Contexto dentro del chat      →      Contexto dentro del repositorio

Cambiar de modelo             →      Cambiar una línea en project.json
significa empezar de nuevo

Cada desarrollador            →      Una única fuente de verdad
explica distinto

Las decisiones                →      memory.md conserva el historial
se pierden

El conocimiento depende       →      El conocimiento pertenece al proyecto,
del proveedor                        no al proveedor
```

---

## Instalar el CLI (opcional)

```bash
go build -o mova ./src/cli
```

Consulta **[COMMANDS.md](COMMANDS.md)** para ver todos los comandos (`run`, `memory`, `search`, `focus`, `mcp`, `http`, etc.).

---

## Ejemplo mínimo

Existe un proyecto de ejemplo completo en:

```
projects/pruebas-locales/
```

Puedes inspeccionar su `project.json` o generar el contexto ejecutando:

```bash
mova run pruebas-locales
```

---

## Ir más profundo

| Quiero... | Documento |
|---|---|
| Ver todos los comandos (incluye memoria, Focus, MCP y HTTP) | [COMMANDS.md](COMMANDS.md) |
| Leer la especificación completa que siguen los modelos | [workflow.md](../../../workflow.md) |
| Entender el código fuente (Resolvers, Adapters y cómo extenderlo) | [SOURCE.md](../SOURCE.md) *(English)* |

---

> **El conocimiento operativo pertenece al proyecto. El razonamiento pertenece al modelo.**
>
> Mova Context es la convención formada por `workflow.md`, `agents/`, `skills/`, `prompts/`, `project.json` y `memory.md`.
>
> El CLI simplemente automatiza el trabajo con esa convención; no la reemplaza.