package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/axllent/mailpit/config"
)

// imageServer serves an image of the size given at /a.png, in the pieces given by the handler.
func imageServer(t *testing.T, write func(w http.ResponseWriter)) string {
	t.Helper()

	config.AllowInternalHTTPRequests = true
	t.Cleanup(func() { config.AllowInternalHTTPRequests = false })

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		write(w)
	}))
	t.Cleanup(srv.Close)

	return srv.URL + "/a.png"
}

// An image or font is accepted up to the limit and refused beyond it, whether the size is sent or not (a response
// of unknown length is read up to the limit and one byte more).
func TestProxyBodyLimit(t *testing.T) {
	for name, tc := range map[string]struct {
		size int
		code int
	}{
		"control: at the limit":       {maxProxyBodySize, http.StatusOK},
		"control: well within":        {1 << 20, http.StatusOK},
		"one byte over the limit":     {maxProxyBodySize + 1, http.StatusBadRequest},
		"far over the limit (60 MiB)": {60 << 20, http.StatusBadRequest},
		"over the limit (20 MiB)":     {20 << 20, http.StatusBadRequest},
		"control: a little smaller":   {maxProxyBodySize - 1, http.StatusOK},
	} {
		t.Run(name, func(t *testing.T) {
			body := []byte(strings.Repeat("x", tc.size))
			uri := imageServer(t, func(w http.ResponseWriter) { _, _ = w.Write(body) })
			const id = "size"
			seedAssets(id)
			recordAsset(id, uri)

			w := proxyRequest(id, uri)
			if w.Code != tc.code {
				t.Fatalf("a %d byte image: status %d, want %d", tc.size, w.Code, tc.code)
			}
			if tc.code == http.StatusOK && w.Body.Len() != tc.size {
				t.Fatalf("a %d byte image was sent as %d bytes", tc.size, w.Body.Len())
			}
		})
	}
}

// An image which never ends is refused once it is over the limit, without reading it all.
func TestProxyStopsReadingEndlessImage(t *testing.T) {
	uri := imageServer(t, func(w http.ResponseWriter) {
		chunk := []byte(strings.Repeat("x", 1<<20))
		for range 4096 { // 4 GiB, if it was read to the end
			if _, err := w.Write(chunk); err != nil {
				return
			}
		}
	})
	const id = "endless"
	seedAssets(id)
	recordAsset(id, uri)

	start := time.Now()
	w := proxyRequest(id, uri)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status %d, want the image to be refused", w.Code)
	}
	if took := time.Since(start); took > 5*time.Second {
		t.Fatalf("took %v", took)
	}
}
