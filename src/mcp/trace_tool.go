// trace_tool.go — exposes `context-trace` as the MCP tool
// "context_trace", reachable identically from stdio and HTTP (same
// executeTool dispatch, see server.go). Uses mova.local/trace.Run -
// the same engine cli/trace_cmd.go and http/server.go use - so CLI,
// MCP, HTTP, and Chat always show the same analysis.
package mcp

import (
	"fmt"
	"os"

	"mova.local/core"
	"mova.local/orchestrator"
	"mova.local/trace"
)

func traceTool(adapter core.Adapter, root string, args map[string]any) (string, error) {
	project := str(args, "project")
	repoURL := str(args, "repo")
	if project == "" && repoURL == "" {
		return "", fmt.Errorf("context_trace: \"project\" or \"repo\" is required")
	}

	if project != "" && orchestrator.IsGroup(root, project) {
		return "", fmt.Errorf("context_trace: %q is a multi-agent group - run context_trace on an individual agent instead (\"%s/<agent>\")", project, project)
	}

	exportFormat := str(args, "export")
	if exportFormat == "" {
		exportFormat = trace.DefaultExportFormat
	}

	origin := str(args, "origin")
	if origin == "" {
		origin = "MCP"
	}

	agentClient := str(args, "agent_client")
	if agentClient == "" {
		agentClient = AgentClientName()
	}
	targetModel := str(args, "target_model")
	if targetModel == "" {
		targetModel = core.TargetModelFor(root, project)
	}

	cwd, _ := os.Getwd()
	opts := trace.Options{
		Root: root, Cwd: cwd, Origin: origin,
		Project: project, Task: str(args, "task"),
		// Same policy precedence as the CLI flags of the same name (see
		// core.ResolvePolicyRequest): comma-separated, bare names or
		// full cross-platform paths.
		PolicyInclude: core.SplitPolicyList(str(args, "policies_include")),
		PolicyExclude: core.SplitPolicyList(str(args, "policies_exclude")),
		RepoURL:       repoURL, Branch: str(args, "branch"),
		ExportFormat: exportFormat, Output: str(args, "output"),
		AgentClient:  agentClient,
		TargetModel:  targetModel,
		PolicyAuthor: core.ResolvePolicyAuthor(root, project),
		// The MCP door is not interactive: an explicit
		// "generate_project_json":"true" is the only way the
		// suggested project.json gets written - a "yes" is never
		// assumed by default.
		GenerateProjectJSON: str(args, "generate_project_json") == "true",
		IgnorePatterns:      splitCommaArg(str(args, "ignore")),
		PruneDocstrings:     str(args, "prune_docstrings") == "true",
	}

	traceAdapter := adapter
	if repoURL != "" {
		traceAdapter = nil
	}

	res, err := trace.Run(traceAdapter, opts)
	if err != nil {
		return "", err
	}

	summary := res.Console
	if res.Data.IsRemote && res.Data.SuggestedProjectJSON != "" && res.SuggestedProjectPath == "" {
		summary += "\n(A suggested project.json is available - call again with \"generate_project_json\": \"true\" to write it to the projects directory.)\n"
	}
	if res.SuggestedProjectPath != "" {
		summary += "\nGenerated project.json at: " + res.SuggestedProjectPath + "\n"
	}
	return summary, nil
}
