package smtpd

import (
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

	client, err := smtp.NewClient(tools.NewIdleConn(conn, relayIdleTimeout), host)
	if err != nil {
		_ = conn.Close()

		if useTLS {
			return nil, fmt.Errorf("SMTP client error: %v", err)
		}

		return nil, fmt.Errorf("error connecting to %s: %v", addr, err)
	}

	return client, nil
}
