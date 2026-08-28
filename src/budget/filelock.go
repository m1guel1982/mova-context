// filelock.go — a tiny per-path mutex so concurrent callers (CLI chat,
// the HTTP API, and the MCP server can all be serving requests for the
// SAME project at the same time) never race on a read-modify-write of
// mova-token-history.json, mova-spend.json, or mova-context-cache.json.
//
// This does not replace the OS file lock a second process on another
// machine might need — it only protects goroutines inside this one
// process, which is exactly where the race actually happens: every
// invocation door (see chat_helpers.go, mcp/chat_tool.go, http/server.go)
// shares this same binary and address space.
package budget

import (
	"path/filepath"
	"sync"
)

// fileLocks holds one *sync.Mutex per absolute path, created lazily and
// kept for the life of the process — cheap even with many projects,
// since each entry is a single pointer plus an unlocked mutex.
var (
	fileLocksMu sync.Mutex
	fileLocks   = map[string]*sync.Mutex{}
)

// lockFor returns the mutex guarding path, creating it on first use.
func lockFor(path string) *sync.Mutex {
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = path
	}
	fileLocksMu.Lock()
	m, ok := fileLocks[abs]
	if !ok {
		m = &sync.Mutex{}
		fileLocks[abs] = m
	}
	fileLocksMu.Unlock()
	return m
}

// withFileLock serializes every read-modify-write cycle against path
// across goroutines in this process, so two concurrent requests for the
// same project (e.g. two chat_completion calls arriving on the HTTP API
// at once) can never interleave and drop one of the updates.
func withFileLock(path string, fn func() error) error {
	m := lockFor(path)
	m.Lock()
	defer m.Unlock()
	return fn()
}
