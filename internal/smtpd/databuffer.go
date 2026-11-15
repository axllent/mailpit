package smtpd

import (
	"sync/atomic"
)

const (
	// defaultReceiveBytes is how many bytes of messages may be held by sessions which are receiving them (or waiting
	// to have them stored) at the same time, unless MaxSize is larger than a tenth of it.
	defaultReceiveBytes = 512 << 20

	// minDataChunk and maxDataChunk are the sizes of the pieces a message is received in: they start small so a small
	// message does not need much, and double to the largest.
	minDataChunk = 4 << 10
	maxDataChunk = 64 << 10
)

// receiveLimit limits how many bytes are held by the sessions which are receiving messages. Without it every session
// holds up to MaxSize while it receives a message and while it waits for its turn to be stored, so as many senders as
// there are connections (a thousand) could use more memory than there is. A session which would go over the limit is
// told to try again later, it does not wait: sessions which wait for each other while each holds part of what it needs
// could all be stuck.
type receiveLimit struct {
	used atomic.Int64
	max  int64
}

func newReceiveLimit(max int64) *receiveLimit {
	return &receiveLimit{max: max}
}

// reserve takes n bytes if there is room for them.
func (l *receiveLimit) reserve(n int64) bool {
	for {
		used := l.used.Load()
		if used+n > l.max {
			return false
		}

		if l.used.CompareAndSwap(used, used+n) {
			return true
		}
	}
}

func (l *receiveLimit) release(n int64) {
	l.used.Add(-n)
}

// dataBuffer holds a message while it is received, in pieces. A slice which is appended to has to be copied each
// time it is full, which leaves many times the size of the message for the garbage collector, and one buffer which
// is reused would keep the memory of the largest message for as long as the session lasts. The pieces are put
// together once, into a slice of exactly the size the message has.
type dataBuffer struct {
	chunks   [][]byte
	size     int
	limit    *receiveLimit // nil if the bytes are not counted
	reserved int64         // how much of the limit this has taken
}

// write adds data. It returns false, and keeps nothing more, if the limit has been reached.
func (d *dataBuffer) write(p []byte) bool {
	for len(p) > 0 {
		if n := len(d.chunks); n == 0 || len(d.chunks[n-1]) == cap(d.chunks[n-1]) {
			size := min(max(d.size, minDataChunk), maxDataChunk)

			if d.limit != nil {
				if !d.limit.reserve(int64(size)) {
					return false
				}
				d.reserved += int64(size)
			}

			d.chunks = append(d.chunks, make([]byte, 0, size))
		}

		last := &d.chunks[len(d.chunks)-1]
		take := min(cap(*last)-len(*last), len(p))
		*last = append(*last, p[:take]...)
		d.size += take
		p = p[take:]
	}

	return true
}

// join returns the prefix followed by the data, in a slice which has no more room than it needs. The pieces are no
// longer held afterwards, but what was taken of the limit is until release is called, as the result is as big as them.
func (d *dataBuffer) join(prefix []byte) []byte {
	out := make([]byte, 0, len(prefix)+d.size)
	out = append(out, prefix...)

	for i, c := range d.chunks {
		out = append(out, c...)
		d.chunks[i] = nil // so each can be freed while the others are copied
	}

	d.chunks, d.size = nil, 0

	return out
}

// release gives back what was taken of the limit, and drops the data. It can be called more than once.
func (d *dataBuffer) release() {
	if d.limit != nil && d.reserved > 0 {
		d.limit.release(d.reserved)
	}

	d.chunks, d.size, d.reserved = nil, 0, 0
}
