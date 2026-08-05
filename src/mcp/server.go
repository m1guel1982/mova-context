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
	"mova.local/budget"
	"mova.local/core"
	"mova.local/logging"
	"os"
	"strings"
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
	logger := logging.Open(root)
	logging.SetDefault(logger)
	defer logger.Close()
	logger.Info("mcp", "stdio server started")

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
	logging.L().Debug("mcp", "tool call: %s", tool)
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
	case "list_jobs":
		result, err = listJobsTool(root, args)
	case "run_job":
		result, err = runJobTool(adapter, root, args)
	case "list_agents":
		result, err = listAgentsTool(root, args)
	case "run_agent":
		result, err = runAgentTool(adapter, root, args)
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

// ── tiny helpers ──────────────────────────────────────────────────────────

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
