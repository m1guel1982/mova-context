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
