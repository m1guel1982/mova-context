// path.go — fully cross-platform output path (--output) resolution
// for context-trace, following the same rule as the rest of Mova
// Context (see documents/pathresolve.go): paths are never
// hand-concatenated with "/" or "\" - everything goes through
// filepath.Clean(filepath.FromSlash(...)).
//
// Accepted formats for --output / "output" (CLI, MCP, HTTP, Chat):
//
//	(empty)                -> see runLocal/runRemote in trace.go
//	                           (analyzed project's own root, or the
//	                           current directory in remote mode)
//	"path://C:/reports"     -> optional "path://" prefix, stripped
//	"C:\\reports\\mova"     -> Windows absolute path (backslash)
//	"D:/reports"            -> Windows absolute path (forward slash)
//	"/mnt/reports"          -> Unix/WSL absolute path
//	"/Users/ana/reports"    -> macOS absolute path
//	"reports/mova"          -> relative to the directory the CLI was
//	                           run from (or to the repo, for HTTP/MCP/Chat)
package trace

import (
	"path/filepath"
	"regexp"
	"strings"
)

var windowsDriveRe = regexp.MustCompile(`^[A-Za-z]:[\\/]`)

// NormalizeOutputPath cleans requested (stripping an optional
// "path://" prefix) and normalizes it for the current OS. cwd is the
// base path to use when requested is relative (the CLI's current
// directory, or whatever the HTTP/MCP/Chat caller decides).
func NormalizeOutputPath(requested, cwd string) string {
	if requested == "" {
		return filepath.Clean(filepath.FromSlash(cwd))
	}

	p := strings.TrimPrefix(requested, "path://")
	p = strings.TrimSpace(p)

	// Windows absolute path ("/" or "\") or Unix/macOS absolute path
	// ("/...") - respected as-is, never prefixed with cwd, same as
	// documents.IsAbsCrossPlatform already does for the rest of Mova
	// Context's file tools.
	if windowsDriveRe.MatchString(p) {
		return filepath.Clean(strings.ReplaceAll(p, "\\", "/"))
	}
	if strings.HasPrefix(p, "/") {
		return filepath.Clean(filepath.FromSlash(p))
	}

	return filepath.Clean(filepath.Join(filepath.FromSlash(cwd), filepath.FromSlash(p)))
}
