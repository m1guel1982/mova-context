# 02 — Gobernanza de PII y secretos antes de la inferencia

**Qué es:** datos reales de clientes (ficticios) van a ser consultados por un LLM.
Mova detecta indicadores de PII, aplica enmascarado técnico, y deja evidencia — antes del envío.

**Escenario ilustrativo:** Ley 21.719 (Chile). Esto es evidencia y control pre-inferencia,
**no** es una certificación de cumplimiento normativo.

## Ejecutar (1 comando)

```bash
mova run 02-pii-compliance-governance --diagram --export png --path ./diagramas/evidencia.png
```

## Qué vas a ver  

| Paso | Resultado |
|---|---|
| Contexto seleccionado | `customers.json`, `customer-profile.pdf`, `privacy-policy.docx`, `system-logs.txt` |
| Control aplicado | Sanitizer ON · **PII Masking ON** (`budget.pii_masking.enabled: true`) |
| Decisión | ~78 de 1.694 tokens candidatos pseudonimizados con `[PII_xxxxxxxx]`; 64% de reducción total |
| Evidencia | `evidencia.png` (diagrama) + `context-report.md` + `mova-budget-report.md` + `pii-audit-log.json` |

`evidencia-ejemplo.png` en esta carpeta es una muestra ya generada.

## Dónde queda cada evidencia (importante, no es redundante)

- **`mova-budget-report.md`** (en `projects/02-pii-compliance-governance/`, se genera con `mova budget`) — el resultado REAL de PII Masking para esta corrida: cuántos tokens se pseudonimizaron y por qué.
- **`pii-audit-log.json`** — aquí su sección `security` queda en cero porque ese detalle fino (archivos con PII, hallazgos clasificados) solo se llena en modo *discovery* (escaneo de un repo remoto sin `project.json`), no en una corrida con `project.json` como esta. `execution.policy_author` / `agent_client` / `target_model` sí quedan siempre completos.

## Ver la traza en texto (sin diagrama)

```bash
mova context-trace 02-pii-compliance-governance --export md
```

**Disclaimer técnico:** el enmascarado es heurístico (forma estructural + entropía), no un diccionario de nombres.
No detecta el 100% de la PII y puede marcar falsos positivos. No reemplaza revisión legal.

`project/project.json` en esta carpeta es una copia de lectura — el archivo real vive en `projects/02-pii-compliance-governance/project.json`.
