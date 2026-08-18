# Analizar contexto de clientes para IA — {{PROJECT}}
Proyecto: `{{PROJECT}}` · Normativa de referencia: `{{REGULATION}}`
Ockham: ver `../engineering/ockham-core.md`.

# Consulta original del negocio
> {{QUERY}}

# Objetivo
Sobre el FOCUS entregado (datos de clientes desde JSON/PDF/DOCX, simulando un CRM/ERP/sistema legado), responder la consulta original desde el rol que te fue asignado (ver tu agente: Data Analyst, Purpose Analyst o AI Privacy Reviewer), usando exclusivamente el formato de respuesta definido en ese agente.

# Pasos
1. Leer el FOCUS completo — no asumir datos que no aparezcan literalmente en él
2. Aplicar el formato de respuesta de tu rol a cada hallazgo relevante
3. Citar siempre la fuente (archivo/sección) de cada hallazgo
4. Si tu rol es AI Privacy Reviewer, cerrar explícitamente con qué información no sería necesaria enviar a un LLM externo para responder la consulta original
5. No afirmar que este análisis por sí solo garantiza cumplimiento de {{REGULATION}} — es una ayuda técnica, no asesoría legal

# Dónde queda el reporte de este análisis (importante)
Este proyecto tiene configurado `budget_path`/`token_history_path` propios — al correr `mova budget {{PROJECT}}` (o el job programado del proyecto), Mova Context genera automáticamente:
- `mova-budget-report.md` en la carpeta de este agente, con el detalle de tokens, Sanitizer, PII Masking (si está activado) y Circuit Breaker.
- Un reporte narrativo adicional en `reports/analisis-ley21719_{date}.md`, vía el job programado de este proyecto (ver `jobs` en `project.json`).

Estos reportes se generan igual sin importar el canal usado (CLI `mova run`/`mova budget`, `mova chat`, `mova jobs run`, HTTP/API, o MCP) — es el mismo motor detrás de los cinco.

# Formato de respuesta
Usar exactamente el formato definido en tu agente — no inventar un formato nuevo aquí.
