package main

import (
	"mova.local/core"
	"mova.local/logging"
)

// enableBackgroundGraphs pone el generador de grafos en modo segundo plano
// (ver core.GraphAsync): en procesos de larga vida (chat, MCP, HTTP) el
// render nunca debe retrasar el arranque, un turno ni la salida. notify
// recibe el estado de cada grafo cuando termina.
func enableBackgroundGraphs(notify func(string)) {
	core.GraphAsync = true
	core.GraphNotify = notify
}

// logGraphNotice: en MCP/HTTP no hay terminal (en stdio, stdout es el canal
// del protocolo), así que los avisos van al log del servidor.
func logGraphNotice(msg string) { logging.L().Info("graph", "%s", msg) }
