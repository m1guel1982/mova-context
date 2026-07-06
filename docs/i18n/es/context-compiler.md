# Context Compiler — `mova compile`

> Documentación: [Español](context-compiler.md) · [English](../en/context-compiler.md)

---

## Qué hace

Reduce el contexto que se envía al LLM antes de enviarlo. Toma lo que ya carga `mova run` (agents + skills + prompt + memory) y, opcionalmente, fragmentos exactos del código o los documentos del proyecto (`repo`), y produce un archivo compacto, sin formato para humanos: `contexto.txt` — junto con un reporte de evidencia (`contexto.report.md` / `.json`, ver más abajo).

No reemplaza `workflow.md` ni `project.json`. Es una capa opcional sobre ellos. Un proyecto sin `contextCompiler` en su `project.json` funciona exactamente igual que siempre — compatibilidad total.

**Nota sobre ediciones:** el Compiler v2 (todo lo descrito en esta página) es la parte comercial de Mova Context. La edición Community (Open Source) puede correr `mova compile` igual, pero cae de vuelta al contexto normal sin distilar, con un aviso explícito — nunca falla, nunca finge haber comprimido algo que no comprimió. Ver [`../premium/es/ediciones.md`](../premium/es/ediciones.md).

---

## Cómo funciona

Dos estrategias, elegidas con `contextCompiler.strategy`:

### `strategy: "semantic"` (default) — Fase 1 sola

Aplica a los archivos Markdown ya cargados (agents, skills, prompts, memory):

- Elimina saludos, agradecimientos y frases de cortesía completas ("Hola, bienvenido...", "Gracias por tu tiempo...").
- Recorta frases de relleno ("cabe destacar que...", "es importante mencionar que...") dejando intacto el contenido sustantivo que sigue.
- Colapsa espacios repetidos y líneas vacías dobles.
- **Nunca** toca bloques de código (` ``` `), placeholders `{{...}}`, ni líneas con marcadores de regla crítica (`NUNCA`, `SIEMPRE`, `OBLIGATORIO`, `PROHIBIDO`, `MUST`, `CRITICAL`...).

Es determinista: no llama a ningún LLM para "resumir" — eso agregaría latencia y costo, justo lo contrario del objetivo.

### `strategy: "full"` — pipeline completo del Compiler v2

Ocho etapas, cada una con una única responsabilidad, siempre en el mismo orden:

```
Knowledge Parser → Semantic Cleanup → Normalization → Deduplication
  → Token Optimization → Priority Ranking → Budget Assembly → contexto.txt
```

| Etapa | Qué hace | Nunca hace |
|---|---|---|
| Knowledge Parser | Clasifica el contenido en unidades (Rule, Instruction, Memory, Example, Code, Table, Variable, Placeholder, Metadata) | Fragmentar código o tablas |
| Semantic Cleanup | Fase 1 de arriba, aplicada solo a prosa | Tocar Code/Table/Metadata |
| Normalization | Viñetas, comillas tipográficas, puntuación repetida | Reescribir una oración o cambiar significado |
| Deduplication | Elimina bloques **idénticos** repetidos | Fusionar bloques solo "parecidos" |
| Token Optimization | Mide con un tokenizer BPE real (nunca caracteres/bytes) | Prometer un % fijo de antemano |
| Priority Ranking | Etiqueta cada bloque (Rule = crítico, nunca se recorta) | Eliminar nada por sí sola |
| Budget Assembly | Si hay `max_tokens`, arma el mejor contexto posible dentro del presupuesto | Recortar una regla crítica, aunque el presupuesto se exceda para respetarlas |

Detalle completo de cada etapa: [`compiler-v2-pipeline.md`](compiler-v2-pipeline.md).

### Poda de `focus` (independiente de la estrategia)

Si `project.json` (o una `task`) define `"focus"`, el compilador nunca envía archivos completos del `repo` sin relación con el foco — para **cualquier** estrategia, incluida `"semantic"`. Cada elemento de `focus` se resuelve contra el **Focus Resolution Engine**, un motor de resolvers extensible (ver [`focus-engine.md`](focus-engine.md) para su arquitectura completa):

| Tipo de elemento | Ejemplo | Extracción |
|---|---|---|
| Archivo | `"archivo.js"` | Contenido completo (fue pedido explícitamente) |
| Directorio | `"src/services"` | Listado compacto de nombres, no el contenido |
| Nodo JSON | `"config.json#database.host"` | Solo ese nodo, no el archivo completo |
| Definición SQL | `"orders"` (nombre de tabla) | Solo el `CREATE TABLE orders (...)`, no el archivo `.sql` completo |
| Símbolo de código | `"crearOrden()"`, `"NombreClase"` | Solo la función/clase/método, por conteo de llaves o indentación |
| Sección de documento | `"Artículo 6"` | Solo esa sección, hasta el siguiente encabezado del mismo nivel (Título/Capítulo/Sección/Artículo/Inciso, o `#`/`##` Markdown) |
| Evento cronológico | palabra clave presente en logs/historiales | Solo los bloques fechados que la contienen |

Si no encuentra una coincidencia estructural exacta, el motor prueba en cascada (código → documento → legal → cronológico) y, como último recurso, entrega un extracto acotado (±10 líneas alrededor de la primera mención textual) en vez de todo el archivo — nunca vacío, nunca el archivo completo por defecto.

---

## Cuándo utilizarlo

- Proyectos con agents/skills/prompts extensos que se repiten en cada `mova run`.
- Tareas que trabajan sobre un archivo, función, tabla o artículo específico dentro de un repositorio o documento grande.
- Perfiles `llm_profile.type: "local"`, donde cada token cuenta más.
- `strategy: "full"` con `max_tokens`, cuando necesitas una garantía dura de presupuesto (p. ej. una ventana de contexto pequeña) y quieres evidencia reproducible de qué se recortó y por qué.

No es necesario para proyectos pequeños o cuando `focus` ya reduce el trabajo lo suficiente sin necesitar distilación de texto.

---

## Configuración (`project.json`)

```json
"contextCompiler": {
  "enabled": true,
  "mode": "manual",
  "strategy": "full",
  "max_tokens": 4000,
  "report_level": "full",
  "focus_exclude": ["tests", "coverage"]
},
"focus": ["archivo.js", "src/services", "nombreFuncion()"]
```

| Campo | Valores | Default | Efecto |
|---|---|---|---|
| `enabled` | `true` / `false` | `false` | Si falta el bloque completo, el compilador nunca se activa solo. |
| `mode` | `"manual"` / `"automatic"` | `"manual"` | `manual`: solo corre con `mova compile`. `automatic`: `mova run` y `get_full_context` (MCP) lo usan siempre. |
| `strategy` | `"semantic"` / `"full"` | `"semantic"` | `full` activa las 8 etapas del pipeline (ver arriba). |
| `max_tokens` | entero | `0` (sin límite) | Solo con `strategy: "full"`. Presupuesto **por bloque de conocimiento** (un agent, un skill, el prompt, o la memoria) — todavía no es un presupuesto único para todo `contexto.txt`. Las reglas críticas (`Rule`) siempre se incluyen, aunque excedan el presupuesto. |
| `report_level` | `"summary"` / `"full"` | `"summary"` | Solo afecta a `contexto.report.md` — ver [Transparencia](#transparencia-qué-muestra-realmente-el-reporte) abajo. `contexto.report.json` siempre tiene todos los números reales que existan, sin importar este valor. |
| `focus_exclude` | array de nombres de carpeta | `[]` | Carpetas adicionales que el escaneo del Focus Resolution Engine debe ignorar, **sumadas a** las que ya ignora siempre por defecto (`.git`, `node_modules`, `vendor`, `dist`, `build`, `__pycache__`, `.venv`, `venv`, `.idea`, `.vscode`) — nunca las reemplaza. |

`focus` puede declararse a nivel global o dentro de una `task` — prioridad `task > global`, igual que `variables` (workflow.md § FOCUS).

---

## Transparencia: qué muestra realmente el reporte

Cada número de `contexto.report.json`/`.md` está medido, nunca estimado
— ver [`ejemplo-roi-compiler.md`](../premium/es/ejemplo-roi-compiler.md)
para ejecuciones reales. Con `focus` configurado, `report_level: "full"`
agrega una segunda tabla con la evidencia real del escaneo del Focus
Resolution Engine:

```text
| Archivos escaneados en `repo`                  | 124 |
| Archivos incluidos en contexto.txt             | 18  |
| Escaneados pero sin match con ningún `focus`   | 104 |
| Ignorados por carpeta `node_modules/`          | 40  |
| Ignorados por carpeta `tests/`                 | 12  |
```

Dos reglas honestas detrás de esta tabla, a propósito:

- **Solo existen dos motivos de exclusión, porque son los únicos dos que
  el motor realmente aplica**: una carpeta que ignoró, o un archivo que
  escaneó y que simplemente no coincidió con ningún target de `focus`
  pedido. No hay categoría de "archivos duplicados" ni "documentación no
  relacionada" — este compilador no detecta ninguna de esas dos cosas,
  así que no dice que sí.
- **`report_level` nunca oculta datos, solo una vista de ellos.** Dejalo
  en `"summary"` (default) y `contexto.report.md` muestra los mismos
  totales de siempre; `contexto.report.json` muestra este desglose de
  cualquier forma — un script que lo necesite no requiere `"full"` en
  absoluto, ese ajuste es solo para el archivo legible por humanos.

Las etiquetas de `contexto.report.md` siguen el `lang` propio del
proyecto (`project.json`). Si `lang` todavía no es `"es"` ni `"en"`, el
reporte cae a inglés y lo dice explícitamente en el archivo —
`contexto.report.json` nunca depende de `lang` en absoluto (sus claves
son snake_case en inglés, universales por diseño, p. ej. `files_scanned`,
no una frase traducida).

---

## Comandos

```bash
mova compile [proyecto] [tarea]     # fuerza el compilador, sin importar "mode"
mova run     [proyecto] [tarea]     # usa el compilador solo si mode: "automatic"
```

`mova compile` siempre escribe, sin pasos adicionales:

```text
projects/[proyecto]/contexto.txt
projects/[proyecto]/contexto.report.md
projects/[proyecto]/contexto.report.json
```

### Reportes — evidencia, no promesas

Cada compilación genera un reporte objetivo. Nunca se afirma un porcentaje de ahorro sin poder demostrarlo: si el binario no tiene el Compiler v2 (edición Community), el reporte dice exactamente eso en vez de inventar una cifra.

```json
{
  "edition": "premium",
  "project": "compiler-demo",
  "task": "revisar-ordenes",
  "tokenizer": "cl100k_base (aproximado)",
  "files_processed": 5,
  "tokens_before": 1814,
  "tokens_after": 1813,
  "blocks_total": 0,
  "blocks_removed": 0,
  "blocks_dropped": 0,
  "overflowed": false,
  "compile_time_ms": 3
}
```

`tokenizer` indica honestamente cuándo el encoding usado es una aproximación — Anthropic no publica un tokenizer offline de Claude, así que se usa `cl100k_base` (el proxy público más cercano) y se documenta como tal en vez de ocultarlo.

---

## Ediciones y perfil de compilación

El Compiler v2 se enlaza o no al binario en tiempo de build (`-tags premium`), y su edición exacta (`trial` / `premium` / `enterprise`) se fija con `-ldflags`, nunca en un archivo editable en runtime. Detalle completo: [`compilacion_del_ejecutable.md`](compilacion_del_ejecutable.md) y [`../premium/es/ediciones.md`](../premium/es/ediciones.md).

Si una licencia Trial expira o agota su cupo de ejecuciones, el binario **nunca se rompe**: degrada automáticamente a comportamiento Community (contexto sin distilar) con un aviso explícito por consola.

---

## Ventajas

- Menos tokens enviados al LLM → menor costo y latencia, medido con un tokenizer real.
- Nunca envía archivos completos de código cuando hay `focus` definido — solo el fragmento relevante.
- 100% determinista y local: no depende de otro LLM ni de red (el tokenizer trae su vocabulario embebido en el binario).
- Cero impacto en proyectos existentes: sin el bloque `contextCompiler`, nada cambia; con `strategy: "semantic"` (default), el comportamiento es idéntico al de siempre.
- Con `strategy: "full"`, presupuesto de tokens garantizado por bloque, con evidencia reproducible de qué se recortó.

## Limitaciones (documentadas, no ocultas)

- La extracción de símbolos de código es heurística (conteo de llaves / indentación), no un parser real. Cubre bien el estilo típico de Go, JS/TS, Java, C#, PHP y Python; símbolos muy inusuales (macros, generics complejos, código minificado) pueden no resolverse — en ese caso se entrega un extracto acotado, nunca vacío ni el archivo completo.
- `max_tokens` presupuesta cada bloque de conocimiento por separado, no `contexto.txt` completo — un presupuesto global requiere que la orquestación vea el documento entero de una vez (ver hoja de ruta).
- Deduplication opera a nivel de bloque/párrafo, no de línea — dos líneas idénticas dentro del mismo párrafo no se deduplican entre sí (evita fragmentar un bloque para ganar un ahorro marginal).
- El tokenizer usado para modelos que no son de OpenAI (Claude, Gemini, Llama, Mistral, ...) es una aproximación pública (`cl100k_base`), no el tokenizer exacto de ese proveedor — el reporte siempre lo indica.
- No interpreta significado — es distillación de texto por patrones, no comprensión semántica real.

---

## Caso de uso real

Ver [`projects/compiler-demo/`](../../../projects/compiler-demo/project.json), con `repo` apuntando a un mini-proyecto de ejemplo (`examples/compiler-demo-repo/`):

```json
{
  "project": "compiler-demo",
  "repo": "examples/compiler-demo-repo",
  "contextCompiler": { "enabled": true, "mode": "manual", "strategy": "semantic" },
  "tasks": {
    "revisar-ordenes": {
      "prompt": "review-project",
      "focus": ["CreateOrder()", "manual.md", "Artículo 6"]
    }
  }
}
```

```bash
mova compile compiler-demo revisar-ordenes
```

Resultado en `contexto.txt`: agents/skills/prompt distilados (sin saludos ni relleno) + solo la función `CreateOrder`, el archivo `manual.md` completo (fue pedido por nombre) y únicamente el `Artículo 6` de ese manual — nunca el proyecto completo. Junto a `contexto.txt` se generan `contexto.report.md` y `contexto.report.json` con la evidencia de la compilación.

---

## Buenas prácticas

- Empieza con `mode: "manual"` y revisa `contexto.txt` antes de pasar a `automatic`.
- Usa `focus` con nombres exactos (`NombreFuncion()`, no descripciones) — la resolución es literal.
- Si un símbolo no se encuentra, revisa el extracto acotado que igual se genera: casi siempre indica que el nombre no coincide exactamente.
- No declares `focus` si necesitas que el modelo vea el proyecto completo — la ausencia de `focus` es la señal para trabajar sin restricciones (igual que en `mova run` normal).
- Antes de fijar `max_tokens`, corre sin límite una vez y mira `contexto.report.json` para saber cuántos tokens usa hoy tu proyecto — evita presupuestos arbitrarios.

---

## Validación

| Comando | Resultado esperado | Error posible | Solución |
|---|---|---|---|
| `mova compile [proyecto]` | Crea `contexto.txt`, `contexto.report.md` y `contexto.report.json`, y lo confirma en consola | `task not found` | Verificar `default_task` o pasar la task explícitamente |
| `mova run [proyecto]` con `mode: "automatic"` | La salida tiene el formato compacto (`PROJECT:... AGENT:...`), no el formato con encabezados `## AGENTS` | Sigue viendo el formato humano | `contextCompiler.enabled` es `false`, o `mode` no es `"automatic"` |
| Foco a un símbolo inexistente | Bloque `FOCUS:` con extracto acotado o `not found: <símbolo>`, nunca vacío | El bloque está vacío | No debería ocurrir — reportar como bug |
| Foco a un directorio | Bloque `FOCUS:` con `dir(N): archivo1, archivo2...` | Aparece contenido de archivos | No debería ocurrir — reportar como bug |
| `mova compile` en edición Community | Genera los 3 archivos; el reporte trae `"edition": "community"` y una nota explicando que no hay distilación | El proceso falla | No debería ocurrir — el fallback es intencional, reportar como bug |
