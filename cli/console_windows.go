//go:build windows

package main

import (
	"os"
	"syscall"
	"unicode/utf16"
	"unsafe"
)

var (
	kernel32      = syscall.NewLazyDLL("kernel32.dll")
	writeConsoleW = kernel32.NewProc("WriteConsoleW")
	getStdHandle  = kernel32.NewProc("GetStdHandle")
	getConsMode   = kernel32.NewProc("GetConsoleMode")

	hStdOut      uintptr
	isConsole    bool
	bomWritten   bool
)

// UTF-8 BOM: le dice a clip, Notepad, Excel y otras apps Windows
// que el stream es UTF-8 y no CP1252.
var utf8BOM = []byte{0xEF, 0xBB, 0xBF}

func init() {
	h, _, _ := getStdHandle.Call(uintptr(0xFFFFFFF5)) // STD_OUTPUT_HANDLE
	hStdOut = h
	var mode uint32
	r, _, _ := getConsMode.Call(h, uintptr(unsafe.Pointer(&mode)))
	isConsole = r != 0
}

// consolePrint escribe s en stdout con soporte completo de Unicode:
//
//   - Consola real (CMD, PowerShell, Windows Terminal):
//     usa WriteConsoleW con UTF-16 nativo → tildes correctas sin importar chcp.
//
//   - Pipe o redirect (| clip, > archivo.txt, | more):
//     escribe BOM UTF-8 al inicio del stream la primera vez, luego UTF-8 puro.
//     El BOM hace que clip, Notepad y apps Windows lean el stream como UTF-8
//     en lugar de asumir CP1252, que es lo que causaba el "CÃ³digo" → "Código".
func consolePrint(s string) {
	if isConsole {
		writeUTF16(s)
		return
	}
	// Pipe/redirect: emitir BOM una sola vez al inicio del stream.
	if !bomWritten {
		os.Stdout.Write(utf8BOM)
		bomWritten = true
	}
	os.Stdout.WriteString(s)
}

func writeUTF16(s string) {
	encoded := utf16.Encode([]rune(s))
	if len(encoded) == 0 {
		return
	}
	const chunk = 8192
	for i := 0; i < len(encoded); i += chunk {
		end := i + chunk
		if end > len(encoded) {
			end = len(encoded)
		}
		part := encoded[i:end]
		var written uint32
		writeConsoleW.Call(
			hStdOut,
			uintptr(unsafe.Pointer(&part[0])),
			uintptr(len(part)),
			uintptr(unsafe.Pointer(&written)),
			0,
		)
	}
}