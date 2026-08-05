// tui_paths.go — small path helpers shared by the TUI screens. Every
// path here mirrors an existing convention already used elsewhere in
// Mova Context (core/file_adapter.go, logging/config.go) — nothing new
// is invented, this file just avoids repeating filepath.Join(...) calls
// across every tui_*.go file.
package main

import "path/filepath"

func projectDir(root, project string) string {
	return filepath.Join(root, "projects", project)
}

func projectJSONPath(root, project string) string {
	return filepath.Join(projectDir(root, project), "project.json")
}

func memoryMDPath(root, project string) string {
	return filepath.Join(projectDir(root, project), "memory.md")
}

func groupConfigPath(root, group string) string {
	return filepath.Join(root, "projects", group, "config.json")
}

func workflowMDPath(root string) string {
	return filepath.Join(root, "workflow.md")
}

// loggingConfigPath mirrors mova.local/logging.ConfigPath exactly —
// duplicated as a one-line path join (not logic) so this file doesn't
// need to import an internal helper for a single filepath.Join.
func loggingConfigPath(root string) string {
	return filepath.Join(root, "config", "log", "logging.json")
}

func repoDir(root, repo string) string {
	if repo == "" || repo == "." {
		return root
	}
	if filepath.IsAbs(repo) {
		return repo
	}
	return filepath.Join(root, repo)
}
