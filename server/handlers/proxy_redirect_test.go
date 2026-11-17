package handlers

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/axllent/mailpit/config"
)

func TestProxyResolvesCSSAssetsAfterRedirect(t *testing.T) {
	const id = "msg-redirect"
	allowInternal := config.AllowInternalHTTPRequests
	config.AllowInternalHTTPRequests = true
	t.Cleanup(func() { config.AllowInternalHTTPRequests = allowInternal })

	finalMux := http.NewServeMux()
	finalMux.HandleFunc("/v2/main.css", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/css")
		_, _ = w.Write([]byte("a { background-image: url(./logo.png); }"))
	})
	finalMux.HandleFunc("/v2/logo.png", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write([]byte("logo"))
	})
	final := httptest.NewServer(finalMux)
	t.Cleanup(final.Close)

	initialMux := http.NewServeMux()
	initialMux.HandleFunc("/theme/main.css", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, final.URL+"/v2/main.css", http.StatusFound)
	})
	initial := httptest.NewServer(initialMux)
	t.Cleanup(initial.Close)

	cssURL := initial.URL + "/theme/main.css"
	seedAssets(id)
	recordAsset(id, cssURL)

	query := url.Values{"data": {base64.StdEncoding.EncodeToString([]byte(id + ":" + cssURL))}}
	css := httptest.NewRecorder()
	ProxyHandler(css, httptest.NewRequest(http.MethodGet, "/proxy?"+query.Encode(), nil))
	if css.Code != http.StatusOK {
		t.Fatalf("stylesheet response = %d: %s", css.Code, css.Body)
	}

	const prefix = "a { background-image: url("
	const suffix = "); }"
	body := css.Body.String()
	if !strings.HasPrefix(body, prefix) || !strings.HasSuffix(body, suffix) {
		t.Fatalf("unexpected CSS: %q", body)
	}
	assetURL := strings.TrimSuffix(strings.TrimPrefix(body, prefix), suffix)
	asset := httptest.NewRecorder()
	ProxyHandler(asset, httptest.NewRequest(http.MethodGet, assetURL, nil))
	if asset.Code != http.StatusOK || asset.Body.String() != "logo" {
		t.Fatalf("asset response = %d: %s", asset.Code, asset.Body)
	}
}
