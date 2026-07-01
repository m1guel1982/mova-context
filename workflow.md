# workflow.md

Sistema de instrucciones para resolver contexto de proyecto.

> Documentación completa: [docs/i18n/es/workflow.md](docs/i18n/es/workflow.md) · [docs/i18n/en/workflow.md](docs/i18n/en/workflow.md)

---

## WORKSPACE

```text
mova-context/              ← repo de la convención (agents, skills, prompts, projects)
├── agents/[domain]/i18n/[lang]/
├── skills/[domain]/i18n/[lang]/
├── prompts/[domain]/i18n/[lang]/
├── projects/
│   └── [PROJECT]/
│       ├── project.json
│       └── memory.md
└── workflow.md             ← estás aquí
```

El código generado puede vivir en cualquier directorio — dentro o fuera de `mova-context`.
La ruta de trabajo se declara en `project.json` como `"repo"`.

```json
"repo": "."                                  ← relativo a mova-context (default)
"repo": "../app-prueba-local"                ← carpeta hermana externa
"repo": "E:/proyectos/mi-proyecto"           ← ruta absoluta en cualquier SO
```

Reglas de resolución del directorio de trabajo:

```text
SI "repo" es "." o está ausente
→ trabajar dentro de mova-context

SI "repo" es una ruta relativa
→ resolver relativa a la ubicación de mova-context

SI "repo" es una ruta absoluta
→ usarla directamente

SI la ruta no existe
→ crearla antes de generar cualquier archivo
→ confirmar al usuario la ruta donde se va a trabajar
```

Búsqueda de agents/skills/prompts/projects: siempre dentro de `mova-context`, independiente de `repo`.
Generación de código y archivos de proyecto: siempre en la ruta indicada por `repo`.

---

## ACTIVACIÓN

Entradas válidas:

```text
Lee workflow.md
Lee workflow.md → [PROJECT]
Lee workflow.md → [PROJECT] → [TASK]

carga [PROJECT]
ejecuta [TASK] de [PROJECT]

Read workflow.md → [PROJECT] → [TASK]
```

---

## ACCESO A ARCHIVOS

```text
SI el entorno permite acceso a filesystem
→ localizar y leer archivos automáticamente, de forma recursiva dentro de "repo" y de mova-context

SI el entorno NO permite acceso a filesystem
→ solicitar únicamente los archivos faltantes
→ continuar cuando estén disponibles
```

---

## DETECCIÓN DE PROYECTO

```text
SI existe un único proyecto
→ usarlo automáticamente

SI existen múltiples proyectos
→ solicitar el nombre una sola vez

SI el usuario indica un proyecto
→ usar ese proyecto
```

---

## RESOLUCIÓN DE PROYECTO

```text
1. Localizar projects/[PROJECT]/project.json

SI no existe
→ informar el error
→ solicitar proyecto válido
→ detener ejecución

SI existe
→ continuar
```

---

## RESOLUCIÓN DE TASK

```text
SI el usuario indica una task
→ usar esa task

SI no indica una task
→ usar default_task

SI la task no existe
→ informar error
→ mostrar tasks disponibles

SI default_task no existe
→ solicitar task válida
→ detener ejecución
```

---

## SECUENCIA DE EJECUCIÓN

```text
1.  Leer project.json
2.  Resolver lang (idioma configurado)
3.  Resolver llm_profile (proveedor + modelo + perfil)
4.  Resolver adapter (almacenamiento: file / postgresql / mongodb)
5.  Resolver task
6.  Cargar agents
7.  Cargar skills
8.  Cargar prompt
9.  Leer memory.md
10. Fusionar variables (task > global)
11. Inyectar variables
12. Ejecutar
13. Actualizar memory.md
```

---

## RESOLUCIÓN DE ARCHIVOS (i18n + domain)

Para `agent: backend-dev`, `domain: software`, `lang: es`:

```text
agents/software/i18n/es/backend-dev.md   ← usar
agents/software/i18n/en/backend-dev.md   ← fallback si es/ no existe
```

Misma lógica de resolución aplica a `skills/` y `prompts/`.

---

## AGENTS

```json
"agents": {
  "domain": "software",
  "use": ["backend-dev", "security-architect"]
}
```

Carga, en orden, para cada nombre en `use`:

```text
1. agents/[domain]/i18n/[lang]/[nombre].md      ← base del dominio
2. agents/custom/i18n/[lang]/[nombre].md        ← override específico del proyecto, si existe
```

Los agents definidos en una task se agregan a los globales (no los reemplazan).

---

## SKILLS

```json
"skills": {
  "domain": "legal",
  "use": ["ley-21719-obligaciones", "derechos-titulares"]
}
```

Carga, en orden, para cada nombre en `use`:

```text
1. skills/[domain]/i18n/[lang]/[nombre].md      ← base del dominio
2. skills/custom/i18n/[lang]/[nombre].md        ← override específico del proyecto, si existe
```

Las skills definidas en una task se agregan a las globales (no las reemplazan).

---

## PROMPTS

```json
"tasks": {
  "analizar-contrato": {
    "prompt": "analizar-contrato-datos"
  }
}
```

Carga:

```text
1. prompts/[domain]/i18n/[lang]/[nombre].md     ← base del dominio
2. prompts/custom/i18n/[lang]/[nombre].md       ← override específico del proyecto, si existe
```

---

## VARIABLES

Origen:

```text
project.json → variables (globales)
task → variables (sobreescriben globales si coincide la clave)
```

Prioridad:

```text
task > global
```

Normalización automática:

```text
Toda clave en snake_case se convierte a {{UPPER_CASE}} para inyección.

Regla: convertir a mayúsculas, envolver en {{ }}

Ejemplos:
  project        → {{PROJECT}}
  api_prefix     → {{API_PREFIX}}
  tipo_documento → {{TIPO_DOCUMENTO}}
  cualquier_clave_nueva → {{CUALQUIER_CLAVE_NUEVA}}
```

No existe lista fija de variables permitidas.
Cualquier clave declarada en `variables` (global o de task) se normaliza e inyecta automáticamente en todos los agents, skills y prompts cargados.

Si un agente, skill o prompt usa `{{NOMBRE_VARIABLE}}` y esa variable no fue declarada en `project.json`, se deja el placeholder sin reemplazar y se informa al usuario qué variable falta.

Variables reservadas del sistema (siempre disponibles sin declararlas):

```text
{{PROJECT}}    → valor de "project" en project.json
{{REPO}}       → valor de "repo" en project.json
{{TASK}}       → nombre de la task activa
{{LANG}}       → idioma configurado
```

---

## FOCUS

Define sobre qué trabajar exactamente: archivos, directorios o símbolos concretos del código fuente.
Cuando está presente, el modelo trabaja solo sobre esos elementos, no sobre el proyecto completo.

Declaración en `project.json` (global o dentro de una task):

```json
"focus": ["archivo.js", "src/services", "nombreFuncion()", "NombreClase"]
```

Reglas de resolución:

```text
SI el elemento tiene ruta absoluta
→ usarla directamente

SI el elemento es solo un nombre (sin separador de ruta)
→ buscarlo recursivamente dentro de "repo"
→ SI aparece en más de un lugar → informar las coincidencias y pedir confirmación
→ SI no se encuentra → informar y continuar sin él

SI el elemento termina en () o tiene la forma Nombre sin extensión
→ tratarlo como símbolo (función, método, clase)
→ buscarlo dentro de los archivos de "focus" o del proyecto si no hay otros focus
```

Prioridad (igual que variables):

```text
task > global
```

`{{FOCUS}}` se inyecta como lista legible en agents, skills y prompts que lo referencien.
Si `focus` no está declarado, el modelo trabaja sobre el proyecto completo.

---

## MEMORIA

Ruta:

```text
projects/[PROJECT]/memory.md
```

Reglas:

```text
SI existe → leer antes de ejecutar
SI no existe → crear

Actualizar al finalizar cada sesión.
```

Formato:

```md
## YYYY-MM-DD — sesión

**Hecho:**
**Resuelto:**
**Pendiente:**
**Decisiones:**
**Errores LLM:**
```

---

## ARCHIVOS CORE

Cada sección tiene un archivo core obligatorio que se carga automáticamente, una sola vez, antes de cualquier otro archivo de esa sección.

| Sección | Core           | Ubicación                                  |
|---------|----------------|--------------------------------------------|
| Agents  | yagni-core.md  | agents/{domain}/i18n/{lang}/yagni-core.md  |
| Skills  | kiss-dry-core.md | skills/{domain}/i18n/{lang}/kiss-dry-core.md |
| Prompts | ockham-core.md | prompts/{domain}/i18n/{lang}/ockham-core.md |

Reglas:

```text
Cada core se carga exactamente una vez por contexto.
Si el core no existe en el dominio activo → buscarlo recursivamente en agents/skills/prompts.
Si un agent/skill/prompt lista el core como nombre → omitirlo (ya fue cargado).
Nunca duplicar contenido de core.
```

El sistema resuelve el core automáticamente sin importar la profundidad del árbol de directorios.

---

## REGLAS DE CARGA

```text
base/domain → custom

custom complementa o sobreescribe base.
```

```text
SI un archivo no existe
→ ignorar y continuar
```

Orden completo:

```text
1.  core de agents (yagni-core) — una sola vez
2.  agents globales del dominio (base)
3.  agents globales custom
4.  agents task del dominio (base)
5.  agents task custom

6.  core de skills (kiss-dry-core) — una sola vez
7.  skills globales del dominio (base)
8.  skills globales custom
9.  skills task del dominio (base)
10. skills task custom

11. core de prompts (ockham-core) — una sola vez
12. prompt del dominio (base)
13. prompt custom
```

---

## REFERENCIAS

Aplicar el contexto activo sobre:

```text
src/[ruta]
archivo.ext
funcion()
clase
ruta absoluta
```

---

## RESULTADO ESPERADO

```text
1. Resolver proyecto y task
2. Resolver lang, llm_profile y adapter
3. Cargar contexto (agents + skills + prompt + memory) de forma recursiva
4. Aplicar y fusionar variables (incluyendo focus)
5. Ejecutar
6. Actualizar memory.md
```