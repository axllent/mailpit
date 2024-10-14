package pop3

import (
	"errors"
	"fmt"
	"net"
	"strings"

	"github.com/axllent/mailpit/internal/auth"
	"github.com/axllent/mailpit/internal/logger"
	"github.com/axllent/mailpit/internal/storage"
	"github.com/axllent/mailpit/server/websockets"
)

func authUser(username, password string) bool {
	return auth.POP3Credentials.Match(username, password)
}

// Send a response with debug logging
func sendResponse(c net.Conn, m string) {
	fmt.Fprintf(c, "%s\r\n", m)
	logger.Log().Debugf("[pop3] response: %s", m)

	if strings.HasPrefix(m, "-ERR ") {
		sub, _ := strings.CutPrefix(m, "-ERR ")
		websockets.BroadCastClientError("error", "pop3", c.RemoteAddr().String(), sub)
	}
}

// Send a response without debug logging (for data)
func sendData(c net.Conn, m string) {
	fmt.Fprintf(c, "%s\r\n", m)
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

// POP3 TOP command returns the headers, followed by the next x lines
func getTop(id string, nr int) (string, string, error) {
	var header, body string
	raw, err := storage.GetMessageRaw(id)
	if err != nil {
		return header, body, errors.New("-ERR no such message")
	}

	parts := strings.SplitN(string(raw), "\r\n\r\n", 2)
	header = parts[0]
	lines := []string{}
	if nr > 0 && len(parts) == 2 {
		lines = strings.SplitN(parts[1], "\r\n", nr)
	}

	return header, strings.Join(lines, "\r\n"), nil
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
