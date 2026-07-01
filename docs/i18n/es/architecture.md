# Arquitectura — Mova Context

## Principio fundamental

> **El conocimiento operativo pertenece al proyecto. El razonamiento pertenece al modelo.**

Mova Context no impone cómo razona el modelo. Organiza qué conocimiento recibe.

---

## Los cuatro principios de diseño

Cada principio vive en un archivo núcleo de dos líneas. Pueden reemplazarse libremente.

### YAGNI → aplicado a Agents

*You Aren't Gonna Need It*

El agente no asume necesidades futuras. No crea abstracciones, endpoints ni estructuras si la tarea no lo solicita explícitamente.

Archivo: `agents/base/i18n/es/yagni-core.md`

### KISS → aplicado a Skills

*Keep It Simple, Stupid*

Cada skill resuelve una sola cosa, de la forma más directa posible, usando lo que ya existe antes de proponer algo nuevo.

Archivo: `skills/base/i18n/es/kiss-dry-core.md`

### DRY → aplicado a Skills

*Don't Repeat Yourself*

Una regla o instrucción existe en un solo lugar. El resto la referencia, no la copia.

### Navaja de Ockham → aplicado a Prompts

Entre múltiples soluciones válidas, el modelo elige la más simple. Sin prosa explicativa innecesaria.

Archivo: `prompts/base/i18n/es/ockham-core.md`

---

## Cómo reemplazar un principio

Para un proyecto específico (sin afectar el repo global):

1. Crear `agents/custom/i18n/es/mi-regla.md` con tu propia regla
2. En `project.json`, agregar tu archivo en agents y quitar el original
3. El resto del repo no cambia

Para cambiar globalmente: editar el archivo núcleo directamente.

---

## Flujo de datos

```text
Usuario
  │
  └─→ Lee workflow.md → [PROJECT] → [TASK]
          │
          └─→ Lee project.json
                │
                ├─→ Resuelve lang, llm, adapter
                │
                ├─→ Carga agents (quién es el modelo)
                ├─→ Carga skills (qué sabe el modelo)
                ├─→ Carga prompt (qué debe hacer el modelo)
                └─→ Lee memory.md (historial)
                          │
                          └─→ Inyecta {{VARIABLES}}
                                    │
                                    └─→ LLM ejecuta
                                              │
                                              └─→ Actualiza memory.md
```

---

## Separación de responsabilidades

| Componente | Responsabilidad |
|------------|-----------------|
| `project.json` | Fuente de verdad. Configuración y orquestación |
| `workflow.md` | Orquestador. Lee config y dirige el flujo |
| `agents/` | Personalidad y restricciones del modelo |
| `skills/` | Conocimiento específico del dominio |
| `prompts/` | Instrucciones concretas de la tarea |
| `memory.md` | Historial de sesiones |

---

## Portabilidad

El mismo proyecto funciona con:

- Cualquier LLM (Claude, GPT, Gemini, Ollama, local)
- Cualquier adaptador (archivos, PostgreSQL, MongoDB)
- Cualquier idioma (es, en, fr, pt, y cualquier otro)
- Cualquier dominio (software, legal, callcenter, healthcare, etc.)

Solo cambia `project.json`. El conocimiento permanece igual.

---

## Extensibilidad

Agregar un idioma → crear carpeta `i18n/{lang}/`

Agregar un dominio → crear carpeta `{domain}/i18n/{lang}/`

Agregar un adaptador → implementar en `adapters/{nombre}/`

Agregar un proyecto → crear `projects/{nombre}/project.json`

Ningún cambio requiere modificar el core del sistema.
