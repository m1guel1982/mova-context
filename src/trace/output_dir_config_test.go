// output_dir_config_test.go — regression coverage for wiring
// config/general/config.json's "output_dir" into context-trace's
// existing "--output not given" fallback (see trace.go's runLocal):
// when the operator configures a non-blank output_dir, it takes
// PRIORITY over the command's historical default (the analyzed
// project's own directory); when unconfigured, that historical
// default is kept exactly as before (mpaths.ConfiguredOutputDir
// returns "" in that case).
package trace

import (
	"os"
	"path/filepath"
	"testing"

	"mova.local/core"
)

func writeTraceProject(t *testing.T) (root, project string) {
	t.Helper()
	root = t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "workflow.md"), []byte("# stub"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "config"), 0o755); err != nil {
		t.Fatal(err)
	}
	prices := `{"currency":"USD","unit":"per_1m_tokens","providers":{"ollama":{"models":{"llama3.1":{"input":0,"output":0,"context_window":128000,"local":true}}}}}`
	if err := os.WriteFile(filepath.Join(root, "config", "prices.json"), []byte(prices), 0o644); err != nil {
		t.Fatal(err)
	}
	project = "trace-output-test"
	projDir := filepath.Join(root, "projects", project)
	if err := os.MkdirAll(projDir, 0o755); err != nil {
		t.Fatal(err)
	}
	repoDir := filepath.Join(root, "repo")
	if err := os.MkdirAll(repoDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repoDir, "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	pj := `{
		"project": "` + project + `", "author": "system:default", "repo": "` + filepath.ToSlash(repoDir) + `",
		"lang": "en", "adapter": "file", "default_task": "t",
		"agents": {"domain": "software", "use": []},
		"skills": {"domain": "software", "use": []},
		"tasks": {"t": {"focus": ["main.go"]}}
	}`
	if err := os.WriteFile(filepath.Join(projDir, "project.json"), []byte(pj), 0o644); err != nil {
		t.Fatal(err)
	}
	return root, project
}

func writeOutputDirConfig(t *testing.T, root, outputDir string) {
	t.Helper()
	dir := filepath.Join(root, "config", "general")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := `{"output_dir": "` + outputDir + `"}`
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestRunLocal_NoOutputFlag_NoConfig_KeepsProjectDirDefault: unchanged
// pre-existing behavior when config/general/config.json doesn't exist.
func TestRunLocal_NoOutputFlag_NoConfig_KeepsProjectDirDefault(t *testing.T) {
	root, project := writeTraceProject(t)
	adapter := core.NewFileAdapter(root)

	res, err := Run(adapter, Options{Root: root, Cwd: root, Origin: "CLI", Project: project, Task: "t"})
	if err != nil {
		t.Fatal(err)
	}
	wantDir := filepath.Join(root, "projects", project)
	if res.OutputDir != wantDir {
		t.Fatalf("got OutputDir %q, want the project's own dir %q", res.OutputDir, wantDir)
	}
}

// TestRunLocal_NoOutputFlag_ConfiguredOutputDir_TakesPriority: THE
// regression test — a configured "output_dir" wins over the project's
// own directory when --output wasn't given.
func TestRunLocal_NoOutputFlag_ConfiguredOutputDir_TakesPriority(t *testing.T) {
	root, project := writeTraceProject(t)
	customOut := filepath.Join(root, "reportes-centralizados")
	writeOutputDirConfig(t, root, "reportes-centralizados") // bare relative -> under root
	adapter := core.NewFileAdapter(root)

	res, err := Run(adapter, Options{Root: root, Cwd: root, Origin: "CLI", Project: project, Task: "t"})
	if err != nil {
		t.Fatal(err)
	}
	if res.OutputDir != customOut {
		t.Fatalf("got OutputDir %q, want the configured output_dir %q", res.OutputDir, customOut)
	}
	if _, err := os.Stat(filepath.Join(customOut, "context-report.md")); err != nil {
		t.Fatalf("expected context-report.md written under the configured output_dir: %v", err)
	}
}

// TestRunLocal_ExplicitOutputFlag_StillWinsOverConfig: an explicit
// --output always wins over BOTH config.json and the project-dir
// default — unchanged, highest priority.
func TestRunLocal_ExplicitOutputFlag_StillWinsOverConfig(t *testing.T) {
	root, project := writeTraceProject(t)
	writeOutputDirConfig(t, root, "reportes-centralizados")
	explicitOut := filepath.Join(root, "explicit-output")
	adapter := core.NewFileAdapter(root)

	res, err := Run(adapter, Options{Root: root, Cwd: root, Origin: "CLI", Project: project, Task: "t", Output: explicitOut})
	if err != nil {
		t.Fatal(err)
	}
	if res.OutputDir != filepath.Clean(explicitOut) {
		t.Fatalf("got OutputDir %q, want the explicit --output %q", res.OutputDir, explicitOut)
	}
}
