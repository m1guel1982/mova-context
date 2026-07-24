package documents

import "os"

// CreateDirectory creates path and every missing parent directory in one
// call — the Go standard library's os.MkdirAll is already recursive
// ("mkdir -p" semantics) and cross-platform: it uses the host OS's native
// path separator handling, so the same call works unchanged on Linux,
// macOS, and Windows. A no-op (no error) if the directory already exists.
func CreateDirectory(path string) error {
	if err := os.MkdirAll(path, 0o755); err != nil {
		return err
	}
	return nil
}
