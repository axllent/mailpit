package pop3

// A small POP3 client used only to test the POP3 server.
//
// It is based on https://github.com/knadh/go-pop3 (borrowed to reduce dependencies), with changes so
// that it behaves as an RFC 1939 client: multi-line responses are un-stuffed (a leading "." is removed
// from any line that has one), lines of any length are read whole, replies that are too short are
// reported as errors instead of causing a panic, and every read and write has a deadline so that a
// server which stops responding fails a test promptly instead of hanging it.

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/mail"
	"strconv"
	"time"
)

// testClient implements a POP3 client.
type testClient struct {
	opt    testClientOpt
	dialer testDialer
}

// testConn is a stateful connection with the POP3 server.
type testConn struct {
	conn    net.Conn
	r       *bufio.Reader
	w       *bufio.Writer
	timeout time.Duration
}

// testClientOpt represents the client configuration.
type testClientOpt struct {
	// Host name
	Host string
	// Port number
	Port int
	// DialTimeout default is 3 seconds.
	DialTimeout time.Duration
	// Timeout is how long a single read or write may take, the default is 10 seconds.
	Timeout time.Duration
	// Dialer
	Dialer testDialer
	// TLSEnabled sets whether TLS is enabled
	TLSEnabled bool
	// TLSSkipVerify skips TLS verification (ie: self-signed)
	TLSSkipVerify bool
}

// testDialer interface
type testDialer interface {
	Dial(network, address string) (net.Conn, error)
}

// testMessageID contains the ID and size of an individual message.
type testMessageID struct {
	// ID is the numerical index (non-unique) of the message.
	ID int
	// Size in bytes
	Size int
	// UID is only present if the response is to the UIDL command.
	UID string
}

var (
	lineBreak   = []byte("\r\n")
	respOK      = []byte("+OK")   // `+OK` without additional info
	respOKInfo  = []byte("+OK ")  // `+OK <info>`
	respErr     = []byte("-ERR")  // `-ERR` without additional info
	respErrInfo = []byte("-ERR ") // `-ERR <info>`
)

// newTestClient returns a new client object.
func newTestClient(opt testClientOpt) *testClient {
	if opt.DialTimeout < time.Millisecond {
		opt.DialTimeout = time.Second * 3
	}
	if opt.Timeout < time.Millisecond {
		opt.Timeout = time.Second * 10
	}

	c := &testClient{
		opt:    opt,
		dialer: opt.Dialer,
	}

	if c.dialer == nil {
		c.dialer = &net.Dialer{Timeout: opt.DialTimeout}
	}

	return c
}

// NewConn creates and returns live POP3 server connection.
func (c *testClient) NewConn() (*testConn, error) {
	addr := fmt.Sprintf("%s:%d", c.opt.Host, c.opt.Port)

	conn, err := c.dialer.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}

	if c.opt.TLSEnabled {
		tlsCfg := tls.Config{} // #nosec
		if c.opt.TLSSkipVerify {
			// skip TLS host verification
			tlsCfg.InsecureSkipVerify = true // #nosec
		} else {
			tlsCfg.ServerName = c.opt.Host
		}

		conn = tls.Client(conn, &tlsCfg)
	}

	pCon := &testConn{
		conn:    conn,
		r:       bufio.NewReader(conn),
		w:       bufio.NewWriter(conn),
		timeout: c.opt.Timeout,
	}

	// Verify the connection by reading the welcome +OK greeting.
	if _, err := pCon.ReadOne(); err != nil {
		_ = conn.Close()

		return nil, err
	}

	return pCon, nil
}

// Send sends a POP3 command to the server. The given command is suffixed with "\r\n".
func (c *testConn) Send(b string) error {
	_ = c.conn.SetWriteDeadline(time.Now().Add(c.timeout))

	if _, err := c.w.WriteString(b + "\r\n"); err != nil {
		return err
	}

	return c.w.Flush()
}

// Cmd sends a command to the server. POP3 responses are either single line or multi-line.
// The first line always with -ERR in case of an error or +OK in case of a successful operation.
// +OK is always followed by a response on the same line which is either the actual response data
// in case of single line responses, or a help message followed by multiple lines of actual response
// data in case of multiline responses.
func (c *testConn) Cmd(cmd string, isMulti bool, args ...any) (*bytes.Buffer, error) {
	cmdLine := cmd
	for _, a := range args {
		cmdLine += fmt.Sprintf(" %v", a)
	}

	if err := c.Send(cmdLine); err != nil {
		return nil, err
	}

	// Read the first line of response to get the +OK/-ERR status.
	b, err := c.ReadOne()
	if err != nil {
		return nil, err
	}

	// Single line response.
	if !isMulti {
		return bytes.NewBuffer(b), err
	}

	return c.ReadAll()
}

// readLine reads one whole line, of any length, without its line ending.
func (c *testConn) readLine() ([]byte, error) {
	_ = c.conn.SetReadDeadline(time.Now().Add(c.timeout))

	var line []byte
	for {
		b, isPrefix, err := c.r.ReadLine()
		if err != nil {
			return nil, err
		}
		line = append(line, b...)
		if !isPrefix {
			return line, nil
		}
	}
}

// ReadOne reads a single line response from the conn.
func (c *testConn) ReadOne() ([]byte, error) {
	b, err := c.readLine()
	if err != nil {
		return nil, err
	}

	return parseResp(b)
}

// ReadAll reads the lines of a multi-line response until the terminating "." line, and returns
// them with the byte-stuffing removed (RFC 1939, section 3), each ended with CRLF.
func (c *testConn) ReadAll() (*bytes.Buffer, error) {
	buf := &bytes.Buffer{}

	for {
		b, err := c.readLine()
		if err != nil {
			return nil, err
		}

		// "." on its own indicates the end of a multi-line response.
		if bytes.Equal(b, []byte(".")) {
			break
		}

		// a line which begins with the termination octet has had another added
		b = bytes.TrimPrefix(b, []byte("."))

		buf.Write(b)
		buf.Write(lineBreak)
	}

	return buf, nil
}

// Auth authenticates the given credentials with the server.
func (c *testConn) Auth(user, password string) error {
	if err := c.User(user); err != nil {
		return err
	}

	if err := c.Pass(password); err != nil {
		return err
	}

	// Issue a NOOP to force the server to respond to the auth.
	return c.Noop()
}

// User sends the username to the server.
func (c *testConn) User(s string) error {
	_, err := c.Cmd("USER", false, s)

	return err
}

// Pass sends the password to the server.
func (c *testConn) Pass(s string) error {
	_, err := c.Cmd("PASS", false, s)

	return err
}

// Stat returns the number of messages and their total size in bytes in the inbox.
func (c *testConn) Stat() (int, int, error) {
	b, err := c.Cmd("STAT", false)
	if err != nil {
		return 0, 0, err
	}

	// count size
	f := bytes.Fields(b.Bytes())
	if len(f) < 2 {
		return 0, 0, fmt.Errorf("invalid STAT response, expected a count and a size: %q", b.String())
	}

	count, err := strconv.Atoi(string(f[0]))
	if err != nil {
		return 0, 0, err
	}

	size, err := strconv.Atoi(string(f[1]))
	if err != nil {
		return 0, 0, err
	}

	return count, size, nil
}

// List returns a list of (message ID, message Size) pairs.
// If the optional msgID > 0, then only that particular message is listed.
// The message IDs are sequential, 1 to N.
func (c *testConn) List(msgID int) ([]testMessageID, error) {
	var (
		buf *bytes.Buffer
		err error
	)

	if msgID <= 0 {
		// Multiline response listing all messages.
		buf, err = c.Cmd("LIST", true)
	} else {
		// Single line response listing one message.
		buf, err = c.Cmd("LIST", false, msgID)
	}
	if err != nil {
		return nil, err
	}

	var out []testMessageID

	for _, l := range bytes.Split(buf.Bytes(), lineBreak) {
		// id size
		f := bytes.Fields(l)
		if len(f) == 0 {
			break
		}
		if len(f) < 2 {
			return nil, fmt.Errorf("invalid LIST line, expected an id and a size: %q", l)
		}

		id, err := strconv.Atoi(string(f[0]))
		if err != nil {
			return nil, err
		}

		size, err := strconv.Atoi(string(f[1]))
		if err != nil {
			return nil, err
		}

		out = append(out, testMessageID{ID: id, Size: size})
	}

	return out, nil
}

// Uidl returns a list of (message ID, message UID) pairs. If the optional msgID
// is > 0, then only that particular message is listed. It works like Top() but only works on
// servers that support the UIDL command. Messages size field is not available in the UIDL response.
func (c *testConn) Uidl(msgID int) ([]testMessageID, error) {
	var (
		buf *bytes.Buffer
		err error
	)

	if msgID <= 0 {
		// Multiline response listing all messages.
		buf, err = c.Cmd("UIDL", true)
	} else {
		// Single line response listing one message.
		buf, err = c.Cmd("UIDL", false, msgID)
	}
	if err != nil {
		return nil, err
	}

	var out []testMessageID

	for _, l := range bytes.Split(buf.Bytes(), lineBreak) {
		// id uid
		f := bytes.Fields(l)
		if len(f) == 0 {
			break
		}
		if len(f) < 2 {
			return nil, fmt.Errorf("invalid UIDL line, expected an id and a unique id: %q", l)
		}

		id, err := strconv.Atoi(string(f[0]))
		if err != nil {
			return nil, err
		}

		out = append(out, testMessageID{ID: id, UID: string(f[1])})
	}

	return out, nil
}

// Retr downloads a message by the given msgID, parses it and returns it as a *mail.Message.
func (c *testConn) Retr(msgID int) (*mail.Message, error) {
	b, err := c.Cmd("RETR", true, msgID)
	if err != nil {
		return nil, err
	}

	return mail.ReadMessage(b)
}

// RetrRaw downloads a message by the given msgID and returns the raw []byte
// of the entire message.
func (c *testConn) RetrRaw(msgID int) (*bytes.Buffer, error) {
	return c.Cmd("RETR", true, msgID)
}

// Top retrieves a message by its ID with full headers and numLines lines of the body.
func (c *testConn) Top(msgID int, numLines int) (*mail.Message, error) {
	b, err := c.Cmd("TOP", true, msgID, numLines)
	if err != nil {
		return nil, err
	}

	return mail.ReadMessage(b)
}

// TopRaw retrieves the headers and numLines lines of the body of a message, as sent.
func (c *testConn) TopRaw(msgID int, numLines int) (*bytes.Buffer, error) {
	return c.Cmd("TOP", true, msgID, numLines)
}

// Dele deletes one or more messages. The server only executes the
// deletions after a successful Quit().
func (c *testConn) Dele(msgID ...int) error {
	for _, id := range msgID {
		if _, err := c.Cmd("DELE", false, id); err != nil {
			return err
		}
	}

	return nil
}

// Rset clears the messages marked for deletion in the current session.
func (c *testConn) Rset() error {
	_, err := c.Cmd("RSET", false)

	return err
}

// Noop issues a do-nothing NOOP command to the server. This is useful for
// prolonging open connections.
func (c *testConn) Noop() error {
	_, err := c.Cmd("NOOP", false)

	return err
}

// Quit sends the QUIT command to server and gracefully closes the connection.
// Message deletions (DELE command) are only executed by the server on a graceful
// quit and close.
func (c *testConn) Quit() error {
	defer func() { _ = c.conn.Close() }()

	_, err := c.Cmd("QUIT", false)

	return err
}

// parseResp checks if the response is an error that starts with `-ERR`
// and returns an error with the message that succeeds the error indicator.
// For success `+OK` messages, it returns the remaining response bytes.
func parseResp(b []byte) ([]byte, error) {
	if len(b) == 0 {
		return nil, nil
	}

	if bytes.Equal(b, respOK) {
		return nil, nil
	} else if after, ok := bytes.CutPrefix(b, respOKInfo); ok {
		return after, nil
	} else if bytes.Equal(b, respErr) {
		return nil, errors.New("unknown error (no info specified in response)")
	} else if after, ok := bytes.CutPrefix(b, respErrInfo); ok {
		return nil, errors.New(string(after))
	}

	return nil, fmt.Errorf("unknown response: %s. Neither -ERR, nor +OK", string(b))
}
