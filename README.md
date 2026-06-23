# Mova Context

**Una capa portable de conocimiento operativo para proyectos asistidos por IA.**

> **El conocimiento operativo pertenece al proyecto. El razonamiento pertenece al modelo.**

Compatible con cualquier herramienta o modelo capaz de trabajar con archivos de texto.

---

## El problema

Cuando un proyecto utiliza IA, gran parte del conocimiento operativo termina distribuido entre chats, configuraciones específicas, documentos aislados y memoria individual de los integrantes del equipo.

Con el tiempo aparecen problemas habituales:

* decisiones técnicas difíciles de rastrear
* convenciones repetidas constantemente
* contexto reconstruido en cada sesión
* dependencia de herramientas o proveedores concretos

**El contexto del proyecto debería permanecer junto al proyecto.**

## La idea

Mova Context propone mantener el conocimiento operativo en archivos versionables que viajan junto al repositorio.

```text
Proyecto
│
├── Código
├── Convenciones
├── Memoria
├── Reglas operativas
├── Workflows
└── Contexto compartido
```

De esta forma distintos modelos y herramientas pueden trabajar sobre una misma base de conocimiento.

Los modelos pueden cambiar.

El contexto del proyecto permanece.

---

## Qué es

Mova Context es una convención organizacional para gestionar conocimiento operativo en proyectos asistidos por IA.

Su objetivo es facilitar:

* reutilización de contexto
* preservación de conocimiento
* colaboración entre equipos
* portabilidad entre herramientas
* trazabilidad de decisiones

Mova Context intenta estandarizar el contexto que recibe un modelo, no la forma en que el modelo razona.

---

## Qué no es

* No es un framework de IA.
* No es un runtime.
* No es un sistema de agentes.
* No es una plataforma de automatización.
* No reemplaza Claude, GPT, Gemini, Codex o herramientas similares.
* No garantiza comportamientos específicos.
* No define una implementación obligatoria.

---

## Beneficios

* El conocimiento permanece asociado al proyecto.
* Facilita cambiar de proveedor sin reconstruir contexto desde cero.
* Permite versionar memoria, convenciones y workflows junto al código.
* Favorece la colaboración entre equipos.
* Reduce la dependencia de configuraciones propietarias.
* Funciona con cualquier modelo capaz de interpretar texto.

---

## Limitaciones

* No garantiza resultados idénticos entre modelos.
* No reemplaza herramientas de IA existentes.
* No elimina la necesidad de mantener documentación actualizada.
* La calidad final sigue dependiendo del modelo utilizado.
* No garantiza ahorro de tokens.
* No existe una especificación oficial adoptada por los proveedores.

---

## Componentes

La estructura del repositorio es una referencia, no una obligación.

```text
project.json
memory.md
agents/
skills/
prompts/
workflows/
```

Cada equipo puede utilizarla, modificarla o reemplazarla completamente.

Mova Context organiza conocimiento.

No impone metodologías.

---

## Principio fundamental

> **El conocimiento operativo pertenece al proyecto. El razonamiento pertenece al modelo.**

Los modelos cambiarán.

Los proveedores cambiarán.

Las herramientas cambiarán.

El conocimiento acumulado del proyecto debería permanecer bajo el control del equipo que construye el proyecto.

---

## Principios de diseño

Mova Context aplica cuatro principios de ingeniería clásicos al diseño de sus archivos de conocimiento. No son obligatorios — son la propuesta por defecto. Cada equipo puede cambiarlos o reemplazarlos según su necesidad.

---

### YAGNI — aplicado a Agents

*You Aren't Gonna Need It*

**Dónde:** en los archivos de `agents/` (base y custom).
**Qué hace:** el agente no asume necesidades futuras. No crea abstracciones, endpoints ni estructuras si la tarea actual no lo pide explícitamente.

| Beneficio | Desventaja |
|---|---|
| El modelo no genera código o arquitectura que nadie pidió | Puede quedarse corto si la tarea está mal especificada |
| Reduce output innecesario y tokens de respuesta | Requiere que los prompts sean precisos sobre qué se necesita |
| Fácil de auditar: lo que existe tiene un motivo | No anticipa patrones recurrentes que podrían ahorrar tiempo |

**Cómo cambiarlo:** si tu equipo prefiere que el agente anticipe necesidades comunes (ej. siempre agregar paginación aunque no se pida), reemplaza la regla YAGNI en `agents/base/engineering/yagni-core.md` por tu propia regla de comportamiento. O simplemente elimina la referencia en el agente que quieras liberar de esa restricción.

---

### KISS — aplicado a Skills

*Keep It Simple, Stupid*

**Dónde:** en los archivos de `skills/` (base y custom).
**Qué hace:** cada skill resuelve una sola cosa, de la forma más directa posible, usando lo que ya existe antes de proponer algo nuevo.

| Beneficio | Desventaja |
|---|---|
| Skills fáciles de combinar sin conflictos | Puede requerir más skills para cubrir un caso complejo |
| El modelo produce soluciones directas, sin rodeos | No siempre la solución más simple es la más correcta en casos límite |
| Fácil de mantener: una skill con una responsabilidad | Exige más granularidad al diseñar el `project.json` |

**Cómo cambiarlo:** si prefieres skills más amplias que cubran varios casos de una vez, escribe tu propia skill sin la restricción de responsabilidad única. Por ejemplo, una skill `full-backend-review` que cubra seguridad + performance + arquitectura en un solo archivo.

---

### DRY — aplicado a Skills

*Don't Repeat Yourself*

**Dónde:** también en `skills/`, y en los archivos núcleo (`yagni-core.md`, `kiss-dry-core.md`, `ockham-core.md`).
**Qué hace:** una regla o instrucción existe en un solo lugar. El resto la referencia, no la copia.

| Beneficio | Desventaja |
|---|---|
| Cambiar una regla en un lugar la actualiza en todo el sistema | Requiere disciplina para no duplicar al crear archivos nuevos |
| El repo no crece innecesariamente | Si la referencia se omite por error, el archivo trabaja sin la regla |
| Reduce tokens de contexto al no repetir el mismo texto varias veces | Un archivo fuera de contexto pierde la regla si el núcleo no fue cargado |

**Cómo cambiarlo:** si tu equipo prefiere que cada archivo sea autónomo y no dependa de referencias (útil cuando no puedes garantizar el orden de carga), copia el contenido del núcleo directamente en cada archivo. Pierdes la centralización pero ganas independencia.

---

### Navaja de Ockham — aplicado a Prompts

**Dónde:** en los archivos de `prompts/` (base y custom).
**Qué hace:** entre dos soluciones válidas, el modelo elige la más simple y compacta. Sin prosa explicativa innecesaria antes o después del código.

| Beneficio | Desventaja |
|---|---|
| Respuestas más directas y con menos tokens de salida | Puede omitir contexto explicativo que un equipo junior necesita |
| El output es más fácil de revisar y aplicar | "Simple" es subjetivo: el modelo puede simplificar en la dirección equivocada |
| Reduce la tendencia del modelo a sobre-explicar | Requiere que quien revisa el output tenga criterio para detectar simplificaciones incorrectas |

**Cómo cambiarlo:** si necesitas que el modelo explique sus decisiones (útil para onboarding o auditorías), reemplaza la regla de Ockham por una que pida razonamiento explícito: "explica cada decisión en una línea antes del código". Eso aumenta tokens de salida pero mejora trazabilidad.

---

### Cómo reemplazar un principio por otro

Cada principio vive en un archivo núcleo de una sola regla:

```text
agents/base/engineering/yagni-core.md      ← 2 líneas
skills/base/engineering/kiss-dry-core.md   ← 2 líneas
prompts/base/engineering/ockham-core.md    ← 2 líneas
```

Para cambiar un principio en un proyecto específico sin afectar el repo global:

1. Crea un archivo `agents/custom/mi-regla-de-agente.md` con tu propia regla
2. En el `project.json` de ese proyecto, agrega tu archivo en `agents.custom` y quita `yagni-core` de `agents.base`
3. El resto del repo no cambia

Para cambiar el principio globalmente para todos los proyectos: edita directamente el archivo núcleo. Al ser solo 2 líneas, el cambio es deliberado y visible en un solo diff.

---

## CLI — usar Mova Context con LLMs web

`mova` es un programa de línea de comandos que empaqueta el contexto de un proyecto (agents + skills + prompts + memory) en un único bloque de texto listo para pegar en cualquier LLM web — Claude, ChatGPT, Gemini, u otro.

```bash
mova list                                    # ver proyectos y tareas
mova run pruebas-locales crear-proyecto  > contexto.txt    # generar contexto → copiar → pegar en el LLM
mova memory pruebas-locales "$(pbpaste)"     # actualizar memory.md con la respuesta del LLM
```

Binarios precompilados para Linux, macOS (Intel + Apple Silicon) y Windows en `cli/dist/`.
Documentación completa en `cli/MANUAL.md`.

---

## Documentación

* **ARCHITECTURE.md** → filosofía y principios de diseño.
GUIDE.md → estructura sugerida, ejemplos y guía de adopción. Incluye cómo integrar el genial concepto de ponytail (modo lazy senior dev), un prompt externo espectacular creado por Dietrich Gebert que puedes descargar desde su repositorio oficial 
https://github.com/DietrichGebert/ponytail/tree/main y que se acopla  a la arquitectura modular de Mova.

* **PRUEBAS-LOCALES.md** → guía paso a paso para probar todo el concepto en local con Cline + Gemini Flash, sin tocar proyectos reales. Funciona con cualquier herramienta de IA. Empieza aquí si quieres validar la idea antes de adoptarla.

## Ejemplos incluidos

* `projects/pruebas-locales/` → proyecto ficticio mínimo (API de tareas) pensado para correr la guía `docs/PRUEBAS-LOCALES.md` de punta a punta: crear, modificar, devops, QA, ponytail y memoria.
* `projects/acme-demo/` → ejemplo mínimo y didáctico de la convención.
