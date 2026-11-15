package server

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"sync"
	"testing"

	"github.com/axllent/mailpit/config"
)

// isHTMLPreviewRoute is what the regular expression it replaced was: the webroot, "view/", an ID of letters and numbers
// and ".html", whatever the webroot is (including one which has characters special to a regular expression).
func TestIsHTMLPreviewRoute(t *testing.T) {
	origWebroot := config.Webroot
	t.Cleanup(func() { config.Webroot = origWebroot })

	paths := []string{
		"view/abc123.html", "view/ABCdef.html", "view/a.html", "view/.html", "view/a.b.html", "view/ab c.html",
		"view/abc.htm", "view/abc.html/", "view/abc.html\n", "view/abc", "view/", "view", "view/abc-123.html",
		"view/abc_123.html", "view/ab/c.html", "view/%61bc.html", "view/abc.HTML", "view/abc.html.html",
		"view/é.html", "viewx/abc.html", "api/v1/messages", "", "x/view/abc.html", "view//abc.html",
	}

	for _, webroot := range []string{"/", "/mailpit/", "/a.b/", "/(x)/", "/a+b/"} {
		config.Webroot = webroot
		re := regexp.MustCompile(`^` + regexp.QuoteMeta(webroot) + `view/[a-zA-Z0-9]+\.html$`)

		for _, p := range paths {
			for _, path := range []string{webroot + p, "/" + p, p} {
				if got, want := isHTMLPreviewRoute(path), re.MatchString(path); got != want {
					t.Errorf("webroot %q, path %q: %v, the regular expression says %v", webroot, path, got, want)
				}
			}
		}
	}

	// control: the ones which are meant to match do
	config.Webroot = "/"
	for _, p := range []string{"/view/abc123.html", "/view/A.html"} {
		if !isHTMLPreviewRoute(p) {
			t.Errorf("%q should be the preview of a message", p)
		}
	}
	for _, p := range []string{"/view/abc123.txt", "/view/.html", "/view/a/b.html", "/api/v1/message/abc"} {
		if isHTMLPreviewRoute(p) {
			t.Errorf("%q should not be the preview of a message", p)
		}
	}
}

// Requests are handled at the same time, which is what the race detector is for (the first request used to set a
// variable which the others read).
func TestMiddlewareConcurrentRequests(t *testing.T) {
	h := middleWareFunc(func(w http.ResponseWriter, r *http.Request) {})

	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 50 {
				h(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/v1/info", nil))
				h(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/view/abc123.html", nil))
			}
		}()
	}
	wg.Wait()
}
