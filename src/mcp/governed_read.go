// governed_read.go — read_file / read_document_layer under Mova's read
// policy (core.CheckRead) and sanitization (budget.GovernText), with
// every decision recorded in the MCP session run.
package mcp

import (
	"fmt"

	"mova.local/budget"
	"mova.local/core"
	"mova.local/evidence"
)

func governedReadTool(adapter core.Adapter, root, tool string, args map[string]any) (string, error) {
	project := str(args, "project")
	proj, err := requireProject(adapter, args)
	if err != nil {
		return "", err
	}
	path, ambiguousMsg, err := resolveSmartFile(adapter, root, args, "filename")
	task := core.ChatTaskName(proj, str(args, "task"))
	run := sessionRun(root, project, task, proj, "mcp:session")
	if err != nil {
		_ = run.Event("read", map[string]any{"tool": tool, "requested": str(args, "filename"), "allowed": false, "rule": "repo_boundary", "reason": err.Error()})
		return "", err
	}
	if ambiguousMsg != "" {
		return ambiguousMsg, nil
	}
	out, ev, err := GovernedRead(root, proj, task, path)
	ev["tool"] = tool
	_ = run.Event("read", ev)
	return out, err
}

// GovernedRead is the single governed read shared by MCP/HTTP read tools
// and the tool loops Mova controls (chat). Returns the sanitized content
// and the evidence event describing the decision.
func GovernedRead(root string, proj *core.Project, task, absPath string) (string, map[string]any, error) {
	content, d, err := core.GovernedRead(root, proj, task, absPath)
	ev := map[string]any{"path": d.RelPathOr(), "allowed": d.Allowed, "rule": d.Rule, "reason": d.Reason, "scope": d.Scope, "mode": d.Mode}
	if err != nil {
		if d.Allowed {
			ev["error"] = err.Error()
		}
		return "", ev, err
	}
	governed, rep := budget.GovernText(root, proj, task, d.RelPath, content)
	ev["sha256"] = evidence.SHA256([]byte(governed))
	ev["bytes"] = len(governed)
	if rep.Changed() {
		ev["governance"] = rep
	}
	if rep.Blocked {
		return "", ev, fmt.Errorf("lectura bloqueada por política %s: %s", rep.Rule, d.RelPath)
	}
	return governed, ev, nil
}

// sessionRun returns the long-lived evidence run of this server process
// for project (door "mcp:session" or "hook:session"). Errors degrade to a
// nil run (events are then dropped) — reads stay governed regardless.
func sessionRun(root, project, task string, proj *core.Project, door string) *evidence.Run {
	m := budget.BaseManifest(root, project, task, proj, budget.RunInfo{Door: door, Agent: agentAttr(), Model: hostModelAttr(proj)})
	r, err := evidence.Session(budget.RunsDir(root, project), m)
	if err != nil {
		return nil
	}
	return r
}
