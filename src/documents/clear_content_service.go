// clear_content_service.go — ClearContent: empties an EXISTING file's
// content without removing it — the fix for a real bug found in QA:
// "elimina el contenido del archivo X" / "elimina el texto que existe
// en X" used to go through full file deletion (documents.Delete)
// instead, because the delete detector didn't tell "remove the file"
// and "empty the file" apart (see documents.DetectDeleteIntent's
// ClearContentOnly field, delete_intent.go).
package documents

import (
	"fmt"
	"os"
)

// ClearContent resolves target the same way ResolveExistingFile
// already does (so it goes through the identical path rules as
// Save/Delete/Rename — cross-platform absolute paths included) and
// overwrites it with an empty string. Refuses to act on something that
// doesn't exist yet — clearing a file that isn't there isn't a
// meaningful operation.
func ClearContent(root, repo, target string) (message string, err error) {
	full, ambiguous, exists, err := ResolveExistingFile(root, repo, target)
	if err != nil {
		return "", err
	}
	if len(ambiguous) > 0 {
		return FormatAmbiguousMessage(target, ambiguous), nil
	}
	if !exists {
		return fmt.Sprintf("%q does not exist.", target), nil
	}
	if err := os.WriteFile(full, []byte(""), 0644); err != nil {
		return "", err
	}
	return fmt.Sprintf("✓ content cleared: %s", full), nil
}
