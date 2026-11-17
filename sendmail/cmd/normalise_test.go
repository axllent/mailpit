package cmd

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
	"time"
)

// hasLoneBreaks is whether there is a <CR> or <LF> on its own.
func hasLoneBreaks(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] == '\r' && (i+1 >= len(s) || s[i+1] != '\n') {
			return true
		}
		if s[i] == '\n' && (i == 0 || s[i-1] != '\r') {
			return true
		}
	}

	return false
}

// The message which is read has <CR><LF> for every line break, as a relayed one has. The <CR> on its own was sent as
// it was, and a body with "<CR>.<CR><LF>" is the end of the message to a server which takes a <CR> as a line break.
// (A <CR> on its own in the headers is text, as it is to Go and Postfix, and is replaced with a space; <CR><CR><LF> is
// one line break.)
func TestReadMessageHasCRLFLineBreaks(t *testing.T) {
	for name, tc := range map[string]struct{ in, want string }{
		"a CR on its own in the body": {
			"Subject: x\r\n\r\nline\r.\rMAIL FROM:<e@f.test>\rrest\r",
			"Subject: x\r\n\r\nline\r\n.\r\nMAIL FROM:<e@f.test>\r\nrest\r\n",
		},
		"the end of data with a CR": {
			"Subject: x\r\n\r\nline\r.\r\nMAIL FROM:<e@f.test>\r\n",
			"Subject: x\r\n\r\nline\r\n.\r\nMAIL FROM:<e@f.test>\r\n",
		},
		"a CR on its own in the headers": {
			"Subject: x\rBcc: e@f.test\r\n\r\nbody\r\n",
			"Subject: x Bcc: e@f.test\r\n\r\nbody\r\n",
		},
		"a LF on its own": {
			"Subject: x\n\nline\n.\nMAIL FROM:<e@f.test>\n",
			"Subject: x\r\n\r\nline\r\n.\r\nMAIL FROM:<e@f.test>\r\n",
		},
		"more than one CR before the LF": {
			"Subject: x\r\r\n\r\r\nbody\r\r\n",
			"Subject: x\r\n\r\nbody\r\n",
		},
		"all of them": {
			"To: a@b.test\nSubject: x\r\r\n\nbody\r.\r\nmore\n",
			"To: a@b.test\r\nSubject: x\r\n\r\nbody\r\n.\r\nmore\r\n",
		},
	} {
		got, _, err := readMessage(strings.NewReader(tc.in))
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}

		if hasLoneBreaks(string(got)) {
			t.Errorf("%s: a line break is not <CR><LF>: %q", name, got)
		}
		if string(got) != tc.want {
			t.Errorf("%s: got %q, want %q", name, got, tc.want)
		}
	}

	// control: a message which has them already is as it was, and so is one which does not end with one
	for name, in := range map[string]string{
		"a message":            "From: a@b.test\r\nTo: c@d.test\r\nSubject: x\r\n\r\nbody\r\nmore\r\n",
		"with no end":          "From: a@b.test\r\n\r\nbody",
		"with the end of data": "From: a@b.test\r\n\r\nline\r\n.\r\nmore\r\n",
		"with a long body":     "From: a@b.test\r\n\r\n" + strings.Repeat("0123456789\r\n", 5000),
	} {
		got, _, err := readMessage(strings.NewReader(in))
		if err != nil || string(got) != in {
			t.Errorf("%s: changed, or an error (%v): %q", name, err, got)
		}
	}
}

// The recipients are the ones which a server would take from the headers: a header with white space before its colon
// is the header ("To : a@b.test" was not the To header to the parser, so there were no recipients).
func TestReadMessageHeadersAreRead(t *testing.T) {
	for name, tc := range map[string]struct {
		in   string
		want string
	}{
		"white space before the colon": {"To : a@example.test\r\nSubject: x\r\n\r\nbody\r\n", "a@example.test"},
		"a tab before the colon":       {"To\t: a@example.test\r\nSubject: x\r\n\r\nbody\r\n", "a@example.test"},
		"with LF line breaks":          {"To: a@example.test\nSubject: x\n\nbody\n", "a@example.test"},
		"control: nothing to change":   {"To: a@example.test\r\nSubject: x\r\n\r\nbody\r\n", "a@example.test"},
	} {
		body, msg, err := readMessage(strings.NewReader(tc.in))
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}

		got := HeaderRecipients(msg.Header)
		if len(got) != 1 || got[0] != tc.want {
			t.Errorf("%s: recipients %q, want %q", name, got, tc.want)
		}
		if !strings.HasPrefix(string(body), "To: a@example.test\r\n") {
			t.Errorf("%s: the message sent has %q", name, body)
		}
	}
}

// What a server is sent: no <CR> or <LF> on its own, and what was in the body is the body (it was not the end of the
// message, and what followed was not a command). The message which was not changed (what ingest sends, as it is meant
// to replay messages) has the <CR> as it was.
func TestReadMessageWhatTheServerGets(t *testing.T) {
	in := "From: a@b.test\r\nTo: c@d.test\r\nSubject: x\r\n\r\nline\r.\r\nMAIL FROM:<evil@y.test>\r\nrest\r"

	body, _, err := readMessage(strings.NewReader(in))
	if err != nil {
		t.Fatal(err)
	}

	addr, rec := newRecorder(t, "tcp", "127.0.0.1:0")
	if err := Send(addr, "a@b.test", []string{"c@d.test"}, body); err != nil {
		t.Fatal(err)
	}

	time.Sleep(50 * time.Millisecond)
	rec.mu.Lock()
	data, after := string(rec.data), rec.after
	rec.mu.Unlock()

	if hasLoneBreaks(data) {
		t.Errorf("the server got a line break which is not <CR><LF>: %q", data)
	}
	if after != "QUIT" {
		t.Errorf("the command after the message was %q, want QUIT", after)
	}
	if !strings.Contains(data, "MAIL FROM:<evil@y.test>") {
		t.Errorf("the body is not all there: %q", data)
	}

	// how it was: sent as it is, the <CR> is on the wire on its own
	addr, rec = newRecorder(t, "tcp", "127.0.0.1:0")
	if err := Send(addr, "a@b.test", []string{"c@d.test"}, []byte(in)); err != nil {
		t.Fatal(err)
	}
	time.Sleep(50 * time.Millisecond)
	rec.mu.Lock()
	raw := string(rec.data)
	rec.mu.Unlock()
	if !hasLoneBreaks(raw) {
		t.Fatalf("the message as it was has no <CR> on its own, so this shows nothing: %q", raw)
	}
}

type failingInput struct{}

func (failingInput) Read([]byte) (int, error) { return 0, errors.New("input/output error") }

// The size of what is read is limited, exactly: a message of the maximum is read, one byte more is not, and what
// is wrong with the input is told apart.
func TestReadMessageSizeLimit(t *testing.T) {
	orig := maxMessageSize
	maxMessageSize = 1024
	t.Cleanup(func() { maxMessageSize = orig })

	msg := func(n int) string {
		head := "From: a@b.test\r\n\r\n"
		return head + strings.Repeat("x", n-len(head))
	}

	if body, _, err := readMessage(strings.NewReader(msg(1024))); err != nil || len(body) != 1024 {
		t.Fatalf("a message of the maximum size: %d bytes, %v", len(body), err)
	}

	_, _, err := readMessage(strings.NewReader(msg(1025)))
	var tooLarge messageTooLargeError
	if !errors.As(err, &tooLarge) || tooLarge.limit != 1024 {
		t.Fatalf("a message one byte over: %v", err)
	}

	// an endless input is not read to the end
	counted := &countingReader{}
	if _, _, err := readMessage(counted); !errors.As(err, &tooLarge) || counted.read > 1024+4096 {
		t.Fatalf("an endless input: %v after reading %d bytes", err, counted.read)
	}

	// control: input which can not be read, and input which is not a message, are not told as too large
	if _, _, err := readMessage(failingInput{}); !errors.Is(err, errReadingInput) || errors.As(err, &tooLarge) {
		t.Fatalf("input which can not be read: %v", err)
	}
	for _, in := range []string{"", "not a message"} {
		_, _, err := readMessage(strings.NewReader(in))
		if err == nil || errors.Is(err, errReadingInput) || errors.As(err, &tooLarge) {
			t.Errorf("%q: %v, want an error for a message which is not valid", in, err)
		}
	}
}

// countingReader is input which never ends.
type countingReader struct{ read int }

func (c *countingReader) Read(p []byte) (int, error) {
	n := min(len(p), 512)
	copy(p, bytes.Repeat([]byte("x"), n))
	c.read += n
	if c.read > 1<<20 {
		return n, io.EOF
	}

	return n, nil
}
