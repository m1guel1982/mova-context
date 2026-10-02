# Skill: Masked PII in context
Mova may replace personal data with pseudonyms `[PII_xxxxxxxx]` (same value → same tag).
- Treat each tag as an opaque value: don't guess, reconstruct or complete the original. Same tag = same data.
- Masking is heuristic (shape + entropy): it may leave PII unmarked (false negative) or mark non-PII such as dates `2024-07-30` (false positive). It is not legal anonymization nor does it guarantee compliance with {{REGULATION}}.
- If you see unmasked PII, don't repeat it in your answer unless the query requires it, and flag it.
