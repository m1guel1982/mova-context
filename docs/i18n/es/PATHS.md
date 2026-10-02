# Rutas de Mova — `config/general/config.json`, multiplataforma y valores por defecto

Mova resuelve 6 directorios propios — `agents`, `skills`, `prompts`, `projects` — y opcionalmente `output_dir`, a través de un único archivo:
`config/general/config.json`, en la raíz de Mova (el mismo directorio donde vive `workflow.md`).

## El archivo tal como se entrega

```json
{
  "agents": "agents",
  "skills": "skills",
  "prompts": "prompts",
  "projects": "projects"
}
```

Estos son exactamente los valores que Mova **ya usaba antes de que este archivo existiera** — está
poblado a propósito, no vacío, para que abrirlo muestre de una vez qué controla cada campo y cuál es
su comportamiento de fábrica. Editar cualquiera de estos 6 valores no rompe nada: sirven como
referencia y como punto de partida para personalizar.

`output_dir` **no** viene en el archivo de fábrica — ver la sección dedicada más abajo.

## Regla de resolución — idéntica a `project.json`'s `"repo"`

Cada uno de estos campos se resuelve con la **misma regla multiplataforma** que ya usa el campo
`repo` de `project.json` (ver `PROJECT_JSON.md`), no con una convención nueva:

| Forma de la ruta | Ejemplo | Resultado |
|---|---|---|
| Letra de unidad Windows | `C:\agentes`, `D:/mova/skills`, `C://agents` | Absoluta, tal cual — funciona en cualquier SO donde esa unidad exista |
| Ruta de red UNC | `\\servidor\recurso\mova` | Absoluta, tal cual |
| Absoluta Unix | `/var/mova/prompts`, `/skills` | Absoluta, tal cual — **en Linux/macOS, `/skills` significa literalmente `/skills` en la raíz del sistema de archivos**, no relativo a Mova |
| Relativa "pelada" (sin `/` ni letra de unidad al inicio) | `agents`, `opt/projects`, `./custom` | Relativa a la raíz de Mova |

Si una ruta absoluta declarada no aplica al sistema operativo actual (por ejemplo, una letra de
unidad de Windows leída en Linux), Mova **nunca rompe el proceso**: cae automáticamente al valor por
defecto de ese campo, como si no se hubiera configurado nada.

## Fallback: en blanco, ausente, o archivo inexistente

Para cada uno de los 6 campos, si el valor es `""`, la clave no está presente en el JSON, o el
archivo `config/general/config.json` directamente no existe, Mova usa exactamente el mismo valor
por defecto que ya traía (la misma tabla de arriba, en su forma "pelada"). Nunca es necesario que el
archivo exista para que Mova funcione.

## El nombre de la carpeta puede ser cualquiera

Mova no asume que la carpeta de agentes se llama literalmente "agents" — solo le importa lo que
declares en `config.json`. Por ejemplo:

```json
{ "agents": "catalogo-de-agentes-custom" }
```

hace que Mova busque agentes en `<raíz>/catalogo-de-agentes-custom`, con el mismo descubrimiento
recursivo por dominio/idioma que ya usa hoy — nada más cambia.

## `output_dir` — el caso especial que NO viene en el archivo de fábrica

A diferencia de los otros 6 campos, `output_dir` solo tiene un consumidor real hoy: el argumento
`--output` de `mova context trace` (y por lo tanto de `context-report.md`, `context-report.pdf` y
`context-diagram.png`). Por eso se decidió **no** incluirlo en el `config.json` de fábrica: ese
comando no tiene un único "comportamiento actual" (usa la carpeta del propio proyecto analizado en
modo local, o el directorio desde donde se ejecutó el comando en modo remoto) — si `output_dir`
viniera con un valor real de fábrica, ese comportamiento habitual cambiaría para todo el mundo desde
el primer uso.

Si en algún momento querés centralizar esos reportes en una única carpeta, agregá la clave a mano:

```json
{ "output_dir": "reportes-centralizados" }
```

Mova la reconoce de inmediato (misma regla de resolución de la tabla de arriba) y le da **prioridad**
sobre el comportamiento por defecto de `mova context trace` — sin tocar código. Si preferís que el
comportamiento habitual siga como está, simplemente no agregues esta clave (o dejala en `""`).

`--output <ruta>` explícito en la línea de comandos sigue ganando siempre, sin importar qué diga
`config.json`.

## `config/log/logging.json` — el mismo criterio para el log

El campo `file.path` de `config/log/logging.json` (por defecto `"logs/mova.log"`) usa exactamente la
misma regla: una ruta "pelada" como la de fábrica se resuelve relativa a la raíz de Mova; una ruta
absoluta Windows/UNC/Unix reconocida se usa tal cual; y si el valor queda en blanco, no existe, o no
aplica al sistema operativo actual, Mova vuelve a guardar el log en `<raíz>/logs/mova.log`.

## Jerarquía por proyecto — `project.json`'s propio `"paths"`

Cualquier proyecto puede declarar su propio bloque `"paths"` en su `project.json`, con los mismos
6 campos que `config/general/config.json` (`agents`, `skills`, `prompts`,
`output_dir` — **nunca** `projects`, ver más abajo por qué). La prioridad de resolución es:

1. **`project.json` del proyecto actual** — si declara el campo y no está en blanco, gana siempre.
2. **`config/general/config.json`** — si el proyecto no declara nada para ese campo.
3. **Ruta por defecto histórica de Mova** — si ninguno de los dos anteriores declara nada.

Las 3 capas usan exactamente la misma regla de resolución (la tabla de más arriba — absoluta
Windows/UNC/Unix se usa tal cual, "pelada" se resuelve relativa a la raíz de Mova). Un ejemplo
completo, funcionando de punta a punta, vive en `projects/02-pii-compliance-governance/project.json`:

```json
{
  "paths": {
    "agents": "projects/02-pii-compliance-governance/local-agents",
    "skills": "projects/02-pii-compliance-governance/local-skills",
    "prompts": "projects/02-pii-compliance-governance/local-prompts"
  }
}
```

Ese proyecto trae sus propias copias de `ai-privacy-reviewer` (agente), `pii-context-reduction`
(skill) y `analizar-contexto-clientes-ia` (prompt) en esas 3 carpetas — con el mismo contenido que
el catálogo global, más un comentario al inicio de cada archivo que confirma que la versión CARGADA
fue la local del proyecto, no la global. Corré `mova run 02-pii-compliance-governance` (o
`mova context-trace --project 02-pii-compliance-governance`) y buscá ese comentario en la sección
AGENTS/SKILLS/PROMPT del contexto ensamblado (o en `context-report.md`) para verlo en acción.

**Por qué `projects` no tiene equivalente por proyecto:** un proyecto declarando dentro de sí mismo
dónde viven "todos los proyectos" (incluyéndose a sí mismo) es circular — Mova necesita saber dónde
buscar `project.json` ANTES de poder leer cualquier `project.json`. Ese campo sigue siendo exclusivo
de `config/general/config.json`.

## Recarga en caliente en las 3 capas

Ni `config/general/config.json` ni ningún `project.json` se cachean — ambos se releen del disco en
cada resolución de ruta / cada `GetProject`. Esto es lo que permite editar cualquiera de los dos
mientras Mova está corriendo (por ejemplo, el servidor MCP/HTTP de larga duración) y ver el cambio
reflejado de inmediato, en la siguiente solicitud, sin reiniciar nada — ni la selección de idioma,
ni las rutas globales, ni las rutas por proyecto.
