// http_client.go — cliente HTTP único, compartido por todos los
// proveedores. Pensado para escenarios de millones de solicitudes:
// reutiliza conexiones TCP/TLS (keep-alive) en vez de abrir una nueva por
// request, lo cual es el costo dominante cuando se llama seguido al mismo
// servidor Ollama/LM Studio/vLLM detrás de docker.
package models

import (
	"net"
	"net/http"
	"time"
)

// SharedClient — un solo *http.Client para todo el proceso. No crear uno
// nuevo por request: eso es lo primero que hay que evitar para soportar
// alto volumen de forma eficiente.
var SharedClient = &http.Client{
	Transport: &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 60 * time.Second,
		}).DialContext,
		MaxIdleConns:          256,
		MaxIdleConnsPerHost:   64, // varias goroutines pueden pegarle al mismo mova_ollama
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		ForceAttemptHTTP2:     true,
	},
}
