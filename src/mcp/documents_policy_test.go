package mcp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"mova.local/core"
	"mova.local/core/focus/astfilter"
	"mova.local/evidence"
	_ "mova.local/graph"
	"mova.local/models" // registers core.ClosureHook, as cli/main.go does
)

// policyFixture: repo with a focused file, an excluded file, a data file
// with PII outside focus, and a file with an excluded symbol.
func policyFixture(t *testing.T, extra map[string]any) (root, project string) {
	t.Helper()
	root = t.TempDir()
	if err := astfilter.Init(root); err != nil {
		t.Fatal(err)
	}
	repo := filepath.Join(root, "repo")
	must := func(p, c string) {
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(c), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	must(filepath.Join(repo, "src", "app.js"), "function visible() { return legacy(); }\nfunction oculta() { return 'SECRETO_SIMBOLO'; }\n")
	must(filepath.Join(repo, "src", "legacy.js"), "function legacy() { return 'LEGACY_MARKER'; }\n")
	must(filepath.Join(repo, "datos", "clientes.json"), `{"nombre": "Ana Perez Soto", "email": "ana@example.com"}`)
	must(filepath.Join(root, "workflow.md"), "# stub")
	cfg := map[string]any{
		"project": "policy", "repo": "repo", "default_task": "t",
		"tasks":             map[string]any{"t": map[string]any{"prompt": "", "focus": []string{"src/app.js"}}},
		"exclude":           []string{"src/legacy.js", "src/app.js::func=oculta"},
		"dependency_policy": "warn",
		"budget":            map[string]any{"pii_masking": map[string]any{"enabled": true}},
	}
	for k, v := range extra {
		cfg[k] = v
	}
	b, _ := json.Marshal(cfg)
	must(filepath.Join(root, "projects", "policy", "project.json"), string(b))
	return root, "policy"
}

func TestReadFile_EnforcesPolicy(t *testing.T) {
	root, project := policyFixture(t, nil)
	cases := []struct {
		name, file, mustHave, mustNot string
	}{
		{"excluded file", "src/legacy.js", "lectura denegada por Mova (exclude)", "LEGACY_MARKER"},
		{"outside focus", "datos/clientes.json", "lectura denegada por Mova (focus_scope)", "Ana Perez"},
		{"outside repo", "/etc/passwd", "fuera del repo", "root:"},
		{"traversal", "../projects/policy/project.json", "fuera del repo", "\"repo\""},
		{"excluded symbol stripped", "src/app.js", "function visible", "SECRETO_SIMBOLO"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			text := callTool(t, root, "read_file", map[string]any{"project": project, "filename": c.file})
			if !strings.Contains(text, c.mustHave) || strings.Contains(text, c.mustNot) {
				t.Fatalf("got: %s", text)
			}
		})
	}
	// Every decision is in the session run's events.
	runs, _ := evidence.Runs(filepath.Join(root, "projects", project, "runs"))
	if len(runs) != 1 {
		t.Fatalf("expected one session run, got %v", runs)
	}
	ev, _ := os.ReadFile(filepath.Join(runs[0], "events.jsonl"))
	if strings.Count(string(ev), `"kind":"read"`) != len(cases) {
		t.Fatalf("expected %d read events, got:\n%s", len(cases), ev)
	}
}

func TestReadFile_RequiresProject(t *testing.T) {
	root, _ := policyFixture(t, nil)
	text := callTool(t, root, "read_file", map[string]any{"filename": filepath.Join(root, "repo", "src", "app.js")})
	if !strings.Contains(text, "\"project\" es obligatorio") {
		t.Fatalf("expected project requirement, got: %s", text)
	}
}

func TestReadFile_RepoScope_SanitizesPII(t *testing.T) {
	root, project := policyFixture(t, map[string]any{"read_scope": "repo"})
	text := callTool(t, root, "read_file", map[string]any{"project": project, "filename": "datos/clientes.json"})
	if strings.Contains(text, "Ana Perez Soto") || strings.Contains(text, "ana@example.com") {
		t.Fatalf("PII leaked: %s", text)
	}
	if !strings.Contains(text, `"nombre": "[PII_`) {
		t.Fatalf("expected the JSON structure kept with a pseudonym: %s", text)
	}
}

func TestCheckReadHook(t *testing.T) {
	root, project := policyFixture(t, nil)
	deny := callTool(t, root, "check_read", map[string]any{"project": project, "tool_name": "Read",
		"tool_input": map[string]any{"file_path": filepath.Join(root, "repo", "src", "legacy.js")}})
	if !strings.Contains(deny, `"permissionDecision":"deny"`) {
		t.Fatalf("expected deny, got %s", deny)
	}
	allow := callTool(t, root, "check_read", map[string]any{"project": project, "tool_name": "Read",
		"tool_input": map[string]any{"file_path": filepath.Join(root, "repo", "src", "app.js")}})
	if allow != "{}" {
		t.Fatalf("allowed read must return no decision ({}), got %s", allow)
	}
	out := callTool(t, root, "sanitize_tool_output", map[string]any{"project": project, "tool_name": "Read",
		"tool_input":    map[string]any{"file_path": filepath.Join(root, "repo", "src", "app.js")},
		"tool_response": "function visible() { return legacy(); }\nfunction oculta() { return 'SECRETO_SIMBOLO'; }\n"})
	if !strings.Contains(out, "updatedToolOutput") || strings.Contains(out, "SECRETO_SIMBOLO") {
		t.Fatalf("expected the governed view as updatedToolOutput, got %s", out)
	}
}

func TestGetFullContext_DependencyClosureBlocks(t *testing.T) {
	root, project := policyFixture(t, map[string]any{"dependency_policy": "block"})
	text := callTool(t, root, "get_full_context", map[string]any{"project": project})
	if !strings.Contains(text, "Dependency closure") || !strings.Contains(text, "src/legacy.js::legacy") {
		t.Fatalf("expected closure conflict, got %s", text)
	}
	runs, _ := evidence.Runs(filepath.Join(root, "projects", project, "runs"))
	m, err := evidence.ReadManifest(runs[len(runs)-1])
	if err != nil || m.Decision.Outcome != "blocked" || m.Decision.Gate != "dependency_closure" || m.Context.Bytes != 0 {
		t.Fatalf("manifest: %+v ctx=%d err=%v", m.Decision, m.Context.Bytes, err)
	}
	_ = core.TaskAll
}

// Tool results inside a loop Mova controls go through the same policy:
// the model cannot read an excluded file, cannot switch project, and the
// result is sanitized and recorded in the session run.
func TestRunLoopTool_GovernsToolResults(t *testing.T) {
	root, project := policyFixture(t, map[string]any{"tools": map[string]any{"enabled": true}, "read_scope": "repo"})
	adapter := core.NewFileAdapter(root)
	proj, err := adapter.GetProject(project)
	if err != nil {
		t.Fatal(err)
	}
	run, err := evidence.Start(filepath.Join(root, "projects", project, "runs"), evidence.Manifest{Door: "chat:provider"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	sess := &models.Session{Root: root, Run: run, ProjectName: project, TaskName: "t"}

	if _, err := RunLoopTool(adapter, root, sess, proj, "read_file", map[string]any{"filename": "src/legacy.js", "project": "otro"}); err == nil || !strings.Contains(err.Error(), "exclude") {
		t.Fatalf("excluded file must be denied in the loop, got %v", err)
	}
	out, err := RunLoopTool(adapter, root, sess, proj, "read_file", map[string]any{"filename": "datos/clientes.json"})
	if err != nil || strings.Contains(out, "Ana Perez Soto") || strings.Contains(out, "ana@example.com") {
		t.Fatalf("tool result not sanitized: %q %v", out, err)
	}
	ev, _ := os.ReadFile(filepath.Join(run.Dir, "events.jsonl"))
	if strings.Count(string(ev), `"kind":"tool_result"`) != 2 {
		t.Fatalf("expected 2 tool_result events:\n%s", ev)
	}
}
