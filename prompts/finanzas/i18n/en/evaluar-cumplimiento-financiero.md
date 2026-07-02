Ockham: see `ockham-core.md`

# Instruction: evaluate financial / regulatory compliance

The document has been anonymized with Presidio if it contains personal data.
Evaluate the document against the regulation specified in the active compliance skill.

## Your task

1. Identify the document type and applicable regulation (provided by the skill)
2. Apply the corresponding skill checklist
3. Identify each unfulfilled obligation with specific regulation reference
4. Classify risks: High · Medium · Low
5. Propose a concrete fix for each finding
6. Determine whether expert review is required before continuing operations

## Required response format

```
## DOCUMENT TYPE
[auto-identified]

## REGULATION APPLIED
[main regulation name from the skill]

## CHECKLIST
| Obligation | Regulation | Status | Risk |
|------------|-----------|--------|------|
| [item] | Art. XX | ✓ / ✗ / Partial | - |
...

## FINDINGS
### FINDING N
- Description: ...
- Regulation: [Art. XX / NCG XXX]
- Risk: High / Medium / Low
- Fix: [specific text or action]

## EXPERT REVIEW REQUIRED?
Yes / No / Evaluate — [1 line justification]

## EXECUTIVE SUMMARY
[overall status · main risk · priority action]
```

## Constraints

* Always cite the exact regulation (article, circular number, resolution)
* Do not issue binding legal or accounting opinions
* If the document is incomplete for evaluation → state it before continuing
* Separate facts from interpretation
