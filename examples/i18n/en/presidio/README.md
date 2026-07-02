# Complete case: Presidio + bge-m3 + Privacy Law Compliance

## The problem

A company receives documents containing personal data (employment contracts, privacy policies,
client forms) and needs to evaluate them against privacy regulations.

It cannot paste those documents directly into Claude or GPT:

```
Contract of John Smith, SSN 123-45-6789, email jsmith@company.com
→ real personal data travels to an external server
→ possible GDPR Art. 32 violation (security measures)
→ potential data breach under applicable national law
```

## The solution

Three-step flow before the LLM sees any data:

```
ORIGINAL DOCUMENT
       │
       ▼
┌─────────────────────┐
│  Microsoft Presidio  │  ← detects: PERSON, EMAIL, PHONE, LOCATION, etc.
│  (runs locally)      │  ← never leaves your machine
└─────────────────────┘
       │ anonymized text
       ▼
┌─────────────────────┐
│  bge-m3 (Ollama)    │  ← vectorizes anonymized text
│  local embedding     │  ← finds relevant agents/skills
└─────────────────────┘
       │ selected context
       ▼
┌─────────────────────┐
│  Mova Context        │  ← builds context: agents + skills + prompt
│  mova run            │
└─────────────────────┘
       │ context.txt
       ▼
┌─────────────────────┐
│  Claude / GPT /      │  ← receives only anonymized text
│  Llama (local)       │  ← never sees real names, emails, IDs
└─────────────────────┘
       │
       ▼
PRIVACY COMPLIANCE ANALYSIS
```

**Key point:** the LLM receives `<PERSON>`, `<EMAIL>`, `<PHONE>` — never the real data.
Presidio and bge-m3 run 100% in your infrastructure.

---

## What to install

### 1. Python 3.9+ and Presidio

```bash
pip install presidio-analyzer presidio-anonymizer

# Language model (English)
python -m spacy download en_core_web_lg

# Spanish too if needed
python -m spacy download es_core_news_lg
```

### 2. Ollama + local models

```bash
# Install Ollama
curl -fsSL https://ollama.ai/install.sh | sh

# Multilingual embedding model (EN + ES + 100 others)
ollama pull bge-m3             # 1.2 GB

# Reranker (natural pair for bge-m3)
ollama pull bge-reranker-v2-m3 # 568 MB

# LLM for generation (optional if using Claude/GPT via API)
ollama pull llama3.1           # 4.7 GB
```

### 3. Verify installation

```bash
python -c "from presidio_analyzer import AnalyzerEngine; print('Presidio OK')"
ollama list     # must show bge-m3 and bge-reranker-v2-m3
mova list       # must show privacidad-presidio
```

---

## Test document

Save as `contract-original.txt`:

```text
EMPLOYMENT CONTRACT

Between Acme Corp, represented by Jane Doe, SSN 123-45-6789,
hereinafter "the Employer", and John Smith, SSN 987-65-4321,
residing at 123 Main Street, New York, email jsmith@gmail.com,
phone +1 212 555 0100, hereinafter "the Employee".

CLAUSE 1 — PURPOSE
The Employee will provide software development services starting February 1, 2026.

CLAUSE 2 — SALARY
Monthly salary of USD 5,000.

CLAUSE 3 — PERSONAL DATA
Employee data will be stored in our systems and shared with our HR provider
located in the European Union for payroll processing.

Signed in New York, January 21, 2026.
```

---

## Step 1 — Anonymize with Presidio

Save as `anonymize.py`:

```python
from presidio_analyzer import AnalyzerEngine
from presidio_anonymizer import AnonymizerEngine
from presidio_anonymizer.entities import OperatorConfig
import sys

analyzer   = AnalyzerEngine()
anonymizer = AnonymizerEngine()

text = open(sys.argv[1]).read()

results = analyzer.analyze(
    text=text,
    language="en",
    entities=["PERSON","EMAIL_ADDRESS","PHONE_NUMBER","LOCATION","DATE_TIME","CREDIT_CARD","US_SSN"],
    score_threshold=0.6
)

operators = {
    "PERSON":        OperatorConfig("replace", {"new_value": "<PERSON>"}),
    "EMAIL_ADDRESS": OperatorConfig("replace", {"new_value": "<EMAIL>"}),
    "PHONE_NUMBER":  OperatorConfig("replace", {"new_value": "<PHONE>"}),
    "LOCATION":      OperatorConfig("replace", {"new_value": "<ADDRESS>"}),
    "DATE_TIME":     OperatorConfig("replace", {"new_value": "<DATE>"}),
    "US_SSN":        OperatorConfig("replace", {"new_value": "<SSN>"}),
}

anonymized = anonymizer.anonymize(text=text, analyzer_results=results, operators=operators)

print("=== DETECTED ENTITIES ===")
for r in sorted(results, key=lambda x: x.start):
    print(f"  {r.entity_type:20} score={r.score:.2f}  [{text[r.start:r.end]}]")

print("\n=== ANONYMIZED TEXT ===")
print(anonymized.text)

with open("contract-anonymized.txt", "w") as f:
    f.write(anonymized.text)
print("\n→ Saved to contract-anonymized.txt")
```

```bash
python anonymize.py contract-original.txt
```

**Expected output:**

```
=== DETECTED ENTITIES ===
  PERSON               score=0.85  [Jane Doe]
  US_SSN               score=0.90  [123-45-6789]
  PERSON               score=0.85  [John Smith]
  US_SSN               score=0.90  [987-65-4321]
  LOCATION             score=0.80  [123 Main Street, New York]
  EMAIL_ADDRESS        score=0.95  [jsmith@gmail.com]
  PHONE_NUMBER         score=0.85  [+1 212 555 0100]
  DATE_TIME            score=0.85  [February 1, 2026]
  DATE_TIME            score=0.85  [January 21, 2026]

=== ANONYMIZED TEXT ===
EMPLOYMENT CONTRACT

Between Acme Corp, represented by <PERSON>, SSN <SSN>,
hereinafter "the Employer", and <PERSON>, SSN <SSN>,
residing at <ADDRESS>, email <EMAIL>,
phone <PHONE>, hereinafter "the Employee".
...
```

---

## Step 2 — Generate context with Mova

```bash
mova run privacidad-presidio analizar-contrato > context.txt

# Verify all three cores loaded
grep "<!-- core:" context.txt
```

```
<!-- core: yagni-core -->
<!-- core: kiss-dry-core -->
<!-- core: ockham-core -->
```

---

## Step 3 — Send to LLM

**Option A — Claude / ChatGPT:**

```bash
cat context.txt > final-prompt.txt
echo "" >> final-prompt.txt
echo "---" >> final-prompt.txt
echo "## DOCUMENT TO ANALYZE" >> final-prompt.txt
cat contract-anonymized.txt >> final-prompt.txt
cat final-prompt.txt | pbcopy   # macOS
```

**Option B — Llama 3.1 local (100% offline):**

```bash
cat context.txt contract-anonymized.txt | ollama run llama3.1
```

---

## Expected LLM response

```
## ANONYMIZATION CHECK
Status: COMPLETE
No visible personal data detected in the text.

## DETECTED ENTITY MAP
| Label     | Count | Data type        |
|-----------|-------|------------------|
| <PERSON>  | 2     | Full name        |
| <SSN>     | 2     | Social Security  |
| <ADDRESS> | 1     | Physical address |
| <EMAIL>   | 1     | Email address    |
| <PHONE>   | 1     | Phone number     |
| <DATE>    | 2     | Date             |

## LEGAL BASIS ASSESSMENT
- <PERSON>, <SSN>, <ADDRESS>, <PHONE>: Art. 6.1.b GDPR — contract execution ✓
- <EMAIL>: Art. 6.1.b GDPR — contract execution ✓

## FINDINGS

### FINDING 1
- Description: Transfer of data to EU provider without declared safeguards
- Article: Art. 46 GDPR — Transfers subject to appropriate safeguards
- Risk: **High**
- Fix: Add clause: "The transfer is made under Standard Contractual Clauses
  approved by the European Commission, ensuring equivalent protection."

### FINDING 2
- Description: Data subject not informed of their rights
- Article: Art. 13 GDPR — Information to be provided
- Risk: **High**
- Fix: Add clause: "The Employee has the right to access, rectify, erase
  and object to processing of their data. Contact: <EMAIL_DPO>."

### FINDING 3
- Description: No data retention period stated
- Article: Art. 13.2.a GDPR
- Risk: **Medium**
- Fix: "Data will be retained for the duration of the contract and
  up to 5 years after termination, per applicable labor law."

## DPIA REQUIRED?
**Evaluate** — international transfer and automated payroll processing
may require a Data Protection Impact Assessment. Consult DPO.

## EXECUTIVE SUMMARY
Anonymization complete. 3 findings: 2 High (international transfer without
safeguards, missing rights information) and 1 Medium (retention period).
Priority action: add international transfer clause and data subject rights.
```

---

## Step 4 — Save session memory

```bash
mova memory privacidad-presidio '```memory
## 2026-01-21 — employment contract analysis
**Done:** contract-original.txt analyzed, 9 PII entities detected, 3 findings
**Resolved:** anonymization verified complete
**Pending:** fix international transfer clause and add data subject rights
**Decisions:** all new contracts must include rights clause before signing
**LLM Errors:** none
```'
```
