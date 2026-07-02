# Context Compiler — `mova compile`

> Documentación: [Español](context-compiler.md) · [English](../en/context-compiler.md)

---

## Qué hace

Reduce el contexto que se envía al LLM antes de enviarlo. Toma lo que ya carga `mova run` (agents + skills + prompt + memory) y, opcionalmente, fragmentos exactos del código o los documentos del proyecto (`repo`), y produce un archivo compacto, sin formato para humanos: `contexto.txt`.

No reemplaza `workflow.md` ni `project.json`. Es una capa opcional sobre ellos. Un proyecto sin `contextCompiler` en su `project.json` funciona exactamente igual que antes — compatibilidad total.

---

## Cómo funciona

Dos fases independientes, activables por separado:

### Fase 1 — Telegrama Semántico (`strategy: "semantic"`)

Aplica a los archivos Markdown ya cargados (agents, skills, prompts, memory):

- Elimina saludos, agradecimientos y frases de cortesía completas ("Hola, bienvenido...", "Gracias por tu tiempo...").
- Recorta frases de relleno ("cabe destacar que...", "es importante mencionar que...") dejando intacto el contenido sustantivo que sigue.
- Colapsa espacios repetidos y líneas vacías dobles.
- **Nunca** toca bloques de código (` ``` `), placeholders `{{...}}`, ni líneas con marcadores de regla crítica (`NUNCA`, `SIEMPRE`, `OBLIGATORIO`, `PROHIBIDO`, `MUST`, `CRITICAL`...).

Es determinista: no llama a ningún LLM para "resumir" — eso agregaría latencia y costo, justo lo contrario del objetivo.

### Fase 2 — Poda quirúrgica (`focus`)

Si `project.json` (o una `task`) define `"focus"`, el compilador nunca envía archivos completos del `repo` sin relación con el foco. Por cada elemento de `focus` resuelve:

| Tipo de elemento | Ejemplo | Extracción |
|---|---|---|
| Archivo | `"archivo.js"` | Contenido completo (fue pedido explícitamente) |
| Directorio | `"src/services"` | Listado compacto de nombres, no el contenido |
| Símbolo de código | `"crearOrden()"`, `"NombreClase"` | Solo la función/clase/método, por conteo de llaves o indentación |
| Nodo de documento | `"Artículo 6"` | Solo esa sección, hasta el siguiente encabezado del mismo nivel (Título/Capítulo/Sección/Artículo/Inciso o `#`/`##`) |
| Evento cronológico | palabra clave presente en logs/historiales | Solo los bloques fechados que la contienen |

Si no encuentra una coincidencia estructural exacta, devuelve un extracto acotado (±10 líneas) en lugar de todo el archivo, y lo indica.

---

## Cuándo utilizarlo

- Proyectos con agents/skills/prompts extensos que se repiten en cada `mova run`.
- Tareas que trabajan sobre un archivo, función o artículo específico dentro de un repositorio o documento grande.
- Perfiles `llm_profile.type: "local"`, donde cada token cuenta más.

No es necesario para proyectos pequeños o cuando `focus` ya reduce el trabajo lo suficiente sin necesitar Fase 1.

---

## Configuración (`project.json`)

```json
"contextCompiler": {
  "enabled": true,
  "mode": "manual",
  "strategy": "semantic"
},
"focus": ["archivo.js", "src/services", "nombreFuncion()"]
```

| Campo | Valores | Default | Efecto |
|---|---|---|---|
| `enabled` | `true` / `false` | `false` | Si falta el bloque completo, el compilador nunca se activa solo. |
| `mode` | `"manual"` / `"automatic"` | `"manual"` | `manual`: solo corre con `mova compile`. `automatic`: `mova run` y `get_full_context` (MCP) lo usan siempre. |
| `strategy` | `"semantic"` / `"full"` | `"semantic"` | `full`: sin Fase 1 (solo aplica Fase 2 si hay `focus`). |

`focus` puede declararse a nivel global o dentro de una `task` — prioridad `task > global`, igual que `variables` (workflow.md § FOCUS).

---

## Comandos

```bash
mova compile [proyecto] [tarea]     # fuerza el compilador, sin importar "mode"
mova run     [proyecto] [tarea]     # usa el compilador solo si mode: "automatic"
```

`mova compile` siempre escribe `projects/[proyecto]/contexto.txt`. Es la vía de inspección y depuración mencionada en workflow.md — funciona exista o no `mode: "automatic"`.

---

## Ventajas

- Menos tokens enviados al LLM → menor costo y latencia.
- Nunca envía archivos completos de código cuando hay `focus` definido — solo el fragmento relevante.
- 100% determinista y local: no depende de otro LLM ni de red.
- Cero impacto en proyectos existentes: sin el bloque `contextCompiler`, nada cambia.

## Limitaciones

- La extracción de símbolos de código es heurística (conteo de llaves / indentación), no un parser real. Cubre bien el estilo típico de Go, JS/TS, Java, C#, PHP y Python; símbolos muy inusuales (macros, generics complejos, código minificado) pueden no resolverse — en ese caso se entrega un extracto acotado, nunca vacío ni el archivo completo.
- La reducción de tokens de la Fase 1 es moderada (instrucciones ya suelen ser concisas). La reducción grande viene de la Fase 2, al evitar enviar archivos o documentos completos cuando solo se necesita una parte.
- No interpreta significado — es distillación de texto por patrones, no comprensión semántica real.

---

## Caso de uso real

Ver [`projects/compiler-demo/`](../../../projects/compiler-demo/project.json), con `repo` apuntando a un mini-proyecto de ejemplo:

```json
{
  "project": "compiler-demo",
  "repo": "/tmp/demo-repo",
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

Resultado en `contexto.txt`: agents/skills/prompt distilados (sin saludos ni relleno) + solo la función `CreateOrder`, el archivo `manual.md` completo (fue pedido por nombre) y únicamente el `Artículo 6` de ese manual — nunca el proyecto completo.

---

## Buenas prácticas

- Empieza con `mode: "manual"` y revisa `contexto.txt` antes de pasar a `automatic`.
- Usa `focus` con nombres exactos (`NombreFuncion()`, no descripciones) — la resolución es literal.
- Si un símbolo no se encuentra, revisa el extracto acotado que igual se genera: casi siempre indica que el nombre no coincide exactamente.
- No declares `focus` si necesitas que el modelo vea el proyecto completo — la ausencia de `focus` es la señal para trabajar sin restricciones (igual que en `mova run` normal).

---

## Validación

| Comando | Resultado esperado | Error posible | Solución |
|---|---|---|---|
| `mova compile [proyecto]` | Crea `projects/[proyecto]/contexto.txt` y lo confirma en consola | `task not found` | Verificar `default_task` o pasar la task explícitamente |
| `mova run [proyecto]` con `mode: "automatic"` | La salida tiene el formato compacto (`PROJECT:... AGENT:...`), no el formato con encabezados `## AGENTS` | Sigue viendo el formato humano | `contextCompiler.enabled` es `false`, o `mode` no es `"automatic"` |
| Foco a un símbolo inexistente | Bloque `FOCUS:` con `not found: <símbolo>` o extracto acotado, nunca vacío | El bloque está vacío | No debería ocurrir — reportar como bug |
| Foco a un directorio | Bloque `FOCUS:` con `dir(N): archivo1, archivo2...` | Aparece contenido de archivos | No debería ocurrir — reportar como bug |
