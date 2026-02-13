// Package server is the HTTP daemon
package server

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync/atomic"
	"text/template"
	"time"

	"github.com/axllent/mailpit/config"
	"github.com/axllent/mailpit/internal/auth"
	"github.com/axllent/mailpit/internal/logger"
	"github.com/axllent/mailpit/internal/pop3"
	"github.com/axllent/mailpit/internal/prometheus"
	"github.com/axllent/mailpit/internal/snakeoil"
	"github.com/axllent/mailpit/internal/stats"
	"github.com/axllent/mailpit/internal/storage"
	"github.com/axllent/mailpit/internal/tools"
	"github.com/axllent/mailpit/server/apiv1"
	"github.com/axllent/mailpit/server/handlers"
	"github.com/axllent/mailpit/server/websockets"
	"github.com/gorilla/mux"
	"github.com/lithammer/shortuuid/v4"
)

var (
	// htmlPreviewRouteRe is a regexp to match the HTML preview route
	htmlPreviewRouteRe *regexp.Regexp
)

// Listen will start the httpd
func Listen() {
	setCORSOrigins()

	isReady := &atomic.Value{}
	isReady.Store(false)
	stats.Track()

	websockets.MessageHub = websockets.NewHub()

	// set allowed websocket origins from configuration
	// websockets.SetAllowedOrigins(AccessControlAllowWSOrigins)

	go websockets.MessageHub.Run()

	go pop3.Run()

	r := apiRoutes()

	// kubernetes probes
	r.HandleFunc(config.Webroot+"livez", handlers.HealthzHandler)
	r.HandleFunc(config.Webroot+"readyz", handlers.ReadyzHandler(isReady))

	// proxy handler for screenshots
	r.HandleFunc(config.Webroot+"proxy", middleWareFunc(handlers.ProxyHandler)).Methods("GET")

	// virtual filesystem for /dist/ & some individual files
	r.PathPrefix(config.Webroot + "dist/").Handler(middleWareFunc(embedController))
	r.PathPrefix(config.Webroot + "api/").Handler(middleWareFunc(embedController))
	r.Path(config.Webroot + "favicon.ico").Handler(middleWareFunc(embedController))
	r.Path(config.Webroot + "favicon.svg").Handler(middleWareFunc(embedController))
	r.Path(config.Webroot + "mailpit.svg").Handler(middleWareFunc(embedController))
	r.Path(config.Webroot + "notification.png").Handler(middleWareFunc(embedController))

	// redirect to webroot if no trailing slash
	if config.Webroot != "/" {
		redirect := strings.TrimRight(config.Webroot, "/")
		r.HandleFunc(redirect, middleWareFunc(addSlashToWebroot)).Methods("GET")
	}

	// UI shortcut
	r.HandleFunc(config.Webroot+"view/latest", middleWareFunc(handlers.RedirectToLatestMessage)).Methods("GET")

	// frontend testing
	r.HandleFunc(config.Webroot+"view/{id}.html", middleWareFunc(apiv1.GetMessageHTML)).Methods("GET")
	r.HandleFunc(config.Webroot+"view/{id}.txt", middleWareFunc(apiv1.GetMessageText)).Methods("GET")

	// web UI via virtual index.html
	r.PathPrefix(config.Webroot + "view/").Handler(middleWareFunc(index)).Methods("GET")
	r.Path(config.Webroot + "search").Handler(middleWareFunc(index)).Methods("GET")
	r.Path(config.Webroot).Handler(middleWareFunc(index)).Methods("GET")

	if auth.UICredentials != nil {
		logger.Log().Info("[http] enabling basic authentication")
	}

	// Mark the application here as ready
	isReady.Store(true)

	server := &http.Server{
		Addr:         config.HTTPListen,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		Handler:      r,
	}

	// add temporary self-signed certificates to get deleted afterwards
	for _, keyPair := range snakeoil.Certificates() {
		storage.AddTempFile(keyPair.Public)
		storage.AddTempFile(keyPair.Private)
	}

	if config.UITLSCert != "" && config.UITLSKey != "" {
		logger.Log().Infof("[http] starting on %s (TLS)", config.HTTPListen)
		logger.Log().Infof("[http] accessible via https://%s%s", logger.CleanHTTPIP(config.HTTPListen), config.Webroot)
		if err := server.ListenAndServeTLS(config.UITLSCert, config.UITLSKey); err != nil {
			storage.Close()
			logger.Log().Fatal(err)
		}

	} else {
		socketAddr, perm, isSocket := tools.UnixSocket(config.HTTPListen)

		if isSocket {
			if err := tools.PrepareSocket(socketAddr); err != nil {
				storage.Close()
				logger.Log().Fatal(err)
			}

			// delete the Unix socket file on exit
			storage.AddTempFile(socketAddr)

			ln, err := net.Listen("unix", socketAddr)
			if err != nil {
				storage.Close()
				logger.Log().Fatal(err)
			}

			if err := os.Chmod(socketAddr, perm); err != nil {
				storage.Close()
				logger.Log().Fatal(err)
			}

			logger.Log().Infof("[http] starting on %s", config.HTTPListen)

			if err := server.Serve(ln); err != nil {
				storage.Close()
				logger.Log().Fatal(err)
			}

		} else {
			logger.Log().Infof("[http] starting on %s", config.HTTPListen)
			logger.Log().Infof("[http] accessible via http://%s%s", logger.CleanHTTPIP(config.HTTPListen), config.Webroot)
			if err := server.ListenAndServe(); err != nil {
				storage.Close()
				logger.Log().Fatal(err)
			}
		}
	}
}

func apiRoutes() *mux.Router {
	r := mux.NewRouter()

	// API V1
	r.HandleFunc(config.Webroot+"api/v1/messages", middleWareFunc(apiv1.GetMessages)).Methods("GET")
	r.HandleFunc(config.Webroot+"api/v1/messages", middleWareFunc(apiv1.SetReadStatus)).Methods("PUT")
	r.HandleFunc(config.Webroot+"api/v1/messages", middleWareFunc(apiv1.DeleteMessages)).Methods("DELETE")
	r.HandleFunc(config.Webroot+"api/v1/search", middleWareFunc(apiv1.Search)).Methods("GET")
	r.HandleFunc(config.Webroot+"api/v1/search", middleWareFunc(apiv1.DeleteSearch)).Methods("DELETE")
	r.HandleFunc(config.Webroot+"api/v1/send", sendAPIAuthMiddleware(apiv1.SendMessageHandler)).Methods("POST")
	r.HandleFunc(config.Webroot+"api/v1/tags", middleWareFunc(apiv1.GetAllTags)).Methods("GET")
	r.HandleFunc(config.Webroot+"api/v1/tags", middleWareFunc(apiv1.SetMessageTags)).Methods("PUT")
	r.HandleFunc(config.Webroot+"api/v1/tags/{tag}", middleWareFunc(apiv1.RenameTag)).Methods("PUT")
	r.HandleFunc(config.Webroot+"api/v1/tags/{tag}", middleWareFunc(apiv1.DeleteTag)).Methods("DELETE")
	r.HandleFunc(config.Webroot+"api/v1/message/{id}/part/{partID}", middleWareFunc(apiv1.DownloadAttachment)).Methods("GET")
	r.HandleFunc(config.Webroot+"api/v1/message/{id}/part/{partID}/thumb", middleWareFunc(apiv1.Thumbnail)).Methods("GET")
	r.HandleFunc(config.Webroot+"api/v1/message/{id}/headers", middleWareFunc(apiv1.GetHeaders)).Methods("GET")
	r.HandleFunc(config.Webroot+"api/v1/message/{id}/raw", middleWareFunc(apiv1.DownloadRaw)).Methods("GET")
	r.HandleFunc(config.Webroot+"api/v1/message/{id}/release", middleWareFunc(apiv1.ReleaseMessage)).Methods("POST")
	r.HandleFunc(config.Webroot+"api/v1/message/{id}/html-check", middleWareFunc(apiv1.HTMLCheck)).Methods("GET")
	r.HandleFunc(config.Webroot+"api/v1/message/{id}/link-check", middleWareFunc(apiv1.LinkCheck)).Methods("GET")
	if config.EnableSpamAssassin != "" {
		r.HandleFunc(config.Webroot+"api/v1/message/{id}/sa-check", middleWareFunc(apiv1.SpamAssassinCheck)).Methods("GET")
	}
	r.HandleFunc(config.Webroot+"api/v1/message/{id}", middleWareFunc(apiv1.GetMessage)).Methods("GET")
	r.HandleFunc(config.Webroot+"api/v1/info", middleWareFunc(apiv1.AppInfo)).Methods("GET")
	r.HandleFunc(config.Webroot+"api/v1/webui", middleWareFunc(apiv1.WebUIConfig)).Methods("GET")
	r.HandleFunc(config.Webroot+"api/v1/swagger.json", middleWareFunc(swaggerBasePath)).Methods("GET")

	// Chaos
	r.HandleFunc(config.Webroot+"api/v1/chaos", middleWareFunc(apiv1.GetChaos)).Methods("GET")
	r.HandleFunc(config.Webroot+"api/v1/chaos", middleWareFunc(apiv1.SetChaos)).Methods("PUT")

	// Prometheus metrics (if enabled and using existing server)
	if prometheus.GetMode() == "integrated" {
		r.HandleFunc(config.Webroot+"metrics", middleWareFunc(func(w http.ResponseWriter, r *http.Request) {
			prometheus.GetHandler().ServeHTTP(w, r)
		})).Methods("GET")
	}

	// web UI websocket
	r.HandleFunc(config.Webroot+"api/events", apiWebsocket).Methods("GET")

	// return blank 200 response for OPTIONS requests for CORS
	r.PathPrefix(config.Webroot + "api/v1/").Handler(middleWareFunc(apiv1.GetOptions)).Methods("OPTIONS")

	return r
}

// BasicAuthResponse returns an basic auth response to the browser
func basicAuthResponse(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", `Basic realm="Login"`)
	w.WriteHeader(http.StatusUnauthorized)
	_, _ = w.Write([]byte("Unauthorized.\n"))
}

// sendAPIAuthMiddleware handles authentication specifically for the send API endpoint
// It can use dedicated send API authentication or accept any credentials based on configuration
func sendAPIAuthMiddleware(fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// If send API auth accept any is enabled, bypass all authentication
		if config.SendAPIAuthAcceptAny {
			// Temporarily disable UI auth for this request
			originalCredentials := auth.UICredentials
			auth.UICredentials = nil
			defer func() { auth.UICredentials = originalCredentials }()
			// Call the standard middleware
			middleWareFunc(fn)(w, r)
			return
		}

		// If Send API credentials are configured, only accept those credentials
		if auth.SendAPICredentials != nil {
			user, pass, ok := r.BasicAuth()

			if !ok {
				basicAuthResponse(w)
				return
			}

			if !auth.SendAPICredentials.Match(user, pass) {
				basicAuthResponse(w)
				return
			}

			// Valid Send API credentials - bypass UI auth and call function directly
			originalCredentials := auth.UICredentials
			auth.UICredentials = nil
			defer func() { auth.UICredentials = originalCredentials }()
			middleWareFunc(fn)(w, r)
			return
		}

		// No Send API credentials configured - fall back to UI auth
		middleWareFunc(fn)(w, r)
	}
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

		// generate a new random nonce on every request
		randomNonce := shortuuid.New()
		// header used to pass nonce through to function
		r.Header.Set("mp-nonce", randomNonce)

		// Prevent JavaScript XSS by adding a nonce for script-src
		cspHeader := strings.Replace(
			config.ContentSecurityPolicy,
			"script-src 'self';",
			fmt.Sprintf("script-src 'nonce-%s';", randomNonce),
			1,
		)

		w.Header().Set("Content-Security-Policy", cspHeader)

		if htmlPreviewRouteRe == nil {
			htmlPreviewRouteRe = regexp.MustCompile(`^` + regexp.QuoteMeta(config.Webroot) + `view/[a-zA-Z0-9]+\.html$`)
		}

		if strings.HasPrefix(r.RequestURI, config.Webroot+"api/") || htmlPreviewRouteRe.MatchString(r.RequestURI) {
			if allowed := corsOriginAccessControl(r); !allowed {
				http.Error(w, "Blocked to to CORS violation", http.StatusForbidden)
				return
			}
			w.Header().Set("Access-Control-Allow-Origin", r.Header.Get("Origin"))
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, PUT, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "*")
		}

		// Check basic authentication headers if configured.
		// OPTIONS requests are skipped if CORS is enabled, since browsers omit credentials for preflight checks.
		isCORSOptionsRequest := AccessControlAllowOrigin != "" && r.Method == http.MethodOptions
		if !isCORSOptionsRequest && auth.UICredentials != nil {
			user, pass, ok := r.BasicAuth()

			if !ok {
				basicAuthResponse(w)
				return
			}

			if !auth.UICredentials.Match(user, pass) {
				basicAuthResponse(w)
				return
			}
		}

		if config.DisableHTTPCompression || !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			fn(w, r)
			return
		}

		w.Header().Set("Content-Encoding", "gzip")
		gz := gzip.NewWriter(w)
		defer func() { _ = gz.Close() }()
		gzr := gzipResponseWriter{Writer: gz, ResponseWriter: w}
		fn(gzr, r)
	}
}

// Redirect to webroot
func addSlashToWebroot(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, config.Webroot, http.StatusFound)
}

// Websocket to broadcast changes
func apiWebsocket(w http.ResponseWriter, r *http.Request) {
	if allowed := corsOriginAccessControl(r); !allowed {
		http.Error(w, "Blocked to to CORS violation", http.StatusForbidden)
		return
	}
	w.Header().Set("Access-Control-Allow-Origin", r.Header.Get("Origin"))
	w.Header().Set("Access-Control-Allow-Headers", "*")
	websockets.ServeWs(websockets.MessageHub, w, r)
	storage.BroadcastMailboxStats()
}

// Wrapper to artificially inject a basePath to the swagger.json if a webroot has been specified
func swaggerBasePath(w http.ResponseWriter, _ *http.Request) {
	f, err := distFS.ReadFile("ui/api/v1/swagger.json")
	if err != nil {
		panic(err)
	}

	if config.Webroot != "/" {
		// artificially inject a path at the start
		replacement := fmt.Sprintf("{\n  \"basePath\": \"%s\",", strings.TrimRight(config.Webroot, "/"))

		f = bytes.Replace(f, []byte("{"), []byte(replacement), 1)
	}

	w.Header().Add("Content-Type", "application/json")
	_, _ = w.Write(f)
}

// Just returns the default HTML template
func index(w http.ResponseWriter, r *http.Request) {

	var h = `<!DOCTYPE html>
<html lang="en" class="h-100">

<head>
	<meta charset="utf-8">
	<meta name="viewport" content="width=device-width,initial-scale=1.0">
	<meta name="referrer" content="no-referrer">
	<meta name="robots" content="noindex, nofollow, noarchive">
	<link rel="icon" href="{{ .Webroot }}favicon.svg">
	<title>Mailpit</title>
	<link rel=stylesheet href="{{ .Webroot }}dist/app.css?{{ .Version }}">
</head>

<body class="h-100">
	<div class="container-fluid h-100 d-flex flex-column" id="app" data-webroot="{{ .Webroot }}" data-version="{{ .Version }}">
		<noscript class="alert alert-warning position-absolute top-50 start-50 translate-middle">
			You need a browser with JavaScript enabled to use Mailpit
		</noscript>
	</div>

	<script src="{{ .Webroot }}dist/app.js?{{ .Version }}" nonce="{{ .Nonce }}"></script>
</body>

</html>`

	t, err := template.New("index").Parse(h)
	if err != nil {
		panic(err)
	}

	data := struct {
		Webroot string
		Version string
		Nonce   string
	}{
		Webroot: config.Webroot,
		Version: config.Version,
		Nonce:   r.Header.Get("mp-nonce"),
	}

	buff := new(bytes.Buffer)

	err = t.Execute(buff, data)
	if err != nil {
		panic(err)
	}

	w.Header().Add("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(buff.Bytes())
}
