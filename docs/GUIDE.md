# GUIDE.md

## Propósito

Mova Context es una convención para organizar conocimiento operativo asociado a proyectos asistidos por IA.

No define una implementación obligatoria.

No reemplaza herramientas como Cursor Rules, Claude Projects, Claude Code, RooCode, Codex o similares.

Puede complementarlas proporcionando una estructura portable y versionable para almacenar contexto del proyecto.

---

## Principio fundamental

> El conocimiento operativo pertenece al proyecto. El razonamiento pertenece al modelo.

El objetivo es conservar junto al proyecto:

* convenciones
* decisiones técnicas
* memoria operativa
* workflows
* documentación relevante

independientemente del proveedor de IA utilizado.

---

## Estructura sugerida

```text
proyecto/
│
├── project.json
├── memory.md
├── agents/
├── skills/
├── prompts/
└── workflows/
```

La estructura es una recomendación.

Cada equipo puede adaptarla libremente.

---

## Qué es obligatorio y qué es opcional

### Recomendado

* conocimiento asociado al proyecto
* documentación versionada
* contexto reutilizable

### Opcional

* project.json
* memory.md
* agents/
* skills/
* prompts/
* workflows/
* cualquier estructura específica

No existe una estructura oficial obligatoria.

---

## project.json

Puede utilizarse como punto central para describir un proyecto.

Ejemplo:

```json
{
  "project": "acme-demo",
  "description": "API interna",
  "stack": "FastAPI + PostgreSQL"
}
```

Cada equipo decide qué información almacenar.



---

## memory.md

Puede utilizarse para registrar:

* decisiones técnicas
* acuerdos del equipo
* pendientes
* aprendizajes

Ejemplo:

```md
## 2026-05-28

Decisión:
Usar JWT RS256.

Motivo:
Rotación de claves y separación entre firma y validación.
```

No existe un formato obligatorio.

---

## Agents, Skills y Prompts

Los directorios:

```text
agents/
skills/
prompts/
```

son ejemplos de organización.

Pueden representar:

* roles
* checklists
* convenciones
* plantillas
* documentación reutilizable

No son obligatorios.

Cada organización puede reemplazarlos completamente.

Ejemplo:

```text
agents/base/
skills/base/
prompts/base/
```

puede convertirse en:

```text
agents/company/
skills/company/
prompts/company/
```

o cualquier otra estructura.

---

## Convención de nombres

Los ejemplos utilizan nombres en inglés:

```text
architect.md
backend-dev.md
jwt-security.md
review-project.md
```

únicamente por familiaridad dentro de la industria.

También podrían llamarse:

```text
arquitecto.md
seguridad-api.md
revision-general.md
```

o cualquier otro nombre.

Lo importante es mantener consistencia dentro del proyecto.

---

## Variables

Las variables son una convención opcional para evitar duplicación.

Ejemplo:

```json
{
  "project": "acme-demo",
  "stack": "FastAPI + PostgreSQL"
}
```

Documentos relacionados podrían referenciar:

```text
{{PROJECT}}
{{STACK}}
```

Mova Context no define cómo resolver esas variables.

Cada equipo puede:

* ignorarlas
* resolverlas manualmente
* utilizar scripts propios
* utilizar herramientas externas

---

## Cuándo usar ponytail

`ponytail.md` (modo "lazy senior dev") es un **prompt custom global**, no parte del workflow base. Vive en `prompts/custom/ponytail.md` y un proyecto lo activa solo cuando lo agrega explícitamente a su `project.json` — no se carga por defecto.

**Conviene activarlo cuando:**

* El proyecto ya tiene una base de código funcionando y el riesgo es sobre-construir, no construir desde cero
* Hay presión a agregar abstracciones, dependencias o capas "por si acaso" sin un caso de uso concreto encima
* Se está haciendo refactor o mantenimiento, donde el objetivo es resolver con el mínimo cambio posible
* El equipo quiere que cada simplificación intencional quede documentada (`ponytail:` / `lazy:` en el código) en vez de quedar implícita

**No conviene activarlo cuando:**

* Se está diseñando la arquitectura inicial de un sistema nuevo y complejo (ahí primero se decide la forma, después se poda)
* La tarea ya es, en sí misma, escribir documentación, tests exhaustivos o cubrir validaciones de seguridad — esas áreas están explícitamente fuera del modo perezoso

**Cómo agregarlo a un proyecto:**

```json
"tasks": {
  "mi-task-de-refactor": {
    "prompt": {"custom": "ponytail"},
    "skills": {"base": ["lazy-minimalism"]}
  }
}
```

La skill base `skills/base/engineering/lazy-minimalism.md` es la versión generalizada y reutilizable del mismo principio (YAGNI + escalera de decisión), pensada para combinarse con cualquier agente — se puede usar sola, sin cargar el prompt completo de ponytail, cuando solo se quiere recordatoria del criterio sin el tono de "modo" completo.

**Dónde vive cada pieza:**

```text
prompts/custom/ponytail.md              → el prompt completo, opt-in por proyecto
skills/base/engineering/lazy-minimalism.md → el criterio reutilizable, sin tono de "modo"
```

---

## Núcleos compartidos (evitar repetir reglas globales)

Cuando una regla aplica a *todos* los agents, *todas* las skills o *todos* los prompts de un workspace (no de un proyecto específico), escribirla una sola vez en un archivo núcleo y referenciarla desde cada documento, en vez de pegarla en cada uno. Esto es DRY aplicado al propio sistema de prompts, no solo al código que el modelo genera.

Convención usada en este repo:

```text
agents/base/engineering/yagni-core.md     → regla de comportamiento para todo agent
skills/base/engineering/kiss-dry-core.md  → regla de resolución para toda skill
prompts/base/engineering/ockham-core.md   → regla de salida para todo prompt
```

Cada agent/skill/prompt referencia su núcleo con una línea (`YAGNI: ver yagni-core.md`), no copia el texto. Si la regla cambia, se edita en un solo lugar.

Esto es opcional — un equipo pequeño con pocos documentos puede preferir reglas inline. Vale la pena centralizar cuando el número de agents/skills/prompts crece lo suficiente como para que copiar y pegar la misma regla en cada uno se vuelva, en sí mismo, una violación de DRY.

---

## Matriz de los 4 principios

| Principio | Dónde vive la regla (una vez) | Quién la hereda |
|---|---|---|
| YAGNI | `agents/base/engineering/yagni-core.md` | todo `agents/base/*` y `agents/custom/*` que lo referencie |
| KISS + DRY | `skills/base/engineering/kiss-dry-core.md` | todo `skills/base/*` y `skills/custom/*` que lo referencie |
| Navaja de Ockham | `prompts/base/engineering/ockham-core.md` | todo `prompts/base/*` y `prompts/custom/*` que lo referencie |
| Minimalismo perezoso (ponytail) | `skills/base/engineering/lazy-minimalism.md` + `prompts/custom/ponytail.md` | opt-in por proyecto, no heredado por defecto |

La "herencia" aquí no es herencia de código — es una convención de una línea: cada documento que adopta un núcleo escribe `YAGNI: ver yagni-core.md` (o el equivalente) en su sección de Rol/Objetivo. No hay mecanismo automático que lo imponga; es texto que tú decides incluir al crear el archivo. Si lo omites, ese documento simplemente no hereda la regla — esa es la naturaleza de que esto sea convención y no framework.

**Cómo agregar un nuevo agent/skill/prompt heredando los núcleos:**

```markdown
# Rol
[lo que hace este agente específico]
YAGNI: ver `yagni-core.md`.

# Reglas
[lo propio de este agente — nunca repetir lo que ya dice el núcleo]
```

Si en el futuro decides cambiar la redacción de la regla YAGNI, se edita una sola vez en `yagni-core.md` y todos los documentos que la referencian quedan actualizados en su significado sin tocarlos — porque "ver yagni-core.md" siempre apunta al texto vigente.

---

## Focus — trabajar sobre archivos o funciones específicas

Cuando no necesitas que el modelo analice el proyecto completo, `focus` le dice exactamente sobre qué trabajar.

```json
"focus": ["taskController.js", "src/services", "crearTarea()", "ModeloAIManager"]
```

Se puede declarar a nivel global del proyecto o dentro de una task. La task tiene prioridad.

Tres tipos de elementos válidos:

| Tipo | Ejemplo | Cómo se resuelve |
|---|---|---|
| Nombre de archivo | `"taskController.js"` | Búsqueda recursiva dentro de `repo` |
| Directorio | `"src/services"` | Relativo a `repo`, o ruta absoluta |
| Símbolo (función, clase) | `"crearTarea()"`, `"ModeloAIManager"` | Búsqueda dentro de los archivos del focus, o del proyecto si no hay otros |

Si un nombre aparece en más de un lugar, el modelo informa las coincidencias antes de continuar — nunca asume cuál.

Si `focus` no está declarado, el modelo trabaja sobre el proyecto completo.

---

## Multi-proyecto y orquestación entre servicios

Para proyectos con varios servicios que necesitan coordinarse (microservicios, monorepo con backends independientes), la estructura recomendada es:

**Un `project.json` por servicio** — para el trabajo diario dentro de cada uno:
```text
projects/proyecto1/project.json     → trabajo en proyecto_api
projects/proyecto2/project.json     → trabajo en proyecto_profile
projects/proyecto3/project.json      → trabajo en proyecto_ai
```

**Un `project.json` de orquestación** — solo para cambios que cruzan más de un servicio:
```text
projects/mova-plataforma/project.json → sincronizar contratos, alinear llamados entre servicios
```

El proyecto de orquestación usa `focus` para apuntar a los archivos de cada servicio que están involucrados en el cambio, sin necesidad de cargar todo el contexto de los tres proyectos:

```json
"focus": [
  "proyecto_api/services",
  "proyecto_profile/backend",
  "proyeto_ai"
]
```

Esto es análogo a tener repositorios de contratos (OpenAPI, Protobuf) en arquitecturas de microservicios reales — el proyecto de orquestación es el punto de sincronización, no el lugar de desarrollo de cada servicio.


---

## Uso individual

Una persona puede utilizar Mova Context para:

* conservar contexto entre sesiones
* registrar decisiones
* reutilizar convenciones
* trabajar con múltiples modelos

---

## Uso en equipos

A medida que crece un equipo, también crece el valor de compartir contexto.

Ejemplo:

```text
backend-team
frontend-team
security-team
platform-team
```

Todos pueden compartir:

```text
agents/base/
skills/base/
prompts/base/
```

y mantener contexto específico por proyecto:

```text
projects/payments/
projects/customers/
projects/analytics/
```

Esto ayuda a evitar duplicación de conocimiento y facilita la colaboración.

---

## Relación con otras herramientas

Mova Context no compite con:

* Cursor Rules
* Claude Projects
* Claude Code
* RooCode
* Codex
* Gemini
* GPT

Su propósito es distinto.

Estas herramientas gestionan interacción con modelos.

Mova Context propone una forma de organizar y conservar el conocimiento operativo del proyecto.

---

## Buenas prácticas

* Mantener documentación actualizada.
* Preferir componentes pequeños y reutilizables.
* Evitar duplicación de contexto.
* Registrar decisiones importantes.
* Revisar periódicamente la memoria del proyecto.
* Mantener el conocimiento asociado al proyecto y no a conversaciones individuales.

---

## FAQ

**¿Funciona con cualquier LLM?**

Sí, porque está basado en texto plano.

**¿Reduce tokens?**

Puede ayudar, pero depende totalmente de qué tan repetitivos estaban tus documentos antes. No es una garantía automática del formato — es el resultado de aplicar YAGNI/KISS/DRY/Ockham deliberadamente. Como referencia, en la reescritura de los agents/skills/prompts de este repo (44 archivos comparables antes/después) la reducción medida fue de ~32% en caracteres. Mide siempre tu propio caso: contar caracteres de archivo antes/después con cualquier script de una línea te da una cifra real, no una promesa.

**¿Se puede combinar con ponytail?**

Sí — son complementarios, no alternativos. Los 4 principios (YAGNI/KISS/DRY/Ockham) rigen cómo se escriben los *documentos* de Mova Context (agents/skills/prompts compactos, sin redundancia). Ponytail rige cómo el modelo escribe *código* dentro de una tarea (mínima superficie, sin abstracción no pedida). Puedes activar ponytail como prompt custom en cualquier task sin que entre en conflicto con los núcleos — de hecho, `skills/base/engineering/lazy-minimalism.md` ya es la versión genérica del mismo criterio, pensada para cargarse junto a cualquier otro agente.

**¿Sirve para otro rubro distinto al de mi primer proyecto?**

Sí, en la medida en que separes bien qué es base y qué es custom — ver "Portabilidad" arriba.

**¿Es un framework?**

No.

**¿Es un sistema de agentes?**

No.

**¿Necesito usar la estructura exacta?**

No.

**¿Reemplaza Cursor Rules o Claude Projects?**

No.

Es una convención complementaria para organizar conocimiento operativo.

---

> Mova Context intenta estandarizar el contexto que recibe un modelo, no la forma en que el modelo razona.
