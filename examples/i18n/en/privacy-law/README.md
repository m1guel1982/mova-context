# Example 1 — Privacy Law Compliance (Adaptable Template)

> **This is the Mova Context flagship example.**
> It demonstrates how to completely decouple legal knowledge from software.

The Spanish version of this example uses Chilean Law 21.719.
This English version is designed as an adaptable template for any privacy regulation:
GDPR (EU), LGPD (Brazil), CCPA (California), PIPEDA (Canada), or any other.

---

## The problem

When privacy law changes, companies typically face:

```text
✗ Legal knowledge in lawyers' heads, not in systems
✗ Each system has its own implementation of legal rules
✗ When the law changes, N systems need updating
✗ Context rebuilt from scratch every AI session
✗ No audit trail of which legal rules drove which decisions
```

---

## With Mova Context

```text
✓ Legal knowledge lives in versioned Markdown files
✓ Same knowledge serves all systems
✓ Backend never changes
✓ Only the knowledge layer updates when law changes
✓ Works with any LLM (Claude, GPT, Gemini, Ollama)
✓ Complete audit trail
```

---

## Configuration

```json
{
  "project": "privacy-law",
  "lang": "en",
  "adapter": "file",
  "llm": "claude",

  "variables": {
    "law": "GDPR",
    "country": "European Union",
    "authority": "Data Protection Authorities (DPAs)",
    "effective_year": "2018"
  },

  "agents": { "domain": "legal", "use": ["data-lawyer"] },
  "skills": { "domain": "legal", "use": ["privacy-law-obligations"] },

  "tasks": {
    "analyze-contract": { "prompt": "analyze-contract-data" },
    "assess-compliance": { "prompt": "assess-compliance" },
    "respond-to-subject": { "prompt": "respond-data-subject-request" }
  }
}
```

---

## End-to-End Flow: Data Subject Access Request

### Step 1 — Message received

```
Customer: "I want to know what personal data you have about me."
```

### Step 2 — workflow.md selects context

```text
1. Read project.json → privacy-law
2. Detect lang: "en"
3. Detect task: "respond-to-subject"
4. Load agents/legal/i18n/en/data-lawyer.md
5. Load skills/legal/i18n/en/privacy-law-obligations.md
6. Load memory.md
```

### Step 3 — Agent loaded

```markdown
# Role
Specialist lawyer in personal data protection under GDPR (European Union).
Cite the specific GDPR article in each observation.
The supervisory authority is: Data Protection Authorities (DPAs).
```

### Step 4 — Skill loaded (relevant excerpt)

```markdown
# GDPR — Key Obligations

## Data subject rights
* Access — know what data is processed and how
* Response deadline: 1 month from request

## Controller obligations
* Respond within 1 month (extendable by 2 months for complex requests)
* Identity verification allowed if reasonable doubt
* Cannot charge for the response
```

### Step 5 — LLM response

```
Hello [Name],

You are exercising your right of access under Article 15 GDPR.

We have 1 month to respond to your request.

To process your request, we need to verify your identity. We will send a verification link to your registered email address.

Once verified, you will receive:
• Confirmation of what data we process about you
• The purposes for which we use it
• Who we share it with
• How long we keep it

This service is free of charge.

Case reference: [CASE-2026-XXXX]

Privacy Team
```

### Step 6 — Why the backend never changed

```text
BEFORE Mova Context:
  Backend → receives message → sends response

WITH Mova Context:
  Backend → receives message → CALLS LLM WITH CONTEXT → sends response

The backend added one LLM call.
The LLM holds the legal knowledge.
The backend knows nothing about the law.
```

---

## Switching between regulations

To switch from GDPR to CCPA, only `project.json` changes:

```json
"variables": {
  "law": "CCPA",
  "country": "California, USA",
  "authority": "California Attorney General",
  "effective_year": "2020"
}
```

Same agent. Same skills structure. Same prompts. Different knowledge.

---

## Files in this example

```text
agents/legal/i18n/en/data-lawyer.md
skills/legal/i18n/en/privacy-law-obligations.md
prompts/legal/i18n/en/respond-data-subject-request.md
projects/privacy-law/project.json
```

---

## Running this example

```bash
mova run privacy-law respond-to-subject > context.txt
# Paste context.txt into Claude, ChatGPT, or Gemini
mova memory privacy-law "$(pbpaste)"
```
