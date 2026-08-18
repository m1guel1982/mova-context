# Analyze customer context for AI — {{PROJECT}}
Project: `{{PROJECT}}` · Reference regulation: `{{REGULATION}}`
Ockham: see `../engineering/ockham-core.md`.

# Original business query
> {{QUERY}}

# Objective
Over the given FOCUS (customer data from JSON/PDF/DOCX, simulating a CRM/ERP/legacy system), answer the original query from the role assigned to you (see your agent: Data Analyst, Purpose Analyst, or AI Privacy Reviewer), using exclusively that agent's defined response format.

# Steps
1. Read the entire FOCUS — never assume data that doesn't literally appear in it
2. Apply your role's response format to every relevant finding
3. Always cite the source (file/section) of each finding
4. If your role is AI Privacy Reviewer, explicitly close with what information would NOT be necessary to send to an external LLM to answer the original query
5. Don't claim this analysis alone guarantees {{REGULATION}} compliance — it's technical assistance, not legal advice

# Where this analysis's report ends up (important)
This project has its own `budget_path`/`token_history_path` configured — running `mova budget {{PROJECT}}` (or the project's scheduled job) automatically generates:
- `mova-budget-report.md` inside this agent's folder, with the token/Sanitizer/PII Masking (if enabled)/Circuit Breaker detail.
- An additional narrative report at `reports/analisis-ley21719_{date}.md`, via this project's scheduled job (see `jobs` in `project.json`).

These reports are generated the same way regardless of the channel used (CLI `mova run`/`mova budget`, `mova chat`, `mova jobs run`, HTTP/API, or MCP) — it's the same engine behind all five.

# Response format
Use exactly the format defined in your agent — don't invent a new one here.
