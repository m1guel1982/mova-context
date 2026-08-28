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
	"os"
	"runtime"
	"strconv"
	"time"

	"mova.local/core"
	"mova.local/logging"
	"mova.local/mcp"
)

// StartServer inicia el servidor MCP sobre HTTP (ideal para Postman/curl).
func StartServer(adapter core.Adapter, root string, port int) error {
	logger := logging.Open(root)
	logging.SetDefault(logger)
	defer logger.Close()
	logger.Info("http", "server starting on port %d", port)

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

	mux.HandleFunc("/jobs/run", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST required", http.StatusMethodNotAllowed)
			return
		}
		var args map[string]any
		if err := json.NewDecoder(r.Body).Decode(&args); err != nil {
			mcpErrorHTTP(w, -32700, "parse error", nil)
			return
		}
		// Body: {"project": "...", "index": "0"} (index optional — runs
		// every job for the project when omitted). Same convenience-shape
		// convention as /save, /delete, /workflow above: reexpressed as
		// tools/call so it runs through mcp.Process → executeTool →
		// "run_job" → mova.local/jobs.RunJob — the exact same flow
		// `mova jobs run` uses. See mova.local/jobs/engine.go.
		req := mcp.Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage("1"),
			Method:  "tools/call",
			Params:  map[string]any{"name": "run_job", "arguments": args},
		}
		responseBytes := mcp.Process(adapter, root, req)
		w.Header().Set("Content-Type", "application/json")
		w.Write(responseBytes)
	})

	mux.HandleFunc("/agents/run", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST required", http.StatusMethodNotAllowed)
			return
		}
		var args map[string]any
		if err := json.NewDecoder(r.Body).Decode(&args); err != nil {
			mcpErrorHTTP(w, -32700, "parse error", nil)
			return
		}
		// Body: {"group": "...", "agent": "...", "task": "..."} (agent
		// optional — runs every agent in the group when omitted). Same
		// convenience-shape convention: reexpressed as tools/call so it
		// runs through mcp.Process → executeTool → "run_agent" →
		// mova.local/orchestrator.RunGroup — the exact same flow
		// `mova agents run` uses. See mova.local/orchestrator/run.go.
		req := mcp.Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage("1"),
			Method:  "tools/call",
			Params:  map[string]any{"name": "run_agent", "arguments": args},
		}
		responseBytes := mcp.Process(adapter, root, req)
		w.Header().Set("Content-Type", "application/json")
		w.Write(responseBytes)
	})

	mux.HandleFunc("/diagram", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST required", http.StatusMethodNotAllowed)
			return
		}
		var args map[string]any
		if err := json.NewDecoder(r.Body).Decode(&args); err != nil {
			mcpErrorHTTP(w, -32700, "parse error", nil)
			return
		}
		// Body: {"project": "...", "task": "...", "export": "svg,png,pdf",
		// "path": "...", "detail": "simple"|"verbose"}. Same
		// reexpressed-as-tools/call convention as every endpoint above —
		// runs through mcp.Process → executeTool → "generate_diagram" →
		// mova.local/diagram, the exact same engine `mova run --diagram`
		// and MCP's own generate_diagram tool use. "origin" is set to
		// "API HTTP" here, server-side, overriding anything the caller
		// sent — see mcp/diagram_tool.go's own doc comment on why this
		// is the one door that must inject it explicitly rather than
		// rely on generate_diagram's default.
		if args == nil {
			args = map[string]any{}
		}
		args["origin"] = "API HTTP"
		req := mcp.Request{
			JSONRPC: "2.0",
			ID:      json.RawMessage("1"),
			Method:  "tools/call",
			Params:  map[string]any{"name": "generate_diagram", "arguments": args},
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

	// net/http already gives every request its own goroutine; what it
	// does NOT give is a ceiling, so a burst of concurrent callers (CLI
	// scripts, a Chat UI, MCP clients, and other HTTP callers all
	// hitting the same shared instance — e.g. on a small Oracle Cloud
	// box) can otherwise spin up unbounded goroutines and file handles
	// at once. httpConcurrencyLimit() bounds that; ReadTimeout/
	// WriteTimeout/IdleTimeout keep a slow or dead client from holding a
	// worker slot forever.
	srv := &http.Server{
		Addr:         addr,
		Handler:      limitConcurrency(mux, httpConcurrencyLimit()),
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 120 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
	return srv.ListenAndServe()
}

// httpConcurrencyLimit reads MOVA_HTTP_MAX_CONCURRENCY, or defaults to
// 4× runtime.NumCPU() (capped at 64) — generous enough for normal
// bursts while still bounded, so this same knob can be tuned down on a
// small shared server and up on a dedicated one.
func httpConcurrencyLimit() int {
	if v := os.Getenv("MOVA_HTTP_MAX_CONCURRENCY"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	n := runtime.NumCPU() * 4
	if n < 8 {
		n = 8
	}
	if n > 64 {
		n = 64
	}
	return n
}

// limitConcurrency wraps handler with a semaphore so at most limit
// requests execute at once; once the limit is reached it replies 503
// immediately instead of letting goroutines pile up unboundedly while
// callers wait.
func limitConcurrency(handler http.Handler, limit int) http.Handler {
	sem := make(chan struct{}, limit)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case sem <- struct{}{}:
			defer func() { <-sem }()
			handler.ServeHTTP(w, r)
		default:
			http.Error(w, `{"error":"server busy, try again"}`, http.StatusServiceUnavailable)
		}
	})
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
