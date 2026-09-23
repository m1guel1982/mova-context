# MCP and HTTP tools — copy-paste reference

Every MCP tool is called **the same way** over stdio or HTTP — same `name`, same `arguments`. One template:

```json
{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"<tool>","arguments":{...}}}
```

**Over MCP (stdio):**
```bash
echo '<JSON above>' | mova mcp start --stdio
```

**Over HTTP:**
```bash
curl -X POST http://localhost:3000/mcp -H "Content-Type: application/json" -d '<JSON above>'
```

From here on, each row shows just `arguments` — drop it into the template above, changing `name`.

**🔒 = honors `egress_audit.dry_run`** (see `PROJECT_JSON.md § Real air-gap`) — with `dry_run: true` in that project's `project.json`, these 7 tools never return the real content, only the audit message. `search_context` has no `project` and searches Mova's shared catalog, not private data — by design, it isn't gated.

## Context and governance (Mova's core)

| Tool | What it does | Example `arguments` |
|---|---|---|
| `context_trace` | Pre-inference audit — tokens, cost, policies, decision. See `GOVERNANCE_CONTROLS.md`. | `{"project": "02-pii-compliance-governance"}` |
| `get_full_context` 🔒 | Full assembled context (= `mova run`). Honors `egress_audit.dry_run`. | `{"project": "02-pii-compliance-governance"}` |
| `chat_completion` 🔒 | Sends a message to a local model, with the context as system prompt. Honors `egress_audit.dry_run` and delegates to the host LLM when no `llm_profile` is set. | `{"project": "02-pii-compliance-governance", "message": "summarize the context"}` |
| `estimate_budget` | Estimates token/USD cost without calling any model — writes `mova-budget-report.md`. | `{"project": "02-pii-compliance-governance"}` |
| `generate_diagram` | Visual diagram of the real pipeline (SVG/PNG/PDF). | `{"project": "02-pii-compliance-governance", "export": "png"}` |

## Projects and knowledge

| Tool | What it does | Example `arguments` |
|---|---|---|
| `list_projects` | Lists registered projects. | `{}` |
| `get_knowledge` | One specific agent/skill/prompt. | `{"kind": "agent", "domain": "base", "name": "backend-dev"}` |
| `search_context` | Searches all knowledge. | `{"query": "PII"}` |
| `get_workflow` 🔒 | Reads `workflow.md` after validating budget. | `{"project": "02-pii-compliance-governance"}` |

## Memory

| Tool | What it does | Example `arguments` |
|---|---|---|
| `get_memory` 🔒 | A project's active memory. | `{"project": "02-pii-compliance-governance"}` |
| `get_memory_all` 🔒 | Active + archived memory. | `{"project": "02-pii-compliance-governance"}` |
| `save_memory` | Appends an entry. | `{"project": "02-pii-compliance-governance", "entry": "decision: use Law 21.719 as reference"}` |

## Multi-agent

| Tool | What it does | Example `arguments` |
|---|---|---|
| `list_agents` | Lists a group's agents. | `{"group": "my-group"}` |
| `run_agent` | Runs one or all agents in the group. | `{"group": "my-group"}` |

## Files (unified — use `save` for anything new)

| Tool | What it does | Example `arguments` |
|---|---|---|
| `save` | Creates/edits any file (.md, .docx, .pdf, .xlsx, .svg, code) — format picked from the extension. | `{"path": "notes.md", "content": "# Hello"}` |
| `delete_path` | Deletes (needs `confirm: true`). | `{"path": "notes.md", "confirm": true}` |
| `create_directory` | Creates a directory. | `{"path": "new-folder"}` |
| `read_document_layer` 🔒 | Extracts text from .docx/.xlsx/.pdf. | `{"project": "02-pii-compliance-governance", "filename": "report.pdf"}` |
| `read_file` 🔒 | Reads a text file. | `{"project": "02-pii-compliance-governance", "filename": "notes.md"}` |
| `patch_file` | Replaces one exact fragment. | `{"filename": "notes.md", "search": "Hello", "replace": "Hello world"}` |
| `write_file` *(legacy, use `save`)* | Creates/overwrites plain text. | `{"filename": "notes.txt", "content": "hi"}` |
| `generate_word_contract` *(legacy, use `save`)* | Markdown → .docx. | `{"filename": "contract.docx", "markdown_content": "# Contract"}` |
| `generate_pdf_document` *(legacy, use `save`)* | HTML/CSS → .pdf. | `{"filename": "report.pdf", "layout_html_css": "<h1>Report</h1>"}` |
| `generate_vector_graphic` *(legacy, use `save`)* | SVG → .svg. | `{"filename": "diagram.svg", "svg_code": "<svg></svg>"}` |
| `generate_excel_report` *(legacy, use `save`)* | Tabular JSON → .xlsx. | `{"filename": "data.xlsx", "sheets_data": {"Sheet1": [["A","B"]]}}` |
| `trigger_diffusion_image` | Generates an image (if a provider is configured). | `{"prompt": "minimalist architecture diagram"}` |

## HTTP — direct endpoints (shortcuts, no `tools/call` wrapper)

| Endpoint | Method | Shortcut for |
|---|---|---|
| `/mcp` | `POST` | Generic — any tool above |
| `/save` | `POST` | `save` |
| `/delete` | `POST` | `delete_path` |
| `/workflow` | `POST` | `get_workflow` |
| `/agents/run` | `POST` | `run_agent` |
| `/diagram` | `POST` | `generate_diagram` |
| `/api/v1/context-trace` | `POST` | `context_trace` |
| `/health` | `GET` | Health check (no body) |

```bash
curl -X POST http://localhost:3000/save -H "Content-Type: application/json" \
  -d '{"path":"notes.md","content":"# Hello"}'
```

Step-by-step guide to test all of this free, no GPU: `MCP_HTTP_TESTING.md`.
