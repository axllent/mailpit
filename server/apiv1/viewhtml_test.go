package apiv1

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/axllent/mailpit/internal/storage"
	"github.com/axllent/mailpit/internal/tools"
)

func storeHTMLEmail(t *testing.T, html string) string {
	t.Helper()

	raw := []byte("From: a@example.test\r\nTo: b@example.test\r\nSubject: html\r\n" +
		"MIME-Version: 1.0\r\nContent-Type: text/html; charset=utf-8\r\n\r\n" + html + "\r\n")
	id, err := storage.Store(&raw, nil)
	if err != nil {
		t.Fatal(err)
	}

	return id
}

func viewHTML(ctx context.Context, id string, query string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/view/"+id+".html"+query, nil).WithContext(ctx)
	rr := httptest.NewRecorder()
	GetMessageHTML(rr, req)

	return rr
}

const viewTestHTML = `<html><head><meta charset="utf-8"></head><body><p>Hello <a href="https://example.test/x">link</a></p></body></html>`

func TestViewHTMLEmbed(t *testing.T) {
	initTestDB(t)
	id := storeHTMLEmail(t, viewTestHTML)

	// embed=1 adds target and rel to the links
	rr := viewHTML(context.Background(), id, "?embed=1")
	body := rr.Body.String()
	if rr.Code != http.StatusOK || !strings.Contains(body, `target="_blank"`) || !strings.Contains(body, `rel="noreferrer noopener"`) || !strings.Contains(body, "window.parent.postMessage") {
		t.Errorf("status %d: %.300q", rr.Code, body)
	}

	// control: without it they are as they were
	rr = viewHTML(context.Background(), id, "")
	if body := rr.Body.String(); rr.Code != http.StatusOK || strings.Contains(body, `target="_blank"`) || !strings.Contains(body, "https://example.test/x") {
		t.Errorf("status %d: %.300q", rr.Code, body)
	}
}

// A client which has gone gets nothing written, with embed or without.
func TestViewHTMLForAClientWhichHasGone(t *testing.T) {
	initTestDB(t)
	id := storeHTMLEmail(t, viewTestHTML)

	gone, cancel := context.WithCancel(context.Background())
	cancel()

	for _, query := range []string{"?embed=1", ""} {
		rr := viewHTML(gone, id, query)
		if rr.Body.Len() != 0 || rr.Header().Get("Content-Type") != "" {
			t.Errorf("%q: something was written for a client which had gone: %d bytes", query, rr.Body.Len())
		}
	}
}

// The embed option parses the whole document, so it waits for a slot as removing the redirects does. It
// was not limited, and the requests used memory in proportion to how many there were.
func TestViewHTMLEmbedWaitsForASlot(t *testing.T) {
	initTestDB(t)
	// with nothing which could be a redirect (a <meta> or <base>), so that only the embed option can wait
	id := storeHTMLEmail(t, `<html><body><p>Hello <a href="https://example.test/x">link</a></p></body></html>`)

	// take every slot
	var releases []func()
	for {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
		release, err := tools.AcquireHTMLSlot(ctx)
		cancel()
		if err != nil {
			break
		}
		releases = append(releases, release)
	}
	if len(releases) == 0 {
		t.Fatal("no slots were taken")
	}
	t.Cleanup(func() {
		for _, r := range releases {
			r()
		}
	})

	done := make(chan *httptest.ResponseRecorder, 1)
	go func() { done <- viewHTML(context.Background(), id, "?embed=1") }()

	select {
	case <-done:
		t.Fatal("the embed option was processed while all the slots were in use")
	case <-time.After(300 * time.Millisecond):
	}

	// without embed, and with nothing to remove, nothing is waited for
	plain := storeHTMLEmail(t, "<p>nothing to remove</p>")
	if rr := viewHTML(context.Background(), plain, ""); rr.Code != http.StatusOK || !strings.Contains(rr.Body.String(), "nothing to remove") {
		t.Errorf("status %d", rr.Code)
	}

	// and when a slot is free it is
	releases[0]()
	releases = releases[1:]
	select {
	case rr := <-done:
		if rr.Code != http.StatusOK || !strings.Contains(rr.Body.String(), `target="_blank"`) {
			t.Errorf("status %d: %.200q", rr.Code, rr.Body.String())
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the waiting request was not processed once a slot was free")
	}
}

// A HTML part over the limit is not rendered, whatever the options are. The cap is of the HTML part, and the
// limit itself is allowed.
func TestViewHTMLTooLarge(t *testing.T) {
	initTestDB(t)

	id := storeHTMLEmail(t, `<html><body><p>Hello <a href="https://example.test/x">link</a></p></body></html>`)
	msg, err := storage.GetMessage(id)
	if err != nil {
		t.Fatal(err)
	}
	size := len(msg.HTML)

	orig := maxRenderHTMLBytes
	t.Cleanup(func() { maxRenderHTMLBytes = orig })

	// control: the limit is the size of the HTML part, which is rendered
	maxRenderHTMLBytes = size
	for _, query := range []string{"", "?embed=1"} {
		if rr := viewHTML(context.Background(), id, query); rr.Code != http.StatusOK || !strings.Contains(rr.Body.String(), "Hello") {
			t.Errorf("%q at the limit: status %d: %.100q", query, rr.Code, rr.Body.String())
		}
	}

	// exploit: one byte over
	maxRenderHTMLBytes = size - 1
	for _, query := range []string{"", "?embed=1"} {
		rr := viewHTML(context.Background(), id, query)
		if rr.Code != http.StatusRequestEntityTooLarge || !strings.Contains(rr.Body.String(), "too large to render") || strings.Contains(rr.Body.String(), "Hello") {
			t.Errorf("%q over the limit: status %d: %.100q", query, rr.Code, rr.Body.String())
		}
		if ct := rr.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/plain") {
			t.Errorf("%q: content type %q", query, ct)
		}
	}

	// it is refused before anything which takes a slot, so it does not wait for one
	var releases []func()
	for {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
		release, err := tools.AcquireHTMLSlot(ctx)
		cancel()
		if err != nil {
			break
		}
		releases = append(releases, release)
	}
	t.Cleanup(func() {
		for _, r := range releases {
			r()
		}
	})

	done := make(chan *httptest.ResponseRecorder, 1)
	go func() { done <- viewHTML(context.Background(), id, "?embed=1") }()
	select {
	case rr := <-done:
		if rr.Code != http.StatusRequestEntityTooLarge {
			t.Errorf("status %d", rr.Code)
		}
	case <-time.After(2 * time.Second):
		t.Error("a message which is too large waited for a slot")
	}

	// and the message is still there to read in the UI and the API
	if m, err := storage.GetMessage(id); err != nil || len(m.HTML) != size {
		t.Errorf("the message is changed: %v", err)
	}
}
