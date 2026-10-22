// Package connlimit provides a hard cap on concurrent connections for Mailpit's
// network listeners (SMTP, POP3 & HTTP), and a backoff helper for accept loops.
//
// Mailpit is a development tool, so the limit is intentionally not configurable.
// It is far above any realistic use, while preventing a single listener from
// exhausting the process' file descriptors and memory.
package connlimit

import (
	"crypto/tls"
	"errors"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/axllent/mailpit/internal/logger"
)

// Max is the maximum number of concurrent connections per service.
const Max = 1000

// warnInterval is the minimum time between "limit reached" log messages.
const warnInterval = time.Minute

// Limiter caps the number of concurrent connections for one service.
type Limiter struct {
	service  string
	sem      chan struct{}
	lastWarn atomic.Int64 // unix nanoseconds
}

// New returns a Limiter allowing up to max concurrent connections.
func New(service string, max int) *Limiter {
	return &Limiter{service: service, sem: make(chan struct{}, max)}
}

// TryAcquire reserves a slot without blocking. It logs a rate-limited
// warning and returns false when the limit has been reached.
func (l *Limiter) TryAcquire() bool {
	select {
	case l.sem <- struct{}{}:
		return true
	default:
		l.warn()
		return false
	}
}

// Release frees a slot previously reserved by Acquire or TryAcquire.
func (l *Limiter) Release() {
	<-l.sem
}

func (l *Limiter) warn() {
	now := time.Now().UnixNano()
	last := l.lastWarn.Load()
	if now-last < int64(warnInterval) || !l.lastWarn.CompareAndSwap(last, now) {
		return
	}
	logger.Log().Warnf("[%s] connection limit of %d reached, new connections are being refused or delayed", l.service, cap(l.sem))
}

// Listener wraps ln so that at most max connections are open at once. Accept
// blocks while the limit is reached, leaving excess connections in the
// kernel's backlog until a slot is freed by a connection closing.
func Listener(service string, ln net.Listener, max int) net.Listener {
	return &limitListener{Listener: ln, limiter: New(service, max), done: make(chan struct{})}
}

// Close closes the listener and unblocks any Accept waiting for a free slot.
func (l *limitListener) Close() error {
	l.closeOnce.Do(func() { close(l.done) })
	return l.Listener.Close()
}

type limitListener struct {
	net.Listener
	limiter   *Limiter
	done      chan struct{}
	closeOnce sync.Once
}

func (l *limitListener) Accept() (net.Conn, error) {
	if !l.limiter.TryAcquire() {
		select {
		case l.limiter.sem <- struct{}{}:
		case <-l.done:
			return nil, net.ErrClosed
		}
	}

	conn, err := l.Listener.Accept()
	if err != nil {
		l.limiter.Release()
		return nil, err
	}

	return &limitConn{Conn: conn, release: l.limiter.Release}, nil
}

type limitConn struct {
	net.Conn
	once    sync.Once
	release func()
}

func (c *limitConn) Close() error {
	err := c.Conn.Close()
	c.once.Do(c.release)
	return err
}

// Refuse sends msg to a connection that is over the limit and closes it.
// TLS connections are closed without a message, as writing would force a
// handshake inside the accept loop, letting a flood of refused connections
// delay the accept of legitimate ones.
func Refuse(conn net.Conn, msg string) {
	if _, isTLS := conn.(*tls.Conn); !isTLS {
		_ = conn.SetWriteDeadline(time.Now().Add(time.Second))
		_, _ = conn.Write([]byte(msg))
	}
	_ = conn.Close()
}

// Retryable reports whether an error returned by Accept is worth retrying,
// such as running out of file descriptors (EMFILE/ENFILE). Only a closed
// listener is permanent.
func Retryable(err error) bool {
	return !errors.Is(err, net.ErrClosed)
}

// Backoff provides exponential delays for retrying a failing Accept, in the
// same way as net/http (5ms doubling up to 1s).
type Backoff struct {
	delay time.Duration
}

// Wait sleeps for the current delay and increases it for the next call.
func (b *Backoff) Wait() {
	switch {
	case b.delay == 0:
		b.delay = 5 * time.Millisecond
	case b.delay < time.Second:
		b.delay = min(b.delay*2, time.Second)
	}
	time.Sleep(b.delay)
}

// Reset clears the delay after a successful Accept.
func (b *Backoff) Reset() {
	b.delay = 0
}
