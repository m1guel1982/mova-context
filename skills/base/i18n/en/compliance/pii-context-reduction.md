# Skill: Technical PII reduction before sending context to an LLM

This skill documents an OPTIONAL, off-by-default Mova Context capability: **PII Masking** (`mova.local/sanitize`, `pii.go`/`pii_policy.go`). It isn't a list of abstract privacy-reasoning instructions — it describes a real engine feature this project can turn on.

## What it does (scope)

Before counting tokens or sending context to any model (local or cloud), Mova can replace tokens in Focus/Memory whose **structural shape** (digit ratio, `.`/`-`/`/` separators, presence of `@`, length, uppercase ratio) and **Shannon entropy** clear a configurable threshold with a deterministic pseudonym `[PII_xxxxxxxx]` (truncated FNV-1a hash). The same original value always produces the same pseudonym — you can see "this repeats" without ever reconstructing the original value from the tag.

The algorithm is **language-agnostic**: no dictionaries, no word lists, no Spanish/English/other grammar rules. The same mathematical (entropy) and structural (word shape) signals fire the same way on a RUT, a phone number, or an email, regardless of the surrounding text's language.

## How to enable it

1. In `project.json` (or at task level), inside `"budget"`:
   ```json
   "budget": { "pii_masking": { "enabled": true } }
   ```
   Off by default — a project that never declares this field sees zero behavior change.
2. Thresholds/weights/tag format live in `config/policy.json` (`pii_masking`), never hardcoded in Go — tunable without recompiling.
3. The result appears in `mova-budget-report.md`'s "PII Masking" section (`mova budget <project>`), and the context Mova assembles via `mova run`/`mova chat`/jobs/MCP/HTTP already carries the replaced tokens when this stage masked anything.

## What it does NOT do (limits — read before using)

* **Not legal anonymization.** It's a heuristic structural mitigation — it does not, by itself, satisfy any legal anonymization standard.
* **Does not guarantee detecting 100% of personal information.** A common name with no digits or structural separators may not clear the threshold and won't be masked (false negative). For the same reason, it can also mask tokens that are NOT PII but share a similar structural shape — for example, separator-formatted dates like `2024-07-30` (false positive).
* **Does not replace legal advice, an internal privacy policy, or a Ley 21.719/GDPR/other-regulation compliance program.**
* Does not replace the rest of the Token Firewall (Sanitizer, Circuit Breaker) — it's an additional, optional stage, not a substitute.

## When to use it in this domain (compliance)

Useful when a Mova Context project builds context from customer data (CRM, ERP, legacy systems) and that context is going to be sent to an external LLM — it technically reduces how much identifiable information travels outside the local environment, without blocking the workflow. Always combine with: preferring a local model when possible, reviewing which fields are actually necessary for the query (see the `ai-privacy-reviewer` agent), and never relying solely on this stage for regulatory compliance.
