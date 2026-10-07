# Example 02 — per-block PII and secrets, with `dry_run`

[Español](README.md)

Fictional Chilean customer data (`customers.json`, a PDF profile, a DOCX policy and a varied system log). The project enables `pii_masking` and `egress_audit.dry_run: true`: **nothing is released**, and the run shows what would have been sent.

```bash
mova run 02-pii-compliance-governance                    # [Evidence] run <id>; dry_run
mova budget 02-pii-compliance-governance --focus         # savings split into selection vs sanitization
```

What to look at in `projects/02-pii-compliance-governance/runs/<id>/`:
- `context.txt`: all 10 names, addresses, RUTs and emails are pseudonymized (`[PII_…]`), also inside the PDF text (known-value propagation from `field_keys`). `FOCUS:` markers and the JSON structure stay intact.
- `manifest.json → governance.changed_blocks`: what was masked in each file and by which detector.

**What this example does NOT show:**
- **Selection:** focus includes all 4 files. The "whole repo vs focus" difference reported by `mova budget --focus` comes from PDF/DOCX text extraction, not from selection.
- **Masking accuracy:** the shape/entropy score over-masks logs (timestamps, identifiers) and has no measured precision or recall. It is mitigation, not compliance.
