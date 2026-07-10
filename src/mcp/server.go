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
        result, err = core.BuildContext(adapter, root, project, task)
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
        var results []core.SearchResult
        results, err = adapter.Search(query, domain)
        if err == nil {
            data, _ := json.MarshalIndent(results, "", "  ")
            result = string(data)
        }
    default:
        return serializeError(-32602, "unknown tool: "+tool, id)
    }

    // SI EL PROYECTO O ACCION FALLA: Lo devolvemos como texto amigable para Claude
    if err != nil {
        return serializeResult(map[string]any{
            "content": []map[string]any{{
                "type": "text",
                "text": fmt.Sprintf("Error running tool: %s. Please use 'list_projects' to see valid projects.", err.Error()),
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
        tool("get_workflow", "The workflow guide.",
            opt("lang")),
        tool("search_context", "Search across all knowledge.",
            req("query"), opt("domain")),
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