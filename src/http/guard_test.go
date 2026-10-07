package http

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGuard(t *testing.T) {
	ok := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })
	h := Guard(ok, "s3cret")
	cases := []struct {
		path, auth, origin string
		want               int
	}{
		{"/mcp", "", "", 401},
		{"/mcp", "Bearer wrong", "", 401},
		{"/mcp", "Bearer s3cret", "", 200},
		{"/mcp", "Bearer s3cret", "http://evil.example", 403},
		{"/mcp", "Bearer s3cret", "http://localhost:3000", 200},
		{"/health", "", "", 200},
	}
	for _, c := range cases {
		r := httptest.NewRequest("POST", c.path, nil)
		if c.auth != "" {
			r.Header.Set("Authorization", c.auth)
		}
		if c.origin != "" {
			r.Header.Set("Origin", c.origin)
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != c.want {
			t.Errorf("%+v → %d", c, w.Code)
		}
	}
	if isLoopbackBind("0.0.0.0") || !isLoopbackBind("127.0.0.1") || !isLoopbackBind("localhost") {
		t.Fatal("loopback detection")
	}
	if err := StartServerOn(nil, t.TempDir(), "0.0.0.0", 0); err == nil {
		t.Fatal("non-loopback bind without MOVA_HTTP_TOKEN must refuse to start")
	}
}
