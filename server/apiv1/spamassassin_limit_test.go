package apiv1

import (
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/axllent/mailpit/internal/spamassassin"
)

const spamdReport = "SPAMD/1.1 0 EX_OK\r\n\r\nSpam: False ; 0.7 / 5.0\r\n\r\n" +
	"Content analysis details:   (0.7 points, 5.0 required)\r\n\r\n" +
	" 0.7 FOO_RULE               Something\r\n"

// spamd is a SpamAssassin server which answers every check with a report after delay, and counts the checks it
// received and the most which were at once.
func spamd(t *testing.T, delay time.Duration) (hits, peak *atomic.Int32) {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	hits, peak = &atomic.Int32{}, &atomic.Int32{}
	var current atomic.Int32
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go func() {
				defer func() { _ = c.Close() }()

				hits.Add(1)
				n := current.Add(1)
				defer current.Add(-1)
				for {
					p := peak.Load()
					if n <= p || peak.CompareAndSwap(p, n) {
						break
					}
				}

				_, _ = io.Copy(io.Discard, c)
				time.Sleep(delay)
				_, _ = io.WriteString(c, spamdReport)
			}()
		}
	}()

	spamassassin.SetService(ln.Addr().String())
	t.Cleanup(func() { spamassassin.SetService("") })

	return hits, peak
}

func saCheck(id string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/message/"+id+"/sa-check", nil)
	req.SetPathValue("id", id)
	w := httptest.NewRecorder()
	SpamAssassinCheck(w, req)

	return w
}

// However many requests there are, only a few messages are sent to SpamAssassin at once, and a message which has
// been checked is not sent again. There was no limit: 25 requests made 25 connections at the same time, and each
// loaded the message (up to the maximum message size).
func TestSpamAssassinCheckIsLimited(t *testing.T) {
	initTestDB(t)
	hits, peak := spamd(t, 300*time.Millisecond)

	var ids []string
	for i := range 12 {
		ids = append(ids, storeSimpleEmail(t, "spam "+string(rune('a'+i))))
	}

	// 12 different messages at once
	var wg sync.WaitGroup
	for _, id := range ids {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if w := saCheck(id); w.Code != http.StatusOK {
				t.Errorf("status %d: %s", w.Code, w.Body.String())
			}
		}()
	}
	wg.Wait()

	if got := peak.Load(); got > 4 || got == 0 {
		t.Fatalf("%d checks were sent to SpamAssassin at once, want at most 4", got)
	}
	if got := hits.Load(); got != 12 {
		t.Fatalf("%d checks were sent, want 12 (one for each message)", got)
	}

	// the same message asked for by 25 requests at once is sent by at most the ones which got a slot before the
	// first result was kept (the others wait, and then use it), and asked for again it is not sent at all
	hits.Store(0)
	again := storeSimpleEmail(t, "again")
	for range 25 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = saCheck(again)
		}()
	}
	wg.Wait()

	burst := hits.Load()
	if burst < 1 || burst > 4 {
		t.Fatalf("the same message was sent %d times by 25 requests, want 4 at most", burst)
	}

	for range 10 {
		_ = saCheck(again)
	}
	if got := hits.Load(); got != burst {
		t.Fatalf("the message was sent %d more times after it had been checked, want none", got-burst)
	}
}

// A message which does not exist is a 404, without reaching SpamAssassin, and the result for one which does is the
// report.
func TestSpamAssassinCheckResults(t *testing.T) {
	initTestDB(t)
	hits, _ := spamd(t, 0)

	if w := saCheck("doesnotexist"); w.Code != http.StatusNotFound {
		t.Fatalf("status %d, want 404 for a message which does not exist", w.Code)
	}
	if hits.Load() != 0 {
		t.Fatal("a message which does not exist was sent to SpamAssassin")
	}

	id := storeSimpleEmail(t, "report")
	w := saCheck(id)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"Score":0.7`) || !strings.Contains(w.Body.String(), "FOO_RULE") {
		t.Fatalf("status %d: %s", w.Code, w.Body.String())
	}
}
