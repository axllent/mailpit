package tools

import (
	"net"
	"time"
)

// IdleConn gives every read and write a deadline of its own, which is the time which it may take to get anywhere,
// not the time which all of them may take (so a large message to a slow server is still sent, as long as it moves).
// The SMTP client of the standard library has none, so a server which accepts the connection and then says nothing
// would hold the program which is sending to it for ever.
//
// The deadline can be changed with SetTimeout, for the reads and writes which follow. It is not safe to change it
// while the connection is being read or written, which for an SMTP client (which is used by one goroutine) it is not.
type IdleConn struct {
	net.Conn
	timeout time.Duration
}

// NewIdleConn returns the connection with a deadline of timeout for each read and write.
func NewIdleConn(conn net.Conn, timeout time.Duration) *IdleConn {
	return &IdleConn{Conn: conn, timeout: timeout}
}

// SetTimeout sets the deadline of the reads and writes which follow.
func (c *IdleConn) SetTimeout(timeout time.Duration) {
	c.timeout = timeout
}

// Read reads from the connection, with a deadline.
func (c *IdleConn) Read(p []byte) (int, error) {
	_ = c.Conn.SetReadDeadline(time.Now().Add(c.timeout))

	return c.Conn.Read(p)
}

// Write writes to the connection, with a deadline.
func (c *IdleConn) Write(p []byte) (int, error) {
	_ = c.Conn.SetWriteDeadline(time.Now().Add(c.timeout))

	return c.Conn.Write(p)
}
