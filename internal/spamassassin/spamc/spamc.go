// Package spamc provides a client for the SpamAssassin spamd protocol.
// http://svn.apache.org/repos/asf/spamassassin/trunk/spamd/PROTOCOL
//
// Modified to add timeouts from https://github.com/cgt/spamc
package spamc

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/axllent/mailpit/internal/tools"
)

// ProtoVersion is the protocol version
const ProtoVersion = "1.5"

// maxResponseBytes is the most that is read from spamd. A report is a few kilobytes, so this is
// generous, but it stops a misbehaving server from filling memory. It is a variable for the tests.
var maxResponseBytes int64 = 2 << 20

var (
	spamInfoRe    = regexp.MustCompile(`(.+)\/(.+) (\d+) (.+)`)
	spamMainRe    = regexp.MustCompile(`^Spam: (.+) ; (.+) . (.+)$`)
	spamDetailsRe = regexp.MustCompile(`^\s?(-?[0-9\.]+)\s([a-zA-Z0-9_]*)(\W*)(.*)`)
	// descriptions wrapped over several lines are indented by this much
	continuationRe = regexp.MustCompile("\n                            ")
)

// connection is like net.Conn except that it also has a CloseWrite method.
// CloseWrite is implemented by net.TCPConn and net.UnixConn, but for some
// reason it is not present in the net.Conn interface.
type connection interface {
	net.Conn
	CloseWrite() error
}

// Client is a spamd client.
type Client struct {
	net     string
	addr    string
	timeout int
}

// NewTCP returns a *Client that connects to spamd via the given TCP address.
// The timeout (in seconds) covers connecting as well as the request.
func NewTCP(addr string, timeout int) *Client {
	return &Client{"tcp", addr, timeout}
}

// NewUnix returns a *Client that connects to spamd via the given Unix socket.
// The timeout (in seconds) covers connecting as well as the request.
func NewUnix(addr string, timeout int) *Client {
	return &Client{"unix", addr, timeout}
}

// Rule represents a matched SpamAssassin rule.
type Rule struct {
	Points      string
	Name        string
	Description string
}

// Result struct
type Result struct {
	ResponseCode int
	Message      string
	Spam         bool
	Score        float64
	Threshold    float64
	Rules        []Rule

	// headerSeen and summarySeen record that the response had the parts of a valid reply
	headerSeen  bool
	summarySeen bool
}

// ParseScore parses a score from spamd, which must be a finite number.
func ParseScore(s string) (float64, error) {
	f, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil {
		return 0, err
	}
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return 0, fmt.Errorf("invalid score %q", s)
	}

	return f, nil
}

// timeoutDuration returns the timeout, which is never zero as that would expire at once.
func (c *Client) timeoutDuration() time.Duration {
	if c.timeout <= 0 {
		return 8 * time.Second
	}

	return time.Duration(c.timeout) * time.Second
}

// dial connects to spamd through TCP or a Unix socket. Connecting, and everything that follows on
// the connection, must finish within the timeout.
func (c *Client) dial() (connection, error) {
	if c.net != "tcp" && c.net != "unix" {
		return nil, fmt.Errorf("unsupported network type: %s", c.net)
	}

	deadline := time.Now().Add(c.timeoutDuration())
	dialer := net.Dialer{Deadline: deadline}

	conn, err := dialer.Dial(c.net, c.addr)
	if err != nil {
		return nil, err
	}

	cc, ok := conn.(connection)
	if !ok {
		_ = conn.Close()
		return nil, fmt.Errorf("unsupported connection type %T", conn)
	}

	if err := cc.SetDeadline(deadline); err != nil {
		_ = cc.Close()
		return nil, err
	}

	return cc, nil
}

// Report checks if message is spam or not, and returns score plus report
func (c *Client) Report(email []byte) (Result, error) {
	output, err := c.report(email)
	if err != nil {
		return Result{}, err
	}

	result, err := c.parseOutput(output)
	if err != nil {
		return Result{}, err
	}

	// a reply that is not a successful report must not look like a message with a score of 0
	if !result.headerSeen {
		return Result{}, errors.New("spamd: unexpected response")
	}
	if result.ResponseCode != 0 {
		return Result{}, fmt.Errorf("spamd: %d %s", result.ResponseCode, result.Message)
	}
	if !result.summarySeen {
		return Result{}, errors.New("spamd: response has no spam summary")
	}

	return result, nil
}

func (c *Client) report(email []byte) ([]string, error) {
	conn, err := c.dial()
	if err != nil {
		return nil, err
	}

	defer func() { _ = conn.Close() }()

	bw := bufio.NewWriter(conn)
	if _, err := bw.WriteString("REPORT SPAMC/" + ProtoVersion + "\r\n"); err != nil {
		return nil, err
	}

	if _, err := bw.WriteString("Content-length: " + strconv.Itoa(len(email)) + "\r\n\r\n"); err != nil {
		return nil, err
	}

	if _, err := bw.Write(email); err != nil {
		return nil, err
	}

	if err := bw.Flush(); err != nil {
		return nil, err
	}

	// Client is supposed to close its writing side of the connection
	// after sending its request.
	if err := conn.CloseWrite(); err != nil {
		return nil, err
	}

	// read one byte more than is allowed, to tell a reply of exactly the limit from a longer one
	data, err := io.ReadAll(io.LimitReader(conn, maxResponseBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maxResponseBytes {
		return nil, fmt.Errorf("spamd: response is larger than %d bytes", maxResponseBytes)
	}

	var lines []string
	for line := range strings.SplitSeq(string(data), "\n") {
		lines = append(lines, strings.TrimRight(line, " \t\r\n"))
	}

	// join lines, and replace multi-line descriptions with single line for each
	n := continuationRe.ReplaceAllString(strings.Join(lines, "\n"), " ")

	//split lines again
	return strings.Split(n, "\n"), nil
}

func (c *Client) parseOutput(output []string) (Result, error) {
	var result Result
	var reachedRules bool

	for i, row := range output {
		// header, which is always the first line
		if i == 0 {
			if res := spamInfoRe.FindStringSubmatch(row); len(res) == 5 {
				resCode, err := strconv.Atoi(res[3])
				if err != nil {
					return Result{}, fmt.Errorf("spamd: invalid response code %q", res[3])
				}
				result.headerSeen = true
				result.ResponseCode = resCode
				result.Message = res[4]
				continue
			}
		}

		// summary
		if res := spamMainRe.FindStringSubmatch(row); len(res) == 4 {
			result.Spam = tools.InArray(res[1], []string{"true", "yes"})

			score, err := ParseScore(res[2])
			if err != nil {
				return Result{}, fmt.Errorf("spamd: %w", err)
			}
			result.Score = score
			result.summarySeen = true

			// the threshold is informational, so one that cannot be read is ignored
			if threshold, err := ParseScore(res[3]); err == nil {
				result.Threshold = threshold
			}

			continue
		}

		if strings.HasPrefix(row, "Content analysis details") {
			reachedRules = true
			continue
		}

		// details
		if reachedRules {
			if res := spamDetailsRe.FindStringSubmatch(row); len(res) == 5 {
				rule := Rule{Points: res[1], Name: res[2], Description: res[4]}
				result.Rules = append(result.Rules, rule)
			}
		}
	}

	return result, nil
}

// Ping the spamd
func (c *Client) Ping() error {
	conn, err := c.dial()
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close() }()

	if _, err := io.WriteString(conn, fmt.Sprintf("PING SPAMC/%s\r\n\r\n", ProtoVersion)); err != nil {
		return err
	}

	if err := conn.CloseWrite(); err != nil {
		return err
	}

	// the reply is "SPAMD/<version> 0 PONG", but whatever follows is read (within a limit) and ignored
	data, err := io.ReadAll(io.LimitReader(conn, maxResponseBytes))
	if err != nil {
		return err
	}

	first, _, _ := strings.Cut(string(data), "\n")
	if !strings.Contains(first, "PONG") {
		return errors.New("spamd: no PONG in the response to PING")
	}

	return nil
}
