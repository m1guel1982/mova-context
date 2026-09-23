# mova — Gobernanza de Contexto Pre-Inferencia para LLMs

**Categoría: Gobernanza y Auditoría de Contexto Pre-Inferencia para LLMs.**

`mova` se coloca entre "el contexto está listo" y "se envía al LLM". Para desarrolladores y agentes
(Claude Code, Cursor, Windsurf), responde **11 preguntas de auditoría** sobre selección, gobernanza,
seguridad, trazabilidad y costo del contexto — con evidencia, antes de gastar un solo token real.

**Alcance:** `mova` no es una plataforma de AI Security completa. Su alcance es el contexto que una
aplicación o agente pretende entregar a un LLM, y la evidencia de las decisiones tomadas sobre ese
contexto antes de la inferencia.

## Empezar en 1 comando

```bash
mova run 02-pii-compliance-governance --diagram --export png --path ./evidencia.png
```

Ver `examples/` — 3 ejemplos, cada uno 1 comando / 15 segundos para entender.

## Matriz de Auditoría Pre-Inferencia

| # | Pregunta de Seguridad / CISO | Cómo responde `mova` | Evidencia / Artefacto |
|---|---|---|---|
| 1 | ¿Qué información llegó al modelo? | Inventario exacto de archivos y símbolos AST seleccionados | `context-report.md`/`.pdf` |
| 2 | ¿Por qué llegó? | Relevancia por tarea + reglas `focus`/`exclude` (incl. AST) | `context-trace` |
| 3 | ¿Qué política/autoridad permitió esta salida? | `PolicyAuthor` — jerarquía `project.json` → `config/policy.json` → `MOVA_POLICY_AUTHOR` → `system:default` | `project.json` / reportes |
| 4 | ¿Qué política aplicó? | Reglas de inclusión/exclusión del escaneo | `config/policy.json` |
| 5 | ¿Tenía PII o secretos? | Detección heurística de patrones sensibles | `pii-audit-log.json` |
| 6 | ¿Se sanitizó? | Sanitizer + PII Masking (opt-in) vs paso sin cambios | Diagrama · `mova-budget-report.md` |
| 7 | ¿Cuántos tokens fueron? | Medición con tokenizer real (`cl100k_base`) | `context-report.md` |
| 8 | ¿Cuánto costó? | Estimación por proveedor antes del envío | `mova budget` |
| 9 | ¿Qué commit se analizó? | Estado exacto del repo en el momento de auditar | `context-report.md` (Execution ID + commit) |
| 10 | ¿Qué agente lo pidió? | `AgentClient` — `mova-cli`, o el cliente MCP real capturado en `initialize` | Reportes / diagrama |
| 11 | ¿Qué modelo lo recibió? | `TargetModel` — `<provider>/<config>` de `llm_profile` | Reportes / diagrama |

## Arquitectura

```
[ Agente / IDE (Claude Code, Cursor, MCP) ]
                   │
                   ▼  Solicitud de contexto
┌───────────────────────────────────────────────────┐
│     MOVA — MOTOR DE GOBERNANZA PRE-INFERENCIA      │
│  Focus/Exclude (AST) · Sanitizer · PII Masking     │
│  Circuit Breaker de presupuesto · Diagrama PNG/PDF │
└───────────────────────────────────────────────────┘
                   │
                   ▼  Contexto auditable, con evidencia
        [ LLM destino (Anthropic, Ollama, etc.) ]
```

Un motor, cuatro puertas — CLI, `mova chat`, MCP (stdio/HTTP), HTTP REST — todas llaman a la misma función.

## Ejemplos (1 clic, 15 segundos)

| Ejemplo | Qué demuestra |
|---|---|
| `examples/01-mcp-agent-governance` | Un agente MCP pide contexto; Mova decide y deja evidencia |
| `examples/02-pii-compliance-governance` | PII/secretos, enmascarado, política aplicada (escenario Ley 21.719) |
| `examples/03-tokenomics-context-trace` | `focus`/`exclude` (AST)/`task`, presupuesto, Context Trace |
| `examples/04-output-mova-trace-fastApi` | Validación de `mova context-trace` sobre el repositorio remoto de FastAPI. Demuestra el filtrado preciso por `focus`/`exclude` mediante parsing de AST, control de presupuesto de tokens y trazabilidad de contexto en proyectos de código real. |
| `examples/05-test-mcp-cursor` | Validación del aislamiento Air-Gap y gobernanza de egresos sobre el proyecto `02-pii-compliance-governance`. Demuestra la interceptación de `chat_completion` bajo `dry_run: true`, sanitización de PII y evidencia de auditoría consumida desde un cliente MCP.|

## Límites honestos del "Air-Gap" (`dry_run`) — en 15 segundos

Con `egress_audit.dry_run: true`, Mova garantiza SU parte: nunca envía el contexto real a ningún
proveedor, y siempre deja evidencia en disco. Lo que Mova **no puede garantizar** es que el **modelo
anfitrión** (Cursor, Claude Code, Grok, etc. — quien invoca la herramienta MCP) obedezca la directiva
de seguridad que viene con el bloqueo: un anfitrión insistente puede intentar leer otros archivos
locales (`context-report.md`, `project.json`, memoria, etc.) para "reconstruir" el contexto por su
cuenta — esto ya ocurrió en pruebas reales. Mova refuerza el mensaje bloqueado con una directiva
explícita anti-elusión (ver `GOVERNANCE_CONTROLS.md § dry_run`), pero el cumplimiento final de esa
directiva depende del anfitrión, no de Mova — Mova no controla, ni puede controlar, qué hace un
proceso externo con el texto que recibe. No se ofrece esto como una garantía absoluta porque no lo es.

## Instalación

```bash
git clone <este-repo> && cd mova/src && make install
```

## Conectar un agente MCP

```json
{ "mcpServers": { "mova": { "command": "mova", "args": ["mcp", "start", "--stdio"] } } }
```

## Documentación

- [`docs/i18n/es/COMMANDS.md`](../es/COMMANDS.md) — comandos (estilo MAN page)
- [`docs/i18n/es/PROJECT_JSON.md`](../es/PROJECT_JSON.md) — referencia de `project.json`
- [`docs/i18n/es/AST_FILTER.md`](../es/AST_FILTER.md) — sintaxis `archivo::kind=nombre`
- [`docs/i18n/es/CONTEXT-TRACE.md`](../es/CONTEXT-TRACE.md) — cómo se toma la decisión de contexto
- [`docs/i18n/es/ARTIFACTS.md`](../es/ARTIFACTS.md) — qué es cada archivo que Mova genera
- [`docs/i18n/es/GOVERNANCE_CONTROLS.md`](../es/GOVERNANCE_CONTROLS.md) — `debug`, `policies`, `on_exceed`, `dry_run`: qué corta el proceso y qué solo explica
- [`docs/i18n/es/MCP_HTTP_TOOLS.md`](../es/MCP_HTTP_TOOLS.md) — todas las herramientas MCP y endpoints HTTP, copiar-pegar
- [`docs/i18n/es/source.md`](../es/source.md) — referencia técnica de arquitectura
- [`docs/i18n/es/FAQ.md`](../es/FAQ.md)

**Nota técnica:** el enmascarado de PII es una mitigación heurística, no una garantía de cumplimiento legal.
