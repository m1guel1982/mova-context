# Compiler v2 — pipeline completo (`strategy: "full"`)

> Documentación: [Español](compiler-v2-pipeline.md) · [English](../en/compiler-v2-pipeline.md)

Detalle técnico de las 8 etapas activadas por `contextCompiler.strategy: "full"`. Para la vista de uso, ver [`context-compiler.md`](context-compiler.md); para la resolución de `focus`, ver [`focus-engine.md`](focus-engine.md).

Cada etapa es un paquete Go independiente bajo `src/compiler/`, con una única responsabilidad, orquestado por `src/compiler/pipeline`:

```
src/compiler/parser      → Knowledge Parser + Classification
src/compiler/semantic     → Semantic Cleanup (Fase 1, reutilizada tal cual)
src/compiler/normalize    → Normalization
src/compiler/dedup        → Deduplication
src/compiler/tokenopt     → Token Optimization (tokenizer real)
src/compiler/priority     → Priority Ranking
src/compiler/budget       → Budget Assembly
src/compiler/pipeline     → orquesta las 7 anteriores
```

---

## 1 — Knowledge Parser (`src/compiler/parser`)

Divide el texto en `Block{Kind, Text}` antes de cualquier transformación. Preserva intactos los fences de código (` ``` `) y las tablas Markdown — nunca los fragmenta línea por línea.

| Kind | Se detecta por |
|---|---|
| `Code` | Fence ` ``` ` |
| `Table` | Líneas `\| ... \|` contiguas |
| `Metadata` | Front-matter YAML (`---`) al inicio del archivo |
| `Memory` | Línea que empieza con fecha (`YYYY-MM-DD` o `DD/MM/YYYY`) |
| `Example` | Párrafo que empieza con "Ejemplo:" / "Example:" |
| `Rule` | Contiene lenguaje normativo (`nunca`, `siempre`, `obligatorio`, `must`, `never`, ...) |
| `Variable` | Contiene `{{...}}` |
| `Placeholder` | Contiene `TODO`/`FIXME`/`XXX` o `<marcador>` |
| `Instruction` | Todo lo demás (catch-all) |

100% determinista: clasificación por patrones, nunca por heurística probabilística ni LLM.

---

## 2 — Semantic Cleanup (`src/compiler/semantic`)

La misma Fase 1 que usa `strategy: "semantic"`, aplicada aquí selectivamente: solo a bloques `Rule`, `Instruction`, `Example` y `Memory`. Nunca a `Code`, `Table`, `Variable`, `Placeholder` o `Metadata`.

---

## 3 — Normalization (`src/compiler/normalize`)

Reglas de superficie, deterministas y configurables (`[]normalize.Rule`, cada una `Pattern → Replace`):

- Comillas tipográficas (`""`) → rectas (`"`).
- Viñetas `*` / `+` → `-`.
- Puntuación repetida (`!!!`, `??`) → un solo carácter.
- Espacios finales de línea, `\u00A0` (NBSP) → espacio normal.

Nunca reescribe una oración ni sustituye una palabra — eso podría cambiar el significado de una regla.

---

## 4 — Deduplication (`src/compiler/dedup`)

Elimina un bloque solo si su *fingerprint* (texto en minúsculas, espacios colapsados) es **idéntico** a uno ya visto del mismo `Kind`. Nunca usa similitud difusa ni distancia de edición — el riesgo de fusionar dos reglas distintas pero parecidas es inaceptable. `Code` nunca se deduplica: dos funciones distintas pueden compartir líneas por coincidencia.

---

## 5 — Token Optimization (`src/compiler/tokenopt`)

Mide **siempre** con un tokenizer BPE real (`github.com/tiktoken-go/tokenizer`, vocabulario embebido en el binario — cero llamadas de red en runtime), nunca por caracteres o bytes.

| Modelo (contiene) | Encoding usado | Nota |
|---|---|---|
| `gpt-4o`, `o1`, `o3`, `o4`, `gpt-4.1`, `gpt-5` | `o200k_base` | Exacto |
| `gpt-4`, `gpt-3.5` | `cl100k_base` | Exacto |
| `claude`, `gemini`, `llama`, `mistral`, `deepseek`, `qwen`, `phi` | `cl100k_base` | **Aproximado** — esos proveedores no publican un tokenizer offline; se documenta, nunca se oculta |
| `davinci`, `curie`, `babbage`, `ada` | `p50k_base` | Exacto |
| (vacío o desconocido) | `cl100k_base` | Default, indicado en el reporte |

Aplica además optimizaciones mecánicas demostrables: colapsar líneas en blanco redundantes, espacios finales, viñetas duplicadas. Nunca sobre bloques `Code`.

---

## 6 — Priority Ranking (`src/compiler/priority`)

Etiqueta cada bloque con un nivel — solo para que Budget Assembly sepa qué recortar primero. No elimina nada por sí sola.

| Nivel | Kind |
|---|---|
| Critical (100) | `Rule` |
| Structural (80) | `Code`, `Table`, `Metadata` |
| Operational (60) | `Memory`, `Variable` |
| Guidance (40) | `Instruction` |
| Illustrative (20) | `Example`, `Placeholder` |

---

## 7 — Budget Assembly (`src/compiler/budget`)

Si `max_tokens > 0`: incluye **todos** los bloques `Critical` primero, sin importar el presupuesto — si eso ya lo excede, lo indica (`overflowed: true`) en vez de recortar una regla. Luego rellena por prioridad decreciente hasta agotar el presupuesto, preservando siempre el orden original del documento.

**Alcance actual, documentado sin ocultar:** el presupuesto se aplica **por bloque de conocimiento** (un agent, un skill, el prompt, o la memoria), no sobre `contexto.txt` completo — eso requiere que una capa de orquestación vea el documento entero de una vez.

---

## Ejemplo medido

```go
content := "Siempre debemos validar el monto.\n\nSiempre debemos validar el monto.\n\nEjemplo: foo()\n"
out, report := pipeline.Run(content, pipeline.Options{Model: "claude-sonnet-4-6", MaxTokens: 15})
```

```text
out:    "Siempre debemos validar el monto."
report: {Tokenizer: cl100k_base (aproximado), TokensBefore: 20, TokensAfter: 14,
         BlocksTotal: 3, BlocksRemoved: 1, BlocksDropped: 1, Overflowed: false, MaxTokens: 15}
```

El párrafo duplicado se deduplicó (`BlocksRemoved: 1`); el ejemplo se recortó por presupuesto (`BlocksDropped: 1`); la regla se mantuvo completa dentro del límite de 15 tokens.
