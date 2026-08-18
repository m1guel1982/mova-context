// tool_registry.go — the tools/list schema (tools()) and the tiny
// tool()/req()/opt() builders it uses. Split out of server.go purely
// to keep that file under 300 lines; tools() is still the single
// source tools/list responds from — executeTool (server.go) is the
// matching dispatch table, kept in sync by hand (adding a tool means
// adding it in both places, same as before this file existed).
package mcp

func tools() []map[string]any {
	return []map[string]any{
		tool("list_projects", "List all available projects inside the Mova registry."),
		tool("get_full_context", "Full assembled context (= mova run): agents + skills + prompt + memory + focus.",
			req("project"), opt("task")),
		tool("get_knowledge", "Get a single agent, skill, or prompt.",
			req("kind"), req("domain"), opt("lang"), req("name")),
		tool("get_memory", "Active memory for a project.",
			req("project")),
		tool("get_memory_all", "Active + all archived memory.",
			req("project")),
		tool("save_memory", "Append an entry to a project's memory.md (most-recent-first) — the only memory WRITE path exposed over MCP/HTTP; get_memory/get_memory_all are read-only. Same underlying AppendMemory as `mova chat`'s /memory and the `mova memory` CLI command use, so all three stay in sync.",
			req("project"), req("entry")),
		tool("get_workflow", "Reads workflow.md for a project, but ONLY after resolving the project, building its context (agents+skills+prompt+focus+memory), and validating the result against its configured Budget (see estimate_budget) — workflow.md is never read directly. \"project\" is required so that Budget check has something to validate against; \"task\" narrows the context the same way get_full_context's task does; \"workflow\" optionally overrides project.json's configured workflow_path with an explicit file.",
			opt("project"), opt("task"), opt("workflow"), opt("lang")),
		tool("search_context", "Search across all knowledge.",
			req("query"), opt("domain")),
		tool("chat_completion", "Send a message to a local model (Ollama, LM Studio, vLLM...) configured under config/models/. Optionally attaches the full Mova context (project+task) as the system prompt. Natural language that asks to MODIFY an existing file (\"fix the bug in auth.go\", \"update report.md\") is detected automatically: the proposed change (a precise line diff) is returned but NOT written unless `apply_edits` is true — there's no interactive y/n on this door, so pass `apply_edits: true` on the same message once you've reviewed the diff to actually write it.",
			req("message"), opt("model"), opt("project"), opt("task"), opt("apply_edits")),
		tool("estimate_budget", "Estimate the token/USD cost of a project's real context (agents+skills+prompt+focus+memory — the same assembly get_full_context produces), broken down per component, using the local tiktoken-go tokenizer and config/prices.json. 100% local: no LLM call, nothing leaves this machine. `project` may also be a multiagent group (its own config.json) — sums one estimate per agent instead of failing, and writes no report file in that case (each agent has its own; pass \"<group>/<agent>\" to get one). Writes mova-budget-report.md for an ordinary project. `focus`=\"true\" also compares full-repo vs. focus-only token cost.",
			req("project"), opt("task"), opt("focus")),
		tool("generate_diagram", "Render a visual architecture diagram (sources -> Context Compiler -> Token Firewall incl. optional PII Masking -> agents/multiagent group -> jobs -> interfaces -> real token/cost metrics) for a project OR a multiagent group, built entirely from its real project.json/config.json plus a live estimate_budget-equivalent count — never simulated data. `export` is a comma-separated list of svg,png,pdf (default \"svg\"); `path` is the output directory (created if missing, defaults to the current directory); `detail` overrides project.json's own \"diagram.detail_level\" (\"simple\" or \"verbose\", default \"verbose\") for this call only.",
			req("project"), opt("task"), opt("export"), opt("path"), opt("detail")),
		tool("list_jobs", "List the scheduled jobs declared in a project's project.json \"jobs\" array (schedule, tasks, save, memory, memory_archive, delete, budget).",
			req("project")),
		tool("run_job", "Run a project's scheduled job(s) right now, bypassing its cron \"schedule\" — the same flow `mova jobs run` and the job scheduler daemon (`mova jobs start`) use. Pass `index` (the job's 0-based position in project.json's \"jobs\" array) to run just one; omit it to run every job declared for the project.",
			req("project"), opt("index")),
		tool("list_agents", "List the agents declared (or auto-discovered) for a multiagent group — a directory under projects/ with its own config.json orchestrating several agent sub-projects (see PROJECT_JSON.md § Multiagent).",
			req("group")),
		tool("run_agent", "Run one agent, several, or an entire multiagent group, sequentially — each agent is an ordinary project (projects/<group>/<agent>/project.json) run through the same assemble+Budget-gate pipeline as `mova run`. Pass `agent` to run just that one; omit it to run every agent in the group's config.json.",
			req("group"), opt("agent"), opt("task")),
		tool("save", "THE unified way to create or edit ANY file or directory — Markdown, plain text, source code, JSON/YAML, .docx, .pdf, .xlsx, .svg... the format is picked automatically from the extension in `path`, so you never need to know which generator handles which format (no generate_pdf_document/generate_word_contract/generate_excel_report/write_file to remember). Pass `directory` instead of `path` to only create a folder — missing parent directories are always created automatically either way. `content` is plain text/Markdown/HTML/CSV — the internal Writer decides how to turn it into the real format. `overwrite`/`append` control what happens if the file already exists (default: overwrite, same as before). Instead of `content`, pass `history` (a JSON array of {\"role\",\"content\"} objects, same shape chat_completion's own `history` uses) plus `mode` (\"all\" for the full conversation, \"range\" with `range`:\"N-M\" for a 1-indexed range of exchanges, or omitted for just the last one) and/or `code_only`/`text_only` (booleans) to save exactly the same current-response/range/full-conversation/code-only/text-only selections `/save` supports in chat.",
			opt("path"), opt("directory"), opt("content"), opt("overwrite"), opt("append"), opt("project"),
			opt("history"), opt("mode"), opt("range"), opt("code_only"), opt("text_only")),
		tool("delete_path", "THE unified way to delete one or more files or directories (see documents.Delete). Pass `path` for a single item or `paths` (comma- or newline-separated) for several. Without `confirm:true`, nothing is deleted — the exact confirmation text (\"Delete \\\"x\\\"? (Y/N)\", one per item) is returned instead, so the caller can show it and re-call with `confirm:true` once the person agrees. Never removes anything without an explicit confirm.",
			opt("path"), opt("paths"), opt("project"), opt("confirm")),
		tool("create_directory", "Create a directory, recursively creating any missing parent directories (like `mkdir -p`). Works cross-platform (Linux, macOS, Windows). `path` may be: empty (defaults to the project's repo), an absolute path in Unix (`/a/b`) or Windows (`C:/a/b`, `C:\\a\\b`) style, a bare name (searched for among existing directories in the repo — asks which one if more than one matches), or an explicit relative path. Kept for backward compatibility — equivalent to `save` with only `directory` set.",
			opt("path"), opt("project")),
		tool("read_document_layer", "Extract the plain-text layer from a .docx, .xlsx, or .pdf file. `filename` resolves the same way as create_directory's `path`.",
			req("filename"), opt("project")),
		tool("read_file", "Read the raw content of any text file (.txt, .md, .json, .yml/.yaml, .xml, source code...).",
			req("filename"), opt("project")),
		tool("write_file", "(legacy — prefer `save`) Create a new text file or fully overwrite an existing one (.txt, .md, .json, .yml/.yaml, .xml). Validates .json/.xml well-formedness before writing.",
			req("filename"), req("content"), opt("project")),
		tool("patch_file", "Surgically replace one exact, unique occurrence of `search` with `replace` inside an existing text file — the rest of the file is untouched.",
			req("filename"), req("search"), req("replace"), opt("project")),
		tool("generate_word_contract", "(legacy — prefer `save`) Compile strongly-structured markdown into a real .docx file.",
			req("filename"), req("markdown_content"), opt("project")),
		tool("generate_pdf_document", "(legacy — prefer `save`) Compile clean HTML/CSS layout text into a real .pdf file.",
			req("filename"), req("layout_html_css"), opt("project")),
		tool("generate_vector_graphic", "(legacy — prefer `save`) Write native SVG code to a .svg file (diagrams, architecture maps).",
			req("filename"), req("svg_code"), opt("project")),
		tool("generate_excel_report", "(legacy — prefer `save`) Compile typed tabular sheets_data (JSON) into a real .xlsx file.",
			req("filename"), req("sheets_data"), opt("project")),
		tool("trigger_diffusion_image", "Route a prompt to the local diffusion server configured at config/models/diffusion/config.json and save the resulting image.",
			req("filename"), req("prompt"), opt("aspect_ratio"), opt("project")),
	}
}

// ── tiny helpers ──────────────────────────────────────────────────────────

func tool(name, desc string, props ...map[string]any) map[string]any {
	// Forzamos un mapa plano de mapas de strings para evitar problemas con interfaces abstractas
	properties := map[string]map[string]string{}
	var required []string

	for _, p := range props {
		// !!! SI EL MAPA ES NIL, SÁLTALO PARA QUE NO SE COMA UN PANIC !!!
		if p == nil {
			continue
		}

		n := p["name"].(string)
		properties[n] = map[string]string{
			"type":        "string",
			"description": p["desc"].(string),
		}
		if p["req"].(bool) {
			required = append(required, n)
		}
	}

	schema := map[string]any{
		"type":       "object",
		"properties": properties, // Si está vacío, se va como {} que es lo que pide Zod
	}

	if len(required) > 0 {
		schema["required"] = required
	}

	return map[string]any{
		"name":        name,
		"description": desc,
		"inputSchema": schema,
	}
}

func req(name string) map[string]any {
	return map[string]any{"name": name, "desc": name, "req": true}
}
func opt(name string) map[string]any {
	return map[string]any{"name": name, "desc": name + " (optional)", "req": false}
}
