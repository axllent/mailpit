package server

import (
	"embed"
	"net/http"
	"path"
	"strings"

	"github.com/axllent/mailpit/config"
)

var (
	//go:embed ui
	distFS embed.FS
)

// EmbedController is a simple controller to return a file from the embedded filesystem.
//
// This controller is replaces Go's default http.FileServer which, as of Go v1.23, removes
// the Content-Encoding header from error responses, breaking pages such as 404's while
// using gzip compression middleware.
func embedController(w http.ResponseWriter, r *http.Request) {
	p := r.URL.Path

	if strings.HasSuffix(p, "/") {
		p = p + "index.html"
	}

	p = strings.TrimPrefix(p, config.Webroot) // server webroot config
	p = path.Join("ui", p)                    // add go:embed path to path prefix

	b, err := distFS.ReadFile(p)
	if err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	// ensure any HTML files have the correct nonce
	if strings.HasSuffix(p, ".html") {
		nonce := r.Header.Get("mp-nonce")
		b = []byte(strings.ReplaceAll(string(b), "%%NONCE%%", nonce))
	}

	// allow browser cache except for ?dev queries and HTML files
	if r.URL.RawQuery != "dev" && !strings.HasSuffix(p, ".html") {
		w.Header().Set("Cache-Control", "max-age=31536000, public, immutable")
	}

	w.Header().Set("Content-Type", contentType(p))
	_, _ = w.Write(b)
}

// ContentType supports only a few content types, limited to this application's needs.
func contentType(p string) string {
	switch {
	case strings.HasSuffix(p, ".html"):
		return "text/html; charset=utf-8"
	case strings.HasSuffix(p, ".css"):
		return "text/css; charset=utf-8"
	case strings.HasSuffix(p, ".js"):
		return "application/javascript; charset=utf-8"
	case strings.HasSuffix(p, ".json"):
		return "application/json"
	case strings.HasSuffix(p, ".svg"):
		return "image/svg+xml"
	case strings.HasSuffix(p, ".ico"):
		return "image/x-icon"
	case strings.HasSuffix(p, ".png"):
		return "image/png"
	case strings.HasSuffix(p, ".jpg"):
		return "image/jpeg"
	case strings.HasSuffix(p, ".gif"):
		return "image/gif"
	case strings.HasSuffix(p, ".woff"):
		return "font/woff"
	case strings.HasSuffix(p, ".woff2"):
		return "font/woff2"
	default:
		return "text/plain"
	}
}
