# Ejemplo funcionando — Token Firewall

Este archivo documenta una corrida real de principio a fin del proyecto
de ejemplo `projects/ejemplo-token-firewall/`, incluido en este
repositorio. Todo el output de abajo es real — capturado corriendo el
binario compilado (`go build -o mova ./src/cli`) contra este mismo
ejemplo. Funciona igual sin importar qué proveedor de modelo tengas
configurado (Claude, GPT, Gemini, Ollama u otro) — el Sanitizer y el
Circuit Breaker son completamente independientes del proveedor, y el
Cache Layout Guard se adapta automáticamente (ver COMMANDS.md § Token
Firewall § "Cache Layout Guard, por proveedor").

## Qué incluye el ejemplo
 
```text
projects/ejemplo-token-firewall/
    project.json                  ← budget con el Token Firewall completo configurado
    memory.md

examples/ejemplo-token-firewall-repo/
    checkout.js                   ← comentario de encabezado grande + líneas en blanco de sobra
    server.log                    ← 53 líneas, 48 casi idénticas (solo cambia el timestamp)
```

`checkout.js` y `server.log` son el `focus` de la única task del
proyecto (`revisar-checkout`) — el caso real y común: un archivo fuente
con boilerplate, y un log con ruido repetitivo.

## Paso 1 — Ver el reporte con el Token Firewall completo

```bash
$ mova budget ejemplo-token-firewall
✓ mova-budget-report.md generated: projects/ejemplo-token-firewall/mova-budget-report.md

Total tokens: 1764 (cl100k_base)
  anthropic/claude: $0.0053 USD
  google/gemini: $0.0022 USD
  openai/gpt-5: $0.0088 USD
```

## Paso 2 — Comparar con el Sanitizer desactivado (medición real)

```bash
# Con "sanitize": {"enabled": false} en project.json:
$ mova budget ejemplo-token-firewall
Total tokens: 2737

# Con la configuración real del ejemplo (sanitize habilitado):
$ mova budget ejemplo-token-firewall
Total tokens: 1764
```

**Ahorro real medido: 973 tokens, un 35.6%** — sin ningún cambio de
significado: el mismo `checkout.js` y el mismo `server.log`, solo sin
la repetición.

## Paso 3 — El reporte completo (`mova-budget-report.md`)

Extracto real, generado por el comando de arriba:

```text
## Sanitizer

Mova Context removed repetitive noise from Focus/Memory before counting
or sending anything — 100% deterministic, no AI involved, microseconds
of work:

- 47 repeated line(s)/header(s) collapsed (e.g. repeated log lines,
  duplicated file headers).
- 1 run(s) of excess blank lines collapsed.

Approximate savings: ~518 tokens (~2075 characters) — already reflected
in every count above.

## Token Firewall — Summary

Before vs. after the full pipeline:

|        | Before | After | Savings |
|--------|--------|-------|---------|
| Tokens | 2737   | 1764  | 35.6%   |
| Cost (claude) | $0.0082 | $0.0053 | 35.6% |
| Cost (gemini) | $0.0034 | $0.0022 | 35.6% |
| Cost (gpt-5)  | $0.0137 | $0.0088 | 35.6% |

## Tokens per File (Focus)

| File | Tokens |
|------|--------|
| checkout.js | 230 |
| server.log  | 142 |

## Cache Layout Guard

"cache_hint" is enabled: the system prompt sent to the model is laid
out as a stable prefix (agents + skills + prompt) followed by
everything that changes every run (timestamp, focus, memory).

Static prefix: 1167 tokens
Prefix fingerprint: d58f4ec76275f850
Estimated tokens reused on a cache hit: ~1050

## Circuit Breaker

Per-run limit: 1764 / 5000 tokens (OK)
Monthly spend: $0.00 / $5.00 (OK) — tracked in mova-spend.json
Status: within budget.
```

## Paso 4 — El Circuit Breaker deteniendo una corrida de verdad

Con `"max_tokens_per_run": 100` y `"on_exceed": "abort"` (a propósito,
muy bajo, para forzar el corte):

```bash
$ mova run ejemplo-token-firewall revisar-checkout
[Project] Loading project configuration...
[Context] Building context...
[Focus] Selected 3 file(s).

Circuit breaker: esta corrida usa 1763 tokens, por encima del límite
por corrida configurado (100).
```

**No se envió nada a ningún modelo.** El proceso terminó ahí mismo,
antes de la llamada HTTP real — exactamente lo que promete
`"on_exceed": "abort"`.

## Paso 5 — Probar que cada etapa se puede apagar individualmente

Cuatro pruebas reales, cada una cambiando un solo campo de
`project.json`:

```bash
# "detailed_reports": false → el reporte deja de incluir las secciones
# nuevas (Sanitizer/Cache Layout Guard/Circuit Breaker/comparación),
# solo quedan los totales de siempre.

# "sanitize": {"enabled": false} → el total vuelve a 2737 (sin optimizar).

# "cache_hint": false → desaparece la sección "Cache Layout Guard".

# "circuit_breaker": false → desaparece la sección "Circuit Breaker",
# aunque los límites sigan configurados en el JSON.
```

Las cuatro se verificaron con el binario real antes de publicar este
ejemplo — cada toggle apaga exactamente lo que dice apagar, y nada más.

## Paso 6 — Tiempo real medido

```text
Primera corrida (sin Context Cache tibio):   ~69 ms
Segunda corrida (con Context Cache tibio):   ~57 ms
```

La mayor parte de esos milisegundos son arranque del proceso y lectura
de archivos, no el Sanitizer en sí — que sobre contenido de este tamaño
corre en microsegundos, sea cual sea el proveedor de modelo detrás.

## Resumen de lo verificado en esta corrida

| Pieza | Verificado con salida real |
|---|---|
| Sanitizer — dedup de logs con timestamp variable | ✓ (47 de 48 líneas colapsadas) |
| Sanitizer — colapso de líneas en blanco | ✓ |
| Sanitizer — preserva el encabezado `## FOCUS` al reescribir | ✓ |
| Cache Layout Guard — prefijo estático + huella | ✓ |
| Circuit Breaker — modo `warn` | ✓ |
| Circuit Breaker — modo `abort` (corta antes de llamar al modelo) | ✓ |
| Comparación antes/después en el reporte | ✓ (35.6% real) |
| Desglose de tokens por archivo | ✓ |
| Los 4 toggles individuales | ✓ (los 4 probados) |
| Funciona igual sin importar el proveedor configurado | ✓ (Sanitizer/Circuit Breaker no dependen del proveedor; Cache Layout Guard se adapta) |
