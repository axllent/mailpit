package spamassassin

import (
	"encoding/json"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/axllent/mailpit/internal/spamassassin/postmark"
)

const report = "SPAMD/1.1 0 EX_OK\r\n\r\nSpam: False ; 0.7 / 5.0\r\n\r\n" +
	"Content analysis details:   (0.7 points, 5.0 required)\r\n\r\n" +
	" 0.7 FOO_RULE               Something\r\n"

func serve(t *testing.T, network, addr, reply string) string {
	t.Helper()

	ln, err := net.Listen(network, addr)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go func() {
				defer c.Close()
				_, _ = io.Copy(io.Discard, c)
				_, _ = io.WriteString(c, reply)
			}()
		}
	}()

	return ln.Addr().String()
}

func use(t *testing.T, s string) {
	t.Helper()

	origService, origTimeout := service, timeout
	t.Cleanup(func() { service, timeout = origService, origTimeout })

	timeout = 5
	SetService(s)
}

func TestRound1dm(t *testing.T) {
	tests := map[float64]float64{
		0.7: 0.7, 0.3: 0.3, 1.1: 1.1, 2.3: 2.3, 4.9: 4.9, 5.0: 5.0,
		4.99: 4.9, 0.29: 0.2, 0.04: 0, -0.5: -0.5, -1.25: -1.3, 12.345: 12.3,
	}
	for in, want := range tests {
		if got := round1dm(in); got != want {
			t.Errorf("round1dm(%v) = %v, want %v", in, got, want)
		}
	}
}

// A score of 0.7 used to be shown as 0.6, as it was parsed as a float32 and then rounded down.
func TestCheckScores(t *testing.T) {
	use(t, serve(t, "tcp", "127.0.0.1:0", report))

	r, err := Check([]byte("Subject: x\r\n\r\nbody"))
	if err != nil || r.Error != "" {
		t.Fatalf("%v %q", err, r.Error)
	}
	if r.Score != 0.7 || r.IsSpam || len(r.Rules) != 1 || r.Rules[0].Score != 0.7 {
		t.Fatalf("score %v, rules %+v", r.Score, r.Rules)
	}
}

func TestCheckIsSpamAtThreshold(t *testing.T) {
	for reply, want := range map[string]bool{
		"SPAMD/1.1 0 EX_OK\r\n\r\nSpam: True ; 5.0 / 5.0\r\n":  true,
		"SPAMD/1.1 0 EX_OK\r\n\r\nSpam: False ; 4.9 / 5.0\r\n": false,
	} {
		use(t, serve(t, "tcp", "127.0.0.1:0", reply))
		if r, _ := Check([]byte("x")); r.IsSpam != want {
			t.Errorf("%q: IsSpam %v", reply, r.IsSpam)
		}
	}
}

// Failures used to give a result with no error and a score of 0, which looks like a clean message.
func TestCheckReportsSpamdFailures(t *testing.T) {
	for name, reply := range map[string]string{
		"too big":   "SPAMD/1.1 65 Too big\r\n\r\n",
		"empty":     "",
		"NaN score": "SPAMD/1.1 0 EX_OK\r\n\r\nSpam: True ; NaN / 5.0\r\n",
	} {
		use(t, serve(t, "tcp", "127.0.0.1:0", reply))
		r, err := Check([]byte("x"))
		if err != nil || r.Error == "" {
			t.Errorf("%s: err=%v Error=%q", name, err, r.Error)
		}
		// the result can always be sent as JSON
		if _, err := json.Marshal(r); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
}

func TestUnixService(t *testing.T) {
	dir, err := os.MkdirTemp("", "sa")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })

	// a relative path starting with characters of "unix:" used to be cut (nix.sock became .sock)
	wd, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })

	for _, sock := range []string{filepath.Join(dir, "abs.sock"), "nix.sock"} {
		serve(t, "unix", sock, report)
		use(t, "unix:"+sock)

		r, err := Check([]byte("x"))
		if err != nil || r.Error != "" || r.Score != 0.7 {
			t.Errorf("unix:%s: err=%v Error=%q score=%v", sock, err, r.Error, r.Score)
		}
	}

	// control: a socket that is not there is an error
	use(t, "unix:missing.sock")
	if r, _ := Check([]byte("x")); r.Error == "" {
		t.Error("no error for a missing socket")
	}
}

func TestNoService(t *testing.T) {
	use(t, "")
	if _, err := Check([]byte("x")); err == nil {
		t.Fatal("no error without a service")
	}
}

func spamdReply(summary string) string {
	return "SPAMD/1.1 0 EX_OK\r\n\r\n" + summary + "\r\n"
}

// SpamAssassin decides what is spam from the required score it is configured with, which is not always
// 5. The verdict and the threshold used to be ignored, and 5 used instead.
func TestCheckUsesTheServersThreshold(t *testing.T) {
	tests := []struct {
		name      string
		summary   string
		isSpam    bool
		score     float64
		threshold float64
	}{
		// a lower required score than 5, and a higher one
		{"exploit: spam below 5", "Spam: True ; 3.5 / 3.0", true, 3.5, 3},
		{"exploit: not spam above 5", "Spam: False ; 6.0 / 8.0", false, 6, 8},
		{"exploit: a required score with decimals", "Spam: False ; 1.0 / 4.25", false, 1, 4.25},
		{"exploit: spam just over a low score", "Spam: True ; 2.1 / 2.0", true, 2.1, 2},

		// controls: the usual required score, where it makes no difference
		{"control: spam at 5", "Spam: True ; 5.0 / 5.0", true, 5, 5},
		{"control: not spam under 5", "Spam: False ; 4.9 / 5.0", false, 4.9, 5},
		{"control: spam well over 5", "Spam: True ; 15.0 / 5.0", true, 15, 5},
		{"control: nothing", "Spam: False ; 0.0 / 5.0", false, 0, 5},

		// the words which spamd can use
		{"yes", "Spam: Yes ; 9.0 / 5.0", true, 9, 5},
		{"no", "Spam: No ; 1.0 / 5.0", false, 1, 5},
		{"upper case", "Spam: TRUE ; 9.0 / 5.0", true, 9, 5},

		// the threshold is shown, the verdict is the server's whatever it is
		{"a threshold which can not be read", "Spam: True ; 7.0 / x", true, 7, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			use(t, serve(t, "tcp", "127.0.0.1:0", spamdReply(tt.summary)))

			r, err := Check([]byte("x"))
			if err != nil || r.Error != "" {
				t.Fatalf("%v %q", err, r.Error)
			}
			if r.IsSpam != tt.isSpam || r.Score != tt.score || r.Threshold != tt.threshold {
				t.Errorf("IsSpam %v, Score %v, Threshold %v, want %v, %v, %v", r.IsSpam, r.Score, r.Threshold, tt.isSpam, tt.score, tt.threshold)
			}
		})
	}

	// the threshold is in the JSON, which the page uses for the score and the graph
	use(t, serve(t, "tcp", "127.0.0.1:0", spamdReply("Spam: False ; 1.0 / 3.5")))
	r, _ := Check([]byte("x"))
	b, err := json.Marshal(r)
	if err != nil || !strings.Contains(string(b), `"Threshold":3.5`) {
		t.Errorf("%s, %v", b, err)
	}
}

// A verdict which is not one which spamd uses is an error, and not a message which is not spam.
func TestCheckRejectsAnUnknownVerdict(t *testing.T) {
	for _, summary := range []string{"Spam: Maybe ; 9.0 / 5.0", "Spam: 1 ; 9.0 / 5.0", "Spam: spam ; 9.0 / 5.0"} {
		use(t, serve(t, "tcp", "127.0.0.1:0", spamdReply(summary)))

		r, err := Check([]byte("x"))
		if err != nil || r.Error == "" || r.IsSpam {
			t.Errorf("%q: err=%v Error=%q IsSpam=%v", summary, err, r.Error, r.IsSpam)
		}
	}
}

// Postmark does not say what its required score is, so it is the usual 5.
func TestFromPostmark(t *testing.T) {
	rules := []postmark.Rule{{Score: "0.7", Name: "FOO", Description: "Foo"}, {Score: "x", Name: "BAR"}}

	tests := []struct {
		name   string
		res    postmark.Response
		isSpam bool
		score  float64
		err    string
	}{
		{"spam at 5", postmark.Response{Success: true, Score: "5.0"}, true, 5, ""},
		{"not spam under 5", postmark.Response{Success: true, Score: "4.9"}, false, 4.9, ""},
		{"spam well over", postmark.Response{Success: true, Score: "12.3"}, true, 12.3, ""},
		{"not a number", postmark.Response{Success: true, Score: "NaN"}, false, 0, "Postmark returned an invalid score"},
		{"an error from Postmark", postmark.Response{Message: "Internal error"}, false, 0, "Internal error"},
		{"no result", postmark.Response{}, false, 0, "Postmark did not return a result"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.res.Rules = rules
			r := fromPostmark(tt.res)

			if r.IsSpam != tt.isSpam || r.Score != tt.score || r.Error != tt.err || r.Threshold != 5 {
				t.Errorf("IsSpam %v, Score %v, Error %q, Threshold %v", r.IsSpam, r.Score, r.Error, r.Threshold)
			}
		})
	}

	r := fromPostmark(postmark.Response{Success: true, Score: "1.0", Rules: rules})
	if len(r.Rules) != 2 || r.Rules[0].Score != 0.7 || r.Rules[0].Name != "FOO" || r.Rules[1].Score != 0 {
		t.Errorf("rules %+v", r.Rules)
	}
}
