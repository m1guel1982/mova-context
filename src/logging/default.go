// default.go — a process-wide default Logger. Every entrypoint
// (cli/main.go, mcp/server.go's StartStdio, http/server.go's
// StartServer) calls SetDefault(Open(root)) exactly once at startup;
// every other package (jobs, orchestrator, documents...) that wants to
// log calls logging.L() instead of threading a *Logger through every
// function signature. Before SetDefault is ever called, L() returns a
// disabled Logger — logging calls are always safe, never nil, and are
// simply no-ops until a real entrypoint opens one.
package logging

import "sync/atomic"

var defaultLogger atomic.Pointer[Logger]

// SetDefault installs l as the process-wide default Logger.
func SetDefault(l *Logger) {
	defaultLogger.Store(l)
}

// L returns the process-wide default Logger — never nil. Safe to call
// from any package at any time, including before SetDefault (returns a
// disabled Logger in that case).
func L() *Logger {
	if l := defaultLogger.Load(); l != nil {
		return l
	}
	return &Logger{cfg: defaultConfig()}
}
