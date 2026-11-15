package apiv1

import (
	"bufio"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"net/mail"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/axllent/mailpit/config"
)

// relayUpstream is an SMTP server which accepts a message, and records the RCPT commands it received.
func relayUpstream(t *testing.T) (host string, port int, rcpts func() []string, reset func()) {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	var mu sync.Mutex
	var got []string

	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go func() {
				defer func() { _ = c.Close() }()
				w := func(s string) { _, _ = c.Write([]byte(s + "\r\n")) }
				r := bufio.NewReader(c)
				w("220 fake")
				for {
					l, err := r.ReadString('\n')
					if err != nil {
						return
					}
					cmd := strings.ToUpper(strings.TrimSpace(l))
					switch {
					case strings.HasPrefix(cmd, "RCPT"):
						mu.Lock()
						got = append(got, strings.TrimSpace(l))
						mu.Unlock()
						w("250 ok")
					case cmd == "DATA":
						w("354 go")
						for {
							d, err := r.ReadString('\n')
							if err != nil || d == ".\r\n" {
								break
							}
						}
						w("250 ok")
					case cmd == "QUIT":
						w("221 bye")
						return
					default:
						w("250 ok")
					}
				}
			}()
		}
	}()

	_, p, _ := net.SplitHostPort(ln.Addr().String())
	port, _ = strconv.Atoi(p)

	return "127.0.0.1", port, func() []string {
			mu.Lock()
			defer mu.Unlock()
			return append([]string(nil), got...)
		}, func() {
			mu.Lock()
			got = nil
			mu.Unlock()
		}
}

func release(t *testing.T, id string, to ...string) *httptest.ResponseRecorder {
	t.Helper()

	body, _ := json.Marshal(struct{ To []string }{to})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/message/"+id+"/release", strings.NewReader(string(body)))
	req.SetPathValue("id", id)
	w := httptest.NewRecorder()
	ReleaseMessage(w, req)

	return w
}

// The address which is sent to is the one which was checked against the allow list. What was given (a name, a quoted
// name, a comment) was sent as it was, so the server it was relayed to received "RCPT TO:<"x@y" <ok@allowed.test>>",
// which it could read as another address than the one which was allowed.
func TestReleaseSendsTheAddressesWhichWereChecked(t *testing.T) {
	initTestDB(t)
	id := storeSimpleEmail(t, "release")

	host, port, rcpts, reset := relayUpstream(t)
	orig := config.SMTPRelayConfig
	config.SMTPRelayConfig = config.SMTPRelayConfigStruct{Host: host, Port: port, AllowedRecipientsRegexp: regexp.MustCompile(`@allowed\.test$`)}
	t.Cleanup(func() { config.SMTPRelayConfig = orig })

	for _, tc := range []struct{ to, want string }{
		{`ok@allowed.test`, `RCPT TO:<ok@allowed.test>`}, // control: an address is sent as it is
		{`<ok@allowed.test>`, `RCPT TO:<ok@allowed.test>`},
		{`Evil <ok@allowed.test>`, `RCPT TO:<ok@allowed.test>`},
		{`"evil@x.test" <ok@allowed.test>`, `RCPT TO:<ok@allowed.test>`},
		{`ok@allowed.test (evil@x.test)`, `RCPT TO:<ok@allowed.test>`},
		// control: a quoted local part is still quoted (the checked address has no quotes, so using it as it is would
		// send evil@x.test@allowed.test, which can be read two ways)
		{`"a b"@allowed.test`, `RCPT TO:<"a b"@allowed.test>`},
		{`"evil@x.test"@allowed.test`, `RCPT TO:<"evil@x.test"@allowed.test>`},
	} {
		reset()

		if w := release(t, id, tc.to); w.Code != http.StatusOK {
			t.Fatalf("%q: status %d: %s", tc.to, w.Code, w.Body.String())
		}

		got := rcpts()
		if len(got) != 1 || got[0] != tc.want {
			t.Errorf("%q: sent %q, want %q", tc.to, got, tc.want)
			continue
		}

		// what was sent is read by the parser as the address which was checked
		sent, err := mail.ParseAddress(strings.TrimSuffix(strings.TrimPrefix(got[0], "RCPT TO:<"), ">"))
		checked, _ := mail.ParseAddress(tc.to)
		if err != nil || checked == nil || sent.Address != checked.Address {
			t.Errorf("%q: what was sent (%q) is not the address which was checked", tc.to, got[0])
		}
	}
}

// The allow and block lists still decide, and nothing is sent for an address which does not pass (the controls for
// the change above).
func TestReleaseStillChecksTheLists(t *testing.T) {
	initTestDB(t)
	id := storeSimpleEmail(t, "release lists")

	host, port, rcpts, reset := relayUpstream(t)
	orig := config.SMTPRelayConfig
	config.SMTPRelayConfig = config.SMTPRelayConfigStruct{
		Host: host, Port: port,
		AllowedRecipientsRegexp: regexp.MustCompile(`@allowed\.test$`),
		BlockedRecipientsRegexp: regexp.MustCompile(`^blocked@`),
	}
	t.Cleanup(func() { config.SMTPRelayConfig = orig })

	for name, tc := range map[string]struct {
		to   []string
		code int
	}{
		"not on the allow list":           {[]string{`x@other.test`}, http.StatusBadRequest},
		"a name does not get it through":  {[]string{`Allowed <x@other.test>`}, http.StatusBadRequest},
		"one which does not pass of many": {[]string{`ok@allowed.test`, `x@other.test`}, http.StatusBadRequest},
		"on the block list":               {[]string{`blocked@allowed.test`}, http.StatusBadRequest},
		"not an address":                  {[]string{`not an address`}, http.StatusBadRequest},
		"none":                            {nil, http.StatusBadRequest},
	} {
		reset()
		if w := release(t, id, tc.to...); w.Code != tc.code {
			t.Errorf("%s: status %d: %s", name, w.Code, w.Body.String())
		}
		if got := rcpts(); len(got) != 0 {
			t.Errorf("%s: %q was sent", name, got)
		}
	}

	// control: several which pass are all sent
	reset()
	if w := release(t, id, `a@allowed.test`, `B <b@allowed.test>`); w.Code != http.StatusOK {
		t.Fatalf("status %d: %s", w.Code, w.Body.String())
	}
	if got := rcpts(); len(got) != 2 || got[0] != `RCPT TO:<a@allowed.test>` || got[1] != `RCPT TO:<b@allowed.test>` {
		t.Fatalf("sent %q", got)
	}
}
