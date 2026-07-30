// Package http wraps mova/mcp's protocol layer (Process) over an HTTP
// transport — a thin adapter, no protocol logic of its own. Named "http"
// to match the directory layout of the architecture proposal; imports
// Go's "net/http" internally without any naming conflict (Go package
// identifiers are file-scoped, never self-referential).
package http

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"mova.local/core"
	"mova.local/mcp"
)

// StartServer inicia el servidor MCP sobre HTTP (ideal para Postman/curl).
func StartServer(adapter core.Adapter, root string, port int) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/mcp", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST required", http.StatusMethodNotAllowed)
			return
		}
		var req mcp.Request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			mcpErrorHTTP(w, -32700, "parse error", nil)
			return
		}
		responseBytes := mcp.Process(adapter, root, req)
		w.Header().Set("Content-Type", "application/json")
		w.Write(responseBytes)
	})

	mux.HandleFunc("/save", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST required", http.StatusMethodNotAllowed)
			return
		}
		var args map[string]any
		if err := json.NewDecoder(r.Body).Decode(&args); err != nil {
			mcpErrorHTTP(w, -32700, "parse error", nil)
			return
		}
		// Reexpressed as an ordinary tools/call so it runs through the exact
		// same mcp.Process → executeTool → documentTool → documents.Save
		// path the "save" MCP tool and the chat's /save command already
		// use — POST /save is a convenience shape on top, not a second
		// implementation. See mova.local/documents/save_service.go.
		req := mcp.Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage("1"),
			Method:  "tools/call",
			Params:  map[string]any{"name": "save", "arguments": args},
		}
		responseBytes := mcp.Process(adapter, root, req)
		w.Header().Set("Content-Type", "application/json")
		w.Write(responseBytes)
	})

	mux.HandleFunc("/delete", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST required", http.StatusMethodNotAllowed)
			return
		}
		var args map[string]any
		if err := json.NewDecoder(r.Body).Decode(&args); err != nil {
			mcpErrorHTTP(w, -32700, "parse error", nil)
			return
		}
		// Same convenience-shape convention as /save above: reexpressed as
		// tools/call so it runs through mcp.Process → executeTool →
		// documentTool → documents.Delete — the exact same unified delete
		// used by the "delete_path" MCP tool and chat's /delete command.
		// Without {"confirm": true} in the body, nothing is deleted; the
		// response is the confirmation prompt to show and re-send with
		// confirm:true once agreed. See mova.local/documents/delete_service.go.
		req := mcp.Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage("1"),
			Method:  "tools/call",
			Params:  map[string]any{"name": "delete_path", "arguments": args},
		}
		responseBytes := mcp.Process(adapter, root, req)
		w.Header().Set("Content-Type", "application/json")
		w.Write(responseBytes)
	})

	mux.HandleFunc("/workflow", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST required", http.StatusMethodNotAllowed)
			return
		}
		var args map[string]any
		if err := json.NewDecoder(r.Body).Decode(&args); err != nil {
			mcpErrorHTTP(w, -32700, "parse error", nil)
			return
		}
		// Body: {"project": "...", "task": "...", "workflow": "..."} (task
		// and workflow optional). Same convenience shape as /save and
		// /delete — runs through mcp.Process → executeTool → "get_workflow"
		// → budget.LoadWorkflow, so workflow.md is only ever loaded AFTER
		// the project is resolved and its Budget validated, identically to
		// "lee workflow.md" in chat and the "get_workflow" MCP tool. See
		// mova.local/budget/workflow.go.
		req := mcp.Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage("1"),
			Method:  "tools/call",
			Params:  map[string]any{"name": "get_workflow", "arguments": args},
		}
		responseBytes := mcp.Process(adapter, root, req)
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

// mcpErrorHTTP helper exclusivo para respuestas rápidas de fallos de
// parsing en HTTP — misma forma de error que usa el protocolo MCP.
func mcpErrorHTTP(w http.ResponseWriter, code int, msg string, id any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"jsonrpc": "2.0", "id": id,
		"error": map[string]any{"code": code, "message": msg},
	})
}
