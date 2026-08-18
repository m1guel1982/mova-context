package diagram

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"mova.local/core"
)

// writeProject is the shared fixture builder: a minimal but real
// project.json under root/projects/<name>/, with a Focus file that
// actually exists on disk (so orchestrator.Count's Focus resolution
// has something real to read, same as a real project would).
func writeProject(t *testing.T, root, name string, extra map[string]any) {
	t.Helper()
	dir := filepath.Join(root, "projects", name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	repoDir := filepath.Join(root, "repo-"+sanitizeFileName(name))
	if err := os.MkdirAll(repoDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repoDir, "notes.txt"), []byte("hello world"), 0o644); err != nil {
		t.Fatal(err)
	}

	proj := map[string]any{
		"project":     name,
		"description": "Test project for diagram package",
		"repo":        repoDir,
		"lang":        "en",
		"adapter":     "file",
		"focus":       []string{"notes.txt"},
		"agents":      map[string]any{"domain": "base", "use": []string{}},
		"skills":      map[string]any{"domain": "base", "use": []string{}},
		"tasks":       map[string]any{},
	}
	for k, v := range extra {
		proj[k] = v
	}
	data, err := json.MarshalIndent(proj, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "project.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestBuildDiagram_SingleProject(t *testing.T) {
	root := t.TempDir()
	writeProject(t, root, "my-project", nil)
	adapter := core.NewFileAdapter(root)

	data, err := BuildDiagram(adapter, root, "my-project", "", "", "")
	if err != nil {
		t.Fatalf("BuildDiagram: %v", err)
	}
	if data.IsGroup {
		t.Fatalf("expected IsGroup=false for a plain project")
	}
	if len(data.Sources) != 1 || data.Sources[0].Path != "notes.txt" {
		t.Fatalf("expected Sources=[notes.txt], got %+v", data.Sources)
	}
	if len(data.Agents) != 1 || data.Agents[0].Name != "my-project" {
		t.Fatalf("expected one AgentNode named my-project, got %+v", data.Agents)
	}
}

func TestBuildDiagram_UnknownProjectErrors(t *testing.T) {
	root := t.TempDir()
	adapter := core.NewFileAdapter(root)
	if _, err := BuildDiagram(adapter, root, "does-not-exist", "", "", ""); err == nil {
		t.Fatalf("expected an error for a project that doesn't exist")
	}
}

func TestBuildDiagram_OriginNormalization(t *testing.T) {
	root := t.TempDir()
	writeProject(t, root, "my-project", nil)
	adapter := core.NewFileAdapter(root)

	cases := map[string]string{
		"CLI": "CLI", "cli": "CLI",
		"Chat": "Chat", "chat": "Chat",
		"API HTTP": "API HTTP", "http": "API HTTP", "api": "API HTTP",
		"MCP": "MCP", "mcp": "MCP", "": "MCP", "something-unrecognized": "MCP",
	}
	for input, want := range cases {
		data, err := BuildDiagram(adapter, root, "my-project", "", "", input)
		if err != nil {
			t.Fatal(err)
		}
		if data.Origin != want {
			t.Errorf("origin %q: got %q, want %q", input, data.Origin, want)
		}
	}
}

func TestBuildDiagram_PIIMaskingReflectsProjectJSON(t *testing.T) {
	root := t.TempDir()
	writeProject(t, root, "pii-on", map[string]any{
		"budget": map[string]any{"pii_masking": map[string]any{"enabled": true}},
	})
	writeProject(t, root, "pii-off", nil)
	adapter := core.NewFileAdapter(root)

	on, err := BuildDiagram(adapter, root, "pii-on", "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if !on.Firewall.PIIMaskingOn {
		t.Fatalf("expected PIIMaskingOn=true when project.json enables it")
	}

	off, err := BuildDiagram(adapter, root, "pii-off", "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if off.Firewall.PIIMaskingOn {
		t.Fatalf("expected PIIMaskingOn=false by default (no budget.pii_masking declared)")
	}
}

func TestRenderSVG_ContainsExpectedSections(t *testing.T) {
	root := t.TempDir()
	writeProject(t, root, "my-project", nil)
	adapter := core.NewFileAdapter(root)
	data, err := BuildDiagram(adapter, root, "my-project", "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	svg := RenderSVG(data)
	for _, want := range []string{"<svg", "SOURCES", "TOKEN FIREWALL", "AVAILABLE INTERFACES"} {
		if !bytesContains(svg, want) {
			t.Errorf("expected SVG output to contain %q", want)
		}
	}
}

func TestRenderSVG_NeverInventsDataOnEmptyFields(t *testing.T) {
	// A project with no jobs must never show a "JOBS" section — this is
	// the "no inventar datos, no dibujar lo que no existe" rule this
	// whole feature is built around.
	root := t.TempDir()
	writeProject(t, root, "no-jobs", nil)
	adapter := core.NewFileAdapter(root)
	data, err := BuildDiagram(adapter, root, "no-jobs", "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	svg := RenderSVG(data)
	if bytesContains(svg, "JOBS (SCHEDULED)") {
		t.Errorf("expected no Jobs section when project.json declares no jobs")
	}
}

func TestRenderPNG_ProducesValidPNGHeader(t *testing.T) {
	root := t.TempDir()
	writeProject(t, root, "my-project", nil)
	adapter := core.NewFileAdapter(root)
	data, err := BuildDiagram(adapter, root, "my-project", "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	png, err := RenderPNG(data)
	if err != nil {
		t.Fatalf("RenderPNG: %v", err)
	}
	pngMagic := []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}
	if !bytes.HasPrefix(png, pngMagic) {
		t.Fatalf("output does not start with the PNG magic header")
	}
}

func TestRenderPDF_ProducesValidPDFHeader(t *testing.T) {
	root := t.TempDir()
	writeProject(t, root, "my-project", nil)
	adapter := core.NewFileAdapter(root)
	data, err := BuildDiagram(adapter, root, "my-project", "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	pdf, err := RenderPDF(data)
	if err != nil {
		t.Fatalf("RenderPDF: %v", err)
	}
	if !bytes.HasPrefix(pdf, []byte("%PDF-1.4")) {
		t.Fatalf("output does not start with a PDF header")
	}
	if !bytes.Contains(pdf, []byte("%%EOF")) {
		t.Fatalf("output has no PDF trailer EOF marker")
	}
}

func TestExport_WritesRequestedFormatsAndCreatesDir(t *testing.T) {
	root := t.TempDir()
	writeProject(t, root, "my-project", nil)
	adapter := core.NewFileAdapter(root)
	data, err := BuildDiagram(adapter, root, "my-project", "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	outDir := filepath.Join(root, "does", "not", "exist", "yet")
	written, err := Export(data, []string{"svg", "png"}, outDir, "")
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	if len(written) != 2 {
		t.Fatalf("expected 2 files written, got %d: %v", len(written), written)
	}
	for _, p := range written {
		if _, err := os.Stat(p); err != nil {
			t.Errorf("expected %s to exist: %v", p, err)
		}
	}
}

func TestExport_UnknownFormatErrors(t *testing.T) {
	root := t.TempDir()
	writeProject(t, root, "my-project", nil)
	adapter := core.NewFileAdapter(root)
	data, err := BuildDiagram(adapter, root, "my-project", "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Export(data, []string{"bmp"}, root, ""); err == nil {
		t.Fatalf("expected an error for an unsupported export format")
	}
}

func bytesContains(s, substr string) bool {
	return len(s) >= len(substr) && (func() bool {
		for i := 0; i+len(substr) <= len(s); i++ {
			if s[i:i+len(substr)] == substr {
				return true
			}
		}
		return false
	})()
}
