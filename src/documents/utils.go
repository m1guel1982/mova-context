package documents

import (
	"os"
	"path/filepath"
)

// ensureDir ensures that the parent directory for a given path exists.
func ensureDir(filePath string) error {
	dir := filepath.Dir(filePath)
	if dir != "" && dir != "." {
		return os.MkdirAll(dir, 0755)
	}
	return nil
}