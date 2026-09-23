# QA Test Plan and Execution: Egress Audit & Token Budget Control

> **Note for Evaluators / Testers:** This document contains the full test suite executed to validate Mova Context's governance controls, PII sanitization , and Air-Gap isolation using an MCP client (e.g., Cline with Claude Anthropic Haiku). You can replicate these commands and scenarios in your own environment to verify test results and audit compliance.

---

## Phase 1 — With `dry_run: true` (Guardrail Mode / Full Block)

**Phase Premise:** Preventative interception and blocking of egress tools targeting external LLMs or restricted context artifacts.

### Test 1 — `chat_completion` (Critical Egress)
* **Command:** Using the `chat_completion` tool from `mova-context`, for the project `02-pii-compliance-governance`, ask it to summarize the context.
* **Criteria:** 

**PASS:** Returns `[MOVA EGRESS AUDIT]`, `dry_run=true`, real evaluated tokens, and `0 tokens sent`, including a security directive that prevents assistant fallbacks. 
**FAIL:** Exposes an actual summary or the LLM guesses content via local host tools.
* **Status:** ✅ PASSED (7,787 tokens evaluated, 0 sent; directive honored without disk fallback).

### Test 2 — `get_full_context`
* **Command:** Retrieve the full context of the project `02-pii-compliance-governance` using `get_full_context` from `mova-context`.
* **Criteria:** 
**PASS:** Returns only the `[MOVA EGRESS AUDIT]` message. 
**FAIL:** Exposes repository structure, `agents`, `skills`, or codebase files.
* **Status:** ✅ PASSED (Fully intercepted).

### Test 3 — `get_memory` (Air-Gap Memory Layer)
* **Command:** Show me the memory of the project `02-pii-compliance-governance` using `get_memory`.
* **Criteria:** 

**PASS:** Returns `[MOVA EGRESS AUDIT]`, even if `memory.md` is empty (0 tokens). 
**FAIL:** Displays original memory content or a default "empty memory" response.
* **Status:** ✅ PASSED (Audit interception consistently applied at the MCP gate).

### Test 4 — `read_file` (Data Leakage Inspection)
* **Command:** Read the file `customers.json` from the project `02-pii-compliance-governance` using `read_file`.
* **Criteria:** 

**PASS:** Total block with no sensitive PII exposed (no RUTs, emails, or names).
 **FAIL:** Leaks any JSON fragment or sensitive key.
* **Status:** ✅ PASSED (Access blocked by audit policy).

### Test 5 — Negative Control: `search_context` (Outside Air-Gap)
* **Command:** Search for "PII" in the `mova-context` knowledge base using `search_context`.
* **Criteria:**

 **PASS:** Returns standard search results (indexed snippets from `agents/skills`).
  **FAIL:** Search is intercepted by the audit guardrail.
* **Status:** ✅ PASSED (Out-of-scope operation executed normally).

### Test 6 — Evidence on Disk (`egress.md`)
* **Command:** Review the report generated on disk to ensure it logs audit evidence without exposing raw PII.
* **Criteria:** 

**PASS:** Generates `egress.md` preserving JSON structure while applying `[PII_xxxxxxxx]` masking.
 **FAIL:** Report contains unmasked personal data.
* **Status:** ✅ PASSED (PII correctly masked with entropy tokens).

---

## Phase 2 — With `dry_run: false` (Delegation / Active Inference)

**Phase Premise:** With guardrails disabled or in passive mode, execution delegates transparently to the host provider.

### Test 7 — Real Host Delegation
* **Command:** Using `chat_completion` from `mova-context` for the project `02-pii-compliance-governance`, ask what type of personal data exists in the project.
* **Criteria:** 

**PASS:** Real inference returned by the LLM (identifying fields like `rut`, `email`, `phone`). 
**FAIL:** Connection error to local models or unexpected audit block.
* **Status:** ✅ PASSED (Successfully delegated to host provider).

### Test 8 — Normal `get_full_context`
* **Command:** Retrieve the full context of the project `02-pii-compliance-governance` using `get_full_context`.
* **Criteria:** 

**PASS:** Returns the entire context (`agents` + `skills` + `prompt` + `focus`). 
**FAIL:** Partial response or blocked by audit.
* **Status:** ✅ PASSED (Full context delivered without interception).