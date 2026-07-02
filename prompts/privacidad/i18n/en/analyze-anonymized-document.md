Ockham: see `ockham-core.md`

# Instruction: analyze a Presidio-anonymized document

The text you receive was processed by Microsoft Presidio.
Real personal data was replaced with labels like `<PERSON>`, `<EMAIL>`, `<PHONE>`, etc.

## Your task

1. **Confirm anonymization** — verify no personal data is visible in the text
2. **Map entities** — list what PII types were detected and how many occurrences
3. **Assess legal basis** — for each data type, state the legal basis under applicable privacy law
4. **Detect risks** — flag labels without a clear legal basis or sensitive data without explicit consent
5. **Detect false negatives** — warn if anything still looks like unreplaced personal data
6. **Deliver findings** — using the standard format: FINDING / ARTICLE / RISK / FIX

## Required response format

```
## ANONYMIZATION CHECK
Status: COMPLETE | INCOMPLETE
[If INCOMPLETE: list what data is still exposed]

## DETECTED ENTITY MAP
| Label    | Count | Data type   |
|----------|-------|-------------|
| <PERSON> | N     | Full name   |
...

## LEGAL BASIS ASSESSMENT
[For each label: legal basis that covers it, or RISK if none]

## FINDINGS
[Using format: FINDING / ARTICLE / RISK / FIX]

## EXECUTIVE SUMMARY
[3 lines max: overall status, main risk, priority action]
```

## Constraints

* Never infer or reconstruct the original data behind any label
* If the document appears non-anonymized → respond only: "Document contains visible personal data. Process with Presidio before continuing."
* Do not issue binding legal opinions
