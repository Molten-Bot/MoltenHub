package api

import (
	_ "embed"
	"net/http"
	"strings"
)

//go:embed ui/index.html
var uiIndexHTML []byte

//go:embed ui/app.js
var uiAppJS []byte

//go:embed ui/login.html
var uiLoginHTML []byte

//go:embed ui/login.js
var uiLoginJS []byte

func (h *Handler) handleUI(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/v1/") || strings.HasPrefix(r.URL.Path, "/health") || strings.HasPrefix(r.URL.Path, "/openapi") {
		writeError(w, http.StatusNotFound, "not_found", "route not found")
		return
	}

	switch r.URL.Path {
	case "/", "/index.html":
		if r.Method != http.MethodGet {
			writeMethodNotAllowed(w)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(uiLoginHTML)
		return
	case "/login.js":
		if r.Method != http.MethodGet {
			writeMethodNotAllowed(w)
			return
		}
		w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(uiLoginJS)
		return
	case "/domains", "/domains/", "/domains/index.html":
		if r.Method != http.MethodGet {
			writeMethodNotAllowed(w)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(uiIndexHTML)
		return
	case "/app.js", "/domains/app.js":
		if r.Method != http.MethodGet {
			writeMethodNotAllowed(w)
			return
		}
		w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(uiAppJS)
		return
	default:
		writeError(w, http.StatusNotFound, "not_found", "route not found")
		return
	}
}
