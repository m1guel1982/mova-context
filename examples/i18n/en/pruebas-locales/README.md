# Minimal Official Example — pruebas-locales

Reference project to validate Mova Context end to end.

## What it includes

- Tasks API (Node.js + Express + SQLite)
- 5 tasks: `crear-proyecto`, `agregar-modulo`, `auditar-modulo`, `configurar-cicd`, `generar-tests`
- Agents: `backend-dev`, `security-architect`, `devops-engineer`, `qa-engineer`
- Skills: `lazy-minimalism`, `generate-tests`
- Strict YAGNI constraints: no Docker, no `.env`, no unnecessary CI/CD

## Running

```bash
# Generate context
mova run pruebas-locales crear-proyecto > context.txt

# Paste context.txt into Claude/GPT and execute.
# After the session, save memory:
mova memory pruebas-locales "```memory
## 2026-01-21 — session
**Done:** server.js with POST /tasks, GET /tasks, PATCH /tasks/:id
**Pending:** categories module
```"

# Next task
mova run pruebas-locales agregar-modulo > context2.txt
```

## Quick validation

```bash
# All three cores must appear in the context
mova run pruebas-locales crear-proyecto | grep "<!-- core:"
```

Expected output:

```text
<!-- core: yagni-core -->
<!-- core: kiss-dry-core -->
<!-- core: ockham-core -->
```

## Expected output structure

```text
.
├── server.js
├── package.json
└── database.js
```

No empty folders. No Dockerfile. No unrequested configuration files.
