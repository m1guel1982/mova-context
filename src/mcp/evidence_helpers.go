// evidence_helpers.go — provenance of agent/model for MCP/HTTP runs.
package mcp

import (
	"mova.local/core"
	"mova.local/evidence"
	"mova.local/models"
)

// agentAttr is the MCP client name sent in "initialize". In stdio it is
// the single client of this process; in HTTP it is the last client that
// initialized this process — the evidence says so instead of guessing.
func agentAttr() evidence.Attr {
	name := AgentClientName()
	if name == "" {
		return evidence.Attr{Value: "unknown", Source: "not_observable: el cliente no envió clientInfo"}
	}
	if httpMode {
		return evidence.Attr{Value: name, Source: "observed:mcp-initialize (HTTP: último cliente que inicializó este proceso)"}
	}
	return evidence.Attr{Value: name, Source: "observed:mcp-initialize (stdio)"}
}

// hostModelAttr: the model of the MCP host is outside Mova's perimeter.
func hostModelAttr(proj *core.Project) evidence.Attr {
	return evidence.Attr{Value: "unknown", Source: "not_observable: el contexto se devuelve al host MCP, que elige el modelo"}
}

// httpMode is set by the HTTP door (http package) at startup.
var httpMode bool

// SetHTTPMode marks this process as serving MCP over HTTP.
func SetHTTPMode(v bool) { httpMode = v }

// runIDOf is the evidence run of the session ("" when none).
func runIDOf(sess *models.Session) string {
	if sess == nil || sess.Run == nil {
		return ""
	}
	return sess.Run.ID
}
