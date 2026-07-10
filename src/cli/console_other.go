//go:build !windows

package main

import "os"

// consolePrint en Linux/macOS simplemente escribe UTF-8 directamente.
// Las terminales modernas lo soportan de forma nativa.
func consolePrint(s string) {
	os.Stdout.WriteString(s)
}