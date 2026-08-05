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

	dispatch(root)
}

// ── generic argument/flag helpers, shared by every *_cmd.go file ──────

func arg(i int, def string) string {
	if i < len(os.Args) {
		return os.Args[i]
	}
	return def
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
		if !strings.HasPrefix(os.Args[i], "--") {
			found = append(found, os.Args[i])
		}
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
