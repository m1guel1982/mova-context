// guard.go — authentication and origin checks for the HTTP door.
package http

import (
	"crypto/subtle"
	"net"
	"net/http"
	"net/url"
	"strings"
)

// Guard wraps h with: Origin check (only localhost/127.0.0.1/[::1] or no
// Origin) and, when token != "", a required "Authorization: Bearer
// <token>". /health stays open (no project data).
func Guard(h http.Handler, token string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			h.ServeHTTP(w, r)
			return
		}
		if o := r.Header.Get("Origin"); o != "" && !localOrigin(o) {
			http.Error(w, "origin not allowed", http.StatusForbidden)
			return
		}
		if token != "" {
			got := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
			if subtle.ConstantTimeCompare([]byte(got), []byte(token)) != 1 {
				w.Header().Set("WWW-Authenticate", "Bearer")
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
		}
		h.ServeHTTP(w, r)
	})
}

func localOrigin(o string) bool {
	u, err := url.Parse(o)
	if err != nil {
		return false
	}
	switch u.Hostname() {
	case "localhost", "127.0.0.1", "::1":
		return true
	}
	return false
}

func isLoopbackBind(bind string) bool {
	if bind == "localhost" {
		return true
	}
	ip := net.ParseIP(bind)
	return ip != nil && ip.IsLoopback()
}
