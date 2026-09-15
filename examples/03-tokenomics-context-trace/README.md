# 03 — Tokens, presupuesto y Context Trace

**Qué es:** `focus`, `exclude` (con AST), `task` y presupuesto/circuit breaker trabajando juntos
sobre un log con líneas repetidas y un archivo con una función legacy que hay que excluir
**sin** partir el archivo en dos.

## Ejecutar (1 comando)

```bash
mova run 03-tokenomics-context-trace --diagram --export png --path ./evidencia.png
```

## Qué vas a ver  

| Paso | Resultado |
|---|---|
| Contexto seleccionado | `checkout.js` + `server.log` (`focus`) |
| AST exclude | `exclude: ["checkout.js::func=procesarPagoLegacy"]` — el archivo se analiza completo, solo el cuerpo de esa función desaparece |
| Sanitizer | 48 líneas idénticas de `server.log` colapsadas en `[×48 repeticiones idénticas omitidas]` |
| Presupuesto | `max_tokens_per_run: 5000` + `max_monthly_usd: 5.0`, `on_exceed: warn` (circuit breaker) |
| Evidencia | `evidencia.png` (pipeline de reducción por archivo) + `context-report.md` |

`evidencia-ejemplo.png` en esta carpeta es una muestra ya generada. Para ver el `exclude` AST
en acción directamente en el contexto ensamblado:

```bash
mova run 03-tokenomics-context-trace | grep -A3 procesarPagoLegacy
```

## Escenario `mova trace` contra un repo remoto (mismo motor, un comando)

```bash
mova trace --repo https://github.com/<usuario>/<repo> --task "revisar el módulo X" --ignore "docs/**,*.lock"
```

```
Repositorio remoto
        ↓
task + --ignore   (AST y relevancia se aplican automáticamente al rankear archivos)
        ↓
Context Trace
        ↓
CANDIDATE / ALLOWED / WOULD SANITIZE / BLOCKED / EXCLUDED
        ↓
tokens + costo
        ↓
context-report.md + context-diagram.png + pii-audit-log.json
```

**Nota honesta:** contra un repo remoto (sin `project.json`), la CLI hoy solo acepta `--task`
e `--ignore` — no `--focus`/`--exclude` explícitos (el ranking por relevancia y AST se aplica
solo automáticamente). El combo completo `focus + exclude (AST) + task` de la tabla de arriba
es exclusivo del modo `project.json`, que es el que este ejemplo ya demuestra funcionando.
`mova trace` es un alias corto de `mova context-trace` (mismo comando, mismo motor).

`project/project.json` en esta carpeta es una copia de lectura — el archivo real vive en
`projects/03-tokenomics-context-trace/project.json`.
