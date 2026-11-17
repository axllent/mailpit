package pop3

import (
	"bufio"
	"bytes"
	"errors"
	"net"
	"strings"
	"time"

	"github.com/axllent/mailpit/internal/auth"
	"github.com/axllent/mailpit/internal/logger"
	"github.com/axllent/mailpit/internal/storage"
	"github.com/axllent/mailpit/server/websockets"
	"golang.org/x/time/rate"
)

var (
	// writeTimeout is how long a single write to a client may take. It is applied to every
	// write rather than to a whole response, so a large message can be sent to a slow client
	// as long as it keeps making progress, while a client that stops reading is dropped.
	// It is a variable so that the tests can shorten it.
	writeTimeout = 60 * time.Second

	// broadcastClientError sends an error to the web UI, a variable for tests.
	broadcastClientError = websockets.BroadCastClientError

	// errorBroadcastLimiter limits how many client errors are broadcast to the web UI, as
	// any client can generate an error with a bad command and would otherwise flood it.
	errorBroadcastLimiter = rate.NewLimiter(rate.Limit(5), 10)
)

func authUser(username, password string) bool {
	return auth.POP3Credentials.Match(username, password)
}

// deadlineWriter sets a write deadline before every write to the connection.
type deadlineWriter struct {
	conn net.Conn
}

func (w deadlineWriter) Write(b []byte) (int, error) {
	_ = w.conn.SetWriteDeadline(time.Now().Add(writeTimeout))

	return w.conn.Write(b)
}

// Send a response with debug logging
func sendResponse(c net.Conn, m string) {
	_, _ = deadlineWriter{c}.Write([]byte(m + "\r\n"))
	logger.Log().Debugf("[pop3] response: %s", m)

	if strings.HasPrefix(m, "-ERR ") && errorBroadcastLimiter.Allow() {
		sub, _ := strings.CutPrefix(m, "-ERR ")
		broadcastClientError("error", "pop3", c.RemoteAddr().String(), sub)
	}
}

// sendMultiline writes data as the body of a multi-line response (RFC 1939, section 3).
// Every line is ended with CRLF, and a line beginning with the termination octet (".")
// is "byte-stuffed" by pre-pending another one. This includes the first line, which must
// not be treated differently to the rest. The terminating "." line is not written.
//
// The data is written as it is processed rather than copied, so sending a message does
// not use memory in addition to the message itself.
func sendMultiline(c net.Conn, data []byte) error {
	w := bufio.NewWriterSize(deadlineWriter{c}, 32*1024)

	for len(data) > 0 {
		var line []byte
		if i := bytes.IndexByte(data, '\n'); i >= 0 {
			line, data = data[:i], data[i+1:]
		} else {
			line, data = data, nil
		}
		line = bytes.TrimSuffix(line, []byte("\r"))

		if len(line) > 0 && line[0] == '.' {
			_ = w.WriteByte('.')
		}
		_, _ = w.Write(line)
		_, _ = w.WriteString("\r\n")
	}

	return w.Flush()
}

// Get the latest 100 messages
func getMessages() ([]message, error) {
	messages := []message{}
	list, err := storage.List(0, 0, 100)
	if err != nil {
		return messages, err
	}

	for _, m := range list {
		msg := message{}
		msg.ID = m.ID
		msg.Size = m.Size
		messages = append(messages, msg)
	}

	return messages, nil
}

// POP3 TOP command returns the headers, the blank line after them, and the next nr lines of the body
func getTop(id string, nr int) ([]byte, error) {
	raw, err := storage.GetMessageRaw(id)
	if err != nil {
		return nil, errors.New("-ERR no such message")
	}

	return topOfMessage(raw, nr), nil
}

// topOfMessage returns the start of a raw message: all of the headers, the blank line
// that ends them, and then up to nr lines of the body. A message with no blank line is
// all headers. Both CRLF and LF line endings are recognised.
func topOfMessage(raw []byte, nr int) []byte {
	// nextLine returns the end of the line starting at pos, including its line ending.
	nextLine := func(pos int) int {
		if i := bytes.IndexByte(raw[pos:], '\n'); i >= 0 {
			return pos + i + 1
		}

		return len(raw)
	}

	pos := 0
	for pos < len(raw) {
		next := nextLine(pos)
		blank := len(bytes.TrimRight(raw[pos:next], "\r\n")) == 0
		pos = next

		if blank {
			// the end of the headers, now add the requested body lines
			for i := 0; i < nr && pos < len(raw); i++ {
				pos = nextLine(pos)
			}

			break
		}
	}

	return raw[:pos]
}

// cuts the line into command and arguments
func getCommand(line string) (string, []string) {
	line = strings.Trim(line, "\r \n")
	cmd := strings.Split(line, " ")

	return cmd[0], cmd[1:]
}

func getSafeArg(args []string, nr int) (string, error) {
	if nr < len(args) {
		return args[nr], nil
	}

	return "", errors.New("-ERR out of range")
}

// sanitizeLogLine redacts sensitive commands (like PASS) from logs
// to prevent credentials from being logged in plaintext
func sanitizeLogLine(line string) string {
	cmd, _ := getCommand(line)
	cmd = strings.ToUpper(cmd)
	if cmd == "PASS" {
		return "PASS [redacted]"
	}

	return line
}
