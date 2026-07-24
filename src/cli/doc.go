// Package main is the Mova Context CLI — the thin dispatcher that wires
// together every package (core, adapters, mcp, http, runtime) into the
// `mova` command.
//
// Este archivo (doc.go) es, a propósito, el único de este paquete cuyo
// comentario queda pegado a "package main" — así `go doc ./cli` siempre
// muestra esta vista general en vez de la cabecera de un archivo
// cualquiera (antes ambiguo: cualquier archivo con un comentario pegado a
// "package main" podía "ganar" según el orden alfabético). Los demás
// archivos conservan sus comentarios descriptivos, separados por una línea
// en blanco antes de "package main" para que sigan siendo comentarios de
// archivo, no candidatos a comentario de paquete.
//
// Responsabilidad de cada archivo:
//
//	main.go            — dispatcher de subcomandos (run, memory, list, ...)
//	help.go            — texto de ayuda (`mova` sin argumentos)
//	adapter_select.go  — decide file vs. db adapter (nunca lo decide core)
//	memory_mgmt.go     — memory-clear / memory-config
//	models_cmd.go      — config / show config / install / model-list / remove
//	budget_cmd.go      — mova budget (estimación local de tokens/costo, ver mova.local/budget)
//	chat_cmd.go        — mova chat (REPL con modelos locales)
//	console_*.go       — helpers de terminal específicos por SO
package main
