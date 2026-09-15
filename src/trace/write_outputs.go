// write_outputs.go — orchestrates writing the physical OUTPUT
// artifacts into the directory path.go already resolved. One
// function for all four doors: CLI/MCP/HTTP/Chat all call
// WriteOutputs and get back the same list of written paths, in the
// order OUTPUT should list them.
package trace

import (
	"fmt"
	"os"
	"path/filepath"
)

// DefaultExportFormat is "md": context-report.md, no external
// dependency to view it, no encoding surprises - PDF is opt-in via
// --export pdf for whoever wants a shareable, printable document.
const DefaultExportFormat = "md"

// WriteOutputs writes context-report.<exportFormat> (md or pdf) and
// context-diagram.png into outDir (created if it does not exist yet)
// - ALWAYS exactly these two, regardless of how many Security
// Findings the run produced (see findings_display.go's
// MaxPDFSecurityFindings: the report body is always capped). A THIRD
// file, pii-audit-log.json, is written ONLY when there is more to
// show than the report body's cap allows - it carries the complete,
// unabridged Findings list the report itself only summarizes, right
// next to the report (same outDir), never silently dropped.
func WriteOutputs(d *Data, exportFormat, outDir string) ([]string, error) {
	if exportFormat != "pdf" && exportFormat != "md" {
		return nil, fmt.Errorf("invalid export format %q - use \"pdf\" or \"md\"", exportFormat)
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return nil, fmt.Errorf("could not create output directory %q: %w", outDir, err)
	}

	var written []string

	contextReportPath := filepath.Join(outDir, "context-report."+exportFormat)
	if exportFormat == "pdf" {
		if err := WriteContextReportPDF(contextReportPath, d); err != nil {
			return nil, fmt.Errorf("could not generate context-report.pdf: %w", err)
		}
	} else {
		if err := os.WriteFile(contextReportPath, []byte(RenderContextReportMarkdown(d)), 0o644); err != nil {
			return nil, fmt.Errorf("could not generate context-report.md: %w", err)
		}
	}
	written = append(written, contextReportPath)

	diagramPath := filepath.Join(outDir, "context-diagram.png")
	pngBytes, err := RenderDiagramPNG(d)
	if err != nil {
		return nil, fmt.Errorf("could not generate context-diagram.png: %w", err)
	}
	if err := os.WriteFile(diagramPath, pngBytes, 0o644); err != nil {
		return nil, fmt.Errorf("could not generate context-diagram.png: %w", err)
	}
	written = append(written, diagramPath)

	// pii-audit-log.json is the forensic artifact / structured source
	// of truth (see prompt requirement: "JSON: artefacto forense
	// completo") - ALWAYS written, every run, right next to the
	// report, scoped to exactly this execution_id (never merged with
	// or reused from a previous run).
	auditLogPath := filepath.Join(outDir, "pii-audit-log.json")
	if err := writeAuditLog(auditLogPath, d); err != nil {
		return nil, fmt.Errorf("could not generate pii-audit-log.json: %w", err)
	}
	written = append(written, auditLogPath)

	return written, nil
}

// OutputNames returns just the base file names (no path), same order
// - used by console.go to display OUTPUT with relative names instead
// of absolute paths (the default across every door, see trace.go).
func OutputNames(written []string) []string {
	names := make([]string, 0, len(written))
	for _, w := range written {
		names = append(names, filepath.Base(w))
	}
	return names
}
