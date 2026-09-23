You can run mova context-trace directly on any public GitHub repository or local path. Running the command below will clone the target repository to a temporary directory, apply exclusion and ignore filters, prune unnecessary docstrings to optimize token budget, and generate the formatted output in output-mova trace-fastApi/


mova context-trace \
  --repo https://github.com/fastapi/fastapi \
  --export pdf \
  --task "solve_dependencies get_dependant in fastapi/dependencies/utils.py" \
  --prune-docstrings \
  --ignore "docs/**, tests/**, *.lock, docs_src/**, .github/**, docs/en/**"


This is the console output, along with the automatically generated reports and diagram in the same directory:



# Mova Context Trace — Remote Analysis

```text
[  0%] Preparing target repository...
[ 15%] Target ready for analysis: C:\Users\mandr\AppData\Local\Temp\mova-trace-2147425420
[ 85%] Temporary directory removed: C:\Users\mandr\AppData\Local\Temp\mova-trace-2147425420
[ 90%] Generating reports...
[100%] Done.
```

## Mova Context Trace (Remote Analysis)

### Input

| Field             | Value                                                               |
| ----------------- | ------------------------------------------------------------------- |
| **Repository**    | `https://github.com/fastapi/fastapi`                                |
| **Branch**        | `master`                                                            |
| **Task**          | `solve_dependencies get_dependant in fastapi/dependencies/utils.py` |
| **Agent**         | `mova-cli`                                                          |
| **Target model**  | `n/a`                                                               |
| **Policy author** | `system:default`                                                    |

**Active `--ignore` patterns:**

```text
docs/**, tests/**, *.lock, docs_src/**, .github/**, docs/en/**
```

---

### Focus

```text
Included   3139 file(s)  (repository scope considered, before content-level discovery)
Excluded     29 file(s)  (suggestion: .git)
```

> **Note:** 3011 of the 3139 in-scope file(s) could not be read as
> text/image during discovery (binary/unreadable/empty) and are reported
> as EXCLUDED below.
>
> **Focus is the SCOPE considered. Discovery is what was actually analyzed.**

---

### Context Governance

**Audit mode — no active `project.json`**

| Control            | Status                                             |
| ------------------ | -------------------------------------------------- |
| Sanitizer          | **DISABLED** — Audit Mode                          |
| PII Masking        | **ON** — 14 file(s) with sensitive-data indicators |
| Cache Layout Guard | **OFF**                                            |
| Circuit Breaker    | **OFF**                                            |

See `GOVERNANCE` below for per-file `SANITIZED` / `BLOCKED` decisions.

---

### Governance

> **STATUS: DISCOVERY ONLY (Default Global Policy)**

| Status             | Files |  Tokens |
| ------------------ | ----: | ------: |
| **DISCOVERED**     |   128 | 207,095 |
| **CANDIDATE**      |    14 | 117,187 |
| **ALLOWED**        |     0 |       0 |
| **WOULD SANITIZE** |    14 | 117,187 |
| **BLOCKED**        |     0 |       0 |
| **EXCLUDED**       | 3,125 |  89,908 |

#### Sendable Context Breakdown

*Audit mode — nothing written to disk.*

| Scenario                      |  Tokens |
| ----------------------------- | ------: |
| Repository context (scanned)  | 207,095 |
| Policy allowed (safe now)     |       0 |
| Would require sanitization    | 117,187 |
| Actually transformed/redacted |       0 |

**Safe to send right now:** `0 tok` — **100.0% reduction**

**Sendable after sanitization is applied:** `117,187 tok` — **43.4% reduction**

**Files with an indicator:** `14` = `would_sanitize 14` + `blocked 0`

#### Estimated Local Cost

| Scenario                  |                      Cost |
| ------------------------- | ------------------------: |
| Safe now                  | No cost — local execution |
| After sanitization        | No cost — local execution |
| Ollama / LM Studio / vLLM |    `$0` in both scenarios |

---

### Budget

```text
N/A (no active project.json)

Total tokens: 117,187
Encoding:     cl100k_base
```

---

### Top Directories by Token Usage

| Directory |      Tokens |    Files |    % Total |
| --------- | ----------: | -------: | ---------: |
| `fastapi` |     104,178 |       11 |      88.9% |
| `scripts` |      13,009 |        3 |      11.1% |
| **TOTAL** | **117,187** | **3139** | **100.0%** |

---

### Projected Cost

> Estimate if the `117,187` candidate tokens were sent.
>
> **Audit mode:** nothing was sent.
> **This is not the cost of the whole repository.**

| Provider / Model          | Estimated Cost |
| ------------------------- | -------------: |
| Anthropic — Claude Haiku  |    `$0.09 USD` |
| Anthropic — Claude Sonnet |    `$0.35 USD` |
| Google — Gemini Flash     |  `$0.0088 USD` |
| Google — Gemini Pro       |    `$0.15 USD` |
| OpenAI — GPT-4.1          |    `$0.23 USD` |
| OpenAI — GPT-4o           |    `$0.29 USD` |
| OpenAI — GPT-4o-mini      |    `$0.02 USD` |
| Local — LM Studio         |        No cost |
| Local — Ollama            |        No cost |
| Local — vLLM              |        No cost |

> All figures above are technical **estimates** based on public price lists
> and `cl100k_base`. They are not absolute provider costs.

---

### Relevance

**Task:**

```text
solve_dependencies get_dependant in fastapi/dependencies/utils.py
```

**Ranking method:** BM25 + structural + AST signal.

| Rank | File                                      |     Score | Role     |
| ---: | ----------------------------------------- | --------: | -------- |
|    1 | `fastapi/dependencies/utils.py`           | `1018.95` | producer |
|    2 | `fastapi/_compat/v2.py`                   |    `3.00` | producer |
|    3 | `fastapi/applications.py`                 |    `3.00` | producer |
|    4 | `fastapi/encoders.py`                     |    `3.00` | producer |
|    5 | `fastapi/openapi/utils.py`                |    `3.00` | producer |
|    6 | `fastapi/security/http.py`                |    `3.00` | producer |
|    7 | `fastapi/security/oauth2.py`              |    `3.00` | producer |
|    8 | `fastapi/security/open_id_connect_url.py` |    `3.00` | producer |
|    9 | `fastapi/utils.py`                        |    `3.00` | producer |
|   10 | `scripts/docs.py`                         |    `3.00` | producer |

---

### Output

```text
context-report.pdf
context-diagram.png
pii-audit-log.json
```

---

### Execution

| Field                | Value                               |
| -------------------- | ----------------------------------- |
| **Execution ID**     | `18d59b76aade82d0-1b096cefb5a060ff` |
| **Repository state** | `50113da`                           |

> All artifacts were generated from this execution result.

---

### Next step

```text
Generate a 'project.json' file with the detected suggestions? [Y/n]:
```
