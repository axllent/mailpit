package spamc

import (
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const okReport = "SPAMD/1.1 0 EX_OK\r\nContent-length: 99\r\n\r\nSpam: False ; 0.7 / 5.0\r\n\r\n" +
	"Content analysis details:   (0.7 points, 5.0 required)\r\n\r\n" +
	" pts rule name              description\r\n" +
	"---- ---------------------- --------------------------------------------------\r\n" +
	" 0.7 FOO_RULE               Something\r\n" +
	" 1.5 BAR_RULE               A long description that\r\n" +
	"                            is wrapped\r\n"

// serve starts a fake spamd which reads the request and then writes reply.
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

func TestReport(t *testing.T) {
	addr := serve(t, "tcp", "127.0.0.1:0", okReport)

	res, err := NewTCP(addr, 5).Report([]byte("Subject: x\r\n\r\nbody"))
	if err != nil {
		t.Fatal(err)
	}
	if res.Score != 0.7 || res.Threshold != 5.0 || res.Spam {
		t.Errorf("score %v threshold %v spam %v", res.Score, res.Threshold, res.Spam)
	}
	if len(res.Rules) != 2 || res.Rules[0].Name != "FOO_RULE" || res.Rules[0].Points != "0.7" {
		t.Fatalf("rules: %+v", res.Rules)
	}
	if res.Rules[1].Description != "A long description that is wrapped" {
		t.Errorf("wrapped description: %q", res.Rules[1].Description)
	}
}

// An error reply used to be returned as a message with a score of 0.
func TestReportRejectsFailedResponses(t *testing.T) {
	tests := []struct {
		name  string
		reply string
		ok    bool
	}{
		{"control: a report", okReport, true},
		{"exploit: bad header", "SPAMD/1.1 76 Bad header line: x\r\n\r\n", false},
		{"exploit: too big", "SPAMD/1.1 65 Too big\r\n\r\n", false},
		{"exploit: empty", "", false},
		{"exploit: error code with a summary", "SPAMD/1.1 76 Bad header\r\n\r\nSpam: False ; 1.0 / 5.0\r\n", false},
		{"exploit: not spamd", "HTTP/1.1 200 OK\r\n\r\nhello", false},
		{"exploit: no summary", "SPAMD/1.1 0 EX_OK\r\n\r\n", false},
		{"exploit: NaN score", "SPAMD/1.1 0 EX_OK\r\n\r\nSpam: True ; NaN / 5.0\r\n", false},
		{"exploit: infinite score", "SPAMD/1.1 0 EX_OK\r\n\r\nSpam: True ; +Inf / 5.0\r\n", false},
		{"exploit: a later line looks like a header", "SPAMD/1.1 0 EX_OK\r\n\r\nSpam: False ; 1.0 / 5.0\r\nx/y 76 fail\r\n", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			addr := serve(t, "tcp", "127.0.0.1:0", tt.reply)
			res, err := NewTCP(addr, 5).Report([]byte("x"))
			if tt.ok && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !tt.ok && err == nil {
				t.Fatalf("no error for %q, got %+v", tt.reply, res)
			}
			if tt.ok && res.ResponseCode != 0 {
				t.Errorf("a later line replaced the response code: %d", res.ResponseCode)
			}
		})
	}
}

// NewUnix used to have a timeout of 0, which expired at once.
func TestUnixSocket(t *testing.T) {
	// a short path, as unix socket paths are limited in length
	dir, err := os.MkdirTemp("", "spamc")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	sock := filepath.Join(dir, "s.sock")

	serve(t, "unix", sock, okReport)

	if res, err := NewUnix(sock, 5).Report([]byte("x")); err != nil || res.Score != 0.7 {
		t.Fatalf("report over a unix socket: %v %v", res.Score, err)
	}

	pingSock := filepath.Join(dir, "p.sock")
	serve(t, "unix", pingSock, "SPAMD/1.5 0 PONG\r\n")
	if err := NewUnix(pingSock, 5).Ping(); err != nil {
		t.Fatalf("ping over a unix socket: %v", err)
	}

	// control: a missing socket still fails
	if err := NewUnix(filepath.Join(dir, "missing.sock"), 5).Ping(); err == nil {
		t.Fatal("ping of a missing socket succeeded")
	}
}

func TestZeroTimeoutIsNotImmediate(t *testing.T) {
	addr := serve(t, "tcp", "127.0.0.1:0", okReport)
	if _, err := NewTCP(addr, 0).Report([]byte("x")); err != nil {
		t.Fatalf("a timeout of 0 expired at once: %v", err)
	}
}

// The timeout used to apply only after connecting, so an unreachable address took the system's
// connect timeout (over 2 minutes).
func TestConnectIsBoundedByTheTimeout(t *testing.T) {
	// 192.0.2.0/24 is reserved for documentation, so is normally not routed. Where the network
	// refuses it at once, this passes trivially.
	start := time.Now()
	err := NewTCP("192.0.2.1:783", 1).Ping()
	if err == nil {
		t.Fatal("connected to a documentation address")
	}
	if d := time.Since(start); d > 5*time.Second {
		t.Fatalf("a timeout of 1s took %v", d)
	}
}

func TestPing(t *testing.T) {
	tests := []struct {
		name  string
		reply string
		ok    bool
	}{
		{"control: PONG", "SPAMD/1.5 0 PONG\r\n", true},
		{"control: a long header line after it", "SPAMD/1.5 0 PONG\r\nX: " + strings.Repeat("a", 6000) + "\r\n", true},
		{"exploit: something else", "220 smtp.example.com ESMTP\r\n", false},
		{"exploit: empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			addr := serve(t, "tcp", "127.0.0.1:0", tt.reply)
			if err := NewTCP(addr, 5).Ping(); (err == nil) != tt.ok {
				t.Fatalf("ping error: %v, want ok=%v", err, tt.ok)
			}
		})
	}
}

func TestResponseSizeLimit(t *testing.T) {
	orig := maxResponseBytes
	maxResponseBytes = 64 << 10
	t.Cleanup(func() { maxResponseBytes = orig })

	filler := strings.Repeat(" 0.1 RULE_NAME               filler\r\n", 1)
	rules := func(n int) string { return strings.Repeat(filler, n) }

	head := "SPAMD/1.1 0 EX_OK\r\n\r\nSpam: False ; 0.7 / 5.0\r\n\r\nContent analysis details:   (0.7 points, 5.0 required)\r\n"
	perRule := len(filler)

	// control: just under the limit is read in full
	n := (int(maxResponseBytes) - len(head)) / perRule
	addr := serve(t, "tcp", "127.0.0.1:0", head+rules(n))
	res, err := NewTCP(addr, 5).Report([]byte("x"))
	if err != nil || len(res.Rules) != n {
		t.Fatalf("a response within the limit: %d rules, %v", len(res.Rules), err)
	}

	// exploit: over the limit is refused
	addr = serve(t, "tcp", "127.0.0.1:0", head+rules(n+200))
	if _, err := NewTCP(addr, 5).Report([]byte("x")); err == nil {
		t.Fatal("a response over the limit was accepted")
	}
}

func TestParseScore(t *testing.T) {
	for in, want := range map[string]float64{"0.7": 0.7, "-1.5": -1.5, " 5 ": 5} {
		if got, err := ParseScore(in); err != nil || got != want {
			t.Errorf("ParseScore(%q) = %v, %v", in, got, err)
		}
	}
	for _, in := range []string{"", "x", "NaN", "Inf", "-Infinity", "1.2.3"} {
		if _, err := ParseScore(in); err == nil {
			t.Errorf("ParseScore(%q) did not fail", in)
		}
	}
}

func FuzzParseOutput(f *testing.F) {
	f.Add("SPAMD/1.1 0 EX_OK\nSpam: True ; 5.1 / 5.0\n\nContent analysis details:   (5.1 points, 5.0 required)\n 0.7 FOO_RULE  Something\n")
	f.Add("Spam: yes ; NaN / Inf\nContent analysis details\n -1.5 A_B ")
	f.Add("a/b 99999999999999999999 x")
	f.Fuzz(func(t *testing.T, s string) {
		rows := strings.Split(s, "\n")
		res, err := (&Client{}).parseOutput(rows)
		if err != nil {
			return
		}
		if len(res.Rules) > len(rows) {
			t.Fatal("more rules than rows")
		}
		for _, r := range res.Rules {
			if strings.ContainsAny(r.Name, " \n\r") {
				t.Fatalf("rule name %q", r.Name)
			}
		}
	})
}
