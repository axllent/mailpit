package smtpd

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"time"
)

var (
	// relayDialTimeout is how long connecting to the server which messages are relayed or forwarded to may take,
	// including the TLS handshake if there is one. It is a variable for the tests.
	relayDialTimeout = 10 * time.Second

	// relayIdleTimeout is how long that server may take to answer (or accept what is sent) once connected. It is
	// not for the whole of the conversation, so a large message to a slow server is still sent as long as it moves.
	// It is a variable for the tests.
	relayIdleTimeout = time.Minute
)

// idleConn gives every read and write a deadline of its own. The SMTP client of the standard library has none, so a
// server which accepts the connection and then says nothing would hold the session of the sender for ever (the
// message is relayed before the sender is answered).
type idleConn struct {
	net.Conn
	timeout time.Duration
}

func (c *idleConn) Read(p []byte) (int, error) {
	_ = c.Conn.SetReadDeadline(time.Now().Add(c.timeout))

	return c.Conn.Read(p)
}

func (c *idleConn) Write(p []byte) (int, error) {
	_ = c.Conn.SetWriteDeadline(time.Now().Add(c.timeout))

	return c.Conn.Write(p)
}

// dialSMTP connects to the server and reads its greeting, with the time limits above. If useTLS is set the
// connection is TLS from the start (tlsConf is used), and host is the name the client knows the server by.
func dialSMTP(addr, host string, useTLS bool, tlsConf *tls.Config) (*smtp.Client, error) {
	dialer := &net.Dialer{Timeout: relayDialTimeout}

	var conn net.Conn
	var err error

	if useTLS {
		conn, err = tls.DialWithDialer(dialer, "tcp", addr, tlsConf)
		if err != nil {
			return nil, fmt.Errorf("TLS dial error: %v", err)
		}
	} else {
		conn, err = dialer.Dial("tcp", addr)
		if err != nil {
			return nil, fmt.Errorf("error connecting to %s: %v", addr, err)
		}
	}

	client, err := smtp.NewClient(&idleConn{Conn: conn, timeout: relayIdleTimeout}, host)
	if err != nil {
		_ = conn.Close()

		if useTLS {
			return nil, fmt.Errorf("SMTP client error: %v", err)
		}

		return nil, fmt.Errorf("error connecting to %s: %v", addr, err)
	}

	return client, nil
}
