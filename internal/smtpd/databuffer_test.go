package smtpd

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// The pieces are put together in order, in a slice which is exactly as big as it needs to be.
func TestDataBufferJoin(t *testing.T) {
	rnd := rand.New(rand.NewSource(1))

	for _, total := range []int{0, 1, minDataChunk - 1, minDataChunk, minDataChunk + 1, 3 * maxDataChunk, 5*maxDataChunk + 123} {
		var want []byte
		d := &dataBuffer{}
		for len(want) < total {
			p := make([]byte, min(1+rnd.Intn(9000), total-len(want)))
			rnd.Read(p)
			want = append(want, p...)
			if !d.write(p) {
				t.Fatal("write failed without a limit")
			}
		}

		for _, c := range d.chunks {
			if cap(c) > maxDataChunk {
				t.Fatalf("a piece of %d bytes, the largest is %d", cap(c), maxDataChunk)
			}
		}

		prefix := []byte("Received: x\r\n")
		got := d.join(prefix)
		if !bytes.Equal(got, append(append([]byte{}, prefix...), want...)) {
			t.Fatalf("total %d: the joined data is not the data", total)
		}
		if cap(got) != len(got) {
			t.Fatalf("total %d: capacity %d for %d bytes, there should be no spare room", total, cap(got), len(got))
		}
		if d.size != 0 || d.chunks != nil {
			t.Fatalf("total %d: the pieces are still held after join", total)
		}
	}

	// control: a small message does not take the largest piece
	d := &dataBuffer{}
	d.write([]byte("small"))
	if cap(d.chunks[0]) != minDataChunk {
		t.Fatalf("the first piece is %d bytes, want %d", cap(d.chunks[0]), minDataChunk)
	}
}

// Nothing more is taken once the limit is reached, and everything which was taken is given back (once).
func TestDataBufferLimit(t *testing.T) {
	limit := newReceiveLimit(100 << 10)
	d := &dataBuffer{limit: limit}

	chunk := make([]byte, 1000)
	var written int
	for d.write(chunk) {
		written += len(chunk)
		if written > 10<<20 {
			t.Fatal("the limit was never reached")
		}
	}

	if used := limit.used.Load(); used > limit.max || used != d.reserved {
		t.Fatalf("used %d, reserved %d, limit %d", used, d.reserved, limit.max)
	}

	d.release()
	d.release()
	if used := limit.used.Load(); used != 0 {
		t.Fatalf("%d is still taken after release", used)
	}

	// control: a limit which is not reached takes what is held, and write works
	limit = newReceiveLimit(10 << 20)
	d = &dataBuffer{limit: limit}
	if !d.write(make([]byte, 200<<10)) {
		t.Fatal("write failed within the limit")
	}
	if limit.used.Load() == 0 {
		t.Fatal("nothing was taken")
	}
	d.release()
	if limit.used.Load() != 0 {
		t.Fatal("not given back")
	}
}

// Sessions at the same time never take more than the limit: of sixteen which each want an eighth of it, eight are
// given it, however the goroutines are scheduled.
func TestReceiveLimitConcurrent(t *testing.T) {
	limit := newReceiveLimit(512 << 10)

	var granted atomic.Int32
	var wg sync.WaitGroup
	start := make(chan struct{})
	for range 16 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			if limit.reserve(64 << 10) {
				granted.Add(1)
			}
		}()
	}
	close(start)
	wg.Wait()

	if granted.Load() != 8 || limit.used.Load() != 512<<10 {
		t.Fatalf("%d were given room, %d bytes taken; want 8 and %d", granted.Load(), limit.used.Load(), 512<<10)
	}

	// control: it is free again when they give it back
	for range 8 {
		limit.release(64 << 10)
	}
	if limit.used.Load() != 0 || !limit.reserve(512<<10) {
		t.Fatal("the limit was not free after it was given back")
	}
}

// referenceReadData is the way the data was read before it was kept in pieces: each line is put together, and then
// looked at as a whole.
func referenceReadData(s *session) ([]byte, error) {
	var data []byte
	for {
		var line []byte
		for {
			fragment, err := s.br.ReadSlice('\n')
			line = append(line, fragment...)
			if err == nil {
				break
			}
			if err != bufio.ErrBufferFull {
				return nil, err
			}
			if s.srv.MaxSize > 0 && len(data)+len(line) > s.srv.MaxSize {
				s.drainData(false)
				return nil, maxSizeExceeded(s.srv.MaxSize)
			}
		}
		if bytes.Equal(line, []byte(".\r\n")) {
			break
		}
		if line[0] == '.' {
			line = line[1:]
		}
		if s.srv.MaxSize > 0 && len(data)+len(line) > s.srv.MaxSize {
			s.drainData(true)
			return nil, maxSizeExceeded(s.srv.MaxSize)
		}
		data = append(data, line...)
	}
	return data, nil
}

type countingReader struct {
	r io.Reader
	n int
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.n += n
	return n, err
}

// randomData makes a stream of lines of different lengths, with lines which start with a period, bare <LF>s, and
// (usually) a terminator and a command after it, which must be left unread.
func randomData(rnd *rand.Rand, dots bool) string {
	var b strings.Builder
	for range rnd.Intn(40) {
		n := rnd.Intn(300)
		if rnd.Intn(8) == 0 {
			n = rnd.Intn(70000) // longer than a piece
		}
		if dots && rnd.Intn(4) == 0 {
			b.WriteString(strings.Repeat(".", 1+rnd.Intn(2)))
		}
		// periods in the middle of a line, which can end up at the start of a piece of it
		line := make([]byte, n)
		for j := range line {
			line[j] = "xxx."[rnd.Intn(4)]
		}
		if !dots && n > 0 {
			line[0] = 'x' // a line which starts with a period and is longer than the buffer is counted a byte less
		}
		b.Write(line)
		switch rnd.Intn(5) {
		case 0:
			b.WriteString("\n")
		case 1:
			b.WriteString("\r")
			b.WriteString("\r\n")
		default:
			b.WriteString("\r\n")
		}
	}
	if rnd.Intn(10) != 0 {
		b.WriteString(".\r\nNOOP\r\n")
	}
	return b.String()
}

// The data which is read, and how much of the stream is used up (so the next command is read from the right place),
// are what they were before, with and without a size limit and with a reader buffer which is often full.
func TestReadChunksSameAsReference(t *testing.T) {
	rnd := rand.New(rand.NewSource(2))

	for i := range 400 {
		maxSize := 0
		if i%2 == 1 {
			maxSize = 1 + rnd.Intn(200000)
		}
		// a line which is longer than the buffer and starts with a period is counted a byte less than it was
		input := randomData(rnd, maxSize == 0)
		bufSize := []int{16, 64, 2048}[rnd.Intn(3)]

		run := func(read func(*session) ([]byte, error)) ([]byte, error, int) {
			cr := &countingReader{r: strings.NewReader(input)}
			s := &session{srv: &Server{MaxSize: maxSize}, br: bufio.NewReaderSize(cr, bufSize)}
			data, err := read(s)
			return data, err, cr.n - s.br.Buffered()
		}

		wantData, wantErr, wantUsed := run(referenceReadData)
		gotData, gotErr, gotUsed := run((*session).readData)

		if !bytes.Equal(gotData, wantData) || fmt.Sprint(gotErr) != fmt.Sprint(wantErr) || gotUsed != wantUsed {
			t.Fatalf("case %d (max %d, buffer %d, %d bytes): data %d/%d bytes, err %v/%v, used %d/%d (got/want)",
				i, maxSize, bufSize, len(input), len(gotData), len(wantData), gotErr, wantErr, gotUsed, wantUsed)
		}
	}
}

// smtpClient is a bare SMTP client for the limit tests.
type smtpClient struct {
	t  *testing.T
	c  net.Conn
	br *bufio.Reader
}

func newSMTPClient(t *testing.T, addr string) *smtpClient {
	t.Helper()
	c, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = c.Close() })

	cl := &smtpClient{t: t, c: c, br: bufio.NewReader(c)}
	cl.expect("220")

	return cl
}

func (cl *smtpClient) read() string {
	cl.t.Helper()
	_ = cl.c.SetReadDeadline(time.Now().Add(5 * time.Second))
	for {
		l, err := cl.br.ReadString('\n')
		if err != nil {
			cl.t.Fatalf("read: %v", err)
		}
		if len(l) < 4 || l[3] != '-' {
			return strings.TrimSpace(l)
		}
	}
}

func (cl *smtpClient) expect(code string) {
	cl.t.Helper()
	if got := cl.read(); !strings.HasPrefix(got, code) {
		cl.t.Fatalf("expected %s, got %q", code, got)
	}
}

func (cl *smtpClient) cmd(line, code string) {
	cl.t.Helper()
	_, _ = fmt.Fprintf(cl.c, "%s\r\n", line)
	cl.expect(code)
}

// begin starts a message and returns once the server is ready for the data.
func (cl *smtpClient) begin() {
	cl.t.Helper()
	cl.cmd("EHLO x", "250")
	cl.cmd("MAIL FROM:<a@b.test>", "250")
	cl.cmd("RCPT TO:<c@d.test>", "250")
	cl.cmd("DATA", "354")
}

func (cl *smtpClient) body(n int) {
	cl.t.Helper()
	_, _ = io.WriteString(cl.c, strings.Repeat(strings.Repeat("x", 76)+"\r\n", n/78+1))
}

func limitServer(t *testing.T, limit *receiveLimit, handler MsgIDHandler, options ...func(*Server)) (string, *Server) {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	srv := &Server{DisableReverseDNS: true, Hostname: "t", AppName: "t", Timeout: 5 * time.Second, MaxSize: 10 << 20, MsgIDHandler: handler, receiving: limit}
	for _, option := range options {
		option(srv)
	}
	go func() { _ = srv.Serve(ln) }()
	t.Cleanup(func() { _ = srv.Close(); _ = ln.Close() })

	return ln.Addr().String(), srv
}

func noopHandler(net.Addr, string, []string, []byte, *string) (string, error) { return "", nil }

// waitUsed waits until the sessions have taken at least n bytes of the limit (what they were sent has been read).
func waitUsed(t *testing.T, limit *receiveLimit, n int64) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for limit.used.Load() < n && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if used := limit.used.Load(); used < n {
		t.Fatalf("only %d bytes were taken of the limit, want %d", used, n)
	}
}

func waitUnused(t *testing.T, limit *receiveLimit) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for limit.used.Load() != 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if used := limit.used.Load(); used != 0 {
		t.Fatalf("%d bytes are still taken of the limit", used)
	}
}

// A message which is received while others hold all that is allowed is told to try again, and the session carries on
// as normal (the rest of the data is not read as commands). Once the others are done it is accepted. With room (the
// control) both are accepted at once.
func TestReceiveLimitRefusesAndRecovers(t *testing.T) {
	limit := newReceiveLimit(256 << 10)
	addr, _ := limitServer(t, limit, noopHandler)

	// A has sent most of its message, which holds its part of the limit
	a := newSMTPClient(t, addr)
	a.begin()
	a.body(200 << 10)
	waitUsed(t, limit, 200<<10)

	b := newSMTPClient(t, addr)
	b.begin()
	b.body(200 << 10)
	_, _ = io.WriteString(b.c, ".\r\n")
	b.expect("452")

	// B is still in sync: its next commands are answered as commands
	b.cmd("NOOP", "250")
	b.cmd("RSET", "250")

	// A finishes, and everything is given back
	_, _ = io.WriteString(a.c, ".\r\n")
	a.expect("250")
	waitUnused(t, limit)

	// B tries again
	b.begin()
	b.body(200 << 10)
	_, _ = io.WriteString(b.c, ".\r\n")
	b.expect("250")
	waitUnused(t, limit)

	// control: with room, both are accepted at the same time
	limit = newReceiveLimit(10 << 20)
	addr, _ = limitServer(t, limit, noopHandler)
	a, b = newSMTPClient(t, addr), newSMTPClient(t, addr)
	a.begin()
	a.body(200 << 10)
	b.begin()
	b.body(200 << 10)
	_, _ = io.WriteString(a.c, ".\r\n")
	_, _ = io.WriteString(b.c, ".\r\n")
	a.expect("250")
	b.expect("250")
	waitUnused(t, limit)
}

// A message which is waiting to be stored holds its part of the limit until it is.
func TestReceiveLimitHeldWhileHandled(t *testing.T) {
	limit := newReceiveLimit(256 << 10)
	started, hold := make(chan struct{}), make(chan struct{})
	var once sync.Once
	addr, _ := limitServer(t, limit, func(net.Addr, string, []string, []byte, *string) (string, error) {
		once.Do(func() { close(started) })
		<-hold
		return "", nil
	})

	a := newSMTPClient(t, addr)
	a.begin()
	a.body(200 << 10)
	_, _ = io.WriteString(a.c, ".\r\n")
	<-started

	b := newSMTPClient(t, addr)
	b.begin()
	b.body(200 << 10)
	_, _ = io.WriteString(b.c, ".\r\n")
	b.expect("452")

	close(hold)
	a.expect("250")
	waitUnused(t, limit)
}

// Whatever way a message ends, what it took is given back: finished, refused for size, handled with an error or a
// panic, for a recipient which was rejected, and a client which goes away part way through.
func TestReceiveLimitIsGivenBack(t *testing.T) {
	type scenario struct {
		name    string
		handler MsgIDHandler
		run     func(t *testing.T, cl *smtpClient)
	}

	send := func(n int) func(*testing.T, *smtpClient) {
		return func(_ *testing.T, cl *smtpClient) {
			cl.begin()
			cl.body(n)
			_, _ = io.WriteString(cl.c, ".\r\n")
		}
	}

	for _, sc := range []scenario{
		{"accepted", noopHandler, func(t *testing.T, cl *smtpClient) { send(100<<10)(t, cl); cl.expect("250") }},
		{"too large", noopHandler, func(t *testing.T, cl *smtpClient) { send(11<<20)(t, cl); cl.expect("552") }},
		{"handler error", func(net.Addr, string, []string, []byte, *string) (string, error) { return "", errors.New("no") },
			func(t *testing.T, cl *smtpClient) { send(100<<10)(t, cl); cl.expect("451") }},
		{"handler panic", func(net.Addr, string, []string, []byte, *string) (string, error) { panic("boom") },
			func(t *testing.T, cl *smtpClient) { send(100<<10)(t, cl); cl.expect("451") }},
		{"client goes away", noopHandler, func(t *testing.T, cl *smtpClient) {
			cl.begin()
			cl.body(100 << 10)
			_ = cl.c.Close()
		}},
	} {
		t.Run(sc.name, func(t *testing.T) {
			limit := newReceiveLimit(64 << 20)
			addr, _ := limitServer(t, limit, sc.handler)
			sc.run(t, newSMTPClient(t, addr))
			waitUnused(t, limit)
		})
	}

	// the message of a recipient which was rejected is not kept either (the session says it is queued, but drops it)
	t.Run("rejected recipient", func(t *testing.T) {
		var handled atomic.Bool
		limit := newReceiveLimit(64 << 20)
		addr, _ := limitServer(t, limit, func(net.Addr, string, []string, []byte, *string) (string, error) {
			handled.Store(true)
			return "", nil
		}, func(srv *Server) {
			srv.IgnoreRejectedRecipients = true
			srv.HandlerRcpt = func(net.Addr, string, string) bool { return false }
		})

		cl := newSMTPClient(t, addr)
		cl.cmd("EHLO x", "250")
		cl.cmd("MAIL FROM:<a@b.test>", "250")
		cl.cmd("RCPT TO:<c@d.test>", "250")
		cl.cmd("DATA", "354")
		cl.body(100 << 10)
		_, _ = io.WriteString(cl.c, ".\r\n")
		cl.expect("250")
		waitUnused(t, limit)

		if handled.Load() {
			t.Fatal("a message for a rejected recipient was handled")
		}
	})
}

// A session which has been refused for the limit does not keep what it had read of the message.
func TestReceiveLimitFirstChunkRefused(t *testing.T) {
	limit := newReceiveLimit(minDataChunk) // room for one piece only
	addr, _ := limitServer(t, limit, noopHandler)

	cl := newSMTPClient(t, addr)
	cl.begin()
	cl.body(100 << 10)
	_, _ = io.WriteString(cl.c, ".\r\n")
	cl.expect("452")
	waitUnused(t, limit)

	// control: a message which fits in the one piece is accepted
	cl.begin()
	_, _ = io.WriteString(cl.c, "short\r\n.\r\n")
	cl.expect("250")
	waitUnused(t, limit)
}

// The size limit is exact: a message of the maximum size is accepted, one byte more is not, however it is made up.
func TestReadChunksSizeBoundary(t *testing.T) {
	const max = 5000

	for _, shape := range []struct {
		name string
		make func(n int) string
	}{
		{"one line", func(n int) string { return strings.Repeat("x", n-2) + "\r\n" }},
		{"short lines", func(n int) string { return strings.Repeat("x", 18) + "\r\n" + strings.Repeat("y", n-20-2) + "\r\n" }},
		{"longer than the buffer", func(n int) string { return strings.Repeat("z", n-2) + "\r\n" }},
	} {
		for _, bufSize := range []int{16, 2048} {
			for n := max - 3; n <= max+3; n++ {
				msg := shape.make(n)
				run := func(read func(*session) ([]byte, error)) ([]byte, error) {
					s := &session{srv: &Server{MaxSize: max}, br: bufio.NewReaderSize(strings.NewReader(msg+".\r\n"), bufSize)}
					return read(s)
				}

				data, err := run((*session).readData)
				if n <= max && (err != nil || len(data) != n) {
					t.Fatalf("%s, buffer %d, %d bytes: %d bytes, %v; want it accepted", shape.name, bufSize, n, len(data), err)
				}
				if n > max && !errors.As(err, new(maxSizeExceededError)) {
					t.Fatalf("%s, buffer %d, %d bytes: err %v, data %d; want it refused", shape.name, bufSize, n, err, len(data))
				}

				// as it was
				refData, refErr := run(referenceReadData)
				if !bytes.Equal(data, refData) || fmt.Sprint(err) != fmt.Sprint(refErr) {
					t.Fatalf("%s, buffer %d, %d bytes: %v differs from the reference %v", shape.name, bufSize, n, err, refErr)
				}
			}
		}
	}
}

// The older Handler (which has no message ID) gives back what the message took as well.
func TestReceiveLimitGivenBackByHandler(t *testing.T) {
	limit := newReceiveLimit(64 << 20)
	addr, _ := limitServer(t, limit, nil, func(srv *Server) {
		srv.Handler = func(net.Addr, string, []string, []byte) error { return nil }
	})

	cl := newSMTPClient(t, addr)
	cl.begin()
	cl.body(100 << 10)
	_, _ = io.WriteString(cl.c, ".\r\n")
	cl.expect("250")
	waitUnused(t, limit)
}
