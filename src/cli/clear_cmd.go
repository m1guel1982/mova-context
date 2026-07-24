// clear_cmd.go — `/clear`, clears the chat's terminal screen. Shells out
// to the OS's own clear command (`cls` on Windows, `clear` elsewhere)
// rather than printing an ANSI escape sequence directly: Windows consoles
// don't reliably interpret VT100 escapes unless virtual terminal
// processing has been explicitly enabled (see console_windows.go, which
// doesn't), so the native command is the one approach that actually
// clears the screen on every terminal this project already supports.
package main

import (
	"os"
	"os/exec"
	"runtime"
)

// clearScreen clears the terminal. Best-effort: if the underlying
// command isn't available for some reason, it silently does nothing
// rather than erroring out a chat session over a cosmetic feature.
func clearScreen() {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/c", "cls")
	} else {
		cmd = exec.Command("clear")
	}
	cmd.Stdout = os.Stdout
	_ = cmd.Run()
}
