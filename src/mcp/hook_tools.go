// hook_tools.go — adapters so a coding agent's OWN tools can be governed
// by the same rules, via hooks (Claude Code: hooks of type "mcp_tool" on
// PreToolUse / PostToolUse). Without hooks, an MCP server cannot see or
// stop the host's Read/Bash/Grep: this is the only way Mova's perimeter
// extends to the host's later turns, and only for the hooks installed.
//
//   - check_read (PreToolUse): repo boundary + exclude + read_scope on the
//     path the host is about to read. Deny → hook JSON with
//     permissionDecision "deny". Allowed → "{}" (no decision: the host's
//     normal permission flow still applies; Mova never auto-approves).
//   - sanitize_tool_output (PostToolUse): for a Read of a file, replaces
//     the output with the governed view (focused symbols only in symbols
//     mode, symbol-level excludes stripped, secrets/PII sanitized); for any
//     other output (Bash, Grep…), sanitizes the text. Returns
//     updatedToolOutput only when something changed.
//
// Limits: directory-wide tools (Grep/Glob) are checked at directory level
// only; content injected without a tool call (e.g. @-mentions) never
// reaches a hook; a host that does not run hooks is not governed.
package mcp

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"mova.local/budget"
	"mova.local/core"
	"mova.local/core/focus/resolvers"
	"mova.local/documents"
)

func hookPath(args map[string]any) string {
	if p := str(args, "path"); p != "" {
		return p
	}
	ti := objArg(args, "tool_input")
	for _, k := range []string{"file_path", "notebook_path", "path"} {
		if v, ok := ti[k].(string); ok && v != "" {
			return v
		}
	}
	return ""
}

func objArg(args map[string]any, k string) map[string]any {
	switch v := args[k].(type) {
	case map[string]any:
		return v
	case string:
		var m map[string]any
		if json.Unmarshal([]byte(v), &m) == nil {
			return m
		}
	}
	return map[string]any{}
}

func hookJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func checkReadHookTool(adapter core.Adapter, root string, args map[string]any) (string, error) {
	project := str(args, "project")
	proj, err := requireProject(adapter, args)
	if err != nil {
		return "", err
	}
	task := core.ChatTaskName(proj, str(args, "task"))
	run := sessionRun(root, project, task, proj, "hook:session")
	p := hookPath(args)
	ev := map[string]any{"hook": "PreToolUse", "tool_name": str(args, "tool_name"), "path": p}
	if p == "" {
		ev["decision"], ev["reason"] = "not_evaluated", "la llamada no trae ruta"
		_ = run.Event("hook_check_read", ev)
		return "{}", nil
	}
	repo := core.RepoDir(root, proj)
	abs := p
	if !filepath.IsAbs(abs) {
		abs = filepath.Join(repo, abs)
	}
	deny := func(rule, reason string) (string, error) {
		ev["decision"], ev["rule"], ev["reason"] = "deny", rule, reason
		_ = run.Event("hook_check_read", ev)
		return hookJSON(map[string]any{"hookSpecificOutput": map[string]any{
			"hookEventName":            "PreToolUse",
			"permissionDecision":       "deny",
			"permissionDecisionReason": fmt.Sprintf("Mova (%s, %s): %s — %s", project, task, p, reason),
		}}), nil
	}
	if isDir(abs) { // Grep/Glob over a directory: boundary + exclude only
		if !documents.WithinRepo(repo, abs) {
			return deny("repo_boundary", "fuera del repo del proyecto")
		}
		if resolvers.PathExcluded(core.ExcludeInScope(proj, task), abs) {
			return deny("exclude", "directorio excluido por \"exclude\"")
		}
		ev["decision"], ev["reason"] = "no_decision", "directorio: solo límite del repo y exclude"
		_ = run.Event("hook_check_read", ev)
		return "{}", nil
	}
	d := core.CheckRead(root, proj, task, abs)
	if !d.Allowed {
		return deny(d.Rule, d.Reason)
	}
	ev["decision"], ev["mode"] = "no_decision", d.Mode
	_ = run.Event("hook_check_read", ev)
	return "{}", nil
}

func sanitizeOutputHookTool(adapter core.Adapter, root string, args map[string]any) (string, error) {
	project := str(args, "project")
	proj, err := requireProject(adapter, args)
	if err != nil {
		return "", err
	}
	task := core.ChatTaskName(proj, str(args, "task"))
	run := sessionRun(root, project, task, proj, "hook:session")
	output := str(args, "output")
	if output == "" {
		switch v := args["tool_response"].(type) {
		case string:
			output = v
		case nil:
		default:
			output = hookJSON(v)
		}
	}
	p := hookPath(args)
	ev := map[string]any{"hook": "PostToolUse", "tool_name": str(args, "tool_name"), "path": p}

	var governed string
	if p != "" && !isDir(p) && (str(args, "tool_name") == "" || str(args, "tool_name") == "Read") {
		abs := p
		if !filepath.IsAbs(abs) {
			abs = filepath.Join(core.RepoDir(root, proj), abs)
		}
		view, rev, rerr := GovernedRead(root, proj, task, abs)
		for k, v := range rev {
			ev[k] = v
		}
		if rerr != nil {
			governed = "[MOVA: contenido retirado — " + rerr.Error() + "]"
		} else {
			governed = view
		}
	} else {
		text, rep := budget.GovernText(root, proj, task, filepath.Base(p)+".out", output)
		governed = text
		if rep.Changed() {
			ev["governance"] = rep
		}
	}
	changed := governed != output
	ev["changed"] = changed
	_ = run.Event("hook_sanitize_output", ev)
	if !changed {
		return "{}", nil
	}
	return hookJSON(map[string]any{"hookSpecificOutput": map[string]any{
		"hookEventName":     "PostToolUse",
		"updatedToolOutput": governed,
	}}), nil
}

func isDir(p string) bool {
	info, err := osStat(p)
	return err == nil && info.IsDir()
}

var osStat = os.Stat
