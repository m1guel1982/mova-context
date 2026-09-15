// apply_file_changes.go — the evolution of Item 15: ONE structured
// tool a model can call, in any language, instead of the person
// having to type manual file commands ("/save -c", etc.) every turn —
// see docs/i18n/{es,en}/COMMANDS.md § "apply_file_changes". Added to
// the SAME provider-agnostic <<<MOVA_TOOL_CALL>>> protocol
// agent_tools.go already uses (see that file's header for why this
// project deliberately does not wire each provider's own native
// function-calling API) rather than a second, parallel mechanism.
//
// This file only knows how to PARSE a proposal and APPLY one already-
// confirmed change (reusing documentTool's "save"/"delete_path" - the
// exact same code /save and the "delete_path" MCP tool already run,
// per the "reutiliza al 100%" requirement). It deliberately does NOT
// decide who gets to confirm what: cli/apply_file_changes.go owns the
// interactive terminal menu (mova chat, mova ui chat); chat_tool.go
// (MCP, and HTTP via /mcp) uses DescribePendingChanges below instead,
// since neither door has a terminal to show a menu on.
package mcp

import (
	"fmt"
	"strings"

	"mova.local/core"
	"mova.local/i18n"
)

type FileChangeAction string

const (
	ActionCreate FileChangeAction = "create"
	ActionModify FileChangeAction = "modify"
	ActionDelete FileChangeAction = "delete"
)

// FileChange is one entry of apply_file_changes' "changes" array —
// the exact shape named in the tool's schema (see argsHintFor).
type FileChange struct {
	Action  FileChangeAction
	Path    string
	Content string
}

// ParseApplyFileChanges decodes the "changes" argument ParseAgentToolCall
// already extracted into a plain map[string]any (it doesn't know Go
// struct types, just JSON) into typed FileChanges. ok is false for a
// malformed or empty proposal (no path, or an action other than
// create/modify/delete) — the caller then reports that as an error to
// the model, exactly like any other bad tool call.
func ParseApplyFileChanges(arguments map[string]any) ([]FileChange, bool) {
	raw, has := arguments["changes"]
	if !has {
		return nil, false
	}
	list, ok := raw.([]any)
	if !ok || len(list) == 0 {
		return nil, false
	}
	var out []FileChange
	for _, item := range list {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		action := FileChangeAction(strings.ToLower(str(m, "action")))
		path := str(m, "path")
		if path == "" {
			continue
		}
		switch action {
		case ActionCreate, ActionModify, ActionDelete:
		default:
			continue
		}
		out = append(out, FileChange{Action: action, Path: path, Content: str(m, "content")})
	}
	return out, len(out) > 0
}

// ApplyFileChange writes/deletes ONE already-confirmed change, reusing
// documentTool's "save" (create/modify — same code path /save and
// the "save" agent tool already use) and "delete_path" (same code
// path the "delete_path" MCP tool and chat's "/delete" use, called
// with confirm:true since the person already confirmed via the menu
// — see cli/apply_file_changes.go — or, on MCP/HTTP, explicitly opted
// in via a second confirm_file_changes call).
func ApplyFileChange(adapter core.Adapter, root string, c FileChange) (string, error) {
	switch c.Action {
	case ActionDelete:
		return documentTool(adapter, root, "delete_path", map[string]any{
			"path": c.Path, "confirm": true,
		})
	default: // create, modify — both are just "write this content to this path"
		return documentTool(adapter, root, "save", map[string]any{
			"path": c.Path, "content": c.Content,
		})
	}
}

// ActionLabel renders an action using the i18n keys the person
// supplied (tools.file_changes.action_create/modify/delete) — used by
// both the interactive CLI menu and this file's non-interactive
// summary, so the wording is identical everywhere.
func ActionLabel(a FileChangeAction) string {
	switch a {
	case ActionCreate:
		return i18n.T("tools.file_changes.action_create")
	case ActionModify:
		return i18n.T("tools.file_changes.action_modify")
	case ActionDelete:
		return i18n.T("tools.file_changes.action_delete")
	default:
		return string(a)
	}
}

// DescribePendingChanges is the MCP/HTTP path (chat_tool.go): neither
// door has a terminal to show cli/apply_file_changes.go's interactive
// menu on, so apply_file_changes NEVER auto-writes there — it always
// stops here and returns this numbered summary as the tool result
// instead, same header/action wording as the interactive menu (see
// tools.file_changes.prompt_confirm_header) plus an explicit note that
// nothing was written and how to actually apply a change on this
// door (call "save"/"delete_path" directly for the specific file/s).
func DescribePendingChanges(changes []FileChange) string {
	var b strings.Builder
	b.WriteString(i18n.T("tools.file_changes.prompt_confirm_header") + "\n")
	for i, c := range changes {
		fmt.Fprintf(&b, "  %d. [%s] %s\n", i+1, ActionLabel(c.Action), c.Path)
	}
	b.WriteString("\nNo file was changed. This door does not support an interactive confirmation menu — to actually apply one of these, call the \"save\" tool (create/modify) or \"delete_path\" tool (delete, with confirm:true) directly for that specific path. Tell the person these proposed changes were NOT applied.")
	return b.String()
}
