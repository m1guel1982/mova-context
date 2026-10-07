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
		tool("get_full_context", "Governed context of a task (= mova run): dependency-closure check, sanitization, budget gate; writes an immutable run under projects/<project>/runs/<run_id>/. The host decides what to do with it afterwards — Mova does not see later turns.",
			req("project"), opt("task")),
		tool("get_knowledge", "Get a single agent, skill, or prompt.",
			req("kind"), req("domain"), opt("lang"), req("name")),
		tool("get_memory", "Active memory for a project.",
			req("project")),
		tool("get_memory_all", "Active + all archived memory.",
			req("project")),
		tool("save_memory", "Register a synthesis (a ```memory block) in a project's memory.md (most-recent-first), tagged with its task and de-duplicated. It is how a host LLM records its own result when Mova does not call the model itself (no llm_profile). Honors project.json's \"memory\" field: false/absent = nothing is saved; true = memory.md next to project.json; a path = that location. `task` is optional (a **Tarea:** line in the entry also works). Same registrar as chat_completion and `mova chat`, so all three doors stay in sync.",
			req("project"), req("entry"), opt("task")),
		tool("get_workflow", "Reads workflow.md for a project, but ONLY after resolving the project, building its context (agents+skills+prompt+focus+memory), and validating the result against its configured Budget (see estimate_budget) — workflow.md is never read directly. \"project\" is required so that Budget check has something to validate against; \"task\" narrows the context the same way get_full_context's task does; \"workflow\" optionally overrides project.json's configured workflow_path with an explicit file.",
			opt("project"), opt("task"), opt("workflow"), opt("lang")),
		tool("search_context", "Search across all knowledge.",
			req("query"), opt("domain")),
		tool("chat_completion", "Send a message to a local model (Ollama, LM Studio, vLLM...) configured under config/models/. Optionally attaches the full Mova context (project+task) as the system prompt. Natural language that asks to MODIFY an existing file (\"fix the bug in auth.go\", \"update report.md\") is detected automatically: the proposed change (a precise line diff) is returned but NOT written unless `apply_edits` is true — there's no interactive y/n on this door, so pass `apply_edits: true` on the same message once you've reviewed the diff to actually write it. Every call writes an immutable run under projects/<project>/runs/<run_id>/ (context.txt + manifest.json + events.jsonl). With \"egress_audit\": {\"dry_run\": true} the provider is never called. Tool results in this loop go through the same read policy and sanitization.",
			req("message"), opt("model"), opt("project"), opt("task"), opt("apply_edits"), opt("apply_changes")),
		tool("estimate_budget", "Estimate the token/USD cost of a project's real context (agents+skills+prompt+focus+memory — the same assembly get_full_context produces), broken down per component, using the local tiktoken-go tokenizer and config/prices.json. 100% local: no LLM call, nothing leaves this machine. `project` may also be a multiagent group (its own config.json) — sums one estimate per agent instead of failing, and writes no report file in that case (each agent has its own; pass \"<group>/<agent>\" to get one). Writes mova-budget-report.md for an ordinary project. `focus`=\"true\" also compares full-repo vs. focus-only token cost.",
			req("project"), opt("task"), opt("focus")),
		tool("generate_diagram", "Render a visual architecture diagram (sources -> Context Compiler -> Context Governance incl. optional PII Masking -> agents/multiagent group -> interfaces -> real token/cost metrics) for a project OR a multiagent group, built entirely from its real project.json/config.json plus a live estimate_budget-equivalent count — never simulated data. `export` is a comma-separated list of svg,png,pdf (default \"svg\"); `path` is the output directory (created if missing, defaults to the current directory); `detail` overrides project.json's own \"diagram.detail_level\" (\"simple\" or \"verbose\", default \"verbose\") for this call only.",
			req("project"), opt("task"), opt("export"), opt("path"), opt("detail")),
		tool("context_trace", "Analyzes context composition, token budget, Context Governance status, and estimated cost (in USD) BEFORE calling an LLM — for a local project (\"project\", optionally \"task\") or a remote repository/local directory with no project.json (\"repo\": a GitHub/GitLab/generic git URL, or a local path). `export` is \"pdf\" or \"md\" (default \"md\"); `output` is the output directory (supports cross-platform paths: \"path://C:/...\", \"/mnt/...\", relative). Always writes EXACTLY two files: context-report.<export> and context-diagram.png (a governance pipeline flowchart with per-stage file/token counts for a discovery run, or a composition chart for a project.json run). In discovery mode (no project.json) the report also includes the full Context Governance & Traceability Engine breakdown: DISCOVERED/CANDIDATE/ALLOWED/SANITIZED/BLOCKED/EXCLUDED token counts, an auditable Security Findings table (detected patterns, action taken, rule that decided it), the active policy cascade (config/policy.json → config/policy/*.json), and a model-compatibility table. It also suggests a project.json (agents/skills/prompt left blank for manual editing, written to the projects/ directory Mova Context actually uses) — pass `generate_project_json`:\"true\" to write it, since this door is not interactive.",
			opt("project"), opt("task"), opt("repo"), opt("branch"), opt("export"), opt("output"), opt("generate_project_json")),
		tool("list_agents", "List the agents declared (or auto-discovered) for a multiagent group — a directory under projects/ with its own config.json orchestrating several agent sub-projects (see PROJECT_JSON.md § Multiagent).",
			req("group")),
		tool("run_agent", "Run one agent, several, or an entire multiagent group, sequentially — each agent is an ordinary project (projects/<group>/<agent>/project.json) run through the same assemble+Budget-gate pipeline as `mova run`. Pass `agent` to run just that one; omit it to run every agent in the group's config.json.",
			req("group"), opt("agent"), opt("task")),
		tool("save", "THE unified way to create or edit ANY file or directory — Markdown, plain text, source code, JSON/YAML, .docx, .pdf, .xlsx, .svg... the format is picked automatically from the extension in `path`, . Paths must resolve inside the project's repo. Pass `directory` instead of `path` to only create a folder — missing parent directories are always created automatically either way. `content` is plain text/Markdown/HTML/CSV — the internal Writer decides how to turn it into the real format. `overwrite`/`append` control what happens if the file already exists (default: overwrite, same as before). Instead of `content`, pass `history` (a JSON array of {\"role\",\"content\"} objects, same shape chat_completion's own `history` uses) plus `mode` (\"all\" for the full conversation, \"range\" with `range`:\"N-M\" for a 1-indexed range of exchanges, or omitted for just the last one) and/or `code_only`/`text_only` (booleans) to save exactly the same current-response/range/full-conversation/code-only/text-only selections `/save` supports in chat.",
			opt("path"), opt("directory"), opt("content"), opt("overwrite"), opt("append"), req("project"),
			opt("history"), opt("mode"), opt("range"), opt("code_only"), opt("text_only")),
		tool("delete_path", "THE unified way to delete one or more files or directories (see documents.Delete). Pass `path` for a single item or `paths` (comma- or newline-separated) for several. Without `confirm:true`, nothing is deleted — the exact confirmation text (\"Delete \\\"x\\\"? (Y/N)\", one per item) is returned instead, so the caller can show it and re-call with `confirm:true` once the person agrees. Never removes anything without an explicit confirm.",
			opt("path"), opt("paths"), req("project"), opt("confirm")),
		tool("create_directory", "Create a directory inside the project's repo (missing parents are created). `path` must resolve inside \"repo\": absolute paths outside it, \"../\" traversal and symlinks escaping it are rejected.",
			opt("path"), req("project")),
		tool("read_document_layer", "Extract the text layer of a .docx/.xlsx/.pdf inside the project's repo, under the SAME read policy as read_file (repo boundary, exclude, read_scope) and the project's sanitization (secrets, PII if enabled). Recorded in the run evidence.",
			req("filename"), req("project"), opt("task")),
		tool("read_file", "Read a file inside the project's repo under Mova's read policy: denied outside \"repo\", denied if matched by \"exclude\", limited to the task's focus when read_scope is \"focus\" (symbol-level focus returns only those symbols); the content is sanitized (secrets always, PII if enabled) and the read is recorded in the run evidence. This governs only reads made THROUGH Mova — the host's own file tools are outside Mova unless you install its hooks (see check_read).",
			req("filename"), req("project"), opt("task")),
		tool("patch_file", "Surgically replace one exact, unique occurrence of `search` with `replace` inside an existing text file in the project's repo.",
			req("filename"), req("search"), req("replace"), req("project")),
		tool("check_read", "Hook adapter (Claude Code PreToolUse, type mcp_tool): decides whether the HOST may read a path, with the same policy as read_file (repo boundary, exclude, read_scope). Pass the hook input as `tool_input` (object with file_path/path/pattern) or `path`. Returns the hook JSON: permissionDecision allow|deny with the reason. Recorded in evidence.",
			req("project"), opt("task"), opt("path"), opt("tool_input"), opt("tool_name")),
		tool("sanitize_tool_output", "Hook adapter (Claude Code PostToolUse, type mcp_tool): applies the project's sanitization (secrets always, PII if enabled) to a host tool output and returns the hook JSON with updatedToolOutput when something changed. Pass `tool_response`/`output` and optionally `tool_input` (to know the file). Recorded in evidence.",
			req("project"), opt("task"), opt("tool_response"), opt("output"), opt("tool_input"), opt("tool_name")),
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
