// Package mcp implements the MCP (Model Context Protocol) JSON-RPC layer
// for Mova Context — transport-agnostic. StartStdio runs it directly over
// stdin/stdout; the http package (mova/http) wraps Process for the HTTP
// transport. Same engine, same tools, either way — exactly what the
// original mcp.go comment promised ("mismo motor base").
package mcp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
    "strings"
	"mova.local/budget"
	"mova.local/core"
)

// Request representa la estructura base del protocolo JSON-RPC 2.0.
type Request struct {
    JSONRPC string          `json:"jsonrpc"`
    ID      json.RawMessage `json:"id"` // <-- Cambiado de any a json.RawMessage
    Method  string          `json:"method"`
    Params  map[string]any  `json:"params"`
}

// StartStdio inicia el servidor usando Entrada/Salida estándar (requerido
// por Claude Desktop/Cursor).
func StartStdio(adapter core.Adapter, root string) error {
    scanner := bufio.NewScanner(os.Stdin)
    for scanner.Scan() {
        var req Request
        inputBytes := scanner.Bytes()
        if len(bytes.TrimSpace(inputBytes)) == 0 {
            continue
        }
        if err := json.Unmarshal(inputBytes, &req); err != nil {
            resp, _ := json.Marshal(serializeError(-32700, "parse error", nil))
            fmt.Println(string(resp))
            continue
        }
        responseBytes := Process(adapter, root, req)
        // Si es una notificación ignorada, responseBytes será nil. No imprimimos nada.
        if responseBytes != nil {
            fmt.Println(string(responseBytes))
        }
    }
    return scanner.Err()
}

// Process centraliza la ejecución de métodos y herramientas de Mova de
// forma unificada — usado tanto por StartStdio como por mova/http.
func Process(adapter core.Adapter, root string, req Request) []byte {
    // 1. Si es una notificación (no tiene ID o el método es de notificación), no respondemos nada
    if len(req.ID) == 0 || req.ID == nil || strings.HasPrefix(req.Method, "notifications/") {
        return nil
    }

    var resp map[string]any

    switch req.Method {
    case "initialize":
        resp = serializeResult(map[string]any{
            "protocolVersion": "2024-11-05",
            "serverInfo":      map[string]string{"name": "mova-context", "version": "3"},
            "capabilities":    map[string]any{"tools": map[string]bool{"listChanged": false}},
        }, req.ID)

    case "tools/list":
        resp = serializeResult(map[string]any{"tools": tools()}, req.ID)

    case "tools/call":
        tool := str(req.Params, "name")
        args, _ := req.Params["arguments"].(map[string]any)
        resp = executeTool(adapter, root, tool, args, req.ID)

    default:
        resp = serializeError(-32601, "method not found: "+req.Method, req.ID)
    }

    data, _ := json.Marshal(resp)
    return data
}

func executeTool(adapter core.Adapter, root, tool string, args map[string]any, id json.RawMessage) map[string]any {
    project := str(args, "project")
    task := str(args, "task")
    kind := str(args, "kind")
    domain := str(args, "domain")
    lang := str(args, "lang")
    name := str(args, "name")
    query := str(args, "query")

    var result string
    var err error

    switch tool {

		
    case "list_projects": // <-- NUEVA LOGICA
        projects, e := core.NewFileAdapter(root).ListProjects()
        if e != nil {
            err = e
        } else {
            var names []string
            for _, p := range projects {
                names = append(names, p.Name)
            }
            result = "Available projects: " + strings.Join(names, ", ")
        }

    case "get_full_context":
        result, err = fullContextTool(adapter, root, project, task)
    case "get_knowledge":
        result, err = adapter.GetKnowledge(kind, domain, lang, name)
    case "get_memory":
        result, err = adapter.GetMemory(project)
    case "get_memory_all":
        result, err = adapter.GetMemoryAll(project)
    case "save_memory":
        // The only memory WRITE path exposed over MCP/HTTP — get_memory/
        // get_memory_all were read-only. Appends "entry" verbatim to
        // projects/<project>/memory.md, most-recent-first, exactly like
        // `mova memory <project> "<entry>"` and `mova chat`'s `/memory`
        // already do (see cli/chat_cmd.go's runChatMemory) — same
        // adapter.AppendMemory underneath, so all three stay consistent.
        entry := str(args, "entry")
        if entry == "" {
            err = fmt.Errorf("save_memory: \"entry\" is required")
        } else {
            err = adapter.AppendMemory(project, entry)
            if err == nil {
                result = "✓ memory saved: projects/" + project + "/memory.md"
            }
        }
    case "get_workflow":
        // Never reads workflow.md directly — see budget.LoadWorkflow's
        // package doc (budget/workflow.go): project resolution, context
        // build, Dedup/Focus, token estimate, and the Budget gate all
        // run first, identically to Chat's "lee/ejecuta workflow.md"
        // and the HTTP /workflow endpoint (http/server.go). Without a
        // "project" argument there is nothing to resolve a Budget
        // against, so this keeps its old placeholder behavior instead
        // of guessing one.
        if project == "" {
            result = "workflow.md — pass \"project\" (and, optionally, \"task\") so its Budget can be validated before loading it."
            break
        }
        wf, e := budget.LoadWorkflow(adapter, root, project, task, str(args, "workflow"), "")
        if e != nil {
            if wf != nil {
                result = wf.RenderLog() + "\n" + e.Error()
                err = nil
                break
            }
            err = e
            break
        }
        result = wf.RenderLog() + "\n" + wf.Content
    case "search_context":
        var results []core.SearchResult
        results, err = adapter.Search(query, domain)
        if err == nil {
            data, _ := json.MarshalIndent(results, "", "  ")
            result = string(data)
        }
    case "chat_completion":
        result, err = chatCompletionTool(adapter, root, args)
    case "estimate_budget":
        result, err = budgetTool(adapter, root, args)
    case "read_document_layer", "generate_word_contract", "generate_pdf_document",
        "generate_vector_graphic", "generate_excel_report", "trigger_diffusion_image",
        "read_file", "write_file", "patch_file", "create_directory", "save", "delete_path":
        result, err = documentTool(adapter, root, tool, args)
    default:
        return serializeError(-32602, "unknown tool: "+tool, id)
    }

    // SI EL PROYECTO O ACCION FALLA: Lo devolvemos como texto amigable para Claude
    if err != nil {
        text := err.Error()
        // Errors that already explain what happened and what to do next
        // (Budget's "ERROR\n\nCurrent context...\n\nSuggestion:\n...") get
        // returned as-is — appending "Please use 'list_projects'" on top
        // would be misleading (it implies the PROJECT name was wrong,
        // when a Budget error has nothing to do with that).
        if !strings.Contains(text, "\nSuggestion:") {
            text = fmt.Sprintf("Error running tool: %s. Please use 'list_projects' to see valid projects.", text)
        }
        return serializeResult(map[string]any{
            "content": []map[string]any{{
                "type": "text",
                "text": text,
            }},
        }, id)
    }

    return serializeResult(map[string]any{
        "content": []map[string]any{{
            "type": "text", "text": result,
        }},
    }, id)
}

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
        tool("estimate_budget", "Estimate the token/USD cost of a project's real context (agents+skills+prompt+focus+memory — the same assembly get_full_context produces), broken down per component, using the local tiktoken-go tokenizer and config/prices.json. 100% local: no LLM call, nothing leaves this machine. Writes mova-budget-report.md. `focus`=\"true\" also compares full-repo vs. focus-only token cost.",
            req("project"), opt("task"), opt("focus")),
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

func str(m map[string]any, k string) string {
	if m == nil {
		return ""
	}
	v, _ := m[k].(string)
	return v
}

func serializeResult(result any, id json.RawMessage) map[string]any {
    return map[string]any{"jsonrpc": "2.0", "id": id, "result": result}
}

func serializeError(code int, msg string, id json.RawMessage) map[string]any {
    return map[string]any{
        "jsonrpc": "2.0",
        "id":      id,
        "error":   map[string]any{"code": code, "message": msg},
    }
}