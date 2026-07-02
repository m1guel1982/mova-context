# PUBLICACION_GITHUB.md

> Documento privado. Guía de publicación por fases de Mova Context en GitHub.
> Cada fase deja el repositorio completamente funcional.

---

## Principio de publicación

> Mejor muchas publicaciones pequeñas que un lanzamiento grande.

Cada fase tiene un objetivo claro, archivos específicos y mensajes de commit sugeridos.
Ninguna fase deja el repositorio en un estado incompleto o inconsistente.

---

## Fase 1 — Organización del proyecto

**Objetivo:** Publicar la estructura base. Cualquiera que abra el repo entiende el proyecto en menos de 5 minutos.

**Archivos:**
```
README.md
workflow.md
docs/i18n/es/README.md
docs/i18n/en/README.md
docs/i18n/es/workflow.md
docs/i18n/en/workflow.md
.gitignore
LICENCE
```

**Commit sugerido:**
```
feat: initial structure — bilingual operational knowledge base

- Single workflow.md as the only entry point
- Bilingual README (es/en) with language selector
- docs/i18n/ convention established
- Clean, self-documenting structure

Closes #1
```

**Publicación LinkedIn:**
```
Estoy trabajando en Mova Context: una convención para organizar el conocimiento operativo de proyectos asistidos por IA.

La idea es simple: el conocimiento del proyecto vive con el proyecto, en archivos Markdown versionados.

Los modelos pueden cambiar. El contexto permanece.

Primera versión en GitHub → [link]

Compatible con Claude · GPT · Gemini · Ollama y cualquier modelo.
```

**Publicación Reddit (r/MachineLearning o r/LocalLLaMA):**
```
[Project] Mova Context — Organizing AI context as a project convention (not a framework)

Instead of rebuilding context every session, keep operational knowledge in versionable Markdown files alongside your code.

No runtime. No dependencies. Works with any LLM.

GitHub: [link]
```

---

## Fase 2 — Estructura i18n

**Objetivo:** Demostrar la convención de internacionalización con carpetas reales.

**Archivos:**
```
agents/base/i18n/es/yagni-core.md
agents/base/i18n/en/yagni-core.md
agents/base/i18n/es/backend-dev.md
agents/base/i18n/en/backend-dev.md
skills/base/i18n/es/kiss-dry-core.md
skills/base/i18n/en/kiss-dry-core.md
prompts/base/i18n/es/ockham-core.md
prompts/base/i18n/en/ockham-core.md
docs/i18n/es/architecture.md
docs/i18n/en/architecture.md
```

**Commit sugerido:**
```
feat: i18n convention — agents/skills/prompts organized by domain and language

- agents/{domain}/i18n/{lang}/ structure established
- Base agents and skills bilingual (es/en)
- New language = new folder, nothing else changes

Closes #2
```

**Publicación LinkedIn:**
```
Actualización de Mova Context: el conocimiento ahora es multilingüe.

La convención: agents/{dominio}/i18n/{idioma}/

Agregar un nuevo idioma es crear una carpeta. Sin configuración adicional. Sin código.

Esto permite que el mismo proyecto funcione en español, inglés, francés o cualquier idioma que necesites.

GitHub: [link]
```

---

## Fase 3 — README

**Objetivo:** README completo en ambos idiomas que explique el proyecto de forma clara para técnicos y no técnicos.

**Archivos:**
```
docs/i18n/es/README.md (completo)
docs/i18n/en/README.md (completo)
```

**Commit sugerido:**
```
docs: complete bilingual README

- Full project explanation (problem, solution, structure)
- project.json reference
- LLM compatibility table
- Adapter comparison
- Clear distinction between what it is and what it is not

Closes #3
```

**Publicación LinkedIn:**
```
README actualizado en Mova Context.

Incluye: el problema que resuelve, cómo funciona, la estructura del proyecto, compatibilidad con cualquier LLM y comparación de adaptadores.

Si tu equipo usa IA y el contexto se reconstruye en cada sesión, esto puede ayudar.

GitHub: [link]
```

---

## Fase 4 — Los dos ejemplos oficiales

**Objetivo:** Publicar los dos ejemplos completos. Este es el corazón del repositorio.

**Archivos:**
```
examples/i18n/es/ley-21719/README.md
examples/i18n/en/privacy-law/README.md
examples/i18n/es/omnicanal/README.md
examples/i18n/en/omnichannel/README.md
projects/ley-21719/project.json
projects/ley-21719/memory.md
projects/omnicanal-demo/project.json
agents/legal/i18n/es/abogado-datos.md
agents/legal/i18n/en/data-lawyer.md
agents/callcenter/i18n/es/ejecutivo-ventas.md
skills/legal/i18n/es/ley-21719-obligaciones.md
skills/legal/i18n/es/derechos-titulares.md
skills/callcenter/i18n/es/politica-ventas.md
prompts/legal/i18n/es/analizar-contrato-datos.md
prompts/legal/i18n/es/responder-solicitud-titular.md
```

**Commit sugerido:**
```
feat: two official examples — Chilean Privacy Law 21.719 and Enterprise Omnichannel

Example 1: Ley 21.719 (flagship)
- Complete end-to-end flow (WhatsApp → backend → audit)
- Same flow demonstrated with 3 adapters (file/postgresql/mongodb)
- Same flow demonstrated with 10 LLMs
- Shows why the backend never changes

Example 2: Enterprise Omnichannel
- Same knowledge, all channels (WhatsApp/WebChat/IVR/CRM)
- Demonstrates knowledge decoupling from channel logic

Closes #4
```

**Publicación LinkedIn:**
```
Mova Context ahora tiene su primer caso insignia: Ley 21.719 de Protección de Datos de Chile.

El ejemplo muestra algo importante: cuando cambió la ley, el backend no cambió.

Solo cambió el conocimiento en Mova Context.

El mismo flujo funciona con archivos, PostgreSQL o MongoDB.
El mismo flujo funciona con Claude, GPT, Gemini o Llama.

Para las empresas que necesitan cumplir la ley en todos sus canales (web, app, call center, IVR, WhatsApp), esto puede ser la diferencia entre una migración de 6 meses o un cambio de configuración.

GitHub: [link]
```

**Publicación Reddit (r/LegalTech o r/Chile):**
```
[Project] How we decoupled Chilean Privacy Law 21.719 knowledge from software systems

Chilean Law 21.719 (Data Protection) came into force in 2026. It affects every company processing personal data in Chile.

Instead of updating each system separately (web, mobile, call center, IVR, CRM), we put the legal knowledge in one place and let all systems consume it via LLM.

Backend never changed. Only the knowledge layer was updated.

GitHub: [link]
```

---

## Fase 5 — CLI

**Objetivo:** Publicar la CLI funcional con binarios precompilados.

**Archivos:**
```
cli/main.go
cli/Makefile
cli/go.mod
cli/dist/mova-linux-amd64
cli/dist/mova-darwin-amd64
cli/dist/mova-darwin-arm64
cli/dist/mova-windows-amd64.exe
docs/i18n/es/cli.md
docs/i18n/en/cli.md
```

**Commit sugerido:**
```
feat: CLI — generate context, update memory, list projects

Commands: run, memory, list, init, search
Binaries: Linux, macOS (Intel + Apple Silicon), Windows

Closes #5
```

**Publicación LinkedIn:**
```
Mova Context ahora tiene CLI.

```bash
mova run ley-21719 analizar-contrato > contexto.txt
mova memory ley-21719 "$(pbpaste)"
```

Dos comandos. El primero genera el contexto. El segundo guarda la respuesta del LLM.

Compatible con cualquier LLM web (Claude, ChatGPT, Gemini).

GitHub: [link]
```

---

## Fase 6 — MCP

**Objetivo:** Publicar la integración MCP para Claude Desktop y herramientas compatibles.

**Commit sugerido:**
```
feat: MCP server — expose project context to Claude Desktop and compatible tools

mova mcp start [--port 3000]

Closes #6
```

**Publicación LinkedIn:**
```
Mova Context ahora se integra con Claude Desktop vía MCP.

En lugar de copiar y pegar el contexto, Claude Desktop lo lee directamente del servidor MCP.

Cada proyecto tiene su contexto disponible automáticamente.

GitHub: [link]
```

---

## Fase 7 — Adaptadores

**Objetivo:** Publicar los adaptadores de PostgreSQL y MongoDB con esquemas completos.

**Archivos:**
```
adapters/filesystem/adapter.go
adapters/postgresql/adapter.go
adapters/mongodb/adapter.go
schema/postgresql.sql
schema/mongodb.md
docs/i18n/es/adapters.md
docs/i18n/en/adapters.md
```

**Commit sugerido:**
```
feat: adapters — filesystem (default), postgresql, mongodb

Same workflow.md for all adapters.
Only "adapter" field in project.json changes.

Closes #7
```

---

## Fase 8 — LLMs locales

**Objetivo:** Demostrar soporte completo para Ollama y modelos locales.

**Commit sugerido:**
```
feat: local LLM support — ollama, llama, deepseek, mistral, gemma, phi, qwen

llm_profile.type: "local" adjusts context for smaller models
Same agents, skills, prompts for all models

Closes #8
```

**Publicación LinkedIn:**
```
Mova Context ahora funciona con LLMs locales: Ollama, Llama, DeepSeek, Mistral, Gemma, Phi, Qwen.

Para empresas que no pueden enviar datos a APIs externas, esto permite tener el mismo workflow con modelos corriendo en infraestructura propia.

GitHub: [link]
```

---

## Fase 9 — Documentación completa

**Objetivo:** Documentación completa en ambos idiomas.

**Archivos:**
```
docs/i18n/es/{architecture,adapters,cli,mcp,schema,memory,faq,roadmap}.md
docs/i18n/en/{architecture,adapters,cli,mcp,schema,memory,faq,roadmap}.md
```

**Commit sugerido:**
```
docs: complete bilingual documentation

All docs available in Spanish and English.
No doc exceeds 250 lines.
Navigation follows Apple/Stripe style: simple, fast, predictable.

Closes #9
```

---

## Fase 10 — Versión estable v1.0

**Objetivo:** Primera versión estable del proyecto. Tag v1.0.0 en GitHub.

**Commit sugerido:**
```
release: v1.0.0 — stable release

- Bilingual documentation (es/en)
- Two complete official examples
- CLI with binaries for Linux, macOS, Windows
- MCP integration
- Three adapters (file, postgresql, mongodb)
- Support for 10+ LLMs
- i18n convention established

Closes #10
```

**Publicación LinkedIn:**
```
Mova Context v1.0 — primera versión estable.

Lo que incluye:
✓ Documentación bilingüe (es/en)
✓ Dos ejemplos oficiales completos
✓ CLI con binarios para Linux, macOS y Windows
✓ Integración MCP
✓ Tres adaptadores (archivos, PostgreSQL, MongoDB)
✓ Soporte para 10+ modelos (Claude, GPT, Gemini, Ollama y más)

El principio sigue siendo el mismo desde el día uno:

El conocimiento operativo pertenece al proyecto. El razonamiento pertenece al modelo.

GitHub: [link]
```

**Publicación Reddit (r/opensource):**
```
[Show HN style] Mova Context v1.0 — A convention for organizing AI context alongside your code

Not a framework. Not a runtime. A convention.

The idea: operational knowledge (agents, skills, prompts, memory) lives in versionable Markdown files alongside your code. Any LLM can consume it.

Works with Claude, GPT, Gemini, Ollama, and any model that can read text.

GitHub: [link]
```
