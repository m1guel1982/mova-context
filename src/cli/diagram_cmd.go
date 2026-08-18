// diagram_cmd.go — `mova run <project> --diagram [--export svg,png,pdf]
// [--path <dir>]`. Thin CLI glue only: everything real happens in
// mova.local/diagram (BuildDiagram + Export) — this file's whole job
// is turning already-parsed CLI flags into that package's call, and
// printing a short, honest summary of what got written.
package main

import (
	"fmt"
	"strings"

	"mova.local/core"
	"mova.local/diagram"
)

func runDiagram(root string, adapter core.Adapter, project, task string, formats []string, path string) {
	if project == "" {
		die("mova run <project> --diagram: no project given and none could be auto-detected")
	}
	for _, f := range formats {
		if !isValidDiagramFormat(f) {
			die(fmt.Sprintf("unknown --export format %q — valid formats: %s", f, strings.Join(diagram.ValidFormats, ", ")))
		}
	}

	data, err := diagram.BuildDiagram(adapter, root, project, task, "", "CLI")
	must(err)

	written, err := diagram.Export(data, formats, path, "")
	must(err)

	fmt.Println("✓ diagram generated for " + project)
	for _, p := range written {
		fmt.Println("  " + p)
	}
	if data.Metrics == nil {
		fmt.Println("  (no budget report found yet — run `mova budget " + project + "` first for token/cost metrics)")
	}
}

func isValidDiagramFormat(f string) bool {
	for _, valid := range diagram.ValidFormats {
		if f == valid {
			return true
		}
	}
	return false
}

// runChatDiagram implements `/diagram [export] [path]` inside `mova
// chat` — same defaults and validation as `mova run --diagram`
// (export defaults to svg, path defaults to the current directory),
// same underlying diagram.BuildDiagram/Export call, just with "Chat"
// as the origin so the rendered diagram highlights the door that
// actually triggered it (see model.go's Data.Origin). Args is
// whatever followed "/diagram" on the line, split on whitespace:
// `/diagram` · `/diagram png` · `/diagram svg,png,pdf ./out`.
func runChatDiagram(root string, adapter core.Adapter, project, task, args string) {
	if project == "" || adapter == nil {
		consolePrint("/diagram requires starting the chat with a project: mova chat <project> [task]\n")
		return
	}
	formats := []string{"svg"}
	path := ""
	if fields := strings.Fields(args); len(fields) > 0 {
		formats = strings.Split(fields[0], ",")
		if len(fields) > 1 {
			path = fields[1]
		}
	}
	for _, f := range formats {
		if !isValidDiagramFormat(f) {
			consolePrint(fmt.Sprintf("unknown format %q — valid formats: %s\n", f, strings.Join(diagram.ValidFormats, ", ")))
			return
		}
	}

	consolePrint("[Diagram] Building...\n")
	data, err := diagram.BuildDiagram(adapter, root, project, task, "", "Chat")
	if err != nil {
		consolePrint("Error: " + err.Error() + "\n")
		return
	}
	written, err := diagram.Export(data, formats, path, "")
	if err != nil {
		consolePrint("Error: " + err.Error() + "\n")
		return
	}
	consolePrint("✓ diagram generated for " + project + "\n")
	for _, p := range written {
		consolePrint("  " + p + "\n")
	}
}
