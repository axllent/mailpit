package server

import (
	"compress/gzip"
	"embed"
	"github.com/axllent/mailpit/config"
	"github.com/axllent/mailpit/server/apiv1"
	"github.com/axllent/mailpit/server/handlers"
	"github.com/axllent/mailpit/server/websockets"
	"github.com/axllent/mailpit/utils/logger"
	"github.com/gorilla/mux"
	"io"
	"io/fs"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
)

//go:embed ui
var embeddedFS embed.FS

// Listen will start the httpd
func Listen() {
	isReady := &atomic.Value{}
	isReady.Store(false)

	serverRoot, err := fs.Sub(embeddedFS, "ui")
	if err != nil {
		logger.Log().Errorf("[http] %s", err)
		os.Exit(1)
	}

	websockets.MessageHub = websockets.NewHub()

	go websockets.MessageHub.Run()

	r := defaultRoutes()

	// kubernetes probes
	r.HandleFunc("/livez", handlers.HealthzHandler)
	r.HandleFunc("/readyz", handlers.ReadyzHandler(isReady))

	// web UI websocket
	r.HandleFunc(config.Webroot+"api/events", apiWebsocket).Methods("GET")

	// virtual filesystem for others
	r.PathPrefix(config.Webroot).Handler(middlewareHandler(http.StripPrefix(config.Webroot, http.FileServer(http.FS(serverRoot)))))

	// redirect to webroot if no trailing slash
	if config.Webroot != "/" {
		redir := strings.TrimRight(config.Webroot, "/")
		r.HandleFunc(redir, middleWareFunc(addSlashToWebroot)).Methods("GET")
	}

	http.Handle("/", r)

	if config.UIAuthFile != "" {
		logger.Log().Info("[http] enabling web UI basic authentication")
	}

	// Mark the application here as ready
	isReady.Store(true)

	if config.UISSLCert != "" && config.UISSLKey != "" {
		logger.Log().Infof("[http] starting secure server on https://%s%s", config.HTTPListen, config.Webroot)
		logger.Log().Fatal(http.ListenAndServeTLS(config.HTTPListen, config.UISSLCert, config.UISSLKey, nil))
	} else {
		logger.Log().Infof("[http] starting server on http://%s%s", config.HTTPListen, config.Webroot)
		logger.Log().Fatal(http.ListenAndServe(config.HTTPListen, nil))
	}

}

func defaultRoutes() *mux.Router {
	r := mux.NewRouter()

	// API V1
	r.HandleFunc(config.Webroot+"api/v1/messages", middleWareFunc(apiv1.GetMessages)).Methods("GET")
	r.HandleFunc(config.Webroot+"api/v1/messages", middleWareFunc(apiv1.SetReadStatus)).Methods("PUT")
	r.HandleFunc(config.Webroot+"api/v1/messages", middleWareFunc(apiv1.DeleteMessages)).Methods("DELETE")
	r.HandleFunc(config.Webroot+"api/v1/tags", middleWareFunc(apiv1.SetTags)).Methods("PUT")
	r.HandleFunc(config.Webroot+"api/v1/search", middleWareFunc(apiv1.Search)).Methods("GET")
	r.HandleFunc(config.Webroot+"api/v1/message/{id}/part/{partID}", middleWareFunc(apiv1.DownloadAttachment)).Methods("GET")
	r.HandleFunc(config.Webroot+"api/v1/message/{id}/part/{partID}/thumb", middleWareFunc(apiv1.Thumbnail)).Methods("GET")
	r.HandleFunc(config.Webroot+"api/v1/message/{id}/raw", middleWareFunc(apiv1.DownloadRaw)).Methods("GET")
	r.HandleFunc(config.Webroot+"api/v1/message/{id}/headers", middleWareFunc(apiv1.Headers)).Methods("GET")
	r.HandleFunc(config.Webroot+"api/v1/message/{id}", middleWareFunc(apiv1.GetMessage)).Methods("GET")
	r.HandleFunc(config.Webroot+"api/v1/info", middleWareFunc(apiv1.AppInfo)).Methods("GET")

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

// Redirect to webroot
func addSlashToWebroot(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, config.Webroot, http.StatusFound)
}

// Websocket to broadcast changes
func apiWebsocket(w http.ResponseWriter, r *http.Request) {
	websockets.ServeWs(websockets.MessageHub, w, r)
}
