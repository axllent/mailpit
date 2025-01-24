// Package smtpd implements a basic SMTP server.
//
// This is a modified version of https://github.com/mhale/smtpd to
// add support for unix sockets and Mailpit Chaos.
package smtpd

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"net"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/axllent/mailpit/internal/smtpd/chaos"
)

var (
	// Debug `true` enables verbose logging.
	Debug      = false
	rcptToRE   = regexp.MustCompile(`(?i)TO: ?<([^<>\v]+)>( |$)(.*)?`)
	mailFromRE = regexp.MustCompile(`(?i)FROM: ?<(|[^<>\v]+)>( |$)(.*)?`) // Delivery Status Notifications are sent with "MAIL FROM:<>"

	// extract mail size from 'MAIL FROM' parameter
	mailFromSizeRE = regexp.MustCompile(`(?U)(^| |,)[Ss][Ii][Zz][Ee]=(.*)($|,| )`)
)

// Handler function called upon successful receipt of an email.
// Results in a "250 2.0.0 Ok: queued" response.
type Handler func(remoteAddr net.Addr, from string, to []string, data []byte) error

// MsgIDHandler function called upon successful receipt of an email. Returns a message ID.
// Results in a "250 2.0.0 Ok: queued as <message-id>" response.
type MsgIDHandler func(remoteAddr net.Addr, from string, to []string, data []byte) (string, error)

// HandlerRcpt function called on RCPT. Return accept status.
type HandlerRcpt func(remoteAddr net.Addr, from string, to string) bool

// AuthHandler function called when a login attempt is performed. Returns true if credentials are correct.
type AuthHandler func(remoteAddr net.Addr, mechanism string, username []byte, password []byte, shared []byte) (bool, error)

// ErrServerClosed is the default message when a server closes a connection
var ErrServerClosed = errors.New("Server has been closed")

// ListenAndServe listens on the TCP network address addr
// and then calls Serve with handler to handle requests
// on incoming connections.
func ListenAndServe(addr string, handler Handler, appName string, hostname string) error {
	srv := &Server{Addr: addr, Handler: handler, AppName: appName, Hostname: hostname}
	return srv.ListenAndServe()
}

// ListenAndServeTLS listens on the TCP network address addr
// and then calls Serve with handler to handle requests
// on incoming connections. Connections may be upgraded to TLS if the client requests it.
func ListenAndServeTLS(addr string, certFile string, keyFile string, handler Handler, appName string, hostname string) error {
	srv := &Server{Addr: addr, Handler: handler, AppName: appName, Hostname: hostname}
	err := srv.ConfigureTLS(certFile, keyFile)
	if err != nil {
		return err
	}
	return srv.ListenAndServe()
}

type maxSizeExceededError struct {
	limit int
}

func maxSizeExceeded(limit int) maxSizeExceededError {
	return maxSizeExceededError{limit}
}

// Error uses the RFC 5321 response message in preference to RFC 1870.
// RFC 3463 defines enhanced status code x.3.4 as "Message too big for system".
func (err maxSizeExceededError) Error() string {
	return fmt.Sprintf("552 5.3.4 Requested mail action aborted: exceeded storage allocation (%d)", err.limit)
}

// LogFunc is a function capable of logging the client-server communication.
type LogFunc func(remoteIP, verb, line string)

// Server is an SMTP server.
type Server struct {
	Addr              string // TCP address to listen on, defaults to ":25" (all addresses, port 25) if empty
	AppName           string
	AuthHandler       AuthHandler
	AuthMechs         map[string]bool // Override list of allowed authentication mechanisms. Currently supported: LOGIN, PLAIN, CRAM-MD5. Enabling LOGIN and PLAIN will reduce RFC 4954 compliance.
	AuthRequired      bool            // Require authentication for every command except AUTH, EHLO, HELO, NOOP, RSET or QUIT as per RFC 4954. Ignored if AuthHandler is not configured.
	DisableReverseDNS bool            // Disable reverse DNS lookups, enforces "unknown" hostname
	Handler           Handler
	HandlerRcpt       HandlerRcpt
	Hostname          string
	LogRead           LogFunc
	LogWrite          LogFunc
	MaxSize           int // Maximum message size allowed, in bytes
	MaxRecipients     int // Maximum number of recipients, defaults to 100.
	MsgIDHandler      MsgIDHandler
	Timeout           time.Duration
	TLSConfig         *tls.Config
	TLSListener       bool        // Listen for incoming TLS connections only (not recommended as it may reduce compatibility). Ignored if TLS is not configured.
	TLSRequired       bool        // Require TLS for every command except NOOP, EHLO, STARTTLS, or QUIT as per RFC 3207. Ignored if TLS is not configured.
	Protocol          string      // Default tcp, supports unix
	SocketPerm        fs.FileMode // if using Unix socket, socket permissions

	inShutdown   int32 // server was closed or shutdown
	openSessions int32 // count of open sessions
	mu           sync.Mutex
	shutdownChan chan struct{} // let the sessions know we are shutting down

	XClientAllowed []string // List of XCLIENT allowed IP addresses
}

// ConfigureTLS creates a TLS configuration from certificate and key files.
func (srv *Server) ConfigureTLS(certFile string, keyFile string) error {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return err
	}
	srv.TLSConfig = &tls.Config{Certificates: []tls.Certificate{cert}} // #nosec
	return nil
}

// // ConfigureTLSWithPassphrase creates a TLS configuration from a certificate,
// // an encrypted key file and the associated passphrase:
// func (srv *Server) ConfigureTLSWithPassphrase(
// 	certFile string,
// 	keyFile string,
// 	passphrase string,
// ) error {
// 	certPEMBlock, err := os.ReadFile(certFile)
// 	if err != nil {
// 		return err
// 	}
// 	keyPEMBlock, err := os.ReadFile(keyFile)
// 	if err != nil {
// 		return err
// 	}
// 	keyDERBlock, _ := pem.Decode(keyPEMBlock)
// 	keyPEMDecrypted, err := x509.DecryptPEMBlock(keyDERBlock, []byte(passphrase))
// 	if err != nil {
// 		return err
// 	}
// 	var pemBlock pem.Block
// 	pemBlock.Type = keyDERBlock.Type
// 	pemBlock.Bytes = keyPEMDecrypted
// 	keyPEMBlock = pem.EncodeToMemory(&pemBlock)
// 	cert, err := tls.X509KeyPair(certPEMBlock, keyPEMBlock)
// 	if err != nil {
// 		return err
// 	}
// 	srv.TLSConfig = &tls.Config{Certificates: []tls.Certificate{cert}}
// 	return nil
// }

// ListenAndServe listens on the either a TCP network address srv.Addr or
// alternatively a Unix socket. and then calls Serve to handle requests on
// incoming connections. If srv.Addr is blank, ":25" is used.
func (srv *Server) ListenAndServe() error {
	if atomic.LoadInt32(&srv.inShutdown) != 0 {
		return ErrServerClosed
	}

	if srv.Addr == "" {
		srv.Addr = ":25"
	}
	if srv.AppName == "" {
		srv.AppName = "smtpd"
	}
	if srv.Hostname == "" {
		srv.Hostname, _ = os.Hostname()
	}
	if srv.Timeout == 0 {
		srv.Timeout = 5 * time.Minute
	}
	if srv.Protocol == "" {
		srv.Protocol = "tcp"
	}

	var ln net.Listener
	var err error

	// If TLSListener is enabled, listen for TLS connections only.
	if srv.TLSConfig != nil && srv.TLSListener {
		ln, err = tls.Listen(srv.Protocol, srv.Addr, srv.TLSConfig)
	} else {
		ln, err = net.Listen(srv.Protocol, srv.Addr)
	}

	if err != nil {
		return err
	}

	if srv.Protocol == "unix" {
		// set permissions
		if err := os.Chmod(srv.Addr, srv.SocketPerm); err != nil {
			return err
		}
	}

	return srv.Serve(ln)
}

// Serve creates a new SMTP session after a network connection is established.
func (srv *Server) Serve(ln net.Listener) error {
	if atomic.LoadInt32(&srv.inShutdown) != 0 {
		return ErrServerClosed
	}

	defer ln.Close()

	for {
		// if we are shutting down, don't accept new connections
		select {
		case <-srv.getShutdownChan():
			return ErrServerClosed
		default:
		}

		conn, err := ln.Accept()
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Temporary() {
				continue
			}
			return err
		}

		session := srv.newSession(conn)
		atomic.AddInt32(&srv.openSessions, 1)
		go session.serve()
	}
}

type session struct {
	srv           *Server
	conn          net.Conn
	br            *bufio.Reader
	bw            *bufio.Writer
	remoteIP      string // Remote IP address
	remoteHost    string // Remote hostname according to reverse DNS lookup
	remoteName    string // Remote hostname as supplied with EHLO
	xClient       string // Information string as supplied with XCLIENT
	xClientADDR   string // Information string as supplied with XCLIENT ADDR
	xClientNAME   string // Information string as supplied with XCLIENT NAME
	xClientTrust  bool   // Trust XCLIENT from current IP address
	tls           bool
	authenticated bool
}

// Create new session from connection.
func (srv *Server) newSession(conn net.Conn) (s *session) {
	s = &session{
		srv:  srv,
		conn: conn,
		br:   bufio.NewReader(conn),
		bw:   bufio.NewWriter(conn),
	}

	// Get remote end info for the Received header.
	s.remoteIP, _, _ = net.SplitHostPort(s.conn.RemoteAddr().String())
	if s.remoteIP == "" {
		s.remoteIP = "127.0.0.1"
	}
	if !s.srv.DisableReverseDNS {
		names, err := net.LookupAddr(s.remoteIP)
		if err == nil && len(names) > 0 {
			s.remoteHost = names[0]
		} else {
			s.remoteHost = "unknown"
		}
	} else {
		s.remoteHost = "unknown"
	}

	// Set tls = true if TLS is already in use.
	_, s.tls = s.conn.(*tls.Conn)

	for _, checkIP := range srv.XClientAllowed {
		if s.remoteIP == checkIP {
			s.xClientTrust = true
		}
	}
	return
}

func (srv *Server) getShutdownChan() <-chan struct{} {
	srv.mu.Lock()
	defer srv.mu.Unlock()
	if srv.shutdownChan == nil {
		srv.shutdownChan = make(chan struct{})
	}

	return srv.shutdownChan
}

func (srv *Server) closeShutdownChan() {
	srv.mu.Lock()
	defer srv.mu.Unlock()
	if srv.shutdownChan == nil {
		srv.shutdownChan = make(chan struct{})
	}

	select {
	case <-srv.shutdownChan:
	default:
		close(srv.shutdownChan)
	}
}

// Close - closes the connection without waiting
func (srv *Server) Close() error {
	atomic.StoreInt32(&srv.inShutdown, 1)
	srv.closeShutdownChan()
	return nil
}

// Shutdown - waits for current sessions to complete before closing
func (srv *Server) Shutdown(ctx context.Context) error {
	atomic.StoreInt32(&srv.inShutdown, 1)
	srv.closeShutdownChan()

	// wait for up to 30 seconds to allow the current sessions to
	// end
	timer := time.NewTimer(100 * time.Millisecond)
	defer timer.Stop()

	for i := 0; i < 300; i++ {
		// wait for open sessions to close
		if atomic.LoadInt32(&srv.openSessions) == 0 {
			break
		}

		select {
		case <-timer.C:
			timer.Reset(100 * time.Millisecond)
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
	}

	return nil
}

// Function called to handle connection requests.
func (s *session) serve() {
	defer atomic.AddInt32(&s.srv.openSessions, -1)
	defer s.conn.Close()

	var from string
	var gotFrom bool
	var to []string
	var buffer bytes.Buffer

	// Send banner.
	s.writef("220 %s %s ESMTP Service ready", s.srv.Hostname, s.srv.AppName)

loop:
	for {
		// Attempt to read a line from the socket.
		// On timeout, send a timeout message and return from serve().
		// On error, assume the client has gone away i.e. return from serve().
		line, err := s.readLine()
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				s.writef("421 4.4.2 %s %s ESMTP Service closing transmission channel after timeout exceeded", s.srv.Hostname, s.srv.AppName)
			}
			break
		}

		verb, args := s.parseLine(line)

		switch verb {
		case "HELO":
			s.remoteName = args
			s.writef("250 %s greets %s", s.srv.Hostname, s.remoteName)

			// RFC 2821 section 4.1.4 specifies that EHLO has the same effect as RSET, so reset for HELO too.
			from = ""
			gotFrom = false
			to = nil
			buffer.Reset()
		case "EHLO":
			s.remoteName = args
			s.writef("%s", s.makeEHLOResponse())

			// RFC 2821 section 4.1.4 specifies that EHLO has the same effect as RSET.
			from = ""
			gotFrom = false
			to = nil
			buffer.Reset()
		case "MAIL":
			if s.srv.TLSConfig != nil && s.srv.TLSRequired && !s.tls {
				s.writef("530 5.7.0 Must issue a STARTTLS command first")
				break
			}
			if s.srv.AuthHandler != nil && s.srv.AuthRequired && !s.authenticated {
				s.writef("530 5.7.0 Authentication required")
				break
			}

			match := mailFromRE.FindStringSubmatch(args)
			if match == nil {
				s.writef("501 5.5.4 Syntax error in parameters or arguments (invalid FROM parameter)")
			} else {
				// Mailpit Chaos
				if fail, code := chaos.Config.Sender.Trigger(); fail {
					s.writef("%d Chaos sender error", code)
					break
				}

				// Validate the SIZE parameter if one was sent.
				if len(match[2]) > 0 { // A parameter is present
					sizeMatch := mailFromSizeRE.FindStringSubmatch(match[3])
					if sizeMatch == nil {
						// ignore other parameter
						from = match[1]
						gotFrom = true
						s.writef("250 2.1.0 Ok")
					} else {
						// Enforce the maximum message size if one is set.
						size, err := strconv.Atoi(sizeMatch[2])
						if err != nil { // Bad SIZE parameter
							s.writef("501 5.5.4 Syntax error in parameters or arguments (invalid SIZE parameter)")
						} else if s.srv.MaxSize > 0 && size > s.srv.MaxSize { // SIZE above maximum size, if set
							err = maxSizeExceeded(s.srv.MaxSize)
							s.writef("%s", err.Error())
						} else { // SIZE ok
							from = match[1]
							gotFrom = true
							s.writef("250 2.1.0 Ok")
						}
					}
				} else { // No parameters after FROM
					from = match[1]
					gotFrom = true
					s.writef("250 2.1.0 Ok")
				}
			}

			to = nil
			buffer.Reset()
		case "RCPT":
			if s.srv.TLSConfig != nil && s.srv.TLSRequired && !s.tls {
				s.writef("530 5.7.0 Must issue a STARTTLS command first")
				break
			}
			if s.srv.AuthHandler != nil && s.srv.AuthRequired && !s.authenticated {
				s.writef("530 5.7.0 Authentication required")
				break
			}
			if !gotFrom {
				s.writef("503 5.5.1 Bad sequence of commands (MAIL required before RCPT)")
				break
			}

			match := rcptToRE.FindStringSubmatch(args)
			if match == nil {
				s.writef("501 5.5.4 Syntax error in parameters or arguments (invalid TO parameter)")
			} else {
				// Mailpit Chaos
				if fail, code := chaos.Config.Recipient.Trigger(); fail {
					s.writef("%d Chaos recipient error", code)
					break
				}

				// RFC 5321 specifies support for minimum of 100 recipients is required.
				if s.srv.MaxRecipients == 0 {
					s.srv.MaxRecipients = 100
				}

				if len(to) == s.srv.MaxRecipients {
					s.writef("452 4.5.3 Too many recipients")
				} else {
					accept := true
					if s.srv.HandlerRcpt != nil {
						accept = s.srv.HandlerRcpt(s.conn.RemoteAddr(), from, match[1])
					}
					if accept {
						to = append(to, match[1])
						s.writef("250 2.1.5 Ok")
					} else {
						s.writef("550 5.1.0 Requested action not taken: mailbox unavailable")
					}
				}
			}
		case "DATA":
			if s.srv.TLSConfig != nil && s.srv.TLSRequired && !s.tls {
				s.writef("530 5.7.0 Must issue a STARTTLS command first")
				break
			}
			if s.srv.AuthHandler != nil && s.srv.AuthRequired && !s.authenticated {
				s.writef("530 5.7.0 Authentication required")
				break
			}
			if !gotFrom || len(to) == 0 {
				s.writef("503 5.5.1 Bad sequence of commands (MAIL & RCPT required before DATA)")
				break
			}

			s.writef("354 Start mail input; end with <CR><LF>.<CR><LF>")

			// Attempt to read message body from the socket.
			// On timeout, send a timeout message and return from serve().
			// On net.Error, assume the client has gone away i.e. return from serve().
			// On other errors, allow the client to try again.
			data, err := s.readData()
			if err != nil {
				switch err.(type) {
				case net.Error:
					if err.(net.Error).Timeout() {
						s.writef("421 4.4.2 %s %s ESMTP Service closing transmission channel after timeout exceeded", s.srv.Hostname, s.srv.AppName)
					}
					break loop
				case maxSizeExceededError:
					s.writef("%s", err.Error())
					continue
				default:
					s.writef("451 4.3.0 Requested action aborted: local error in processing")
					continue
				}
			}

			// Create Received header & write message body into buffer.
			buffer.Reset()
			buffer.Write(s.makeHeaders(to))
			buffer.Write(data)

			// Pass mail on to handler.
			if s.srv.Handler != nil {
				err := s.srv.Handler(s.conn.RemoteAddr(), from, to, buffer.Bytes())
				if err != nil {
					checkErrFormat := regexp.MustCompile(`^([2-5][0-9]{2})[\s\-](.+)$`)
					if checkErrFormat.MatchString(err.Error()) {
						s.writef("%s", err.Error())
					} else {
						s.writef("451 4.3.5 Unable to process mail")
					}
					break
				}
				s.writef("250 2.0.0 Ok: queued")
			} else if s.srv.MsgIDHandler != nil {
				msgID, err := s.srv.MsgIDHandler(s.conn.RemoteAddr(), from, to, buffer.Bytes())
				if err != nil {
					checkErrFormat := regexp.MustCompile(`^([2-5][0-9]{2})[\s\-](.+)$`)
					if checkErrFormat.MatchString(err.Error()) {
						s.writef("%s", err.Error())
					} else {
						s.writef("451 4.3.5 Unable to process mail")
					}
					break
				}

				if msgID != "" {
					s.writef("250 2.0.0 Ok: queued as %s", msgID)
				} else {
					s.writef("250 2.0.0 Ok: queued")
				}
			} else {
				s.writef("250 2.0.0 Ok: queued")
			}

			// Reset for next mail.
			from = ""
			gotFrom = false
			to = nil
			buffer.Reset()
		case "QUIT":
			s.writef("221 2.0.0 %s %s ESMTP Service closing transmission channel", s.srv.Hostname, s.srv.AppName)
			break loop
		case "RSET":
			if s.srv.TLSConfig != nil && s.srv.TLSRequired && !s.tls {
				s.writef("530 5.7.0 Must issue a STARTTLS command first")
				break
			}
			s.writef("250 2.0.0 Ok")
			from = ""
			gotFrom = false
			to = nil
			buffer.Reset()
		case "NOOP":
			s.writef("250 2.0.0 Ok")
		case "XCLIENT":
			s.xClient = args
			if s.xClientTrust {
				xCArgs := strings.Split(args, " ")
				for _, xCArg := range xCArgs {
					xCParse := strings.Split(strings.TrimSpace(xCArg), "=")
					if strings.ToUpper(xCParse[0]) == "ADDR" && (net.ParseIP(xCParse[1]) != nil) {
						s.xClientADDR = xCParse[1]
					}
					if strings.ToUpper(xCParse[0]) == "NAME" && len(xCParse[1]) > 0 {
						if xCParse[1] != "[UNAVAILABLE]" {
							s.xClientNAME = xCParse[1]
						}
					}
				}
				if len(s.xClientADDR) > 7 {
					s.remoteIP = s.xClientADDR
					if len(s.xClientNAME) > 4 {
						s.remoteHost = s.xClientNAME
					} else {
						names, err := net.LookupAddr(s.remoteIP)
						if err == nil && len(names) > 0 {
							s.remoteHost = names[0]
						} else {
							s.remoteHost = "unknown"
						}
					}
				}
			}
			s.writef("250 2.0.0 Ok")
		case "HELP", "VRFY", "EXPN":
			// See RFC 5321 section 4.2.4 for usage of 500 & 502 response codes.
			s.writef("502 5.5.1 Command not implemented")
		case "STARTTLS":
			// Parameters are not allowed (RFC 3207 section 4).
			if args != "" {
				s.writef("501 5.5.2 Syntax error (no parameters allowed)")
				break
			}

			// Handle case where TLS is requested but not configured (and therefore not listed as a service extension).
			if s.srv.TLSConfig == nil {
				s.writef("502 5.5.1 Command not implemented")
				break
			}

			// Handle case where STARTTLS is received when TLS is already in use.
			if s.tls {
				s.writef("503 5.5.1 Bad sequence of commands (TLS already in use)")
				break
			}

			s.writef("220 2.0.0 Ready to start TLS")

			// Establish a TLS connection with the client.
			tlsConn := tls.Server(s.conn, s.srv.TLSConfig)
			err := tlsConn.Handshake()
			if err != nil {
				s.writef("403 4.7.0 TLS handshake failed")
				break
			}

			// TLS handshake succeeded, switch to using the TLS connection.
			s.conn = tlsConn
			s.br = bufio.NewReader(s.conn)
			s.bw = bufio.NewWriter(s.conn)
			s.tls = true

			// RFC 3207 specifies that the server must discard any prior knowledge obtained from the client.
			s.remoteName = ""
			from = ""
			gotFrom = false
			to = nil
			buffer.Reset()
		case "AUTH":
			if s.srv.TLSConfig != nil && s.srv.TLSRequired && !s.tls {
				s.writef("530 5.7.0 Must issue a STARTTLS command first")
				break
			}
			// Handle case where AUTH is requested but not configured (and therefore not listed as a service extension).
			if s.srv.AuthHandler == nil {
				s.writef("502 5.5.1 Command not implemented")
				break
			}

			// Handle case where AUTH is received when already authenticated.
			if s.authenticated {
				s.writef("503 5.5.1 Bad sequence of commands (already authenticated for this session)")
				break
			}

			// RFC 4954 specifies that AUTH is not permitted during mail transactions.
			if gotFrom || len(to) > 0 {
				s.writef("503 5.5.1 Bad sequence of commands (AUTH not permitted during mail transaction)")
				break
			}

			// RFC 4954 requires a mechanism parameter.
			authType, authArgs := s.parseLine(args)
			if authType == "" {
				s.writef("501 5.5.4 Malformed AUTH input (argument required)")
				break
			}

			// RFC 4954 requires rejecting unsupported authentication mechanisms with a 504 response.
			allowedAuth := s.authMechs()
			if allowed, found := allowedAuth[authType]; !found || !allowed {
				s.writef("504 5.5.4 Unrecognized authentication type")
				break
			}

			// Mailpit Chaos
			if fail, code := chaos.Config.Authentication.Trigger(); fail {
				s.writef("%d Chaos authentication error", code)
				break
			}

			// RFC 4954 also specifies that ESMTP code 5.5.4 ("Invalid command arguments") should be returned
			// when attempting to use an unsupported authentication type.
			// Many servers return 5.7.4 ("Security features not supported") instead.
			switch authType {
			case "PLAIN":
				s.authenticated, err = s.handleAuthPlain(authArgs)
			case "LOGIN":
				s.authenticated, err = s.handleAuthLogin(authArgs)
			case "CRAM-MD5":
				s.authenticated, err = s.handleAuthCramMD5()
			}

			if err != nil {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					s.writef("421 4.4.2 %s %s ESMTP Service closing transmission channel after timeout exceeded", s.srv.Hostname, s.srv.AppName)
					break loop
				}

				s.writef("%s", err.Error())
				break
			}

			if s.authenticated {
				s.writef("235 2.7.0 Authentication successful")
			} else {
				s.writef("535 5.7.8 Authentication credentials invalid")
			}
		default:
			// See RFC 5321 section 4.2.4 for usage of 500 & 502 response codes.
			s.writef("500 5.5.2 Syntax error, command unrecognized")
		}
	}
}

// Wrapper function for writing a complete line to the socket.
func (s *session) writef(format string, args ...interface{}) {
	if s.srv.Timeout > 0 {
		_ = s.conn.SetWriteDeadline(time.Now().Add(s.srv.Timeout))
	}

	line := fmt.Sprintf(format, args...)
	fmt.Fprintf(s.bw, "%s\r\n", line)
	_ = s.bw.Flush()

	if Debug {
		verb := "WROTE"
		if s.srv.LogWrite != nil {
			s.srv.LogWrite(s.remoteIP, verb, line)
		} else {
			log.Println(s.remoteIP, verb, line)
		}
	}
}

// Read a complete line from the socket.
func (s *session) readLine() (string, error) {
	if s.srv.Timeout > 0 {
		_ = s.conn.SetReadDeadline(time.Now().Add(s.srv.Timeout))
	}

	line, err := s.br.ReadString('\n')
	if err != nil {
		return "", err
	}
	line = strings.TrimSpace(line) // Strip trailing \r\n

	if Debug {
		verb := "READ"
		if s.srv.LogRead != nil {
			s.srv.LogRead(s.remoteIP, verb, line)
		} else {
			log.Println(s.remoteIP, verb, line)
		}
	}

	return line, err
}

// Parse a line read from the socket.
func (s *session) parseLine(line string) (verb string, args string) {
	if idx := strings.Index(line, " "); idx != -1 {
		verb = strings.ToUpper(line[:idx])
		args = strings.TrimSpace(line[idx+1:])
	} else {
		verb = strings.ToUpper(line)
		args = ""
	}
	return verb, args
}

// Read the message data following a DATA command.
func (s *session) readData() ([]byte, error) {
	var data []byte
	for {
		if s.srv.Timeout > 0 {
			_ = s.conn.SetReadDeadline(time.Now().Add(s.srv.Timeout))
		}

		line, err := s.br.ReadBytes('\n')
		if err != nil {
			return nil, err
		}
		// Handle end of data denoted by lone period (\r\n.\r\n)
		if bytes.Equal(line, []byte(".\r\n")) {
			break
		}
		// Remove leading period (RFC 5321 section 4.5.2)
		if line[0] == '.' {
			line = line[1:]
		}

		// Enforce the maximum message size limit.
		if s.srv.MaxSize > 0 {
			if len(data)+len(line) > s.srv.MaxSize {
				_, _ = s.br.Discard(s.br.Buffered()) // Discard the buffer remnants.
				return nil, maxSizeExceeded(s.srv.MaxSize)
			}
		}

		data = append(data, line...)
	}
	return data, nil
}

// Create the Received header to comply with RFC 2821 section 3.8.2.
// TODO: Work out what to do with multiple to addresses.
func (s *session) makeHeaders(to []string) []byte {
	var buffer bytes.Buffer
	now := time.Now().Format("Mon, 2 Jan 2006 15:04:05 -0700 (MST)")
	buffer.WriteString(fmt.Sprintf("Received: from %s (%s [%s])\r\n", s.remoteName, s.remoteHost, s.remoteIP))
	buffer.WriteString(fmt.Sprintf("        by %s (%s) with SMTP\r\n", s.srv.Hostname, s.srv.AppName))
	buffer.WriteString(fmt.Sprintf("        for <%s>; %s\r\n", to[0], now))
	return buffer.Bytes()
}

// Determine allowed authentication mechanisms.
// RFC 4954 specifies that plaintext authentication mechanisms such as LOGIN and PLAIN require a TLS connection.
// This can be explicitly overridden e.g. setting s.srv.AuthMechs["LOGIN"] = true.
func (s *session) authMechs() (mechs map[string]bool) {
	mechs = map[string]bool{"LOGIN": s.tls, "PLAIN": s.tls, "CRAM-MD5": true}

	for mech := range mechs {
		allowed, found := s.srv.AuthMechs[mech]
		if found {
			mechs[mech] = allowed
		}
	}

	return
}

// Create the greeting string sent in response to an EHLO command.
func (s *session) makeEHLOResponse() (response string) {
	response = fmt.Sprintf("250-%s greets %s\r\n", s.srv.Hostname, s.remoteName)

	// RFC 1870 specifies that "SIZE 0" indicates no maximum size is in force.
	response += fmt.Sprintf("250-SIZE %d\r\n", s.srv.MaxSize)

	// Only list STARTTLS if TLS is configured, but not currently in use.
	if s.srv.TLSConfig != nil && !s.tls {
		response += "250-STARTTLS\r\n"
	}

	// Only list AUTH if an AuthHandler is configured and at least one mechanism is allowed.
	if s.srv.AuthHandler != nil {
		var mechs []string
		for mech, allowed := range s.authMechs() {
			if allowed {
				mechs = append(mechs, mech)
			}
		}
		if len(mechs) > 0 {
			response += "250-AUTH " + strings.Join(mechs, " ") + "\r\n"
		}
	}

	response += "250 ENHANCEDSTATUSCODES"
	return
}

func (s *session) handleAuthLogin(arg string) (bool, error) {
	var err error

	if arg == "" {
		s.writef("334 %s", base64.StdEncoding.EncodeToString([]byte("Username:")))
		arg, err = s.readLine()
		if err != nil {
			return false, err
		}
	}

	username, err := base64.StdEncoding.DecodeString(arg)
	if err != nil {
		return false, errors.New("501 5.5.2 Syntax error (unable to decode)")
	}

	s.writef("334 %s", base64.StdEncoding.EncodeToString([]byte("Password:")))
	line, err := s.readLine()
	if err != nil {
		return false, err
	}

	password, err := base64.StdEncoding.DecodeString(line)
	if err != nil {
		return false, errors.New("501 5.5.2 Syntax error (unable to decode)")
	}

	// Validate credentials.
	authenticated, err := s.srv.AuthHandler(s.conn.RemoteAddr(), "LOGIN", username, password, nil)

	return authenticated, err
}

func (s *session) handleAuthPlain(arg string) (bool, error) {
	var err error

	// If fast mode (AUTH PLAIN [arg]) is not used, prompt for credentials.
	if arg == "" {
		s.writef("334 ")
		arg, err = s.readLine()
		if err != nil {
			return false, err
		}
	}

	data, err := base64.StdEncoding.DecodeString(arg)
	if err != nil {
		return false, errors.New("501 5.5.2 Syntax error (unable to decode)")
	}

	parts := bytes.Split(data, []byte{0})
	if len(parts) != 3 {
		return false, errors.New("501 5.5.2 Syntax error (unable to parse)")
	}

	// Validate credentials.
	authenticated, err := s.srv.AuthHandler(s.conn.RemoteAddr(), "PLAIN", parts[1], parts[2], nil)

	return authenticated, err
}

func (s *session) handleAuthCramMD5() (bool, error) {
	shared := "<" + strconv.Itoa(os.Getpid()) + "." + strconv.Itoa(time.Now().Nanosecond()) + "@" + s.srv.Hostname + ">"

	s.writef("334 %s", base64.StdEncoding.EncodeToString([]byte(shared)))

	data, err := s.readLine()
	if err != nil {
		return false, err
	}

	if data == "*" {
		return false, errors.New("501 5.7.0 Authentication cancelled")
	}

	buf, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return false, errors.New("501 5.5.2 Syntax error (unable to decode)")
	}

	fields := strings.Split(string(buf), " ")
	if len(fields) < 2 {
		return false, errors.New("501 5.5.2 Syntax error (unable to parse)")
	}

	// Validate credentials.
	authenticated, err := s.srv.AuthHandler(s.conn.RemoteAddr(), "CRAM-MD5", []byte(fields[0]), []byte(fields[1]), []byte(shared))

	return authenticated, err
}
