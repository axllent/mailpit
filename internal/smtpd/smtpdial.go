package smtpd

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"time"

	"github.com/axllent/mailpit/internal/tools"
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

// dialSMTP connects to the server and reads its greeting, with the time limits above. If useTLS is set the
// connection is TLS from the start (tlsConf is used), and host is the name the client knows the server by.
func dialSMTP(addr, host string, useTLS bool, tlsConf *tls.Config) (*smtp.Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), relayDialTimeout)
	defer cancel()

	dialer := &net.Dialer{}

	raw, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		if useTLS {
			return nil, fmt.Errorf("TLS dial error: %v", err)
		}

		return nil, fmt.Errorf("error connecting to %s: %v", addr, err)
	}

	// The idle time limits are on the network connection, below TLS if there is one.
	var conn net.Conn = tools.NewIdleConn(raw, relayIdleTimeout)

	if useTLS {
		// The SMTP client must be given the *tls.Conn itself: it only knows the connection is encrypted by its type,
		// and PLAIN authentication refuses to send the credentials over a connection it does not know is encrypted.
		tlsConn := tls.Client(conn, tlsConf)
		if err := tlsConn.HandshakeContext(ctx); err != nil {
			_ = raw.Close()
			return nil, fmt.Errorf("TLS dial error: %v", err)
		}

		conn = tlsConn
	}

	client, err := smtp.NewClient(conn, host)
	if err != nil {
		_ = conn.Close()

		if useTLS {
			return nil, fmt.Errorf("SMTP client error: %v", err)
		}

		return nil, fmt.Errorf("error connecting to %s: %v", addr, err)
	}

	return client, nil
}
