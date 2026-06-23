# ARCHITECTURE.md

## Filosofía

Mova Context es una convención para organizar conocimiento operativo asociado a proyectos asistidos por IA.

No es un framework.

No es un runtime.

No es un sistema de agentes.

No define una implementación obligatoria.

Su objetivo es mantener el conocimiento del proyecto separado de cualquier proveedor, herramienta o modelo específico.

---

## Principio fundamental

> El conocimiento operativo pertenece al proyecto. El razonamiento pertenece al modelo.

Mova Context parte de una idea simple:

El conocimiento acumulado durante el desarrollo de un proyecto debería permanecer junto al proyecto.

No dentro de un chat.

No dentro de una plataforma.

No dentro de una configuración propietaria.

De la misma forma que el código se conserva mediante Git, el contexto operativo puede conservarse mediante archivos versionables.

---

## Separación de responsabilidades

### El proyecto conserva

* convenciones
* decisiones técnicas
* memoria operativa
* documentación
* workflows
* reglas de negocio
* conocimiento acumulado

### El modelo aporta

* razonamiento
* análisis
* generación de contenido
* síntesis
* resolución de problemas

Mova Context no intenta controlar cómo razona un modelo.

Intenta organizar el contexto sobre el que razona.

---

## Independencia del proveedor

Los modelos cambian.

Los proveedores cambian.

Las herramientas cambian.

Por ejemplo:

* Claude
* GPT
* Gemini
* Codex
* Cursor
* RooCode
* Ollama
* modelos locales

pueden evolucionar o ser reemplazados con el tiempo.

La propuesta de Mova Context es que el conocimiento operativo permanezca asociado al proyecto independientemente de la herramienta utilizada para consultarlo.

---

## Portabilidad

Una consecuencia directa de esta filosofía es la portabilidad.

Si el conocimiento operativo se almacena junto al proyecto, puede acompañarlo cuando se cambie de:

* proveedor
* modelo
* interfaz
* herramienta

La interpretación del contexto puede variar entre modelos.

La base de conocimiento permanece.

---

## Uso en equipos

Mova Context puede utilizarse por una sola persona o por organizaciones grandes.

A medida que aumenta el número de proyectos, equipos o herramientas de IA utilizadas, también aumenta el valor de disponer de una base de conocimiento compartida y versionada.

Por ejemplo:

```text
backend-team
frontend-team
security-team
platform-team
```

pueden compartir convenciones comunes mientras cada proyecto mantiene su propio contexto operativo.

El objetivo es reducir la dependencia de conocimiento disperso en conversaciones individuales.

---

## Componentes como conocimiento

Elementos como:

```text
project.json
memory.md
agents/
skills/
prompts/
workflows/
```

deben entenderse como formas posibles de organizar conocimiento.

No representan componentes ejecutables.

No son obligatorios.

Cada organización puede adaptarlos, reemplazarlos o eliminarlos según sus necesidades.

---

## Nivel de estandarización

Mova Context no define un estándar formal.

No existe una especificación obligatoria.

No existe validación automática.

No existe compatibilidad certificada entre herramientas.

No existe una implementación oficial que todos los proveedores deban seguir.

La estructura propuesta es una convención abierta orientada a facilitar:

* reutilización
* colaboración
* documentación
* portabilidad

Cada equipo puede adaptarla libremente.

---

## El rol de workflow.md

Cuando existe, `workflow.md` debe entenderse como una guía operativa escrita en lenguaje natural.

No es un programa ejecutable.

No es un motor de orquestación.

No garantiza comportamientos específicos.

Su propósito es describir cómo un equipo recomienda organizar y utilizar el contexto de un proyecto.

La interpretación concreta dependerá siempre del modelo y de la herramienta utilizada.

Por esta razón distintos modelos pueden interpretar el mismo workflow de formas diferentes.


---

## Por qué texto plano

La elección de Markdown, JSON y archivos de texto responde a criterios prácticos:

* legibilidad humana
* facilidad de edición
* compatibilidad entre herramientas
* versionado con Git
* independencia tecnológica
* portabilidad

La propuesta principal no es una tecnología específica.

La propuesta principal es conservar conocimiento operativo de forma simple y portable.

---

## Qué permanece estable y qué cambia

Lo que cambia:

* modelos
* proveedores
* interfaces
* herramientas

Lo que debería permanecer estable:

* convenciones
* memoria
* decisiones técnicas
* documentación operativa
* conocimiento del proyecto

Ese es el objetivo central de Mova Context.

Porque los modelos cambiarán.

Los proveedores cambiarán.

Las herramientas cambiarán.

Pero el conocimiento del proyecto debería permanecer bajo el control de quienes construyen.
