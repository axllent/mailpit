// Package pop3 is a simple POP3 server for Mailpit.
// By default it is disabled unless password credentials have been loaded.
//
// References: https://github.com/r0stig/golang-pop3 | https://github.com/inbucket/inbucket
// See RFC: https://datatracker.ietf.org/doc/html/rfc1939
package pop3

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/axllent/mailpit/config"
	"github.com/axllent/mailpit/internal/auth"
	"github.com/axllent/mailpit/internal/logger"
	"github.com/axllent/mailpit/internal/storage"
	"github.com/axllent/mailpit/server/websockets"
)

const (
	// UNAUTHORIZED state
	UNAUTHORIZED = 1
	// TRANSACTION state
	TRANSACTION = 2
	// UPDATE state
	UPDATE = 3
)

// Run will start the pop3 server if enabled
func Run() {
	if auth.POP3Credentials == nil || config.POP3Listen == "" {
		// POP3 server is disabled without authentication
		return
	}

	var listener net.Listener
	var err error

	if config.POP3TLSCert != "" {
		cer, err2 := tls.LoadX509KeyPair(config.POP3TLSCert, config.POP3TLSKey)
		if err2 != nil {
			logger.Log().Errorf("[pop3] %s", err2.Error())
			return
		}

		tlsConfig := &tls.Config{
			Certificates: []tls.Certificate{cer},
			MinVersion:   tls.VersionTLS12,
		}

		listener, err = tls.Listen("tcp", config.POP3Listen, tlsConfig)
	} else {
		// unencrypted
		listener, err = net.Listen("tcp", config.POP3Listen)
	}

	if err != nil {
		logger.Log().Errorf("[pop3] %s", err.Error())
		return
	}

	logger.Log().Infof("[pop3] starting on %s", config.POP3Listen)

	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}

		// run as goroutine
		go handleClient(conn)
	}
}

type message struct {
	ID   string
	Size float64
}

func handleClient(conn net.Conn) {
	var (
		user     = ""
		state    = 1
		toDelete = []string{}
	)

	defer func() {
		if state == UPDATE {
			for _, id := range toDelete {
				_ = storage.DeleteMessages([]string{id})
			}
			if len(toDelete) > 0 {
				// update web UI to remove deleted messages
				websockets.Broadcast("prune", nil)
			}
		}

		if err := conn.Close(); err != nil {
			logger.Log().Errorf("[pop3] %s", err.Error())
		}
	}()

	reader := bufio.NewReader(conn)

	messages := []message{}

	// State
	// 1 = Unauthorized
	// 2 = Transaction mode
	// 3 = update mode

	logger.Log().Debugf("[pop3] connection opened by %s", conn.RemoteAddr().String())

	// First welcome the new connection
	sendResponse(conn, "+OK Mailpit POP3 server")

	timeoutDuration := 30 * time.Second

	for {
		// POP3 server enforced a timeout of 30 seconds
		if err := conn.SetDeadline(time.Now().Add(timeoutDuration)); err != nil {
			logger.Log().Errorf("[pop3] %s", err.Error())
			return
		}

		// Reads a line from the client
		rawLine, err := reader.ReadString('\n')
		if err != nil {
			logger.Log().Errorf("[pop3] %s", err.Error())
			return
		}

		// Parses the command
		cmd, args := getCommand(rawLine)

		logger.Log().Debugf("[pop3] received: %s (%s)", strings.TrimSpace(rawLine), conn.RemoteAddr().String())

		if cmd == "CAPA" {
			// List our capabilities per RFC2449
			sendResponse(conn, "+OK Capability list follows")
			sendResponse(conn, "TOP")
			sendResponse(conn, "USER")
			sendResponse(conn, "UIDL")
			sendResponse(conn, "IMPLEMENTATION Mailpit")
			sendResponse(conn, ".")
			continue
		} else if cmd == "USER" && state == UNAUTHORIZED {
			if len(args) != 1 {
				sendResponse(conn, "-ERR must supply a user")
				return
			}
			// always true - stash for PASS
			sendResponse(conn, "+OK")
			user = args[0]

		} else if cmd == "PASS" && state == UNAUTHORIZED {
			if len(args) != 1 {
				sendResponse(conn, "-ERR must supply a password")
				return
			}

			pass := args[0]
			if authUser(user, pass) {
				sendResponse(conn, "+OK signed in")
				messages, err = getMessages()
				if err != nil {
					logger.Log().Errorf("[pop3] %s", err.Error())
				}
				state = 2
			} else {
				sendResponse(conn, "-ERR invalid password")
				logger.Log().Warnf("[pop3] failed login: %s", user)
			}

		} else if cmd == "STAT" && state == TRANSACTION {
			totalSize := float64(0)
			for _, m := range messages {
				totalSize = totalSize + m.Size
			}

			sendResponse(conn, fmt.Sprintf("+OK %d %d", len(messages), int64(totalSize)))

		} else if cmd == "LIST" && state == TRANSACTION {
			totalSize := float64(0)
			for _, m := range messages {
				totalSize = totalSize + m.Size
			}
			sendData(conn, fmt.Sprintf("+OK %d messages (%d octets)", len(messages), int64(totalSize)))

			// print all sizes
			for row, m := range messages {
				sendData(conn, fmt.Sprintf("%d %d", row+1, m.Size))
			}
			// end
			sendData(conn, ".")

		} else if cmd == "UIDL" && state == TRANSACTION {
			totalSize := float64(0)
			for _, m := range messages {
				totalSize = totalSize + m.Size
			}

			sendData(conn, "+OK unique-id listing follows")

			// print all message IDS
			for row, m := range messages {
				sendData(conn, fmt.Sprintf("%d %s", row+1, m.ID))
			}
			// end
			sendData(conn, ".")

		} else if cmd == "RETR" && state == TRANSACTION {
			if len(args) != 1 {
				sendResponse(conn, "-ERR no such message")
				return
			}

			nr, err := strconv.Atoi(args[0])
			if err != nil {
				sendResponse(conn, "-ERR no such message")
				return
			}

			if nr < 1 || nr > len(messages) {
				sendResponse(conn, "-ERR no such message")
				return
			}

			m := messages[nr-1]
			raw, err := storage.GetMessageRaw(m.ID)
			if err != nil {
				sendResponse(conn, "-ERR no such message")
				return
			}

			size := len(raw)
			sendData(conn, fmt.Sprintf("+OK %d octets", size))

			// When all lines of the response have been sent, a
			// final line is sent, consisting of a termination octet (decimal code
			// 046, ".") and a CRLF pair. If any line of the multi-line response
			// begins with the termination octet, the line is "byte-stuffed" by
			// pre-pending the termination octet to that line of the response.
			// @see: https://www.ietf.org/rfc/rfc1939.txt
			sendData(conn, strings.Replace(string(raw), "\n.", "\n..", -1))
			sendData(conn, ".")

		} else if cmd == "TOP" && state == TRANSACTION {
			arg, err := getSafeArg(args, 0)
			if err != nil {
				sendResponse(conn, "-ERR TOP requires two arguments")
				return
			}
			nr, err := strconv.Atoi(arg)
			if err != nil {
				sendResponse(conn, "-ERR TOP requires two arguments")
				return
			}

			if nr < 1 || nr > len(messages) {
				sendResponse(conn, "-ERR no such message")
				return
			}
			arg2, err := getSafeArg(args, 1)
			if err != nil {
				sendResponse(conn, "-ERR TOP requires two arguments")
				return
			}

			lines, err := strconv.Atoi(arg2)
			if err != nil {
				sendResponse(conn, "-ERR TOP requires two arguments")
				return
			}

			m := messages[nr-1]
			headers, body, err := getTop(m.ID, lines)
			if err != nil {
				sendResponse(conn, err.Error())
				return
			}

			sendData(conn, "+OK Top of message follows")
			sendData(conn, headers+"\r\n")
			sendData(conn, body)
			sendData(conn, ".")

		} else if cmd == "NOOP" && state == TRANSACTION {
			sendData(conn, "+OK")
		} else if cmd == "DELE" && state == TRANSACTION {
			arg, _ := getSafeArg(args, 0)
			nr, err := strconv.Atoi(arg)
			if err != nil {
				logger.Log().Warnf("[pop3] -ERR invalid DELETE integer: %s", arg)
				sendResponse(conn, "-ERR invalid integer")
				return
			}

			if nr < 1 || nr > len(messages) {
				logger.Log().Warnf("[pop3] -ERR no such message")
				sendResponse(conn, "-ERR no such message")
				return
			}
			toDelete = append(toDelete, messages[nr-1].ID)

			sendResponse(conn, "+OK")

		} else if cmd == "RSET" && state == TRANSACTION {
			toDelete = []string{}
			sendData(conn, "+OK")

		} else if cmd == "QUIT" {
			state = UPDATE
			return
		} else {
			logger.Log().Warnf("[pop3] -ERR %s not implemented", cmd)
			sendResponse(conn, fmt.Sprintf("-ERR %s not implemented", cmd))
		}
	}
}
