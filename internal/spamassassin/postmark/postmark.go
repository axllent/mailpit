// Package postmark uses the free https://spamcheck.postmarkapp.com/
// See https://spamcheck.postmarkapp.com/doc/ for more details.
package postmark

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

// endpoint is the Postmark API. It is a variable so that the tests can use their own server.
var endpoint = "https://spamcheck.postmarkapp.com/filter"

// maxResponseBytes is the most that is read from Postmark. A response is a few kilobytes. It is a
// variable for the tests.
var maxResponseBytes int64 = 4 << 20

// maxDescriptionLength is the longest rule description that is looked up in the report. Real
// ones are one line.
const maxDescriptionLength = 1024

var (
	lineBreakRe    = regexp.MustCompile("\r?\n")
	continuationRe = regexp.MustCompile("\n                                ")
)

// Response struct
type Response struct {
	Success bool   `json:"success"`
	Message string `json:"message"` // for errors only
	Score   string `json:"score"`
	Rules   []Rule `json:"rules"`
	Report  string `json:"report"` // used to find the names of the rules
}

// Rule struct
type Rule struct {
	Score string `json:"score"`
	// Name not returned by postmark but rather extracted from description
	Name        string `json:"name"`
	Description string `json:"description"`
}

// Check will post the email data to Postmark
func Check(email []byte, timeout int) (Response, error) {
	r := Response{}

	client := http.Client{
		Timeout: time.Duration(timeout) * time.Second,
		// the message must only go to Postmark, so a redirect is not followed
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	req, err := newRequest(email)
	if err != nil {
		return r, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return r, err
	}

	defer func() { _ = resp.Body.Close() }()

	body := io.LimitReader(resp.Body, maxResponseBytes+1)
	counter := &countingReader{r: body}

	decodeErr := json.NewDecoder(counter).Decode(&r)

	// checked first, as a response cut off at the limit also fails to decode
	if counter.n > maxResponseBytes {
		return Response{}, fmt.Errorf("postmark response is larger than %d bytes", maxResponseBytes)
	}

	if decodeErr != nil {
		if resp.StatusCode != http.StatusOK {
			return r, fmt.Errorf("postmark responded with status %d", resp.StatusCode)
		}
		return r, decodeErr
	}

	if resp.StatusCode != http.StatusOK && r.Message == "" {
		return r, fmt.Errorf("postmark responded with status %d", resp.StatusCode)
	}

	// remove trailing line spaces for all lines in report
	lines := lineBreakRe.Split(r.Report, -1)
	reportLines := []string{}
	for _, l := range lines {
		line := strings.TrimRight(l, " ")
		reportLines = append(reportLines, line)
	}
	reportRaw := strings.Join(reportLines, "\n")

	// join description lines to make a single line per rule
	report := continuationRe.ReplaceAllString(reportRaw, "")
	for i, rule := range r.Rules {
		// populate rule name
		r.Rules[i].Name = nameFromReport(rule.Score, rule.Description, report)
	}

	return r, nil
}

// The request is {"email":"<the raw message>","options":"long"}. "options" must be "long" for a full
// report of the rules, or "short" for the score only. It is not made with json.Marshal, as that
// needs the message as a string (a copy), builds the result in a buffer (which grows by doubling)
// and copies that (about 11 times the size of the message in all, 570 MB for one of 50 MB). The
// message is instead escaped a little at a time while it is sent.
const (
	requestPrefix = `{"email":"`
	requestSuffix = `","options":"long"}`

	// escapeChunk is how much of the message is escaped at a time
	escapeChunk = 4096
)

// newRequest returns the request for the message, which is sent without being copied.
func newRequest(email []byte) (*http.Request, error) {
	// The length is found first (without keeping anything), so it is sent with the request. It
	// is not sent in chunks, as that is not known to be accepted.
	length := int64(len(requestPrefix) + len(requestSuffix))
	buf := make([]byte, 0, 2*escapeChunk)
	for pos := 0; pos < len(email); {
		end := chunkEnd(email, pos)
		buf = appendEscaped(buf[:0], email[pos:end])
		length += int64(len(buf))
		pos = end
	}

	body := func() io.Reader {
		return io.MultiReader(strings.NewReader(requestPrefix), &escapedReader{src: email}, strings.NewReader(requestSuffix))
	}

	req, err := http.NewRequest(http.MethodPost, endpoint, io.NopCloser(body()))
	if err != nil {
		return nil, err
	}

	req.ContentLength = length
	req.GetBody = func() (io.ReadCloser, error) { return io.NopCloser(body()), nil }
	req.Header.Set("Content-Type", "application/json")

	return req, nil
}

// chunkEnd returns where the chunk of the message which starts at pos ends. It does not end in
// the middle of a character, as each part is escaped on its own.
func chunkEnd(src []byte, pos int) int {
	end := min(pos+escapeChunk, len(src))

	for i := 0; i < utf8.UTFMax-1 && end < len(src) && end > pos+1 && !utf8.RuneStart(src[end]); i++ {
		end--
	}

	return end
}

// appendEscaped appends the bytes as they are in a JSON string, without the quotes. Bytes which are
// not valid UTF-8 are replaced with U+FFFD, one for each, as encoding/json does.
func appendEscaped(dst, b []byte) []byte {
	const hex = "0123456789abcdef"

	for i := 0; i < len(b); {
		c := b[i]

		if c < utf8.RuneSelf {
			switch {
			case c == '"' || c == '\\':
				dst = append(dst, '\\', c)
			case c == '\n':
				dst = append(dst, '\\', 'n')
			case c == '\r':
				dst = append(dst, '\\', 'r')
			case c == '\t':
				dst = append(dst, '\\', 't')
			case c < 0x20:
				dst = append(dst, '\\', 'u', '0', '0', hex[c>>4], hex[c&0xF])
			default:
				dst = append(dst, c)
			}
			i++

			continue
		}

		r, size := utf8.DecodeRune(b[i:])
		switch {
		case r == utf8.RuneError && size == 1:
			dst = append(dst, `\ufffd`...)
		case r == '\u2028' || r == '\u2029':
			// valid in JSON, but not in all of what reads it
			dst = append(dst, '\\', 'u', '2', '0', '2', hex[r&0xF])
		default:
			dst = append(dst, b[i:i+size]...)
		}
		i += size
	}

	return dst
}

// escapedReader reads the message as it is in a JSON string, escaping a chunk at a time.
type escapedReader struct {
	src     []byte
	pos     int
	pending []byte
	buf     []byte
}

func (e *escapedReader) Read(p []byte) (int, error) {
	n := 0

	for n < len(p) {
		if len(e.pending) > 0 {
			c := copy(p[n:], e.pending)
			e.pending = e.pending[c:]
			n += c

			continue
		}

		if e.pos >= len(e.src) {
			if n == 0 {
				return 0, io.EOF
			}

			break
		}

		end := chunkEnd(e.src, e.pos)
		e.buf = appendEscaped(e.buf[:0], e.src[e.pos:end])
		e.pending = e.buf
		e.pos = end
	}

	return n, nil
}

// countingReader counts the bytes read.
type countingReader struct {
	r io.Reader
	n int64
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.n += int64(n)

	return n, err
}

// Extract the name of the test from the report as Postmark does not include this in the JSON reports
func nameFromReport(score, description, report string) string {
	// the values come from the response, so they are bounded and a pattern that does not compile
	// gives no name instead of stopping Mailpit
	if len(score) > maxDescriptionLength || len(description) > maxDescriptionLength {
		return ""
	}

	score = regexp.QuoteMeta(score)
	description = regexp.QuoteMeta(description)
	str := fmt.Sprintf("%s\\s+([A-Z0-9\\_]+)\\s+%s", score, description)
	re, err := regexp.Compile(str)
	if err != nil {
		return ""
	}

	matches := re.FindAllStringSubmatch(report, 1)
	if len(matches) > 0 && len(matches[0]) == 2 {
		return strings.TrimSpace(matches[0][1])
	}

	return ""
}
