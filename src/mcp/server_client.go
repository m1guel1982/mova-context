// server_client.go — identidad del cliente MCP conectado. Movido tal cual
// desde server.go (≤300 líneas por archivo).
package mcp

import (
	"strings"
)

// AgentClientName returns the MCP client identified in the last
// "initialize" handshake — "mcp-agent" by default when the client
// didn't send "clientInfo" (see captureClientInfo).
func AgentClientName() string { return currentAgentClient }

// captureClientInfo reads params.clientInfo.name/version (standard
// MCP shape, e.g. {"name":"Claude-Code","version":"1.2.0"}) and
// builds a readable identifier like "Claude-Code/1.2.0". When the
// client doesn't declare "clientInfo", or declares it empty,
// currentAgentClient keeps its default "mcp-agent" — never empty.
func captureClientInfo(params map[string]any) {
	ci, ok := params["clientInfo"].(map[string]any)
	if !ok {
		return
	}
	name, _ := ci["name"].(string)
	name = strings.TrimSpace(name)
	if name == "" {
		return
	}
	if version, _ := ci["version"].(string); strings.TrimSpace(version) != "" {
		name = name + "/" + strings.TrimSpace(version)
	}
	currentAgentClient = name
}
