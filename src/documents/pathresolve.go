// pathresolve.go implements the path-resolution intelligence shared by
// every file/directory tool (write_file, patch_file, create_directory,
// generate_word_contract, generate_excel_report, generate_pdf_document,
// generate_vector_graphic, trigger_diffusion_image):
//
//  1. An absolute path (Unix or Windows style) is accepted ONLY if it lies
//     inside the project's repo (see confine/WithinRepo). Anything outside —
//     absolute, "../" traversal, or a symlink escaping the repo — is
//     rejected with ErrOutsideRepo, on every door (chat/MCP/HTTP).
//  2. No path given → the project's `repo` (from project.json), same
//     default as before.
//  3. A bare directory name with no path separators ("config", "reportes")
//     is treated as "does this already exist somewhere in the project?" —
//     the repo tree is searched for a directory with that exact name. Zero
//     matches → create it fresh under repo. Exactly one match → reuse it.
//     More than one → the tool returns the candidate list instead of
//     guessing, so the caller can ask which one was meant.
//  4. An explicit path with segments ("output/reports", "output/reports/x.md")
//     is unambiguous by construction and resolved directly under repo.
//
// Directories and files are resolved slightly differently: ResolveDirectoryPath
// treats the *entire* requested string as a directory name (used by
// create_directory), while ResolveFilePath treats only the directory
// *portion* of the requested string as search-worthy — the final path
// segment is always the file itself, never something to search for.
package documents

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
)

var windowsDriveRe = regexp.MustCompile(`^[A-Za-z]:[\\/]`)
var windowsUNCRe = regexp.MustCompile(`^\\\\[^\\]+\\`)

// IsAbsCrossPlatform reports whether p is an absolute path in either Unix
// style (leading "/") or Windows style (drive letter or UNC share) —
// recognized the same way regardless of which OS this binary runs on, so a
// path typed in chat parses consistently everywhere.
func IsAbsCrossPlatform(p string) bool {
	if strings.HasPrefix(p, "/") {
		return true
	}
	return windowsDriveRe.MatchString(p) || windowsUNCRe.MatchString(p)
}

// NormalizeAbsPath prepares a recognized absolute path for the current host
// OS, and rejects a path style the host genuinely cannot satisfy — a
// Windows drive letter has no meaning on Linux/macOS, so that combination
// gets a clear, honest error instead of silently writing somewhere wrong.
func NormalizeAbsPath(p string) (string, error) {
	isWindowsStyle := windowsDriveRe.MatchString(p) || windowsUNCRe.MatchString(p)
	if isWindowsStyle && runtime.GOOS != "windows" {
		return "", fmt.Errorf(
			"la ruta %q usa el formato de Windows (letra de unidad), pero este servidor de Mova Context corre en %s — usá una ruta nativa de %s o una ruta relativa al proyecto",
			p, runtime.GOOS, runtime.GOOS)
	}
	if isWindowsStyle {
		p = strings.ReplaceAll(p, "\\", "/")
	}
	return filepath.Clean(p), nil
}

// ResolveDirectoryPath resolves requested as a directory in its entirety —
// used by create_directory. See the package doc for the resolution order.
func ResolveDirectoryPath(root, repo, requested string) (resolved string, ambiguous []string, err error) {
	repoDir := repoDirFor(root, repo)

	if requested == "" {
		if err := os.MkdirAll(repoDir, 0o755); err != nil {
			return "", nil, fmt.Errorf("no se pudo crear el directorio de trabajo %q: %w", repoDir, err)
		}
		return repoDir, nil, nil
	}

	if IsAbsCrossPlatform(requested) {
		normalized, err := NormalizeAbsPath(requested)
		if err != nil {
			return "", nil, err
		}
		return confine(repoDir, normalized)
	}

	cleaned := toSlash(requested)
	if !strings.Contains(cleaned, "/") {
		resolved, ambiguous, err := searchOrPlaceUnder(repoDir, cleaned)
		if err != nil || len(ambiguous) > 0 {
			return resolved, ambiguous, err
		}
		return confine(repoDir, resolved)
	}
	return confine(repoDir, filepath.Join(repoDir, filepath.FromSlash(cleaned)))
}

// ResolveFilePath resolves requested as a file: only the directory portion
// (everything before the final path segment) is search-worthy when it's a
// bare name — the file's own name is never treated as a directory to
// search for. Used by write_file, patch_file, read_file,
// read_document_layer, and every generate_* tool.
func ResolveFilePath(root, repo, requested string) (resolved string, ambiguous []string, err error) {
	repoDir := repoDirFor(root, repo)

	if requested == "" {
		return "", nil, fmt.Errorf("filename is required")
	}

	if IsAbsCrossPlatform(requested) {
		normalized, err := NormalizeAbsPath(requested)
		if err != nil {
			return "", nil, err
		}
		return confine(repoDir, normalized)
	}

	cleaned := toSlash(requested)
	dir, base := path.Split(cleaned)
	dir = strings.TrimSuffix(dir, "/")

	switch {
	case dir == "":
		return confine(repoDir, filepath.Join(repoDir, base))
	case !strings.Contains(dir, "/"):
		resolvedDir, ambiguous, err := searchOrPlaceUnder(repoDir, dir)
		if err != nil || len(ambiguous) > 0 {
			return "", ambiguous, err
		}
		return confine(repoDir, filepath.Join(resolvedDir, base))
	default:
		return confine(repoDir, filepath.Join(repoDir, filepath.FromSlash(dir), base))
	}
}

// ErrOutsideRepo is returned when a requested path resolves outside the
// project's repo (absolute path elsewhere, "../" traversal, or a symlink
// that points out of the repo). Every file/directory tool goes through
// ResolveFilePath/ResolveDirectoryPath, so this is the single boundary.
var ErrOutsideRepo = fmt.Errorf("ruta fuera del repo del proyecto: denegada")

// confine is the repo boundary: candidate must stay inside repoDir after
// cleaning and after resolving symlinks of the deepest existing ancestor.
func confine(repoDir, candidate string) (string, []string, error) {
	if WithinRepo(repoDir, candidate) {
		return filepath.Clean(candidate), nil, nil
	}
	return "", nil, fmt.Errorf("%w: %s", ErrOutsideRepo, candidate)
}

// WithinRepo reports whether candidate is inside repoDir (or is repoDir),
// following symlinks of every existing path component.
func WithinRepo(repoDir, candidate string) bool {
	base := realPath(filepath.Clean(repoDir))
	target := realPath(filepath.Clean(candidate))
	rel, err := filepath.Rel(base, target)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}

// realPath resolves symlinks of the longest existing prefix of p and
// re-appends the non-existing remainder (files about to be created).
func realPath(p string) string {
	abs, err := filepath.Abs(p)
	if err != nil {
		abs = p
	}
	rest := ""
	cur := abs
	for {
		if r, err := filepath.EvalSymlinks(cur); err == nil {
			if rest == "" {
				return r
			}
			return filepath.Join(r, rest)
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			return abs
		}
		if rest == "" {
			rest = filepath.Base(cur)
		} else {
			rest = filepath.Join(filepath.Base(cur), rest)
		}
		cur = parent
	}
}

func toSlash(p string) string {
	return strings.ReplaceAll(p, "\\", "/")
}

// searchOrPlaceUnder looks for an existing directory named name anywhere
// under repoDir. Zero matches → a fresh path directly under repoDir. One
// match → reuse it. More than one → ambiguous, caller must ask.
func searchOrPlaceUnder(repoDir, name string) (string, []string, error) {
	matches, err := findDirsByName(repoDir, name)
	if err != nil {
		return "", nil, err
	}
	switch len(matches) {
	case 0:
		return filepath.Join(repoDir, name), nil, nil
	case 1:
		return matches[0], nil, nil
	default:
		return "", matches, nil
	}
}

func repoDirFor(root, repo string) string {
	switch {
	case repo == "" || repo == ".":
		return root
	case IsAbsCrossPlatform(repo):
		if normalized, err := NormalizeAbsPath(repo); err == nil {
			return normalized
		}
		return repo
	default:
		return filepath.Join(root, repo)
	}
}

// findDirsByName walks searchRoot looking for every directory whose
// basename exactly equals name. Unreadable subtrees are skipped rather than
// failing the whole search.
func findDirsByName(searchRoot, name string) ([]string, error) {
	var matches []string
	err := filepath.WalkDir(searchRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() && d.Name() == name {
			matches = append(matches, path)
		}
		return nil
	})
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	sort.Strings(matches)
	return matches, nil
}
