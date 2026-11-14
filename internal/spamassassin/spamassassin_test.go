package spamassassin

import (
	"encoding/json"
	"io"
	"net"
	"os"
	"path/filepath"
	"testing"
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
