# 02 — PII and Secrets Governance Before Inference

**What it is:** real customer data (fictionalized) will be queried by an LLM.

Mova detects PII indicators, applies technical masking, and leaves evidence — before anything is sent.

**Illustrative scenario:** Chilean Law 21.719. This is pre-inference evidence and control,

**not** a regulatory compliance certification.

## Run (1 command)

```bash
mova run 02-pii-compliance-governance --diagram --export png --path ./diagrams/evidence.png
```

## What you'll see

| Step             | Result                                                                                          |
| ---------------- | ----------------------------------------------------------------------------------------------- |
| Context selected | `customers.json`, `customer-profile.pdf`, `privacy-policy.docx`, `system-logs.txt`              |
| Control applied  | Sanitizer ON · **PII Masking ON** (`budget.pii_masking.enabled: true`)                          |
| Decision         | ~78 of 1,694 candidate tokens pseudonymized with `[PII_xxxxxxxx]`; 64% total reduction          |
| Evidence         | `evidence.png` (diagram) + `context-report.md` + `mova-budget-report.md` + `pii-audit-log.json` |

`evidence-example.png` in this folder is an already-generated sample.

## Where each piece of evidence lives (important — not redundant)

* **`mova-budget-report.md`** (in `projects/02-pii-compliance-governance/`, generated with `mova budget`) — the REAL PII Masking result for this run: how many tokens were pseudonymized and why.

* **`pii-audit-log.json`** — its `security` section is zero here because that level of detail (files containing PII, classified findings) is only populated in *discovery* mode (scanning a remote repo without `project.json`), not in a run with `project.json` like this one. `execution.policy_author` / `agent_client` / `target_model` are always fully recorded.

## View the trace as text (without a diagram)

```bash
mova context-trace 02-pii-compliance-governance --export md
```

**Technical disclaimer:** masking is heuristic (structural patterns + entropy), not a name dictionary.

It does not detect 100% of PII and may produce false positives. It does not replace legal review.

`project/project.json` in this folder is a read-only copy — the actual file lives at `projects/02-pii-compliance-governance/project.json`.
