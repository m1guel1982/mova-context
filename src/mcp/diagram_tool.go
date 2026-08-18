// diagram_tool.go — exposes `mova run <project> --diagram` as the MCP
// tool "generate_diagram", reachable identically from stdio and HTTP
// (same executeTool dispatch, see server.go) as any other tool. Same
// mova.local/diagram.BuildDiagram/Export the CLI (cli/diagram_cmd.go)
// calls — one engine, two doors, exactly like estimate_budget/
// budget_tool.go already does for `mova budget`.
package mcp

import (
	"fmt"
	"strings"

	"mova.local/core"
	"mova.local/diagram"
	"mova.local/orchestrator"
)

func diagramTool(adapter core.Adapter, root string, args map[string]any) (string, error) {
	project := str(args, "project")
	if project == "" {
		return "", fmt.Errorf("generate_diagram: \"project\" is required")
	}
	// Same group-vs-project adapter rule every other group-aware tool
	// follows (see run_agent/estimate_budget): a group has no
	// project.json/adapter of its own.
	if orchestrator.IsGroup(root, project) {
		adapter = core.NewFileAdapter(root)
	}

	formatsArg := str(args, "export")
	if formatsArg == "" {
		formatsArg = "svg"
	}
	formats := strings.Split(formatsArg, ",")

	data, err := diagram.BuildDiagram(adapter, root, project, str(args, "task"), str(args, "detail"), str(args, "origin"))
	if err != nil {
		return "", err
	}
	written, err := diagram.Export(data, formats, str(args, "path"), "")
	if err != nil {
		return "", err
	}

	summary := fmt.Sprintf("✓ diagram generated for %s\n", project)
	for _, p := range written {
		summary += "  " + p + "\n"
	}
	if data.Metrics == nil {
		summary += fmt.Sprintf("(no budget report found yet — call estimate_budget for %q first for token/cost metrics)\n", project)
	}
	return summary, nil
}
