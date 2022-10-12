package server

import (
	"compress/gzip"
	"embed"
	"io"
	"io/fs"
	"net/http"
	"os"
	"strings"

	"github.com/axllent/mailpit/config"
	"github.com/axllent/mailpit/logger"
	"github.com/axllent/mailpit/server/apiv1"
	"github.com/axllent/mailpit/server/websockets"
	"github.com/gorilla/mux"
)

//go:embed ui
var embeddedFS embed.FS

// Listen will start the httpd
func Listen() {
	serverRoot, err := fs.Sub(embeddedFS, "ui")
	if err != nil {
		logger.Log().Errorf("[http] %s", err)
		os.Exit(1)
	}

	websockets.MessageHub = websockets.NewHub()

	go websockets.MessageHub.Run()

	r := defaultRoutes()

	// web UI websocket
	r.HandleFunc("/api/events", apiWebsocket).Methods("GET")

	// virtual filesystem for others
	r.PathPrefix("/").Handler(middlewareHandler(http.FileServer(http.FS(serverRoot))))
	http.Handle("/", r)

	if config.UIAuthFile != "" {
		logger.Log().Info("[http] enabling web UI basic authentication")
	}

	if config.UISSLCert != "" && config.UISSLKey != "" {
		logger.Log().Infof("[http] starting secure server on https://%s", config.HTTPListen)
		logger.Log().Fatal(http.ListenAndServeTLS(config.HTTPListen, config.UISSLCert, config.UISSLKey, nil))
	} else {
		logger.Log().Infof("[http] starting server on http://%s", config.HTTPListen)
		logger.Log().Fatal(http.ListenAndServe(config.HTTPListen, nil))
	}
}

func defaultRoutes() *mux.Router {
	r := mux.NewRouter()

	// API V1
	r.HandleFunc("/api/v1/messages", middleWareFunc(apiv1.Messages)).Methods("GET")
	r.HandleFunc("/api/v1/messages", middleWareFunc(apiv1.SetReadStatus)).Methods("PUT")
	r.HandleFunc("/api/v1/messages", middleWareFunc(apiv1.DeleteMessages)).Methods("DELETE")
	r.HandleFunc("/api/v1/search", middleWareFunc(apiv1.Search)).Methods("GET")
	r.HandleFunc("/api/v1/message/{id}/part/{partID}", middleWareFunc(apiv1.DownloadAttachment)).Methods("GET")
	r.HandleFunc("/api/v1/message/{id}/part/{partID}/thumb", middleWareFunc(apiv1.Thumbnail)).Methods("GET")
	r.HandleFunc("/api/v1/message/{id}/raw", middleWareFunc(apiv1.DownloadRaw)).Methods("GET")
	r.HandleFunc("/api/v1/message/{id}/headers", middleWareFunc(apiv1.Headers)).Methods("GET")
	r.HandleFunc("/api/v1/message/{id}", middleWareFunc(apiv1.Message)).Methods("GET")
	r.HandleFunc("/api/v1/info", middleWareFunc(apiv1.AppInfo)).Methods("GET")

	return r
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
		w.Header().Set("Content-Security-Policy", config.ContentSecurityPolicy)

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
		w.Header().Set("Content-Security-Policy", config.ContentSecurityPolicy)

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

// Websocket to broadcast changes
func apiWebsocket(w http.ResponseWriter, r *http.Request) {
	websockets.ServeWs(websockets.MessageHub, w, r)
}
