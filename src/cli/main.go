// main.go — Mova Context CLI v3 (Unified Engine)
//
// Build: go build -o mova ./cli   (desde la raíz del repo, módulo "mova")
//
// main() only bootstraps (root, logging) and hands off to dispatch()
// (dispatch.go) — the actual `switch os.Args[1]` command table. Split
// out purely to keep this file under 300 lines; dispatch() is still
// the single place every command is routed from.
package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"mova.local/core/focus/astfilter"
	"mova.local/i18n"
	"mova.local/logging"
	"mova.local/runtime"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		return
	}

	root, err := runtime.FindRoot()
	if err != nil {
		die(err.Error())
	}
	logger := logging.Open(root)
	logging.SetDefault(logger)
	defer logger.Close()
	logger.Info("cli", "command: %s", strings.Join(os.Args[1:], " "))

	// i18n.Init loads config/lang/{es,en}.json + lang_active.json and
	// starts the hot-reload poller (see mova.local/i18n) - called once
	// here, at the CLI's single entry point, so every subcommand (and
	// the same call in mcp.Process / http's server bootstrap) shares
	// one translator. Errors here are non-fatal: i18n.T falls back to
	// the literal key when nothing loaded, so a broken config/lang/
	// tree degrades output readability, not availability.
	if err := i18n.Init(root); err != nil {
		logger.Info("i18n", "could not load config/lang/: %v", err)
	}

	// astfilter.Init loads config/ast/keywords_*.json (embedded
	// defaults + optional project-level overrides under root) — see
	// mova.local/core/focus/astfilter's config.go. Called here, right
	// after i18n.Init, for the exact same reason: this is the single
	// entry point every subcommand (chat, context-trace, mcp, ui, the
	// http server — see dispatch()) is routed through, so one call
	// here is enough for every door. Errors are non-fatal: a language
	// whose JSON fails to parse just isn't available for AST filtering
	// (Extract/Strip report ok=false for it), it doesn't break
	// anything else.
	if err := astfilter.Init(root); err != nil {
		logger.Info("astfilter", "could not fully load config/ast/: %v", err)
	}

	dispatch(root)
}

// ── generic argument/flag helpers, shared by every *_cmd.go file ──────

func arg(i int, def string) string {
	if i < len(os.Args) {
		return os.Args[i]
	}
	return def
}

// valueFlags lists every long flag that consumes the NEXT argument as
// its value. positionalArgs uses it to skip those values, so a flag's
// value is never mistaken for a positional argument (project/task).
// Boolean flags (--focus, --diagram, --prune-docstrings, ...) are
// deliberately absent: they consume nothing.
var valueFlags = map[string]bool{
	"--export":           true,
	"--output":           true,
	"--path":             true,
	"--repo":             true,
	"--branch":           true,
	"--task":             true,
	"--ignore":           true,
	"--port":             true,
	"--policies_include": true,
	"--policies_exclude": true,
}

// positionalArgs returns the first two non-flag arguments starting at
// os.Args[startIdx] — used by commands like `mova budget [project] [task]
// --focus` that combine positional args with boolean flags, so a flag
// occupying what would otherwise be a positional slot is never mistaken
// for it (e.g. "mova budget my-project --focus" must not read "--focus"
// as the task name).
func positionalArgs(startIdx int) (first, second string) {
	var found []string
	for i := startIdx; i < len(os.Args); i++ {
		if strings.HasPrefix(os.Args[i], "--") {
			// Skip this flag AND, for the flags that take one, its
			// value — otherwise `mova context-trace proj --export md`
			// reads "md" as the task name (see valueFlags).
			if valueFlags[os.Args[i]] {
				i++
			}
			continue
		}
		found = append(found, os.Args[i])
	}
	if len(found) > 0 {
		first = found[0]
	}
	if len(found) > 1 {
		second = found[1]
	}
	return first, second
}

func needArg(i int, label string) string {
	if i < len(os.Args) {
		return os.Args[i]
	}
	die("missing argument: " + label)
	return ""
}

func flagBool(flag string) bool {
	for _, a := range os.Args {
		if a == flag {
			return true
		}
	}
	return false
}

func flagStr(flag, def string) string {
	for i, a := range os.Args {
		if a == flag && i+1 < len(os.Args) {
			return os.Args[i+1]
		}
	}
	return def
}

// flagStrAll collects EVERY occurrence of flag (e.g. multiple
// "--ignore x --ignore y" invocations), each split on commas - used
// by --ignore so both "acepta cadenas separadas por comas" and
// "invocaciones múltiples" are supported natively, per spec.
func flagStrAll(flag string) []string {
	var out []string
	for i, a := range os.Args {
		if a == flag && i+1 < len(os.Args) {
			for _, part := range strings.Split(os.Args[i+1], ",") {
				part = strings.TrimSpace(part)
				if part != "" {
					out = append(out, part)
				}
			}
		}
	}
	return out
}

// flagIgnorePatterns is --ignore's own reader: accepts the flag as
// "--ignore" or its short form "-i", either given once with several
// comma-separated glob patterns ("--ignore \"docs/**, tests/**,
// *.lock\"") or repeated ("--ignore docs/** --ignore tests/**") — both
// forms merge into the same list (flagStrAll already splits on comma
// per occurrence; this just also checks "-i" and combines both).
func flagIgnorePatterns() []string {
	return append(flagStrAll("--ignore"), flagStrAll("-i")...)
}

func flagInt(flag string, def int) int {
	s := flagStr(flag, "")
	if s == "" {
		return def
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return v
}

func must(err error) {
	if err != nil {
		die(err.Error())
	}
}

func die(msg string) {
	fmt.Fprintln(os.Stderr, "error: "+msg)
	os.Exit(1)
}
