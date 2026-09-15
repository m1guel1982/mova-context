// trace_cmd.go — `mova context-trace [<project>] [<task>] [--repo <url>]
// [--export pdf|md] [--output <path>]` (see dispatch.go) and its
// equivalent inside `mova chat` (`/context-trace`, see chat_cmd.go).
//
// Both doors call mova.local/trace.Run - the same engine the MCP
// server (mcp/trace_tool.go) and the HTTP endpoint
// (/api/v1/context-trace, see http/server.go) use - so all four doors
// always show the exact same analysis, never a "simpler" one for
// CLI/Chat and a "fuller" one for MCP/HTTP.
package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"mova.local/core"
	"mova.local/i18n"
	"mova.local/trace"
)

// runContextTrace implements `mova context-trace` as a direct terminal
// command (outside of `mova chat`).
func runContextTrace(root, project, task string) {
	repoURL := flagStr("--repo", "")
	exportFormat := flagStr("--export", trace.DefaultExportFormat)
	output := flagStr("--output", "")
	if t := flagStr("--task", ""); t != "" {
		task = t
	}

	cwd, err := os.Getwd()
	must(err)

	opts := trace.Options{
		Root: root, Cwd: cwd, Origin: "CLI",
		Project: project, Task: task,
		RepoURL: repoURL, ExportFormat: exportFormat, Output: output,
		IgnorePatterns:  flagIgnorePatterns(),
		PruneDocstrings: flagBool("--prune-docstrings"),
		OnProgress:      printProgress,
		AgentClient:     "mova-cli",
		TargetModel:     core.TargetModelFor(root, project),
		PolicyAuthor:    core.ResolvePolicyAuthor(root, project),
	}

	var adapter core.Adapter
	if repoURL == "" {
		fa := core.NewFileAdapter(root)
		proj, err := fa.GetProject(project)
		must(err)
		adapter = newAdapter(root, proj)
	}

	res, err := trace.Run(adapter, opts)
	must(err)

	consolePrint(res.Console + "\n")

	if res.Data.IsRemote && res.Data.SuggestedProjectJSON != "" && res.SuggestedProjectPath == "" {
		promptGenerateProjectJSON(bufio.NewScanner(os.Stdin), res, root)
	}
}

// printProgress renders a simple, dependency-free progress line: no
// ANSI escape codes (some Windows terminals don't support them without
// extra setup - see console_windows.go), just "\r" plus enough
// trailing spaces to fully overwrite whatever the previous, possibly
// longer, message left behind.
// printProgress mantiene los progresos repetitivos (como tokenizado) en la misma
// línea usando '\r', y hace salto de línea '\n' únicamente cuando cambia de evento.
var lastProgressMessage string

func printProgress(percent int, message string) {
	const width = 96

	// Extraer el prefijo clave del mensaje para detectar si cambió de etapa
	// (ej: de "Cloning..." a "Tokenizing...", o cuando finaliza "Done.")
	currentKey := message
	if idx := strings.Index(message, " ("); idx != -1 {
		currentKey = message[:idx]
	}

	// Si cambió el tipo de evento respecto al anterior, cerramos la línea anterior con un salto
	if lastProgressMessage != "" && lastProgressMessage != currentKey {
		consolePrint("\n")
	}
	lastProgressMessage = currentKey

	// Formatear la línea actual con relleno de espacios
	line := fmt.Sprintf("[%3d%%] %s", percent, message)
	if len(line) < width {
		line += strings.Repeat(" ", width-len(line))
	} else {
		line = line[:width]
	}

	// Sobrescribir la línea actual
	consolePrint("\r" + line)

	// Al llegar al final, agregar el salto de línea final
	if percent >= 100 {
		consolePrint("\n")
		lastProgressMessage = ""
	}
}

// promptGenerateProjectJSON asks the console format's "[Y/n]" question
// (see console.go's renderConsoleRemote). On an affirmative answer:
//   - Local repository: writes project.json pointing at the existing
//     local path immediately (see trace.WriteSuggestedProjectJSON).
//   - Remote repository: asks for a definitive target directory
//     (iteratively, until a usable path is given), clones the
//     repository there (see trace.CloneToDirectory - the original
//     temporary clone was already removed by Run's own cleanup, see
//     git_fetch.go's header for why this is a fresh clone rather than
//     a move), then writes project.json pointing at that new path.
//
// On "N": nothing is deleted here for a remote repository - its
// temporary clone was already removed automatically as part of the
// analysis itself (see trace.Run); for a local repository, nothing
// on disk is touched at all, per the spec.
func promptGenerateProjectJSON(scanner *bufio.Scanner, res *trace.Result, root string) {
	if !scanner.Scan() {
		return
	}
	answer := strings.ToLower(strings.TrimSpace(scanner.Text()))
	if answer != "" && answer != "y" && answer != "yes" {
		return
	}

	content := res.Data.SuggestedProjectJSON
	if trace.LooksLikeGitURL(res.Data.RepoURL) {
		targetDir := promptTargetDirectory(scanner)
		if targetDir == "" {
			consolePrint("No target directory provided - project.json was not generated.\n")
			return
		}
		// A relative path (e.g. "fast", matching a typical answer to
		// "Enter target directory") is resolved against the SAME
		// resolved project root as project.json itself, not against
		// whatever the current working directory happens to be - so
		// it lands under $MOVA_PROJECT_ROOT/... too, consistent with
		// the FindRoot fix above. An absolute path is still honored
		// exactly as typed.
		if !filepath.IsAbs(targetDir) {
			targetDir = filepath.Join(root, targetDir)
		}
		force := flagBool("--force")
		consolePrint("Cloning repository into: " + targetDir + "\n")
		err := trace.CloneToDirectory(res.Data.RepoURL, res.Data.Branch, targetDir, force, consolePrint)
		switch {
		case err == trace.ErrTargetNotEmpty:
			// Safe, non-destructive default: the target directory
			// already has content (e.g. it's the very directory that
			// was just analyzed) - never re-clone over it or fail with
			// a raw git error. Just point project.json at it and move
			// on, exactly as the person most likely intended.
			consolePrint(i18n.T("cli.messages.dir_not_empty_skip_clone") + "\n")
		case err != nil:
			consolePrint("Could not clone the repository into the target directory: " + err.Error() + "\n")
			return
		}
		updated, err := trace.SetSuggestedProjectRepo(content, targetDir)
		if err != nil {
			consolePrint("Could not update project.json's repo path: " + err.Error() + "\n")
			return
		}
		content = updated
	}

	path, err := trace.WriteSuggestedProjectJSON(root, res.Data.SuggestedProjectName, content)
	if err != nil {
		consolePrint("Could not generate project.json: " + err.Error() + "\n")
		return
	}
	consolePrint("Generated project.json at: " + path + "\n")
}

// promptTargetDirectory asks "Enter target directory path to save the
// cloned repository:" and keeps asking until a usable path is given -
// pressing ENTER with no input asks again, exactly as the prompt spec
// requires ("iterativamente hasta que se proporcione una ubicación
// válida").
func promptTargetDirectory(scanner *bufio.Scanner) string {
	for {
		consolePrint(i18n.T("cli.messages.enter_target_dir") + " ")
		if !scanner.Scan() {
			return ""
		}
		path := strings.TrimSpace(scanner.Text())
		if path == "" {
			consolePrint(i18n.T("cli.messages.target_dir_required") + "\n")
			continue
		}
		return path
	}
}

// runChatTrace implements `/context-trace [--repo <url>] [--export pdf|md]
// [--output <path>]` inside `mova chat` - reuses the project/task
// already active in the chat session when --repo isn't given, same as
// `/diagram` and `/budget` already do (see runChatDiagram/runChatBudget).
func runChatTrace(root string, adapter core.Adapter, project, task, rest string, scanner *bufio.Scanner) {
	repoURL := extractChatFlag(rest, "--repo")
	exportFormat := extractChatFlag(rest, "--export")
	if exportFormat == "" {
		exportFormat = trace.DefaultExportFormat
	}
	output := extractChatFlag(rest, "--output")

	if repoURL == "" && project == "" {
		consolePrint("Usage: /context-trace  (requires an active project, or --repo <url>)\n")
		return
	}

	cwd, _ := os.Getwd()
	opts := trace.Options{
		Root: root, Cwd: cwd, Origin: "Chat",
		Project: project, Task: task,
		RepoURL: repoURL, ExportFormat: exportFormat, Output: output,
		IgnorePatterns:  extractChatIgnorePatterns(rest),
		PruneDocstrings: extractChatFlag(rest, "--prune-docstrings") != "" || strings.Contains(rest, "--prune-docstrings"),
		OnProgress:      printProgress,
		AgentClient:     "mova-cli",
		TargetModel:     core.TargetModelFor(root, project),
		PolicyAuthor:    core.ResolvePolicyAuthor(root, project),
	}

	traceAdapter := adapter
	if repoURL != "" {
		traceAdapter = nil
	}

	res, err := trace.Run(traceAdapter, opts)
	if err != nil {
		consolePrint("Error: " + err.Error() + "\n")
		return
	}
	consolePrint(res.Console + "\n")

	if res.Data.IsRemote && res.Data.SuggestedProjectJSON != "" {
		promptGenerateProjectJSON(scanner, res, root)
	}
}

// extractChatFlag looks for "--flag value" inside a one-line chat
// string (rest already comes without the command name) - a minimal
// version of flagStr for the chat door, which has no os.Args.
func extractChatFlag(rest, flag string) string {
	parts := strings.Fields(rest)
	for i, p := range parts {
		if p == flag && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return ""
}

// extractChatFlagRest is extractChatFlag's counterpart for a value
// that may itself contain spaces (--ignore's comma-separated pattern
// list, e.g. "--ignore docs/**, tests/**, *.lock"): everything after
// the flag up to the next recognized "--xxx" flag or end of line,
// instead of just the next whitespace-delimited token.
func extractChatFlagRest(rest, flag string) string {
	idx := strings.Index(rest, flag)
	if idx == -1 {
		return ""
	}
	tail := strings.TrimSpace(rest[idx+len(flag):])
	if next := regexp.MustCompile(`\s--[a-z-]+\b`).FindStringIndex(tail); next != nil {
		tail = tail[:next[0]]
	}
	return strings.TrimSpace(tail)
}

func extractChatIgnorePatterns(rest string) []string {
	raw := extractChatFlagRest(rest, "--ignore")
	if raw == "" {
		raw = extractChatFlagRest(rest, "-i")
	}
	if raw == "" {
		return nil
	}
	var out []string
	for _, part := range strings.Split(raw, ",") {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	return out
}
