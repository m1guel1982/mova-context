# Validation Guide — Presidio + Privacy Law Compliance

Checklist to verify the complete case works correctly.

---

## PART 1 — Installation

### ✅ Presidio

```bash
python -c "
from presidio_analyzer import AnalyzerEngine
from presidio_anonymizer import AnonymizerEngine
print('Presidio OK')
"
```

**Expected:** `Presidio OK`
**Common error:** `ModuleNotFoundError` → `pip install presidio-analyzer presidio-anonymizer`

```bash
python -m spacy info en_core_web_lg | grep "Name"
```

**Expected:** `Name: en_core_web_lg`
**Common error:** model not found → `python -m spacy download en_core_web_lg`

---

### ✅ Ollama

```bash
ollama list | grep -E "bge-m3|bge-reranker"
```

**Expected:**
```
bge-m3                   1.2 GB
bge-reranker-v2-m3       568 MB
```

---

### ✅ Mova Context

```bash
mova list | grep privacidad-presidio
```

**Expected:**
```
  privacidad-presidio    [es] Análisis de documentos con PII anonimizada...
    tasks: analizar-contrato, analizar-terminos, evaluar-formulario, evaluar-politica
```

---

## PART 2 — Presidio anonymization

### ✅ Basic detection test

```bash
python -c "
from presidio_analyzer import AnalyzerEngine
analyzer = AnalyzerEngine()
text = 'Contact Jane Doe at jane.doe@company.com, phone +1 555 123 4567'
results = analyzer.analyze(text=text, language='en', score_threshold=0.6)
for r in results:
    print(f'{r.entity_type}: {text[r.start:r.end]} (score={r.score:.2f})')
"
```

**Expected:**
```
PERSON: Jane Doe (score=0.85)
EMAIL_ADDRESS: jane.doe@company.com (score=0.95)
PHONE_NUMBER: +1 555 123 4567 (score=0.85)
```

---

### ✅ Post-anonymization check

```bash
# After running anonymize.py, verify no real data remains
grep -E "[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}" contract-anonymized.txt \
  && echo "⚠ Email not anonymized" || echo "✓ No visible emails"
```

---

## PART 3 — Mova context

### ✅ Context generation

```bash
mova run privacidad-presidio analizar-contrato | grep "<!-- "
```

**Expected:**
```
<!-- core: yagni-core -->
<!-- agent: analista-presidio -->
<!-- agent: abogado-privacidad -->
<!-- core: kiss-dry-core -->
<!-- skill: deteccion-pii -->
<!-- skill: cumplimiento-ley-21719 -->
<!-- core: ockham-core -->
<!-- prompt: analizar-documento-anonimizado -->
```

### ✅ Each core loaded exactly once

```bash
mova run privacidad-presidio analizar-contrato | grep "<!-- core:" | sort | uniq -c
```

**Expected:**
```
      1 <!-- core: kiss-dry-core -->
      1 <!-- core: ockham-core -->
      1 <!-- core: yagni-core -->
```

---

## PART 4 — Full pipeline

```bash
# 1. Anonymize
python anonymize.py contract-original.txt

# 2. Verify clean
grep -E "[a-zA-Z0-9._%+\-]+@" contract-anonymized.txt \
  && echo "⚠ Email still present" || echo "✓ No emails visible"

# 3. Build Mova context
mova run privacidad-presidio analizar-contrato > context.txt

# 4. Combine
cat context.txt contract-anonymized.txt > final-prompt.txt

# 5. Check size
wc -w final-prompt.txt
```

---

## Checklist summary

| Step | Command | ✓ |
|------|---------|---|
| Presidio installed | `python -c "from presidio_analyzer..."` | □ |
| spaCy model EN | `python -m spacy info en_core_web_lg` | □ |
| bge-m3 in Ollama | `ollama list \| grep bge-m3` | □ |
| project visible | `mova list \| grep privacidad` | □ |
| cores loaded (×1) | `mova run ... \| grep core: \| uniq -c` | □ |
| PERSON detected | python test → `PERSON: Jane Doe` | □ |
| email detected | python test → `EMAIL_ADDRESS: ...` | □ |
| no real data post-anon | `grep -E "..."` → `✓ No emails` | □ |
| final-prompt.txt built | `wc -w final-prompt.txt` | □ |
| memory saved | `mova memory-read privacidad-presidio` | □ |

**All ✓ → the case is working correctly.**
