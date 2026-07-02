KISS + DRY: see `kiss-dry-core.md`

# PII Detection with Microsoft Presidio

## Entities detected by Presidio

| Presidio Entity  | Real data          | Sensitive |
|------------------|--------------------|-----------|
| `PERSON`         | Full name          | No        |
| `EMAIL_ADDRESS`  | Email              | No        |
| `PHONE_NUMBER`   | Phone              | No        |
| `LOCATION`       | Address, city      | No        |
| `DATE_TIME`      | Birth date, medical dates | Depends |
| `CREDIT_CARD`    | Card number        | No (financial) |
| `IBAN_CODE`      | Bank account       | No (financial) |
| `MEDICAL_LICENSE`| Medical licence no | No        |
| `HEALTH_DATA`    | Diagnoses, meds *(custom)* | **Yes** |

## Confidence score

Presidio assigns a score 0.0–1.0 to each detection:

```
score ≥ 0.85   → high confidence, always anonymize
score 0.6–0.85 → review manually before processing
score < 0.6    → likely false positive, discard
```

## Common false negatives (Presidio may miss these)

```
- Names in ALL CAPS: JOHN SMITH
- Usernames or handles inside text
- Internal file/ticket numbers that identify a person
- Addresses described narratively: "lives next to Central Park"
```

## Recommended anonymization operators

```python
operators = {
    "PERSON":        OperatorConfig("replace", {"new_value": "<PERSON>"}),
    "EMAIL_ADDRESS": OperatorConfig("replace", {"new_value": "<EMAIL>"}),
    "PHONE_NUMBER":  OperatorConfig("replace", {"new_value": "<PHONE>"}),
    "LOCATION":      OperatorConfig("replace", {"new_value": "<ADDRESS>"}),
    "DATE_TIME":     OperatorConfig("replace", {"new_value": "<DATE>"}),
    "CREDIT_CARD":   OperatorConfig("replace", {"new_value": "<CARD>"}),
}
```

## Post-anonymization verification rule

Before sending to the LLM, verify the anonymized text does not contain:
- 16-digit sequences (card number)
- Patterns `@domain.com`
- Obvious proper names (minimal common-name list)
