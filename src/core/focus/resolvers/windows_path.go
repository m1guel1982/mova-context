// windows_path.go — comparableHostPath: the one normalization helper
// that makes path comparisons (exclude matching, "is this the repo
// root" checks) immune to Windows drive-letter case differences,
// without ever touching the rest of a path's casing (which stays
// significant on case-sensitive filesystems). Split out of fsutil.go
// purely to keep it under the 300-line limit — see fsutil.go's
// relOrBase and exclude.go's newExcludeMatcher/excludesPath for where
// this is actually used.
package resolvers

import "strings"

// comparableHostPath is normalizeHostPath PLUS case-folding of a
// leading Windows drive letter, used ONLY when comparing two already
// host-absolute paths for prefix/equality (exclude matching, "is this
// the repo root" checks) — never for the actual os.Stat/WalkDir calls
// themselves, which must keep native casing (a Linux/macOS filesystem
// is case-SENSITIVE; folding case there would silently merge two
// different files/directories). Windows itself is case-INSENSITIVE
// for both the drive letter and the rest of the path, but folding the
// entire path risks the same false-merge on a case-sensitive network
// share mounted under Windows — folding only the drive letter is the
// narrow, safe fix for the one real ambiguity Windows itself
// guarantees (E: vs e:), without touching the rest of the path.
//
// This exists because of a real, reported bug: a project.json whose
// "repo" used an absolute Windows path with a drive letter caused
// focus/exclude to silently stop matching.
func comparableHostPath(target string) string {
	norm := normalizeHostPath(target)
	if winDriveRe.MatchString(target) && len(norm) > 0 {
		return strings.ToLower(norm[:1]) + norm[1:]
	}
	return norm
}
