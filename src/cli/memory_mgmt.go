// memory_mgmt.go — memory management commands for Mova Context.
//
// Commands added here (all prefixed with "memory"):
//   mova memory-clear [project]              delete ALL memory (active + archives)
//   mova memory-clear [project] --archived   delete only archived months
//   mova memory-clear [project] --date 2024-06-15   delete a specific day
//   mova memory-clear [project] --from 2024-06-01 --to 2024-06-30   date range
//   mova memory-clear [project] --keep-active   delete archives, keep memory.md
//   mova memory-config [project] enable      enable auto-archive
//   mova memory-config [project] disable     disable auto-archive
//   mova memory-config [project] days N      set retention to N days
//
// Every destructive command asks for confirmation unless --yes is passed.
// Configuration lives entirely in project.json under "archive".
// The Adapter decides where data lives. This file never knows.

package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"mova.local/core"
)

// runMemoryClear handles: mova memory-clear [project] [flags]
func runMemoryClear(adapter core.Adapter, root, project string) {
	req := core.MemoryDeleteRequest{}
	yes := flagBool("--yes")

	// Determine what to delete based on flags
	switch {
	case flagBool("--archived"):
		req.Archived = true
	case flagBool("--keep-active"):
		req.KeepActive = true
	case flagStr("--date", "") != "":
		req.Date = flagStr("--date", "")
	case flagStr("--from", "") != "" && flagStr("--to", "") != "":
		req.From = flagStr("--from", "")
		req.To = flagStr("--to", "")
	default:
		req.All = true
	}

	// Load project config to check confirm_delete setting
	fa := core.NewFileAdapter(root)
	proj, _ := fa.GetProject(project)
	needConfirm := true
	if proj != nil {
		needConfirm = core.ConfirmDeleteRequired(proj.Archive)
	}

	// Confirmation
	if needConfirm && !yes {
		msg := clearDescription(req, project)
		if !askConfirm(msg) {
			consolePrint("cancelled — nothing was deleted\n")
			return
		}
	}

	n, err := adapter.DeleteMemory(project, req)
	must(err)
	consolePrint(fmt.Sprintf("deleted %d entries from %s\n", n, project))
}

// runMemoryConfig handles: mova memory-config [project] [action] [value]
func runMemoryConfig(root, project, action, value string) {
	projPath := filepath.Join(root, "projects", project, "project.json")
	data, err := os.ReadFile(projPath)
	must(err)

	// Parse as raw JSON to preserve all existing fields
	var raw map[string]any
	must(json.Unmarshal(data, &raw))

	// Get or create "archive" block
	archRaw, _ := raw["archive"].(map[string]any)
	if archRaw == nil {
		archRaw = map[string]any{}
	}

	switch action {
	case "enable":
		archRaw["enabled"] = true
		consolePrint("archive enabled for: " + project + "\n")
	case "disable":
		archRaw["enabled"] = false
		consolePrint("archive disabled for: " + project + "\n")
	case "days":
		n, err := strconv.Atoi(value)
		if err != nil || n <= 0 {
			die("days must be a positive integer (e.g. 10, 30, 90)")
		}
		archRaw["retention_days"] = n
		consolePrint(fmt.Sprintf("retention set to %d days for: %s\n", n, project))
	case "confirm":
		// mova memory-config project confirm true|false
		archRaw["confirm_delete"] = value != "false"
		consolePrint(fmt.Sprintf("confirm_delete set to %v for: %s\n", value != "false", project))
	case "keep-memory-only":
		archRaw["keep_memory_only"] = true
		consolePrint("keep_memory_only enabled for: " + project + "\n")
	default:
		consolePrint(`memory-config actions:
  enable           enable auto-archive
  disable          disable auto-archive
  days N           set retention days (any positive integer: 10, 30, 90...)
  confirm true|false  require confirmation on delete
  keep-memory-only delete archives, always keep memory.md
`)
		return
	}

	raw["archive"] = archRaw
	out, err := json.MarshalIndent(raw, "", "  ")
	must(err)
	must(os.WriteFile(projPath, out, 0644))
}

// ── helpers ───────────────────────────────────────────────────────────────────

// clearDescription returns a human-readable description of what will be deleted.
func clearDescription(req core.MemoryDeleteRequest, project string) string {
	switch {
	case req.All:
		return fmt.Sprintf("Delete ALL memory (active + archives) for project %q?", project)
	case req.Archived:
		return fmt.Sprintf("Delete all ARCHIVED months for project %q? (memory.md kept)", project)
	case req.KeepActive:
		return fmt.Sprintf("Delete all archives for project %q? (memory.md kept)", project)
	case req.Date != "":
		return fmt.Sprintf("Delete entries from %s for project %q?", req.Date, project)
	case req.From != "" && req.To != "":
		return fmt.Sprintf("Delete entries from %s to %s for project %q?", req.From, req.To, project)
	default:
		return fmt.Sprintf("Delete memory for project %q?", project)
	}
}

// askConfirm prints msg and waits for y/yes. Returns true if confirmed.
func askConfirm(msg string) bool {
	consolePrint(msg + " [y/N]: ")
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		ans := strings.TrimSpace(strings.ToLower(scanner.Text()))
		return ans == "y" || ans == "yes"
	}
	return false
}
