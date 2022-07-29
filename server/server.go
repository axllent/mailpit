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
	r.HandleFunc("/api/mailboxes", gzipHandlerFunc(apiListMailboxes))
	r.HandleFunc("/api/{mailbox}/messages", gzipHandlerFunc(apiListMailbox))
	r.HandleFunc("/api/{mailbox}/search", gzipHandlerFunc(apiSearchMailbox))
	r.HandleFunc("/api/{mailbox}/delete", gzipHandlerFunc(apiDeleteAll))
	r.HandleFunc("/api/{mailbox}/events", apiWebsocket)
	r.HandleFunc("/api/{mailbox}/{id}/source", gzipHandlerFunc(apiDownloadSource))
	r.HandleFunc("/api/{mailbox}/{id}/part/{partID}", gzipHandlerFunc(apiDownloadAttachment))
	r.HandleFunc("/api/{mailbox}/{id}/delete", gzipHandlerFunc(apiDeleteOne))
	r.HandleFunc("/api/{mailbox}/{id}/unread", gzipHandlerFunc(apiUnreadOne))
	r.HandleFunc("/api/{mailbox}/{id}", gzipHandlerFunc(apiOpenMessage))
	r.HandleFunc("/api/{mailbox}/search", gzipHandlerFunc(apiSearchMailbox))
	r.PathPrefix("/").Handler(gzipHandler(http.FileServer(http.FS(serverRoot))))
	http.Handle("/", r)

	if config.SSLCert != "" && config.SSLKey != "" {
		logger.Log().Infof("[http] starting secure server on https://%s", config.HTTPListen)
		log.Fatal(http.ListenAndServeTLS(config.HTTPListen, config.SSLCert, config.SSLKey, nil))
	} else {
		logger.Log().Infof("[http] starting server on http://%s", config.HTTPListen)
		log.Fatal(http.ListenAndServe(config.HTTPListen, nil))
	}

}

type gzipResponseWriter struct {
	io.Writer
	http.ResponseWriter
}

func (w gzipResponseWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

// GzipHandlerFunc http middleware
func gzipHandlerFunc(fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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

func gzipHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

// FourOFour returns a standard 404 meesage
func fourOFour(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNotFound)
	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprint(w, "404 page not found")
}

// HTTPError returns a standard 404 meesage
func httpError(w http.ResponseWriter, msg string) {
	w.WriteHeader(http.StatusBadRequest)
	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprint(w, msg)
}

// Get the start and limit based on query params. Defaults to 0, 50
func getStartLimit(req *http.Request) (start int, limit int) {
	start = 0
	limit = 50

	s := req.URL.Query().Get("start")
	if n, e := strconv.ParseInt(s, 10, 64); e == nil && n > 0 {
		start = int(n)
	}

	l := req.URL.Query().Get("limit")
	if n, e := strconv.ParseInt(l, 10, 64); e == nil && n > 0 {
		if n > 500 {
			n = 500
		}
		limit = int(n)
	}

	return start, limit
}
