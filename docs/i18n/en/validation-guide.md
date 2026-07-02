# Validation Guide — Mova Context

Complete checklist to verify everything works correctly.

---

## 1. Basic CLI

### `mova list`

```bash
mova list
```

**Expected result:**

```text
  acme-demo              [es] API interna multi-tenant de ejemplo
    tasks: audit-auth, audit-payments, new-module, review-complete
  ley-21719              [es] Ley 21.719 — ...
    tasks: analizar-contrato, evaluar-cumplimiento, redactar-politica, responder-titular
  omnicanal-demo         [es] Acme Services — Omnicanal Empresarial
    tasks: cobranza-tardia, cobranza-temprana, ivr-menu, ventas-webchat, ventas-whatsapp
  pruebas-locales        [es] Minimal official example — ...
    tasks: agregar-modulo, auditar-modulo, configurar-cicd, crear-proyecto, generar-tests
```

**Validation:** all 4 projects appear.  
**Common error:** running from outside `mova-context` directory (`workflow.md` not found).

---

## 2. Context generation

### `mova run`

```bash
mova run pruebas-locales crear-proyecto
```

**Expected result:** context with AGENTS, SKILLS, PROMPT, INSTRUCTION sections.

```bash
# Verify all three cores are present
mova run pruebas-locales crear-proyecto | grep "<!-- core:"
```

**Expected output:**

```text
<!-- core: yagni-core -->
<!-- core: kiss-dry-core -->
<!-- core: ockham-core -->
```

**Common error:** empty context → verify `project.json` uses correct schema (`agents.domain`, `agents.use`).

---

## 3. All projects

```bash
mova run ley-21719 analizar-contrato | head -20
mova run omnicanal-demo ivr-menu | head -20
mova run omnicanal-demo ventas-whatsapp | head -20
mova run acme-demo review-complete | grep "<!-- agent:"
mova run pruebas-locales crear-proyecto | grep "<!-- "
```

All must run without errors.

---

## 4. Core files

```bash
# Each core loaded exactly once
mova run pruebas-locales crear-proyecto | grep "<!-- core:" | sort | uniq -c
# Expected: 1 yagni-core, 1 kiss-dry-core, 1 ockham-core
```

---

## 5. Memory

```bash
# Save memory
mova memory pruebas-locales '```memory
## 2026-01-21 — test
**Done:** validation OK
**Pending:** nothing
```'

# Read memory
mova memory-read pruebas-locales

# Verify it appears in the next run
mova run pruebas-locales crear-proyecto | grep -A5 "## MEMORY"
```

---

## 6. MCP

```bash
# Terminal 1: start server
mova mcp start --port 3000

# Terminal 2: test
curl -s -X POST http://localhost:3000/mcp \
  -H "Content-Type: application/json" \
  -d '{"tool":"list_projects","arguments":{}}' | jq .

curl -s -X POST http://localhost:3000/mcp \
  -H "Content-Type: application/json" \
  -d '{"tool":"run_context","arguments":{"project":"pruebas-locales","task":"crear-proyecto"}}' \
  | jq .content | head -5
```

---

## 7. Ollama

```bash
# Set llm_profile in project.json:
# "type": "local", "provider": "ollama", "model": "llama3.1"

mova run pruebas-locales crear-proyecto | grep "Profile:"
# Expected: Profile: local/ollama:llama3.1
```

---

## 8. PostgreSQL

```bash
MOVA_ADAPTER=db MOVA_DSN="postgres://user:pass@localhost/mova_db" \
  mova run pruebas-locales crear-proyecto
```

**Expected with no DB:** `db connect failed ... falling back to file`  
**Expected with active DB:** full context from PostgreSQL.

---

## 9. Claude / GPT

```bash
mova run pruebas-locales crear-proyecto > context.txt
```

Paste `context.txt` as the first message in Claude or ChatGPT.  
The model should respond by generating code according to the loaded agents and skills.

---

## 10. Context Compiler (`mova compile`)

```bash
mova compile compiler-demo revisar-ordenes
cat projects/compiler-demo/contexto.txt
```

**Expected result:** compact file (no `## AGENTS` headers) with `FOCUS:CreateOrder()` showing only that function, `FOCUS:manual.md` with the full manual, and `FOCUS:Artículo 6` with just that section.

```bash
# Verify it never sends the whole project when focus is set
grep -c "FOCUS:" projects/compiler-demo/contexto.txt
# expected: 3 (one per focus item)
```

**Common error:** `task not found` — check `default_task` in `projects/compiler-demo/project.json`.
**Common error:** symbol not found — matching is literal; check exact casing/name in the source file.

See [context-compiler.md](context-compiler.md) for the full Phase 1 / Phase 2 breakdown.

---

## Checklist summary

| Item                         | Command                              | ✓ |
|------------------------------|--------------------------------------|---|
| list shows 4 projects        | `mova list`                          | □ |
| run generates context        | `mova run pruebas-locales crear-proyecto` | □ |
| yagni-core present           | `grep "<!-- core: yagni"`            | □ |
| kiss-dry-core present        | `grep "<!-- core: kiss"`             | □ |
| ockham-core present          | `grep "<!-- core: ockham"`           | □ |
| omnicanal no errors          | `mova run omnicanal-demo ivr-menu`   | □ |
| memory works                 | `mova memory-read pruebas-locales`   | □ |
| MCP responds                 | curl to port 3000                    | □ |
| Ollama profile applied       | `grep "Profile: local"`              | □ |
| compile generates contexto.txt | `mova compile compiler-demo`       | □ |
| focus prunes without full dump | `grep -c "FOCUS:" contexto.txt`    | □ |
