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
	"io"
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
	// AUTHORIZATION is the initial state
	AUTHORIZATION = 1
	// TRANSACTION is the state after login
	TRANSACTION = 2
	// UPDATE is the state before closing
	UPDATE = 3
)

// Run will start the POP3 server if enabled
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
			logger.Log().Errorf("[pop3] accept error: %s", err.Error())
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
		state    = AUTHORIZATION // Start with AUTHORIZATION state
		toDelete []string        // Track messages marked for deletion
		messages []message
	)

	defer func() {
		if state == UPDATE {
			if len(toDelete) > 0 {
				if err := storage.DeleteMessages(toDelete); err != nil {
					logger.Log().Errorf("[pop3] error deleting: %s", err.Error())
				}
				// Update web UI to remove deleted messages
				websockets.Broadcast("prune", nil)
			}
		}

		if err := conn.Close(); err != nil {
			logger.Log().Errorf("[pop3] %s", err.Error())
		}
	}()

	reader := bufio.NewReader(conn)

	logger.Log().Debugf("[pop3] connection opened by %s", conn.RemoteAddr().String())

	// First welcome the new connection
	serverName := "Mailpit"
	if config.Label != "" {
		serverName = fmt.Sprintf("Mailpit (%s)", config.Label)
	}
	sendResponse(conn, fmt.Sprintf("+OK %s POP3 server", serverName))

	// Set 10 minutes timeout according to RFC1939
	timeoutDuration := 600 * time.Second

	for {
		// Set read deadline
		if err := conn.SetReadDeadline(time.Now().Add(timeoutDuration)); err != nil {
			logger.Log().Errorf("[pop3] %s", err.Error())
			return
		}

		// Reads a line from the client
		rawLine, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				logger.Log().Debugf("[pop3] client disconnected: %s", conn.RemoteAddr().String())
			} else {
				logger.Log().Errorf("[pop3] read error: %s", err.Error())
			}
			return
		}

		// Parses the command
		cmd, args := getCommand(rawLine)
		cmd = strings.ToUpper(cmd) // Commands in the POP3 are case-insensitive

		logger.Log().Debugf("[pop3] received: %s (%s)", strings.TrimSpace(rawLine), conn.RemoteAddr().String())

		switch cmd {
		case "CAPA":
			// List our capabilities per RFC2449
			sendResponse(conn, "+OK capability list follows")
			sendResponse(conn, "TOP")
			sendResponse(conn, "USER")
			sendResponse(conn, "UIDL")
			sendResponse(conn, "IMPLEMENTATION Mailpit")
			sendResponse(conn, ".")
		case "USER":
			if state == AUTHORIZATION {
				if len(args) != 1 {
					sendResponse(conn, "-ERR must supply a user")
					return
				}
				sendResponse(conn, "+OK")
				user = args[0]
			} else {
				sendResponse(conn, "-ERR user already specified")
			}
		case "PASS":
			if state == AUTHORIZATION {
				if user == "" {
					sendResponse(conn, "-ERR must supply a user")
					return
				}
				if len(args) != 1 {
					sendResponse(conn, "-ERR must supply a password")
					return
				}

				pass := args[0]
				if authUser(user, pass) {
					sendResponse(conn, "+OK signed in")
					var err error
					messages, err = getMessages()
					if err != nil {
						logger.Log().Errorf("[pop3] %s", err.Error())
					}
					state = TRANSACTION
				} else {
					sendResponse(conn, "-ERR invalid password")
					logger.Log().Warnf("[pop3] failed login: %s", user)
				}
			} else {
				sendResponse(conn, "-ERR user not specified")
			}
		case "STAT", "LIST", "UIDL", "RETR", "TOP", "NOOP", "DELE", "RSET":
			if state == TRANSACTION {
				handleTransactionCommand(conn, cmd, args, messages, &toDelete)
			} else {
				sendResponse(conn, "-ERR user not authenticated")
			}
		case "QUIT":
			sendResponse(conn, "+OK goodbye")
			state = UPDATE
			return
		default:
			sendResponse(conn, "-ERR unknown command")
		}
	}
}

func handleTransactionCommand(conn net.Conn, cmd string, args []string, messages []message, toDelete *[]string) {
	switch cmd {
	case "STAT":
		totalSize := float64(0)
		for _, m := range messages {
			totalSize += m.Size
		}
		sendResponse(conn, fmt.Sprintf("+OK %d %d", len(messages), int64(totalSize)))
	case "LIST":
		totalSize := float64(0)
		for _, m := range messages {
			totalSize += m.Size
		}
		sendResponse(conn, fmt.Sprintf("+OK %d messages (%d octets)", len(messages), int64(totalSize)))

		for row, m := range messages {
			sendResponse(conn, fmt.Sprintf("%d %d", row+1, int64(m.Size))) // Convert Size to int64 when printing
		}
		sendResponse(conn, ".")
	case "UIDL":
		sendResponse(conn, "+OK unique-id listing follows")
		for row, m := range messages {
			sendResponse(conn, fmt.Sprintf("%d %s", row+1, m.ID))
		}
		sendResponse(conn, ".")
	case "RETR":
		if len(args) != 1 {
			sendResponse(conn, "-ERR no such message")
			return
		}

		nr, err := strconv.Atoi(args[0])
		if err != nil || nr < 1 || nr > len(messages) {
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
		sendResponse(conn, fmt.Sprintf("+OK %d octets", size))

		// When all lines of the response have been sent, a
		// final line is sent, consisting of a termination octet (decimal code
		// 046, ".") and a CRLF pair. If any line of the multi-line response
		// begins with the termination octet, the line is "byte-stuffed" by
		// pre-pending the termination octet to that line of the response.
		// @see: https://www.ietf.org/rfc/rfc1939.txt
		sendData(conn, strings.Replace(string(raw), "\n.", "\n..", -1))
		sendResponse(conn, ".")
	case "TOP":
		arg, err := getSafeArg(args, 0)
		if err != nil {
			sendResponse(conn, "-ERR TOP requires two arguments")
			return
		}
		nr, err := strconv.Atoi(arg)
		if err != nil || nr < 1 || nr > len(messages) {
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

		sendResponse(conn, "+OK top of message follows")
		sendData(conn, headers+"\r\n")
		sendData(conn, body)
		sendResponse(conn, ".")
	case "NOOP":
		sendResponse(conn, "+OK")
	case "DELE":
		arg, _ := getSafeArg(args, 0)
		nr, err := strconv.Atoi(arg)
		if err != nil || nr < 1 || nr > len(messages) {
			sendResponse(conn, "-ERR no such message")
			return
		}

		m := messages[nr-1]
		*toDelete = append(*toDelete, m.ID)
		sendResponse(conn, "+OK message marked for deletion")
	case "RSET":
		*toDelete = []string{}
		sendResponse(conn, "+OK")
	default:
		sendResponse(conn, "-ERR unknown command")
	}
}
