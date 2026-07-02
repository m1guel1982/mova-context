# CAMBIOS_REALIZADOS.md

Resumen de todas las correcciones y mejoras realizadas en Mova Context.  
La arquitectura, filosofía y esencia del proyecto se mantienen intactas.

---

## Nueva funcionalidad — Context Compiler (`mova compile`)

Extensión 100% Core, opt-in, cero impacto en proyectos existentes (sin el bloque `contextCompiler` en `project.json`, ningún comportamiento cambia).

- **Fase 1 — Telegrama Semántico** (`cli/compiler_semantic.go`): distila agents/skills/prompts/memory quitando saludos y frases de relleno, preservando siempre reglas críticas y placeholders.
- **Fase 2 — Poda quirúrgica** (`cli/compiler_focus.go`): cuando `project.json` declara `focus`, extrae solo el símbolo de código, sección de documento o bloque cronológico solicitado — nunca archivos completos sin relación.
- **Glue** (`cli/compiler_run.go`): `mova compile` (manual, siempre disponible) y el enrutamiento automático de `mova run` / `get_full_context` (MCP) cuando `contextCompiler.mode` es `"automatic"`.
- Nuevos campos en `project.json`: `contextCompiler` (`enabled`/`mode`/`strategy`) y `focus` (global o por task, prioridad `task > global`).
- Documentación: [`docs/i18n/es/context-compiler.md`](docs/i18n/es/context-compiler.md) / [`en`](docs/i18n/en/context-compiler.md), arquitectura [Core + Extensions](docs/i18n/es/core-extensions.md).
- Caso de uso real: [`projects/compiler-demo/`](projects/compiler-demo/project.json).

---

## Correcciones realizadas

### Comandos corregidos

**`mova list`** — ahora muestra todos los proyectos correctamente.
- Causa del bug: `acme-demo` y `pruebas-locales` tenían `project.json` con esquema incorrecto, causando errores silenciosos al parsearlos.
- Fix: `ListProjects` ahora recorre recursivamente `projects/` usando `filepath.WalkDir`. Detecta automáticamente cualquier `project.json` a cualquier profundidad. Nunca usa listas hardcodeadas.

**`mova run omnicanal-demo`** — ya no falla con error de unmarshal.
- Causa: `task.agents` y `task.skills` eran objetos `{"domain":...,"use":[...]}` pero el struct `Task` espera `[]string`.
- Fix: `project.json` corregido para usar la sintaxis actual.

**Contexto vacío en `mova run ley-21719`** — ahora carga todos los recursos.
- Causa: `GetKnowledge` no manejaba la estructura `i18n/` ni las subcarpetas (`engineering/`, `business/`, `legal/`).
- Fix: búsqueda en 10 pasos con fallback recursivo completo.

### Errores solucionados

| Error | Causa | Solución |
|-------|-------|----------|
| `cannot unmarshal object into []string` | task.agents/skills eran objetos | Corregir project.json de omnicanal-demo |
| Contexto vacío (agents/skills/prompts no cargados) | `GetKnowledge` no resolvía rutas i18n | Búsqueda recursiva en file_adapter.go |
| `mova list` solo mostraba ley-21719 | project.json inválidos provocaban skip silencioso | Corregir todos los project.json |

### Proyectos corregidos

**`omnicanal-demo/project.json`**
- Añadidos `agents` y `skills` al nivel del proyecto (requerido por el engine).
- Task-level `agents`/`skills` cambiados de objeto a `[]string`.

**`acme-demo/project.json`**
- `agents`/`skills` cambiados de `{"base":[...],"custom":[...]}` a `{"domain":"base","use":[...],"custom":[...]}`.
- Task-level: `prompt` de objeto a string, `agents`/`skills` de objeto a `[]string`.
- Añadido `lang: "es"` y `llm_profile`.

**`pruebas-locales/project.json`**
- Completamente reescrito al esquema actual.
- Task-level `prompt` de `{"base":"..."}` a `"..."` (string).
- Task-level `agents`/`skills` de objetos a `[]string`.
- Añadido `lang: "es"` y `llm_profile`.
- Simplificado (removidos campos no estándar como `focus` del project.json).

### Mejoras en `file_adapter.go`

**`GetKnowledge`** — nueva búsqueda en 10 pasos:

```text
1. domain/i18n/lang/name.md          (ruta i18n exacta)
2. domain/i18n/en/name.md            (fallback en, exacta)
3. domain/lang/name.md               (legacy flat, exacta)
4. domain/en/name.md                 (legacy en, exacta)
5. domain/name.md                    (sin lang)
6. agents/name.md                    (root legacy)
7. walk domain/i18n/lang/            (recursivo, maneja engineering/ business/ etc.)
8. walk domain/i18n/en/              (recursivo, fallback en)
9. walk domain/                      (recursivo, todo el dominio)
10. walk agents/                     (global, encuentra custom/ etc.)
```

**`ListProjects`** — ahora usa `filepath.WalkDir` para detectar proyectos automáticamente a cualquier profundidad. Sin listas hardcodeadas.

### Mejoras en `engine.go`

Añadida carga automática de archivos Core:

- `yagni-core` — cargado antes de cualquier agent, exactamente una vez.
- `kiss-dry-core` — cargado antes de cualquier skill, exactamente una vez.
- `ockham-core` — cargado antes del prompt, exactamente una vez.

Si el nombre de un core aparece en la lista de agents/skills del proyecto, se omite (ya fue cargado). Sin duplicados.

---

## Cambios en la resolución recursiva

### Antes
```text
GetKnowledge buscaba solo:
  agents/<domain>/<lang>/name.md
  agents/<domain>/en/name.md
  agents/<domain>/name.md
  agents/name.md
```
No encontraba nada en la estructura `i18n/` ni en subdirectorios.

### Ahora
La búsqueda tiene 10 pasos con fallback completo. Funciona para:

- **Agents**: `agents/legal/i18n/es/abogado-datos.md` ✓
- **Skills**: `skills/callcenter/i18n/es/politica-ventas.md` ✓
- **Prompts**: `prompts/base/i18n/es/engineering/audit-module.md` ✓ (recursivo, maneja subdirs)
- **Core files**: `agents/base/i18n/es/yagni-core.md` ✓
- **Custom**: `agents/custom/acme-backend.md` ✓ (encontrado por búsqueda global)
- **Cualquier profundidad**: sin importar cuántos subdirectorios haya
- **Cualquier idioma**: con fallback a `en` y luego sin idioma
- **Todos los adaptadores**: la lógica está en `file_adapter.go`; `db_adapter.go` sigue la misma interfaz

---

## Documentación actualizada

| Archivo | Modificación |
|---------|-------------|
| `workflow.md` | Añadida sección `## ARCHIVOS CORE` explicando la carga automática. Actualizado orden de carga en `## REGLAS DE CARGA`. |
| `docs/i18n/es/validation-guide.md` | Nuevo — guía de validación completa en español. |
| `docs/i18n/en/validation-guide.md` | Nuevo — guía de validación completa en inglés. |
| `examples/i18n/es/pruebas-locales/README.md` | Nuevo — ejemplo mínimo oficial. |
| `examples/i18n/en/pruebas-locales/README.md` | Nuevo — minimal official example. |
| `examples/i18n/es/mcp/README.md` | Nuevo — ejemplo MCP con CURL y Postman. |
| `examples/i18n/en/mcp/README.md` | Nuevo — MCP example with CURL and Postman. |
| `examples/i18n/es/ollama/README.md` | Nuevo — ejemplo Ollama con Llama 3.1. |
| `examples/i18n/en/ollama/README.md` | Nuevo — Ollama example with Llama 3.1. |
| `examples/i18n/es/postgresql/README.md` | Nuevo — ejemplo PostgreSQL. |
| `examples/i18n/en/postgresql/README.md` | Nuevo — PostgreSQL example. |

---

## Compatibilidad

| Requisito | Estado |
|-----------|--------|
| No se rompió compatibilidad | ✅ Los project.json existentes válidos siguen funcionando |
| No cambió la arquitectura | ✅ Misma estructura de directorios, mismos archivos |
| No cambió la filosofía | ✅ KISS, DRY, YAGNI, Navaja de Ockham, convención sobre configuración |
| `workflow.md` mantiene su esencia | ✅ Solo se añadió la sección de archivos core |
| `project.json` sigue siendo la fuente única | ✅ Sin nueva configuración ni archivos adicionales |
| Compatibilidad con modelos potentes y locales | ✅ `llm_profile` intacto |
| Compatibilidad con adaptadores actuales y futuros | ✅ Interfaz `Adapter` sin cambios |

---

## Resumen final

Los cambios realizados son únicamente correcciones de bugs y completado de funcionalidades pendientes:

1. **Bug crítico en `GetKnowledge`**: no resolvía la estructura `i18n/` real del proyecto. Todos los contextos salían vacíos. Corregido con búsqueda en 10 pasos con fallback recursivo completo.

2. **Bug en `project.json` de tres proyectos**: usaban un esquema anterior incompatible con el engine actual. Corregidos para usar el esquema `{"domain":"...","use":[...]}` correcto.

3. **`mova list` incompleto**: solo mostraba proyectos con `project.json` válido. Al corregir los `project.json`, ahora muestra los 4 proyectos. Además se mejoró para recorrer recursivamente.

4. **Core files no se cargaban**: `yagni-core`, `kiss-dry-core` y `ockham-core` existían pero no se incluían en el contexto generado. Ahora se cargan automáticamente, una sola vez, antes de su sección respectiva.

5. **Documentación**: añadidos ejemplos para MCP, Ollama y PostgreSQL que ya existían como funcionalidades pero carecían de ejemplos. Añadidas guías de validación bilingües.

Todo esto mantiene Mova Context simple, limpio y fácil de entender.
