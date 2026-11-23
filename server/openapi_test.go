package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/axllent/mailpit/config"
	"github.com/axllent/mailpit/server/apiv1"
)

var pathParam = regexp.MustCompile(`\{[^}]+\}`)

// Every documented route is registered, as the one which is documented, whatever the webroot is.
// The routes which are not registered from the table are exactly the ones which are known to be registered elsewhere.
func TestAPIRoutesAreRegistered(t *testing.T) {
	origWebroot, origSA, origRspamd := config.Webroot, config.EnableSpamAssassin, config.EnableRspamd
	t.Cleanup(func() {
		config.Webroot, config.EnableSpamAssassin, config.EnableRspamd = origWebroot, origSA, origRspamd
	})

	config.EnableSpamAssassin = "127.0.0.1:783"
	config.EnableRspamd = "127.0.0.1:11334"

	// documented here, registered by the web UI's own handlers
	registeredElsewhere := map[string]bool{"GET /view/{id}.html": true, "GET /view/{id}.txt": true}

	for _, webroot := range []string{"/", "/mailpit/"} {
		config.Webroot = webroot
		mux := apiRoutes()

		for _, rt := range apiv1.Routes {
			key := rt.Method + " " + rt.Path

			if rt.Handler == nil {
				if !registeredElsewhere[key] {
					t.Errorf("%s has no handler and is not a known exception", key)
				}

				continue
			}

			if registeredElsewhere[key] {
				t.Errorf("%s is registered elsewhere, but has a handler", key)
			}

			target := webroot + strings.TrimPrefix(pathParam.ReplaceAllString(rt.Path, "x"), "/")

			_, pattern := mux.Handler(httptest.NewRequest(rt.Method, target, nil))
			if want := rt.Method + " " + webroot + strings.TrimPrefix(rt.Path, "/"); pattern != want {
				t.Errorf("webroot %q: %s is served by %q, want %q", webroot, key, pattern, want)
			}
		}
	}
}

// A route which is not enabled is documented, but not registered (control: it is when it is enabled).
func TestOptionalAPIRoutes(t *testing.T) {
	origSA, origRspamd := config.EnableSpamAssassin, config.EnableRspamd
	t.Cleanup(func() { config.EnableSpamAssassin, config.EnableRspamd = origSA, origRspamd })

	registered := func() (sa, rspamd bool) {
		mux := apiRoutes()

		_, p1 := mux.Handler(httptest.NewRequest(http.MethodGet, "/api/v1/message/x/sa-check", nil))
		_, p2 := mux.Handler(httptest.NewRequest(http.MethodGet, "/api/v1/message/x/rspamd-check", nil))

		// a path without a registered route matches no pattern (it is not caught by the {id} route)
		return p1 == "GET /api/v1/message/{id}/sa-check", p2 == "GET /api/v1/message/{id}/rspamd-check"
	}

	config.EnableSpamAssassin, config.EnableRspamd = "", ""
	if sa, rs := registered(); sa || rs {
		t.Errorf("disabled routes are registered: spamassassin %v, rspamd %v", sa, rs)
	}

	config.EnableSpamAssassin, config.EnableRspamd = "127.0.0.1:783", "127.0.0.1:11334"
	if sa, rs := registered(); !sa || !rs {
		t.Errorf("enabled routes are not registered: spamassassin %v, rspamd %v", sa, rs)
	}
}

// The specification is served (also by its original name), with the webroot as the server if there is one.
func TestOpenAPISpec(t *testing.T) {
	origWebroot := config.Webroot
	t.Cleanup(func() { config.Webroot = origWebroot })

	get := func(webroot, path string) map[string]any {
		t.Helper()

		config.Webroot = webroot

		w := httptest.NewRecorder()
		apiRoutes().ServeHTTP(w, httptest.NewRequest(http.MethodGet, webroot+path, nil))

		if w.Code != http.StatusOK {
			t.Fatalf("%s%s: status %d", webroot, path, w.Code)
		}

		var doc map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &doc); err != nil {
			t.Fatalf("%s%s: %v", webroot, path, err)
		}

		return doc
	}

	for _, name := range []string{"openapi.json", "swagger.json"} {
		path := "api/v1/" + name

		// control: no webroot, nothing is added
		doc := get("/", path)
		if doc["openapi"] == nil || doc["paths"] == nil {
			t.Errorf("%s is not an OpenAPI specification", path)
		}

		if _, ok := doc["servers"]; ok {
			t.Errorf("%s has servers without a webroot", path)
		}

		// a webroot is the server, and the rest is unchanged
		withRoot := get("/mailpit/", path)

		servers, _ := withRoot["servers"].([]any)
		if len(servers) != 1 || servers[0].(map[string]any)["url"] != "/mailpit" {
			t.Errorf("%s servers with webroot = %v, want /mailpit", path, withRoot["servers"])
		}

		if len(withRoot["paths"].(map[string]any)) != len(doc["paths"].(map[string]any)) {
			t.Errorf("%s: the paths changed with a webroot", path)
		}
	}
}
