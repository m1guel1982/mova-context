// mcp.go — MCP (Model Context Protocol) server for Mova.
// Puede iniciarse en modo HTTP o STDIO manteniendo el mismo motor base.
package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
)

// mcpRequest representa la estructura base del protocolo JSON-RPC 2.0
type mcpRequest struct {
	JSONRPC string         `json:"jsonrpc"`
	ID      any            `json:"id"`
	Method  string         `json:"method"`
	Params  map[string]any `json:"params"`
}

// startMCPHttp inicia el servidor utilizando protocolo HTTP (ideal para Postman/Curl)
func startMCPHttp(adapter Adapter, root string, port int) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/mcp", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST required", http.StatusMethodNotAllowed)
			return
		}

		var req mcpRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			mcpErrorHTTP(w, -32700, "parse error", nil)
			return
		}

		responseBytes := processMCP(adapter, root, req)
		w.Header().Set("Content-Type", "application/json")
		w.Write(responseBytes)
	})

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok", "version": "3"})
	})

	addr := fmt.Sprintf(":%d", port)
	log.Printf("Mova MCP (HTTP) running → http://localhost%s/mcp", addr)
	return http.ListenAndServe(addr, mux)
}

// startMCPStdio inicia el servidor usando Entrada/Salida estándar (Requerido por Claude Desktop/Cursor)
func startMCPStdio(adapter Adapter, root string) error {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		var req mcpRequest
		inputBytes := scanner.Bytes()
		if len(bytes.TrimSpace(inputBytes)) == 0 {
			continue
		}

		if err := json.Unmarshal(inputBytes, &req); err != nil {
			resp, _ := json.Marshal(serializeError(-32700, "parse error", nil))
			fmt.Println(string(resp))
			continue
		}

		responseBytes := processMCP(adapter, root, req)
		fmt.Println(string(responseBytes))
	}
	return scanner.Err()
}

// processMCP centraliza la ejecución de métodos y herramientas de Mova de forma unificada
func processMCP(adapter Adapter, root string, req mcpRequest) []byte {
	var resp map[string]any

	switch req.Method {
	case "initialize":
		resp = serializeResult(map[string]any{
			"protocolVersion": "2024-11-05",
			"serverInfo":      map[string]string{"name": "mova-context", "version": "3"},
			"capabilities":    map[string]any{"tools": map[string]bool{"listChanged": false}},
		}, req.ID)

	case "tools/list":
		resp = serializeResult(map[string]any{"tools": mcpTools()}, req.ID)

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

func executeTool(adapter Adapter, root, tool string, args map[string]any, id any) map[string]any {
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
		// Retorna el contexto compilado automáticamente cuando project.json lo habilita
		result, err = resolveContext(adapter, root, project, task)
	case "compile_context":
		// Fuerza la compilación del contexto ignorando el modo por defecto
		result, err = buildCompiledContext(adapter, root, project, task)
	case "get_knowledge":
		result, err = adapter.GetKnowledge(kind, domain, lang, name)
	case "get_memory":
		result, err = adapter.GetMemory(project)
	case "get_memory_all":
		result, err = adapter.GetMemoryAll(project)
	case "get_workflow":
		result, err = adapter.GetKnowledge("workflow", "", lang, "workflow")
		if err != nil {
			result = "workflow.md — read from repository root"
			err = nil
		}
	case "search_context":
		var results []SearchResult
		results, err = adapter.Search(query, domain)
		if err == nil {
			data, _ := json.MarshalIndent(results, "", "  ")
			result = string(data)
		}
	default:
		return serializeError(-32602, "unknown tool: "+tool, id)
	}

	if err != nil {
		return serializeError(-32603, err.Error(), id)
	}

	return serializeResult(map[string]any{
		"content": []map[string]any{{"type": "text", "text": result}},
	}, id)
}

func mcpTools() []map[string]any {
	return []map[string]any{
		tool("get_full_context", "Full assembled context (= mova run). Compiled automatically when project.json enables it. Primary tool.",
			req("project"), opt("task")),
		tool("compile_context", "Force the Context Compiler regardless of mode (= mova compile). Distilled + focus-pruned.",
			req("project"), opt("task")),
		tool("get_knowledge", "Get a single agent, skill, or prompt.",
			req("kind"), req("domain"), opt("lang"), req("name")),
		tool("get_memory", "Active memory for a project.",
			req("project")),
		tool("get_memory_all", "Active + all archived memory.",
			req("project")),
		tool("get_workflow", "The workflow guide.",
			opt("lang")),
		tool("search_context", "Search across all knowledge.",
			req("query"), opt("domain")),
	}
}

// ── tiny helpers ──────────────────────────────────────────────────────────────

func tool(name, desc string, props ...map[string]any) map[string]any {
	properties := map[string]any{}
	var required []string
	for _, p := range props {
		n := p["name"].(string)
		properties[n] = map[string]string{"type": "string", "description": p["desc"].(string)}
		if p["req"].(bool) {
			required = append(required, n)
		}
	}
	schema := map[string]any{"type": "object", "properties": properties}
	if len(required) > 0 {
		schema["required"] = required
	}
	return map[string]any{"name": name, "description": desc, "inputSchema": schema}
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

// mcpErrorHTTP helper exclusivo para respuestas rápidas de fallos de parsing en HTTP
func mcpErrorHTTP(w http.ResponseWriter, code int, msg string, id any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(serializeError(code, msg, id))
}

func serializeResult(result any, id any) map[string]any {
	return map[string]any{"jsonrpc": "2.0", "id": id, "result": result}
}

func serializeError(code int, msg string, id any) map[string]any {
	return map[string]any{"jsonrpc": "2.0", "id": id,
		"error": map[string]any{"code": code, "message": msg}}
}