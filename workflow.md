# workflow.md

Sistema de instrucciones para resolver contexto de proyecto.

## WORKSPACE

```text
mova-context/              ← repo de la convención (agents, skills, prompts, projects)
├── agents/
│   ├── base/
│   └── custom/
├── skills/
│   ├── base/
│   └── custom/
├── prompts/
│   ├── base/
│   └── custom/
├── projects/
│   └── [PROJECT]/
│       ├── project.json
│       └── memory.md
└── workflow.md
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
```

---

## ACCESO A ARCHIVOS

```text
SI el entorno permite acceso a filesystem
→ localizar y leer archivos automáticamente

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
1. Leer project.json
2. Resolver task
3. Cargar agents
4. Cargar skills
5. Cargar prompts
6. Leer memory.md
7. Fusionar variables
8. Inyectar variables
9. Ejecutar
10. Actualizar memory.md
```

---

## AGENTS

```json
"agents": {
  "base": ["agent-a"],
  "custom": ["agent-b"]
}
```

Carga:

```text
agents/base/[nombre].md
agents/custom/[nombre].md
```

Orden:

```text
base → custom
```

Los agents definidos en una task se agregan a los globales.

---

## SKILLS

```json
"skills": {
  "base": ["skill-a"],
  "custom": ["skill-b"]
}
```

Carga:

```text
skills/base/[nombre].md
skills/custom/[nombre].md
```

Orden:

```text
base → custom
```

Las skills definidas en una task se agregan a las globales.

---

## PROMPTS

```json
"prompt": {
  "base": "prompt-base",
  "custom": "prompt-custom"
}
```

Carga:

```text
prompts/base/[nombre].md
prompts/custom/[nombre].md
```

Orden:

```text
base → custom
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

Regla: reemplazar "_" por "_", convertir a mayúsculas, envolver en {{ }}

Ejemplos:
  project        → {{PROJECT}}
  api_prefix     → {{API_PREFIX}}
  test_framework → {{TEST_FRAMEWORK}}
  ci_provider    → {{CI_PROVIDER}}
  module_name    → {{MODULE_NAME}}
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
SI existe
→ leer

SI no existe
→ crear
→ continuar

Leer antes de ejecutar.
Actualizar al finalizar.
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

## REGLAS DE CARGA

```text
base → custom

custom complementa o sobreescribe base.
```

```text
SI un archivo no existe
→ ignorar y continuar
```

Orden completo:

```text
1. agents globales base
2. agents globales custom
3. agents task base
4. agents task custom

5. skills globales base
6. skills globales custom
7. skills task base
8. skills task custom

9. prompt base
10. prompt custom
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
1. Resolver proyecto
2. Resolver task
3. Cargar contexto requerido
4. Aplicar variables
5. Ejecutar usando agents, skills, prompts y memory.md
6. Actualizar memory.md al finalizar
```
