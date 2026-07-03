// mcp.go — MCP server universal compatible (Claude, Cursor, Codex, Gemini)

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
)

/* ─────────────────────────────────────────────────────────────
   CORE MCP TYPES (STRICT JSON-RPC 2.0 COMPLIANT)
───────────────────────────────────────────────────────────── */

type mcpRequest struct {
	JSONRPC string                 `json:"jsonrpc"`
	ID      any                    `json:"id,omitempty"`
	Method  string                 `json:"method"`
	Params  map[string]any        `json:"params,omitempty"`
}

type mcpResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      any         `json:"id,omitempty"`
	Result  any         `json:"result,omitempty"`
	Error   *mcpError   `json:"error,omitempty"`
}

type mcpError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

/* ─────────────────────────────────────────────────────────────
   MCP CONTENT TYPES (REQUIRED)
───────────────────────────────────────────────────────────── */

type mcpTextContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

/* ─────────────────────────────────────────────────────────────
   HTTP SERVER
───────────────────────────────────────────────────────────── */

func startMCPHttp(adapter Adapter, root string, port int) error {
	mux := http.NewServeMux()

	mux.HandleFunc("/mcp", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST required", http.StatusMethodNotAllowed)
			return
		}

		var req mcpRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErrorHTTP(w, -32700, "parse error", nil)
			return
		}

		resp := processMCP(adapter, root, req)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"status":  "ok",
			"service": "mova-context",
			"version": "3.0.0",
		})
	})

	addr := fmt.Sprintf(":%d", port)
	log.Printf("MCP server running http://localhost%s/mcp", addr)

	return http.ListenAndServe(addr, mux)
}

/* ─────────────────────────────────────────────────────────────
   STDIO TRANSPORT (CLAUDE DESKTOP / CURSOR / CODEx)
───────────────────────────────────────────────────────────── */

func startMCPStdio(adapter Adapter, root string) error {

	decoder := json.NewDecoder(os.Stdin)
	encoder := json.NewEncoder(os.Stdout)

	for {

		var req mcpRequest

		if err := decoder.Decode(&req); err != nil {
			continue
		}

		// notifications → IGNORAR
		if req.ID == nil {
			continue
		}

		resp := processMCP(adapter, root, req)

		if err := encoder.Encode(resp); err != nil {
			continue
		}
	}
}

/* ─────────────────────────────────────────────────────────────
   MCP CORE ROUTER
───────────────────────────────────────────────────────────── */

func processMCP(adapter Adapter, root string, req mcpRequest) mcpResponse {

	switch req.Method {

   case "initialize":
	return mcpResponse{
		JSONRPC: "2.0",
		ID: req.ID,
		Result: map[string]any{
			"protocolVersion": "2024-11-05",
			"serverInfo": map[string]any{
				"name": "mova-context",
				"version": "3.0.0",
			},
			"capabilities": map[string]any{
				"tools": map[string]any{},
			},
		},
	}

	case "notifications/initialized":
		return mcpResponse{JSONRPC: "2.0"}

	case "ping":
		return serializeResult(map[string]any{"pong": true}, req.ID)

	case "tools/list":
		return serializeResult(map[string]any{
			"tools": mcpTools(),
		}, req.ID)

	case "tools/call":
		name := str(req.Params, "name")
		args, _ := req.Params["arguments"].(map[string]any)
		return executeTool(adapter, root, name, args, req.ID)

	default:
		return serializeError(-32601, "method not found: "+req.Method, req.ID)
	}
}

/* ─────────────────────────────────────────────────────────────
   TOOL EXECUTION
───────────────────────────────────────────────────────────── */

func executeTool(adapter Adapter, root, tool string, args map[string]any, id any) mcpResponse {

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

	case "get_full_context":
		result, err = resolveContext(adapter, root, project, task)

	case "compile_context":
		result, err = buildCompiledContext(adapter, root, project, task)

	case "get_knowledge":
		result, err = adapter.GetKnowledge(kind, domain, lang, name)

	case "get_memory":
		result, err = adapter.GetMemory(project)

	case "get_memory_all":
		result, err = adapter.GetMemoryAll(project)

	case "get_workflow":
		result, err = adapter.GetKnowledge("workflow", "", lang, "workflow")

	case "search_context":
		results, e := adapter.Search(query, domain)
		err = e
		if err == nil {
			b, _ := json.MarshalIndent(results, "", "  ")
			result = string(b)
		}

	default:
		return serializeError(-32602, "unknown tool: "+tool, id)
	}

	/* ───── MCP STANDARD RESPONSE ───── */

	if err != nil {
		return serializeResult(map[string]any{
			"content": []mcpTextContent{{
				Type: "text",
				Text: "Error: " + err.Error(),
			}},
			"isError": true,
		}, id)
	}

	return serializeResult(map[string]any{
		"content": []mcpTextContent{{
			Type: "text",
			Text: result,
		}},
		"isError": false,
	}, id)
}

/* ─────────────────────────────────────────────────────────────
   TOOLS DEFINITION (STRICT MCP SCHEMA)
───────────────────────────────────────────────────────────── */

func mcpTools() []map[string]any {
	return []map[string]any{
		tool("get_full_context", "Compile full context", true,
			map[string]any{
				"type": "object",
				"properties": map[string]any{
					"project": map[string]any{"type": "string"},
					"task":    map[string]any{"type": "string"},
				},
				"required": []string{"project"},
				"additionalProperties": false,
			},
		),

		tool("compile_context", "Force compilation", true, nil),
		tool("get_knowledge", "Get knowledge", true, nil),
		tool("get_memory", "Get memory", true, nil),
		tool("get_memory_all", "Get all memory", true, nil),
		tool("get_workflow", "Get workflow", false, nil),
		tool("search_context", "Search context", true, nil),
	}
}

func tool(name, desc string, strict bool, schema map[string]any) map[string]any {
	if schema == nil {
		schema = map[string]any{
			"type": "object",
			"properties": map[string]any{},
			"additionalProperties": false,
		}
	}

	return map[string]any{
		"name":        name,
		"description": desc,
		"inputSchema": schema,
	}
}

/* ─────────────────────────────────────────────────────────────
   HELPERS
───────────────────────────────────────────────────────────── */

func str(m map[string]any, k string) string {
	if m == nil {
		return ""
	}
	v, _ := m[k].(string)
	return v
}

func serializeResult(result any, id any) mcpResponse {
	return mcpResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}
}

func serializeError(code int, msg string, id any) mcpResponse {
	return mcpResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: &mcpError{
			Code:    code,
			Message: msg,
		},
	}
}

func mcpEncodeError(code int, msg string, id any) string {
	b, _ := json.Marshal(serializeError(code, msg, id))
	return string(b)
}

func writeErrorHTTP(w http.ResponseWriter, code int, msg string, id any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(serializeError(code, msg, id))
}