// git_fetch.go — RepositoryFetcher: an extensible abstraction for
// bringing a remote repository into a local temporary directory before
// analyzing it (see analyzer.go's AnalyzeRemote). GitHubFetcher/
// GitLabFetcher/GenericGitFetcher share the same implementation today
// (cloning via the system's `git` binary, --depth 1 to skip
// unnecessary history) - kept as separate types because each provider
// is the natural place to add provider-specific logic later (for
// example, resolving the default branch via the GitHub/GitLab REST API
// instead of letting `git` detect it) without touching the others.
package trace

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"mova.local/i18n"
)

// RepositoryFetcher brings a remote repository into a local temporary
// directory or handles a local path. cleanup() must ALWAYS be called
// (see Run in trace.go, which does so with defer) to remove that directory
// as soon as the analysis finishes - a temporary clone is never left behind on disk.
type RepositoryFetcher interface {
	// Fetch clones url or resolves local path (branchHint if given, or the
	// repository's default branch when branchHint == "") into a directory.
	// Returns the directory path, the branch actually used, and a cleanup
	// function to call when done.
	Fetch(urlOrPath, branchHint string) (dir string, branch string, cleanup func(), err error)
}

// LocalDirectoryFetcher maneja rutas locales del sistema de archivos sin clonar.
type LocalDirectoryFetcher struct{}

func (LocalDirectoryFetcher) Fetch(urlOrPath, branchHint string) (string, string, func(), error) {
	cleanPath := cleanInputPath(urlOrPath)
	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		absPath = cleanPath
	}

	// Como es una carpeta local del usuario, no hay directorio temporal que eliminar.
	cleanup := func() {}

	branch := branchHint
	if branch == "" {
		branch = detectDefaultBranch(absPath)
	}
	if branch == "" || branch == "HEAD" {
		branch = "local"
	}

	return absPath, branch, cleanup, nil
}

// NewFetcherFor picks the right RepositoryFetcher for url's host or local path.
func NewFetcherFor(urlOrPath string) RepositoryFetcher {
	cleanPath := cleanInputPath(urlOrPath)

	// Detección prioritaria de directorio local (Windows / Unix)
	if isDir(cleanPath) {
		return LocalDirectoryFetcher{}
	}

	switch {
	case strings.Contains(urlOrPath, "github.com"):
		return GitHubFetcher{}
	case strings.Contains(urlOrPath, "gitlab.com"):
		return GitLabFetcher{}
	default:
		return GenericGitFetcher{}
	}
}

// GenericGitFetcher clones any URL `git clone` supports (https://,
// git@..., ssh://, file://) - the base case GitHubFetcher and
// GitLabFetcher reuse.
type GenericGitFetcher struct{}

// GitHubFetcher - today delegates entirely to GenericGitFetcher;
// exists as its own type so RepositoryFetcher stays extensible per
// provider (see the file header) without analyzer.go ever needing to
// know the difference.
type GitHubFetcher struct{}

// GitLabFetcher - same reasoning as GitHubFetcher.
type GitLabFetcher struct{}

func (GitHubFetcher) Fetch(url, branchHint string) (string, string, func(), error) {
	return cloneGeneric(url, branchHint)
}

func (GitLabFetcher) Fetch(url, branchHint string) (string, string, func(), error) {
	return cloneGeneric(url, branchHint)
}

func (GenericGitFetcher) Fetch(url, branchHint string) (string, string, func(), error) {
	return cloneGeneric(url, branchHint)
}

// cloneGeneric does the real work: creates the temporary directory,
// runs `git clone --depth 1 [-b <branch>] <url> <dir>`, and if it
// fails due to missing permissions (a private repository with no
// configured credentials) returns the exact instructional message
// requested for this command.
func cloneGeneric(url, branchHint string) (string, string, func(), error) {
	dir, err := os.MkdirTemp("", "mova-trace-*")
	if err != nil {
		return "", "", nil, fmt.Errorf("could not create the temporary directory for cloning: %w", err)
	}
	cleanup := func() { os.RemoveAll(dir) }

	args := []string{"clone", "--depth", "1"}
	if branchHint != "" {
		args = append(args, "-b", branchHint)
	}
	args = append(args, url, dir)

	cmd := exec.Command("git", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		cleanup()
		if looksLikeAuthFailure(string(out)) {
			return "", "", nil, fmt.Errorf(
				"[ERROR] Could not access the private repository. Please make sure you are already logged in on this machine (for example: 'gh auth login', 'glab auth login', or that your SSH/Git credentials are configured).")
		}
		return "", "", nil, fmt.Errorf("could not clone repository %q: %s", url, strings.TrimSpace(string(out)))
	}

	branch := branchHint
	if branch == "" {
		branch = detectDefaultBranch(dir)
	}
	return dir, branch, cleanup, nil
}

// CloneToDirectory clones url straight into targetDir (creating it if
// needed) - used by the CLI's interactive "Enter target directory path
// to save the cloned repository" flow (see cli/trace_cmd.go) to
// persist a remote analysis's repository at a definitive location the
// person chooses, AFTER the original temporary clone (see
// cloneGeneric) has already been analyzed and removed by Run's normal
// cleanup. This is a fresh clone, not a move/copy of the now-deleted
// temporary directory - see this file's header for why: cleanup()
// always runs unconditionally right after analysis (no temporary
// clone is ever left on disk, for every door, interactive or not),
// so "copying from the temp folder" is not actually available by the
// time an interactive answer could arrive; re-cloning gets the person
// the same end result (their own permanent copy of the repository)
// without ever risking a leaked temporary directory on the other,
// non-interactive doors (MCP/HTTP).
// ErrTargetNotEmpty is returned by CloneToDirectory when targetDir
// already exists and has content and force is false - a caller (see
// cli/trace_cmd.go's promptGenerateProjectJSON) should treat this as
// "skip cloning, just write project.json pointing at this existing
// directory", never as a fatal error: the reported bug was exactly a
// person pointing the target directory at the SAME local path that
// was just analyzed, which is a completely normal, non-destructive
// thing to do.
var ErrTargetNotEmpty = fmt.Errorf("target directory already exists and has content")

// CloneToDirectory clones url straight into targetDir. If targetDir
// already exists and is non-empty: with force=false it returns
// ErrTargetNotEmpty without touching anything on disk; with
// force=true it removes the existing content first and calls notice
// (if non-nil) with a clear, explicit message before doing so - the
// person is never left wondering why their directory changed, and
// this never requires them to pass a --force flag just to get past a
// crash (see cli/trace_cmd.go's default behavior: it prefers the
// safe, non-destructive path automatically).
func CloneToDirectory(url, branch, targetDir string, force bool, notice func(string)) error {
	if info, err := os.Stat(targetDir); err == nil {
		if !info.IsDir() {
			return fmt.Errorf("target path %q exists and is not a directory", targetDir)
		}
		entries, _ := os.ReadDir(targetDir)
		if len(entries) > 0 {
			if !force {
				return ErrTargetNotEmpty
			}
			if notice != nil {
				notice(i18n.T("cli.messages.overwriting_dir", map[string]any{"path": targetDir}))
			}
			if err := os.RemoveAll(targetDir); err != nil {
				return fmt.Errorf("could not clear existing target directory %q: %w", targetDir, err)
			}
			if err := os.MkdirAll(targetDir, 0o755); err != nil {
				return fmt.Errorf("could not recreate target directory %q: %w", targetDir, err)
			}
		}
	} else if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return fmt.Errorf("could not create target directory %q: %w", targetDir, err)
	}

	args := []string{"clone", "--depth", "1"}
	if branch != "" && branch != "local" {
		args = append(args, "-b", branch)
	}
	args = append(args, url, targetDir)

	cmd := exec.Command("git", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		if looksLikeAuthFailure(string(out)) {
			return fmt.Errorf("[ERROR] Could not access the private repository. Please make sure you are already logged in on this machine (for example: 'gh auth login', 'glab auth login', or that your SSH/Git credentials are configured)")
		}
		return fmt.Errorf("could not clone repository %q into %q: %s", url, targetDir, strings.TrimSpace(string(out)))
	}
	return nil
}

// LooksLikeGitURL reports whether urlOrPath is a remote git URL (as
// opposed to a local filesystem path) - used by the CLI's interactive
// flow to decide between the "remote repository" (ask for a target
// directory, clone there) and "local repository" (just point
// project.json at the existing path) cases from the prompt's spec.
func LooksLikeGitURL(urlOrPath string) bool {
	if isDir(cleanInputPath(urlOrPath)) {
		return false
	}
	lower := strings.ToLower(urlOrPath)
	return strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") ||
		strings.HasPrefix(lower, "git@") || strings.HasPrefix(lower, "ssh://") || strings.HasPrefix(lower, "git://")
}

// cleanInputPath limpia comillas, espacios y convierte barras de ruta al OS actual.
func cleanInputPath(path string) string {
	s := strings.TrimSpace(path)
	s = strings.Trim(s, "\"'`")
	s = filepath.FromSlash(s)
	return filepath.Clean(s)
}

// isDir comprueba si la ruta limpia o absoluta existe y es un directorio en el sistema.
func isDir(path string) bool {
	if path == "" {
		return false
	}

	if info, err := os.Stat(path); err == nil && info.IsDir() {
		return true
	}

	if abs, err := filepath.Abs(path); err == nil {
		if info, err := os.Stat(abs); err == nil && info.IsDir() {
			return true
		}
	}

	return false
}

// looksLikeAuthFailure recognizes the typical messages git/ssh return
// when a repository is private and no credentials are available.
func looksLikeAuthFailure(output string) bool {
	lower := strings.ToLower(output)
	markers := []string{
		"permission denied", "authentication failed", "could not read username",
		"could not read password", "403", "401",
		"fatal: could not read",
	}
	for _, m := range markers {
		if strings.Contains(lower, m) {
			return true
		}
	}
	return false
}

// detectDefaultBranch asks git, inside the clone or local directory, which
// branch ended up active (HEAD) - avoids guessing "main" vs "master".
func detectDefaultBranch(dir string) string {
	cmd := exec.Command("git", "-C", dir, "rev-parse", "--abbrev-ref", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return "main"
	}
	branch := strings.TrimSpace(string(out))
	if branch == "" || branch == "HEAD" {
		return "main"
	}
	return branch
}
