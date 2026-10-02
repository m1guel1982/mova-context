// server_util.go — helpers del servidor MCP/HTTP (lectura de argumentos y
// serialización JSON-RPC). Movidos tal cual desde server.go (≤300 líneas).
package mcp

import (
	"encoding/json"
	"strings"
)

func str(m map[string]any, k string) string {
	if m == nil {
		return ""
	}
	v, _ := m[k].(string)
	return v
}

// splitCommaArg splits a comma-separated MCP argument (e.g.
// "ignore":"docs/**,scripts/**") into a clean slice - MCP has no
// native repeated-flag concept like the CLI's flagStrAll, so comma
// separation is the one form it supports.
func splitCommaArg(s string) []string {
	if s == "" {
		return nil
	}
	var out []string
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
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
