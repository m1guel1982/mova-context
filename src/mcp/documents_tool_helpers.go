// documents_tool_helpers.go — arg parsing (boolArg/hasArg), path
// resolution (resolveSmartDir/resolveSmartFile/repoFor), disambiguation
// messages, and the two remaining JSON-shaped argument helpers
// (parseSheetsData/loadDiffusionConfig) for documents_tool.go. Split into
// its own file so documents_tool.go stays a readable, single-purpose
// executeTool switch.
package mcp

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"mova.local/core"
	"mova.local/documents"
)

// boolArg reads k from args as a boolean — accepts a native JSON boolean
// (the natural shape an HTTP/API caller sends) as well as the string
// convention some existing tools already use for boolean-ish flags (e.g.
// estimate_budget's "focus"), so "save" behaves the same from chat, MCP,
// or raw HTTP JSON regardless of which shape the caller used.
func boolArg(m map[string]any, k string) bool {
	if m == nil {
		return false
	}
	switch v := m[k].(type) {
	case bool:
		return v
	case string:
		return strings.EqualFold(v, "true")
	default:
		return false
	}
}

// hasArg reports whether k was present in args at all — used to tell
// "overwrite explicitly set to false" apart from "overwrite not
// mentioned at all" (see documents.SaveRequest.OverwriteExplicit).
func hasArg(m map[string]any, k string) bool {
	if m == nil {
		return false
	}
	_, ok := m[k]
	return ok
}

// ensureDir makes sure filePath's parent directory exists.
func ensureDir(filePath string) error {
	dir := filepath.Dir(filePath)
	if dir != "" && dir != "." {
		return os.MkdirAll(dir, 0755)
	}
	return nil
}

// resolveSmartDir resolves the "path" argument as a directory in its
// entirety — used only by create_directory.
func resolveSmartDir(adapter core.Adapter, root string, args map[string]any, argName string) (resolved string, ambiguousMsg string, err error) {
	requested := str(args, argName)
	repo := repoFor(adapter, args)
	path, ambiguous, err := documents.ResolveDirectoryPath(root, repo, requested)
	if err != nil {
		return "", "", err
	}
	if len(ambiguous) > 0 {
		return "", formatAmbiguousMessage(requested, ambiguous), nil
	}
	return path, "", nil
}

// resolveSmartFile resolves the "filename" argument as a file — only its
// directory portion is search-worthy, the final segment is always the file
// itself. Used by every other file/document tool.
func resolveSmartFile(adapter core.Adapter, root string, args map[string]any, argName string) (resolved string, ambiguousMsg string, err error) {
	requested := str(args, argName)
	repo := repoFor(adapter, args)
	path, ambiguous, err := documents.ResolveFilePath(root, repo, requested)
	if err != nil {
		return "", "", err
	}
	if len(ambiguous) > 0 {
		return "", formatAmbiguousMessage(ambiguousDirLabel(requested), ambiguous), nil
	}
	return path, "", nil
}

// ambiguousDirLabel extracts just the directory portion of a "dir/file.ext"
// argument, so the disambiguation question names the folder that's
// ambiguous ("config") rather than the whole requested path
// ("config/settings.json").
func ambiguousDirLabel(requested string) string {
	cleaned := strings.ReplaceAll(requested, "\\", "/")
	if idx := strings.LastIndex(cleaned, "/"); idx != -1 {
		return cleaned[:idx]
	}
	return cleaned
}

func repoFor(adapter core.Adapter, args map[string]any) string {
	repo := "."
	if project := str(args, "project"); project != "" {
		if proj, err := adapter.GetProject(project); err == nil && proj.Repo != "" {
			repo = proj.Repo
		}
	}
	return repo
}

// chatTurnsArg parses a "history" argument (a JSON array of
// {"role": "...", "content": "..."} objects — the same shape
// chat_completion's own "history" argument already uses) into
// []documents.ChatTurn, so the "save" tool's mode/range selection (see
// documents/save_selection.go) works from an ordinary JSON body exactly
// the way cli/chat_save.go's chatTurns adapts Session.History.
func chatTurnsArg(raw any) []documents.ChatTurn {
	list, ok := raw.([]any)
	if !ok {
		return nil
	}
	turns := make([]documents.ChatTurn, 0, len(list))
	for _, item := range list {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		role, _ := m["role"].(string)
		content, _ := m["content"].(string)
		if role == "" {
			continue
		}
		turns = append(turns, documents.ChatTurn{Role: role, Content: content})
	}
	return turns
}

// pathsArg reads "paths" (comma- or newline-separated) and/or the
// singular "path" argument and returns every non-empty entry, in order.
// This is unrelated to project.json's config fields (repo/workflow_path/
// budget_path/token_history_path are single values, see core/types.go) —
// it's specifically /delete's own "one or more files to remove in the
// same call" list (see documents/delete_service.go).
// Used by delete_path so a person, an MCP client, or an HTTP caller can
// pass either one path or several without three different argument
// shapes to support.
func pathsArg(args map[string]any) []string {
	var out []string
	if p := str(args, "path"); p != "" {
		out = append(out, p)
	}
	raw := str(args, "paths")
	for _, line := range strings.FieldsFunc(raw, func(r rune) bool { return r == ',' || r == '\n' }) {
		if t := strings.TrimSpace(line); t != "" {
			out = append(out, t)
		}
	}
	return out
}

// formatAmbiguousMessage builds the "which one did you mean?" question
// shown when a bare name matches more than one existing folder.
func formatAmbiguousMessage(name string, candidates []string) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d folders named %q. Which one should I use?\n", len(candidates), name))
	for i, c := range candidates {
		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, c))
	}
	sb.WriteString("Ask again with the full path (one of the above, or a new one).")
	return sb.String()
}

// parseSheetsData re-marshals the generically-decoded JSON-RPC params
// (map[string]any) back into documents.SheetsData, so the JSON payload's
// explicit type/value pairs survive the round trip without ambiguity.
func parseSheetsData(raw any) (documents.SheetsData, error) {
	if raw == nil {
		return nil, fmt.Errorf("sheets_data is required")
	}
	var b []byte
	if s, ok := raw.(string); ok {
		b = []byte(s)
	} else {
		marshaled, err := json.Marshal(raw)
		if err != nil {
			return nil, err
		}
		b = marshaled
	}
	var sheets documents.SheetsData
	if err := json.Unmarshal(b, &sheets); err != nil {
		return nil, fmt.Errorf("sheets_data: invalid JSON: %w", err)
	}
	return sheets, nil
}

// loadDiffusionConfig reads config/models/diffusion/config.json directly —
// same file location convention as config/models/ollama and
// config/models/lmstudio, kept as a plain read here so the documents
// package doesn't need to depend on models' provider-specific types.
func loadDiffusionConfig(root string) (documents.DiffusionConfig, error) {
	path := filepath.Join(root, "config", "models", "diffusion", "config.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return documents.DiffusionConfig{}, fmt.Errorf("%s: %w (create this file with the local diffusion server's provider/base_url)", path, err)
	}
	var dc documents.DiffusionConfig
	if err := json.Unmarshal(data, &dc); err != nil {
		return documents.DiffusionConfig{}, err
	}
	return dc, nil
}
