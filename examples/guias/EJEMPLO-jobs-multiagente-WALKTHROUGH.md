# Ejemplo funcionando — Job Engine + Logging + Multiagente + Cron

Este archivo documenta una corrida real de principio a fin del proyecto
de ejemplo `projects/ejemplo-jobs-multiagente/`, incluido en este
repositorio. Todo el output de abajo es real — capturado corriendo el
binario compilado (`go build -o mova ./src/cli`) contra este mismo
ejemplo.
 
## Qué incluye el ejemplo

```text
projects/ejemplo-jobs-multiagente/
    config.json                 ← orquestador del grupo (multiagente)
    auditor-checkout/
        project.json             ← 1 job programado (schedule + save + memory + budget)
        memory.md
    auditor-cookies/
        project.json             ← 2 jobs (auditoría + archivado mensual de memoria)
        memory.md

examples/ejemplo-jobs-multiagente-repo/
    checkout.html                ← checkbox que mezcla T&C + privacidad + marketing
    cookies.html                 ← banner de cookies sin opción de rechazo
```

Dos agentes de compliance (`auditor-checkout`, `auditor-cookies`), cada
uno un proyecto Mova normal con su propio `project.json`, `memory.md` y
`jobs`, orquestados desde `config.json`. Ninguno de los dos necesita un
modelo/LLM configurado para este ejemplo — la acción `tasks` de un job
solo arma el contexto (agents+skills+prompt+focus), no llama a ningún
proveedor.

## Paso 1 — Ver los jobs configurados

```bash
$ mova jobs list ejemplo-jobs-multiagente/auditor-checkout
  [0] schedule="0 2 * * *"  Auditoría nocturna del checkout — corre todos los días a las 2 AM

$ mova jobs list ejemplo-jobs-multiagente/auditor-cookies
  [0] schedule="0 2 * * *"  Auditoría nocturna del banner de cookies — corre todos los días a las 2 AM
  [1] schedule="0 3 1 * *"  Archivado mensual de memoria — el día 1 de cada mes a las 3 AM, sin tasks
```

## Paso 2 — Correr los jobs bajo demanda (sin esperar al cron)

```bash
$ mova jobs run ejemplo-jobs-multiagente/auditor-checkout
[ejemplo-jobs-multiagente/auditor-checkout] 2026-07-31 02:22:14
  ✓ task "auditar" executed (1671 tokens)
  ✓ file saved: examples/ejemplo-jobs-multiagente-repo/reports/auditoria-checkout_2026-07-31.md
  ✓ memory updated: projects/ejemplo-jobs-multiagente/auditor-checkout/memory.md
  ✓ budget report: projects/ejemplo-jobs-multiagente/auditor-checkout/mova-budget-report.md (1721 tokens)

$ mova jobs run ejemplo-jobs-multiagente/auditor-cookies
[ejemplo-jobs-multiagente/auditor-cookies] 2026-07-31 02:22:14
  ✓ task "auditar" executed (1632 tokens)
  ✓ file saved: examples/ejemplo-jobs-multiagente-repo/reports/auditoria-cookies_2026-07-31.md
  ✓ memory updated: projects/ejemplo-jobs-multiagente/auditor-cookies/memory.md
  ✓ budget report: projects/ejemplo-jobs-multiagente/auditor-cookies/mova-budget-report.md (1682 tokens)
[ejemplo-jobs-multiagente/auditor-cookies] 2026-07-31 02:22:14
  ✓ memory archived: ejemplo-jobs-multiagente/auditor-cookies (entries older than 30 days)
```

Cada job corrió sus 4 (o 5) acciones en el orden fijo del motor: tasks
→ save → memory → memory_archive → delete → budget. El segundo job de
`auditor-cookies` (solo `memory_archive`, sin `tasks`) también corrió,
mostrando que las acciones son completamente independientes entre sí.

**`memory.md` resultante** (`projects/.../auditor-checkout/memory.md`):
```text
Auditoría de checkout ejecutada el 2026-07-31 a las 02:22 — ver reports/auditoria-checkout_2026-07-31.md

---
```

**Extracto real de `reports/auditoria-checkout_2026-07-31.md`** (el `.md`
generado por la acción `save`, formato elegido automáticamente por la
extensión del path — igual que `/save` en el chat):
```text
## Task: auditar

# Mova Context — ejemplo-jobs-multiagente/auditor-checkout / auditar
Generated: 2026-07-31 02:22 | Repo: examples/ejemplo-jobs-multiagente-repo | Lang: es | LLM: not set | Profile: powerful

---
## AGENTS

<!-- agent: privacy-auditor -->
# Rol
Privacy Auditor. Audita flujos de UI/UX y código de captura de consentimiento
contra normativa de protección de datos (Ley 21.719 y equivalentes tipo GDPR)...
```

## Paso 3 — Correr el grupo multiagente completo

```bash
$ mova agents list ejemplo-jobs-multiagente
Group: ejemplo-jobs-multiagente
Dos agentes de compliance (auditor-checkout, auditor-cookies) que auditan
distintas partes del mismo sitio...
Agents:
  - ejemplo-jobs-multiagente/auditor-checkout
  - ejemplo-jobs-multiagente/auditor-cookies

$ mova agents run ejemplo-jobs-multiagente
=== ejemplo-jobs-multiagente/auditor-checkout ===
# Mova Context — ejemplo-jobs-multiagente/auditor-checkout / auditar
Generated: 2026-07-31 02:22 | Repo: examples/ejemplo-jobs-multiagente-repo | Lang: es | ...
[... contexto completo armado (agents+skills+prompt+focus), igual que "mova run" ...]

=== ejemplo-jobs-multiagente/auditor-cookies ===
# Mova Context — ejemplo-jobs-multiagente/auditor-cookies / auditar
Generated: 2026-07-31 02:22 | ...
[... contexto completo del segundo agente ...]
```

`mova agents run` corrió ambos agentes **secuencialmente**, uno
después del otro, cada uno a través del mismo pipeline de
ensamblado+Budget-gate que usa `mova run` para cualquier proyecto —
"agente" nunca es un caso especial, es un proyecto llamado
`<grupo>/<agente>`.

## Paso 4 — Activar el logging y ver el log real

Por defecto `config/log/logging.json` tiene `"enabled": false`. Para
esta corrida se cambió a `true` (dejando el resto de la config por
defecto: nivel `info`, todas las categorías, rotación diaria). Después
de correr los pasos 1-3, el archivo `logs/mova.log` quedó así:

```text
2026-07-31 02:22:10 [info] [cli] command: jobs list ejemplo-jobs-multiagente/auditor-checkout
2026-07-31 02:22:10 [info] [cli] command: jobs list ejemplo-jobs-multiagente/auditor-cookies
2026-07-31 02:22:14 [info] [cli] command: jobs run ejemplo-jobs-multiagente/auditor-checkout
2026-07-31 02:22:14 [info] [jobs] starting job for project=ejemplo-jobs-multiagente/auditor-checkout schedule="0 2 * * *"
2026-07-31 02:22:14 [info] [jobs] finished job for project=ejemplo-jobs-multiagente/auditor-checkout (4 step(s))
2026-07-31 02:22:14 [info] [cli] command: jobs run ejemplo-jobs-multiagente/auditor-cookies
2026-07-31 02:22:14 [info] [jobs] starting job for project=ejemplo-jobs-multiagente/auditor-cookies schedule="0 2 * * *"
2026-07-31 02:22:14 [info] [jobs] finished job for project=ejemplo-jobs-multiagente/auditor-cookies (4 step(s))
2026-07-31 02:22:14 [info] [jobs] starting job for project=ejemplo-jobs-multiagente/auditor-cookies schedule="0 3 1 * *"
2026-07-31 02:22:14 [info] [jobs] finished job for project=ejemplo-jobs-multiagente/auditor-cookies (1 step(s))
2026-07-31 02:22:17 [info] [cli] command: agents list ejemplo-jobs-multiagente
2026-07-31 02:22:17 [info] [cli] command: agents run ejemplo-jobs-multiagente
2026-07-31 02:22:17 [info] [orchestrator] running agent ejemplo-jobs-multiagente/auditor-checkout
2026-07-31 02:22:17 [info] [orchestrator] running agent ejemplo-jobs-multiagente/auditor-cookies
```

Cada categoría (`cli`, `jobs`, `orchestrator`) quedó registrada por
separado — se puede filtrar cualquiera de ellas en
`config/log/logging.json` → `"categories"` sin tocar las demás.

**Para reproducirlo tu mismo:**
```bash
# En config/log/logging.json:
"enabled": true
```

## Paso 5 — El daemon (`mova jobs start`) disparando por horario real

Para probar el scheduler disparando por horario (no por `jobs run`
manual), se agregó temporalmente un tercer agente con un job que
corre **cada minuto** (`"schedule": "* * * * *"`) y luego se corrió el
daemon ~68 segundos:

```bash
$ mkdir -p projects/ejemplo-jobs-multiagente/demo-cron
$ cat > projects/ejemplo-jobs-multiagente/demo-cron/project.json << 'EOF'
{
  "project": "ejemplo-jobs-multiagente/demo-cron",
  "repo": "examples/ejemplo-jobs-multiagente-repo",
  "jobs": [
    { "schedule": "* * * * *", "memory": "Latido del daemon a las {time} del {date}" }
  ]
}
EOF
$ touch projects/ejemplo-jobs-multiagente/demo-cron/memory.md

$ mova jobs start
[Jobs] scheduler started — checking every project once a minute (Ctrl+C to stop)
[ejemplo-jobs-multiagente/demo-cron] 2026-07-31 02:22:32
  ✓ memory updated: projects/ejemplo-jobs-multiagente/demo-cron/memory.md
[ejemplo-jobs-multiagente/demo-cron] 2026-07-31 02:23:32
  ✓ memory updated: projects/ejemplo-jobs-multiagente/demo-cron/memory.md
```

**`memory.md` resultante:**
```text
Latido del daemon a las 02:23 del 2026-07-31

---

Latido del daemon a las 02:22 del 2026-07-31

---
```

Disparó dos veces, cada 60 segundos, exactamente como especifica
`"* * * * *"` — el daemon revisa **todos** los proyectos una vez por
minuto y corre cualquier job cuyo `schedule` coincida con el minuto
actual, sin importar cuántos proyectos o jobs haya declarados.

El `demo-cron` no se incluye en el repositorio (dispararía cada minuto
apenas alguien corra `mova jobs start`) — se muestra acá solo como
receta para que lo repliques tú mismo si quieres ver el daemon en vivo.

## Paso 6 — Lo mismo desde HTTP y MCP (mismo motor, otra puerta)

```bash
# HTTP
$ mova mcp start --port 3000 &
$ curl -X POST http://localhost:3000/jobs/run \
    -d '{"project": "ejemplo-jobs-multiagente/auditor-checkout"}'
$ curl -X POST http://localhost:3000/agents/run \
    -d '{"group": "ejemplo-jobs-multiagente"}'
```

```bash
# MCP (stdio) — tools "run_job", "list_jobs", "run_agent", "list_agents"
$ mova mcp start --stdio
```

Ambos terminan llamando exactamente `jobs.RunJob` /
`orchestrator.RunGroup` — el mismo código que corrió en los pasos 2 y
3, solo que disparado por HTTP/MCP en lugar del CLI.

## Resumen de lo verificado en esta corrida

| Pieza | Verificado con salida real |
|---|---|
| Job Engine — acción `tasks` | ✓ (armó contexto, 1671/1632 tokens) |
| Job Engine — acción `save` | ✓ (escribió los `.md` en `reports/`) |
| Job Engine — acción `memory` | ✓ (`memory.md` actualizado con `{date}`/`{time}` expandidos) |
| Job Engine — acción `memory_archive` | ✓ (corrió sin `tasks`, job independiente) |
| Job Engine — acción `budget` | ✓ (`mova-budget-report.md` generado) |
| Cron — `schedule` con `mova jobs run` | ✓ (bajo demanda, ignora el horario) |
| Cron — `schedule` con `mova jobs start` (daemon) | ✓ (2 disparos reales, cada 60s) |
| Logging — categorías `cli`/`jobs`/`orchestrator` | ✓ (`logs/mova.log` real) |
| Multiagente — `config.json` + auto-listado | ✓ |
| Multiagente — `mova agents run` secuencial | ✓ (2 agentes, en orden) |
