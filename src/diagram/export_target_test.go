package diagram

import (
	"os"
	"path/filepath"
	"testing"
)

// TestExport_FullFilePathTarget covers the exact one-command shape
// documented in examples/02-pii-compliance-governance/README.md:
// `--diagram --export png --path ./diagramas/evidencia.png` must
// write that literal file, not a directory named "evidencia.png".
func TestExport_FullFilePathTarget(t *testing.T) {
	tmp := t.TempDir()
	target := filepath.Join(tmp, "diagramas", "evidencia.png")

	data := &Data{ProjectName: "02-pii-compliance-governance", Origin: "CLI", Interfaces: []string{"CLI"}}
	written, err := Export(data, []string{"png"}, target, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(written) != 1 {
		t.Fatalf("expected exactly 1 file written, got %d: %v", len(written), written)
	}
	if filepath.Base(written[0]) != "evidencia.png" {
		t.Fatalf("expected evidencia.png, got %s", written[0])
	}
	if _, err := os.Stat(target); err != nil {
		t.Fatalf("expected file at %s: %v", target, err)
	}
	if info, err := os.Stat(filepath.Join(tmp, "diagramas")); err != nil || !info.IsDir() {
		t.Fatalf("expected diagramas/ to be a directory")
	}
}

// TestExport_DirectoryTargetUnchanged is the existing behavior: a
// plain directory (no matching extension) still auto-names the file
// after the project, and MULTIPLE formats always use the directory
// form even if "path" happens to end in one of their extensions.
func TestExport_DirectoryTargetUnchanged(t *testing.T) {
	tmp := t.TempDir()
	data := &Data{ProjectName: "my-project", Origin: "CLI", Interfaces: []string{"CLI"}}

	written, err := Export(data, []string{"svg"}, tmp, "")
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(written[0]) != "my-project.svg" {
		t.Fatalf("expected my-project.svg, got %s", written[0])
	}

	written, err = Export(data, []string{"svg", "png"}, filepath.Join(tmp, "out.svg"), "")
	if err != nil {
		t.Fatal(err)
	}
	for _, w := range written {
		if filepath.Base(filepath.Dir(w)) != "out.svg" {
			t.Fatalf("with 2+ formats, %q must stay a plain directory, got %s", "out.svg", w)
		}
	}
}
