# mova — Gobernanza de Contexto Pre-Inferencia para LLMs

**Categoría: Gobernanza y Auditoría de Contexto Pre-Inferencia para LLMs.**

`mova` se coloca entre "el contexto está listo" y "se envía al LLM". Para desarrolladores y agentes
(Claude Code, Cursor, Windsurf), responde **11 preguntas de auditoría** sobre selección, gobernanza,
seguridad, trazabilidad y costo del contexto — con evidencia, antes de realizar la inferencia y consumir tokens del proveedor.

**Alcance:** `mova` no es una plataforma de AI Security completa. Su alcance es el contexto que una
aplicación o agente pretende entregar a un LLM, y la evidencia de las decisiones tomadas sobre ese
contexto antes de la inferencia.

## Empezar en 1 comando

```bash
mova run 02-pii-compliance-governance --diagram --export png --path ./evidencia.png
```

Ver `examples/` — 3 ejemplos, cada uno 1 comando .

## Matriz de Auditoría Pre-Inferencia

| # | Pregunta de Gobernanza / Auditoría | Cómo responde `mova` | Evidencia / Artefacto |
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
| 11 | ¿Qué modelo estaba destinado a recibirlo? | `TargetModel` — `<provider>/<config>` de `llm_profile` | Reportes / diagrama |

## Arquitectura

```
[ Agente / IDE (Claude Code, Cursor, MCP) ]
                   │
                   ▼  Solicitud de contexto
┌───────────────────────────────────────────────────┐
│     MOVA — GOBERNANZA DE CONTEXTO PRE-INFERENCIA      │
│  Focus/Exclude (AST) · Sanitizer · PII Masking     │
│  Circuit Breaker de presupuesto · Diagrama PNG/PDF │
└───────────────────────────────────────────────────┘
                   │
                   ▼  Contexto auditable, con evidencia
        [ LLM destino (Anthropic, Ollama, etc.) ]
```

Una misma política de gobernanza, cuatro formas de integración: CLI, mova chat, MCP (stdio/HTTP) y HTTP REST.

## Ejemplos 

| Ejemplo | Qué demuestra |
|---|---|
| `examples/01-mcp-agent-governance` | Un agente MCP pide contexto; Mova decide y deja evidencia |
| `examples/02-pii-compliance-governance` | PII/secretos, enmascarado, política aplicada (escenario Ley 21.719) |
| `examples/03-tokenomics-context-trace` | `focus`/`exclude` (AST)/`task`, presupuesto, Context Trace |

## Instalación

```bash
git clone <este-repo> && cd mova/src && make install
```

## Conectar un agente MCP

```json
{ "mcpServers": { "mova": { "command": "mova", "args": ["mcp"] } } }
```

## Documentación

- [`docs/i18n/es/COMMANDS.md`](../es/COMMANDS.md) — comandos (estilo MAN page)
- [`docs/i18n/es/PROJECT_JSON.md`](../es/PROJECT_JSON.md) — referencia de `project.json`
- [`docs/i18n/es/AST_FILTER.md`](../es/AST_FILTER.md) — sintaxis `archivo::kind=nombre`
- [`docs/i18n/es/context-trace.md`](../es/context-trace.md) — cómo se toma la decisión de contexto
- [`docs/i18n/es/ARTIFACTS.md`](../es/ARTIFACTS.md) — qué es cada archivo que Mova genera
- [`docs/i18n/es/source.md`](../es/source.md) — referencia técnica de arquitectura
- [`docs/i18n/es/FAQ.md`](../es/FAQ.md)

**Nota técnica:** el enmascarado de PII es una mitigación heurística, no una garantía de cumplimiento legal.
