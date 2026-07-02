# Mova Context

**Una capa portable de conocimiento operativo para proyectos asistidos por IA.**

> **El conocimiento operativo pertenece al proyecto. El razonamiento pertenece al modelo.**

Compatible con cualquier herramienta o modelo capaz de trabajar con archivos de texto.

---

## El problema

Cuando un proyecto utiliza IA, gran parte del conocimiento operativo termina distribuido entre chats, configuraciones específicas, documentos aislados y memoria individual de los integrantes del equipo.

Con el tiempo aparecen problemas habituales:

- decisiones técnicas difíciles de rastrear
- convenciones repetidas constantemente
- contexto reconstruido en cada sesión
- dependencia de herramientas o proveedores concretos

**El contexto del proyecto debería permanecer junto al proyecto.**

---

## La idea

Mova Context mantiene el conocimiento operativo en archivos versionables que viajan junto al repositorio.

```text
Proyecto
│
├── Código
├── Convenciones
├── Memoria
├── Reglas operativas
└── Contexto compartido
```

Los modelos pueden cambiar. El contexto del proyecto permanece.

---

## Qué es

Una convención organizacional para gestionar conocimiento operativo en proyectos asistidos por IA.

Su objetivo es facilitar:

- reutilización de contexto
- preservación de conocimiento
- colaboración entre equipos
- portabilidad entre herramientas
- trazabilidad de decisiones

---

## Qué no es

- No es un framework de IA
- No es un runtime
- No es una plataforma de automatización
- No reemplaza Claude, GPT, Gemini ni ningún modelo

---

## Estructura

```text
mova-context/
│
├── README.md                    ← selector de idioma
├── workflow.md                  ← único punto de entrada
│
├── docs/i18n/{es,en}/           ← documentación bilingüe
│
├── agents/{domain}/i18n/{lang}/ ← quién es el modelo
├── skills/{domain}/i18n/{lang}/ ← qué sabe el modelo
├── prompts/{domain}/i18n/{lang}/← qué debe hacer el modelo
│
├── projects/{PROJECT}/
│   ├── project.json             ← fuente de verdad
│   └── memory.md                ← historial de sesiones
│
├── adapters/                    ← filesystem · postgresql · mongodb
├── schema/                      ← esquemas de base de datos
├── cli/                         ← herramienta de línea de comandos
└── mcp/                         ← integración MCP
```

---

## project.json — fuente de verdad

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
    "empresa": "Acme Corp"
  },
  "agents": { "domain": "software", "use": ["backend-dev"] },
  "skills": { "domain": "software", "use": ["api-security"] },

  "tasks": {
    "mi-tarea": {
      "prompt": "review-project",
      "variables": { "modulo": "auth" }
    }
  }
}
```

Un solo archivo controla todo. El resto son archivos Markdown.

---

## LLM soportados

El proyecto funciona igual con cualquier modelo. Solo cambia `project.json`.

| Campo | Valor |
|-------|-------|
| `"llm": "claude"` | Anthropic Claude |
| `"llm": "gpt"` | OpenAI GPT |
| `"llm": "gemini"` | Google Gemini |
| `"llm": "ollama"` | Ollama (local) |
| `"llm": "openrouter"` | OpenRouter (multi-modelo) |

El código, los agentes, las skills y los prompts nunca cambian.

---

## Adaptadores

| Adaptador | Descripción |
|-----------|-------------|
| `"adapter": "file"` | Archivos Markdown (default) |
| `"adapter": "postgresql"` | Base de datos PostgreSQL |
| `"adapter": "mongodb"` | Base de datos MongoDB |

Solo cambia el adaptador. El workflow permanece igual.

---

## CLI

```bash
mova list                                    # ver proyectos disponibles
mova run [proyecto] [tarea]                  # generar contexto
mova compile [proyecto] [tarea]              # contexto distilado + podado → contexto.txt
mova memory [proyecto] "respuesta del LLM"   # actualizar memoria
mova init [nombre]                           # crear nuevo proyecto
mova mcp start                               # iniciar servidor MCP
```

---

## Ejemplos oficiales

- [Ley 21.719 — Protección de Datos (Chile)](../../examples/i18n/es/ley-21719/)
- [Omnicanal Empresarial](../../examples/i18n/es/omnicanal/)

---

## Documentación

- [workflow.md](workflow.md) — guía operacional completa
- [architecture.md](architecture.md) — filosofía y principios
- [cli.md](cli.md) — referencia de la CLI
- [mcp.md](mcp.md) — integración MCP
- [adapters.md](adapters.md) — adaptadores de almacenamiento
- [schema.md](schema.md) — esquema de base de datos
- [context-compiler.md](context-compiler.md) — `mova compile`, Fase 1 y Fase 2
- [core-extensions.md](core-extensions.md) — arquitectura Core + Extensions

---

## Principio fundamental

> **El conocimiento operativo pertenece al proyecto. El razonamiento pertenece al modelo.**

Los modelos cambiarán. Los proveedores cambiarán. Las herramientas cambiarán.

El conocimiento acumulado del proyecto debería permanecer bajo el control del equipo que lo construye.
