package server

import (
	"compress/gzip"
	"embed"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/axllent/mailpit/config"
	"github.com/axllent/mailpit/logger"
	"github.com/axllent/mailpit/server/websockets"
	"github.com/gorilla/mux"
)

//go:embed ui
var embeddedFS embed.FS

var contentSecurityPolicy = "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; frame-src 'self'; img-src * data: blob:; font-src 'self' data:; media-src 'self'; connect-src 'self'; object-src 'none'; base-uri 'self';"

// Listen will start the httpd
func Listen() {
	serverRoot, err := fs.Sub(embeddedFS, "ui")
	if err != nil {
		logger.Log().Errorf("[http] %s", err)
		os.Exit(1)
	}

	websockets.MessageHub = websockets.NewHub()

	go websockets.MessageHub.Run()

	r := mux.NewRouter()
	r.HandleFunc("/api/stats", middleWareFunc(apiMailboxStats)).Methods("GET")
	r.HandleFunc("/api/messages", middleWareFunc(apiListMessages)).Methods("GET")
	r.HandleFunc("/api/search", middleWareFunc(apiSearchMessages)).Methods("GET")
	r.HandleFunc("/api/delete", middleWareFunc(apiDeleteAll)).Methods("GET")
	r.HandleFunc("/api/delete", middleWareFunc(apiDeleteSelected)).Methods("POST")
	r.HandleFunc("/api/events", apiWebsocket).Methods("GET")
	r.HandleFunc("/api/read", apiMarkAllRead).Methods("GET")
	r.HandleFunc("/api/read", apiMarkSelectedRead).Methods("POST")
	r.HandleFunc("/api/unread", apiMarkSelectedUnread).Methods("POST")
	r.HandleFunc("/api/{id}/raw", middleWareFunc(apiDownloadRaw)).Methods("GET")
	r.HandleFunc("/api/{id}/part/{partID}", middleWareFunc(apiDownloadAttachment)).Methods("GET")
	r.HandleFunc("/api/{id}/part/{partID}/thumb", middleWareFunc(apiAttachmentThumbnail)).Methods("GET")
	r.HandleFunc("/api/{id}/delete", middleWareFunc(apiDeleteOne)).Methods("GET")
	r.HandleFunc("/api/{id}/unread", middleWareFunc(apiUnreadOne)).Methods("GET")
	r.HandleFunc("/api/{id}", middleWareFunc(apiOpenMessage)).Methods("GET")
	r.PathPrefix("/").Handler(middlewareHandler(http.FileServer(http.FS(serverRoot))))
	http.Handle("/", r)

	if config.UIAuthFile != "" {
		logger.Log().Info("[http] enabling web UI basic authentication")
	}

	if config.UISSLCert != "" && config.UISSLKey != "" {
		logger.Log().Infof("[http] starting secure server on https://%s", config.HTTPListen)
		log.Fatal(http.ListenAndServeTLS(config.HTTPListen, config.UISSLCert, config.UISSLKey, nil))
	} else {
		logger.Log().Infof("[http] starting server on http://%s", config.HTTPListen)
		log.Fatal(http.ListenAndServe(config.HTTPListen, nil))
	}
}

// BasicAuthResponse returns an basic auth response to the browser
func basicAuthResponse(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", `Basic realm="Login"`)
	w.WriteHeader(http.StatusUnauthorized)
	_, _ = w.Write([]byte("Unauthorised.\n"))
}

type gzipResponseWriter struct {
	io.Writer
	http.ResponseWriter
}

func (w gzipResponseWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

// MiddleWareFunc http middleware adds optional basic authentication
// and gzip compression.
func middleWareFunc(fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Content-Security-Policy", contentSecurityPolicy)

		if config.UIAuthFile != "" {
			user, pass, ok := r.BasicAuth()

			if !ok {
				basicAuthResponse(w)
				return
			}

			if !config.UIAuth.Match(user, pass) {
				basicAuthResponse(w)
				return
			}
		}

		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			fn(w, r)
			return
		}
		w.Header().Set("Content-Encoding", "gzip")
		gz := gzip.NewWriter(w)
		defer gz.Close()
		gzr := gzipResponseWriter{Writer: gz, ResponseWriter: w}
		fn(gzr, r)
	}
}

// MiddlewareHandler http middleware adds optional basic authentication
// and gzip compression
func middlewareHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Content-Security-Policy", contentSecurityPolicy)

		if config.UIAuthFile != "" {
			user, pass, ok := r.BasicAuth()

			if !ok {
				basicAuthResponse(w)
				return
			}

			if !config.UIAuth.Match(user, pass) {
				basicAuthResponse(w)
				return
			}
		}

		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			h.ServeHTTP(w, r)
			return
		}
		w.Header().Set("Content-Encoding", "gzip")
		gz := gzip.NewWriter(w)
		defer gz.Close()
		h.ServeHTTP(gzipResponseWriter{Writer: gz, ResponseWriter: w}, r)
	})
}

// FourOFour returns a basic 404 message
func fourOFour(w http.ResponseWriter) {
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("Content-Security-Policy", contentSecurityPolicy)
	w.WriteHeader(http.StatusNotFound)
	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprint(w, "404 page not found")
}

// HTTPError returns a basic error message (400 response)
func httpError(w http.ResponseWriter, msg string) {
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("Content-Security-Policy", contentSecurityPolicy)
	w.WriteHeader(http.StatusBadRequest)
	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprint(w, msg)
}

// Get the start and limit based on query params. Defaults to 0, 50
func getStartLimit(req *http.Request) (start int, limit int) {
	start = 0
	limit = 50

	s := req.URL.Query().Get("start")
	if n, err := strconv.Atoi(s); err == nil && n > 0 {
		start = n
	}

	l := req.URL.Query().Get("limit")
	if n, err := strconv.Atoi(l); err == nil && n > 0 {
		limit = n
	}

	return start, limit
}
