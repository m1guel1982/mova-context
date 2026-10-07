# Ejemplo 02 — PII y secretos por bloque, con `dry_run`

[English](README_EN.md)

Datos ficticios de clientes chilenos (`customers.json`, una ficha PDF, una política DOCX y un log de sistema variado). El proyecto tiene `pii_masking` activo y `egress_audit.dry_run: true`: **nada se libera**, y el run muestra qué se habría enviado.

```bash
mova run 02-pii-compliance-governance                    # [Evidence] run <id>; dry_run
mova budget 02-pii-compliance-governance --focus         # ahorro separado por selección y por sanitización
```

Qué mirar en `projects/02-pii-compliance-governance/runs/<id>/`:
- `context.txt`: los 10 nombres, direcciones, RUT y emails aparecen seudonimizados (`[PII_…]`), también dentro del texto del PDF (propagación de valores conocidos desde `field_keys`). Los marcadores `FOCUS:` y la estructura JSON quedan intactos.
- `manifest.json → governance.changed_blocks`: qué se enmascaró en cada archivo y con qué detector (`field_values_masked`, `typed_pii_masked`, `shape_pii_masked`, `known_values_masked`).

**Lo que este ejemplo NO demuestra:**
- **Selección:** el focus incluye los 4 archivos. La diferencia entre "repo completo" y "focus" que muestra `mova budget --focus` viene de extraer texto de PDF/DOCX, no de seleccionar.
- **Exactitud del enmascarado:** el puntaje de forma/entropía enmascara de más en los logs (timestamps, identificadores) y no tiene precisión ni recall medidos. Es mitigación, no cumplimiento de la Ley 21.719.
