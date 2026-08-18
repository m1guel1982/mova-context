# Ejemplo funcionando — Ley 21.719, datos de clientes y PII Masking

Este archivo documenta una corrida real de principio a fin del proyecto
de ejemplo `projects/ejemplo-ley21719-pii-context/`, incluido en este
repositorio. Todo el output de abajo es real — capturado corriendo el
binario compilado (`go build -o mova ./src/cli`) contra este mismo
ejemplo. Los datos de clientes son 100% FICTICIOS.

No confundir con `projects/ejemplo-ley21719/` (otro ejemplo, sobre un
checkbox de consentimiento en un checkout) — este ejemplo es sobre un
problema distinto: **qué datos personales existen en un contexto que se
le va a armar a un LLM, y cuánta de esa información es realmente
necesaria enviar**.

## Qué incluye el ejemplo

```text
projects/ejemplo-ley21719-pii-context/
    config.json                  ← orquestador del grupo (multiagente)
    data-analyst/project.json     ← rol 1: qué datos personales existen
    purpose-analyst/project.json  ← rol 2: para qué se usan / qué sistemas
    ai-privacy-reviewer/project.json ← rol 3: qué NO es necesario enviar a un LLM
                                        (único sub-proyecto con pii_masking activado)

examples/ejemplo-ley21719-pii-context-repo/data/
    customers.json         ← 10 clientes ficticios (export típico de CRM/ERP)
    customer-profile.pdf   ← ficha ampliada de 3 clientes (sistema legado, PDF real)
    privacy-policy.docx    ← política de privacidad ficticia (Word real)
    system-logs.txt        ← log técnico con líneas repetidas (para el Sanitizer)

agents/base/i18n/{es,en}/compliance/
    data-analyst.md · purpose-analyst.md · ai-privacy-reviewer.md

skills/base/i18n/{es,en}/compliance/
    pii-context-reduction.md   ← documenta la función PII Masking, alcance y límites

prompts/base/i18n/{es,en}/compliance/
    analizar-contexto-clientes-ia.md   ← task compartida por los 3 agentes
```

Tres agentes de compliance, cada uno un proyecto Mova normal con su
propio `project.json`, `budget` y `jobs`, orquestados desde
`config.json` — mismo mecanismo multiagente que
`examples/ejemplo-jobs-multiagente-repo`. Ninguno necesita una API key
configurada para este ejemplo: `mova run`/`mova budget`/`mova jobs run`
arman el contexto igual, sin llamar a ningún proveedor.

## Paso 0 — El PDF y el DOCX son documentos reales, no texto disfrazado

`customer-profile.pdf` y `privacy-policy.docx` se generaron con las
mismas funciones que usa Mova para crear documentos
(`documents.GeneratePDFDocument`/`documents.GenerateWordContract`), y
Mova los lee de vuelta con su extractor real
(`documents.ReadDocumentLayer`) — el mismo que usa `mova chat`. Antes
de este ejemplo, el Focus Resolver leía `.pdf`/`.docx` como bytes
crudos (`os.ReadFile`), lo que mandaba basura binaria al contexto en
vez del texto real; se corrigió como parte de este trabajo (ver
`docs/SOURCE.md § PDF/DOCX focus-reading fix`) — este ejemplo es la
prueba end-to-end de que ahora funciona.

## Paso 1 — Ver los datos ficticios

```bash
$ mova budget ejemplo-ley21719-pii-context/data-analyst
✓ mova-budget-report.md generated
Total tokens: 7276 (cl100k_base)
  anthropic/claude: $0.0218 USD
  google/gemini: $0.0091 USD
  openai/gpt-5: $0.0364 USD
```

## Paso 2 — El Sanitizer reduce ruido real, medible

`data/system-logs.txt` incluye 20 líneas de log casi idénticas (solo
cambia el timestamp), a propósito — el mismo patrón que
`examples/ejemplo-token-firewall-repo/server.log`. En
`mova-budget-report.md`:

```text
## Sanitizer

Mova Context removed repetitive noise from Focus/Memory before counting
or sending anything — 100% deterministic, no AI involved, microseconds
of work:

- 19 repeated line(s)/header(s) collapsed (e.g. repeated log lines,
  duplicated file headers).

Approximate savings: ~312 tokens (~1250 characters) — already reflected
in every count above.
```

## Paso 3 — PII Masking, desactivado por defecto

`data-analyst` y `purpose-analyst` **no** activan PII Masking en su
`budget` — su `mova-budget-report.md` no incluye la sección "PII
Masking" en absoluto, porque la etapa nunca corrió. Es el comportamiento
por defecto de todo el repositorio: nadie ve un cambio a menos que lo
pida explícitamente.

## Paso 4 — PII Masking, activado explícitamente en un solo sub-proyecto

`ai-privacy-reviewer/project.json` es el único de los tres con:

```json
"budget": {
  "...": "...",
  "pii_masking": { "enabled": true }
}
```

```bash
$ mova budget ejemplo-ley21719-pii-context/ai-privacy-reviewer
✓ mova-budget-report.md generated
Total tokens: 7527 (cl100k_base)
```

Y en el reporte:

```text
## PII Masking

78 of 1757 scanned token(s) in Focus/Memory matched the structural
PII-shape threshold (config/policy.json) and were replaced with a
deterministic `[PII_xxxxxxxx]` pseudonym before counting or sending
anything.

This is a heuristic, structural mitigation (word shape + Shannon
entropy, no word lists) — **not** a legal anonymization or Ley
21.719/GDPR compliance guarantee. It does not detect 100% of PII (false
negatives), and can also mask some non-PII tokens that share a similar
structural shape, such as dates (false positives). It does not replace
legal review, an internal privacy policy, or a compliance program.
```

Mirando el contexto real que arma `mova run` para este sub-proyecto, el
mismo valor original siempre produce el mismo pseudónimo — por ejemplo,
el RUT de un cliente que aparece tanto en `customers.json` como en
`customer-profile.pdf` sale con **el mismo** `[PII_xxxxxxxx]` en ambos
lugares:

```text
"direccion": "Pasaje El Roble 45, Rancagua",
"fecha_registro": "[PII_12656e68]",
"historial_interacciones": [
  "[PII_12656e68] - Alta de cuenta vía app móvil",
  ...
```

Nóta: `fecha_registro` (una fecha con
separadores, `2024-07-30`) también quedó enmascarada con el mismo
pseudónimo que el resto de las apariciones de ese texto — es exactamente
el **falso positivo** que el propio reporte advierte: una fecha
formateada comparte forma estructural con un RUT, y el algoritmo no
distingue el significado, solo la forma. Es información a tener en
cuenta antes de confiar ciegamente en la etapa.

Sobre el log técnico, el Sanitizer y el PII Masking se combinan en el
orden correcto (Sanitizer primero, PII Masking después, sobre el texto
ya reducido):

```text
FOCUS:data/system-logs.txt
...
[PII_fb307417] 03:00:01 sync-worker: INFO conector CRM-legado enlace OK
  [×20 repeticiones idénticas omitidas]
```

## Paso 5 — Correr el grupo completo (los 3 roles)

```bash
$ mova agents list ejemplo-ley21719-pii-context
Group: ejemplo-ley21719-pii-context
Agents:
  - ejemplo-ley21719-pii-context/data-analyst
  - ejemplo-ley21719-pii-context/purpose-analyst
  - ejemplo-ley21719-pii-context/ai-privacy-reviewer

$ mova agents run ejemplo-ley21719-pii-context --all
```

## Paso 6 — Correr los jobs bajo demanda (dónde queda guardado el reporte)

Cada sub-proyecto tiene un job programado (`jobs` en su `project.json`)
que corre la task `analizar` y guarda el resultado narrativo en
`examples/ejemplo-ley21719-pii-context-repo/reports/`, además del
`mova-budget-report.md` técnico en la carpeta del propio agente:

```bash
$ mova jobs run ejemplo-ley21719-pii-context/ai-privacy-reviewer
[ejemplo-ley21719-pii-context/ai-privacy-reviewer] 2026-08-13 01:06:31
  · sanitizer: task "analizar" cleaned (19 repeated line(s), 0 blank-line run(s))
  ✓ task "analizar" executed (7579 tokens)
  ✓ file saved: examples/ejemplo-ley21719-pii-context-repo/reports/analisis-ley21719-ia_2026-08-13.md
  ✓ budget report: projects/ejemplo-ley21719-pii-context/ai-privacy-reviewer/mova-budget-report.md (7583 tokens)
```

Mismo mecanismo para `data-analyst` (`reports/analisis-ley21719-datos_{date}.md`)
y `purpose-analyst` (`reports/analisis-ley21719-finalidad_{date}.md`).

## Para probarlo con un modelo real

Este ejemplo usa `llm_profile: { "config": "llama3.2.3b" }` (Ollama
local) — instalá [Ollama](https://ollama.ai), corré
`ollama pull llama3.2:3b`, y `mova run ejemplo-ley21719-pii-context/ai-privacy-reviewer`
va a generar una respuesta real del modelo, en vez de solo armar el
contexto. Para probar con Claude/GPT/Gemini en la nube, cambiá
`llm_profile` por el proveedor que prefieras (ver
[PROJECT_JSON.md](../docs/i18n/es/PROJECT_JSON.md)) — y considerá
activar `pii_masking` antes de mandar estos datos a un proveedor
externo, sabiendo sus límites (ver Paso 4).

## Ver también

- `docs/i18n/es/COMMANDS.md` § Token Firewall — la explicación completa
  de cada etapa, incluida PII Masking.
- `skills/base/i18n/es/compliance/pii-context-reduction.md` — alcance y
  límites de PII Masking en detalle.
- `config/policy.json` — los umbrales/pesos que usa el algoritmo,
  editables sin recompilar.
- `examples/EJEMPLO-token-firewall-WALKTHROUGH.md` — el ejemplo
  original del Sanitizer/Cache Layout Guard/Circuit Breaker.
