// Package tools provides various methods for various things
package tools

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/mail"
	"net/textproto"
	"strings"

	"github.com/axllent/mailpit/internal/logger"
	"golang.org/x/text/encoding/htmlindex"
)

// AddressList parses an address header. It is as strict as mail.Header.AddressList,
// with the exception that it also decodes display names in charsets other than
// UTF-8, ISO-8859-1 and US-ASCII. The address itself is never decoded.
func AddressList(h mail.Header, key string) ([]*mail.Address, error) {
	hdr := h.Get(key)
	if hdr == "" {
		return nil, mail.ErrHeaderNotPresent
	}

	parser := mail.AddressParser{WordDecoder: &mime.WordDecoder{CharsetReader: charsetReader}}

	return parser.ParseList(hdr)
}

// charsetReader converts text in the named charset to UTF-8.
func charsetReader(charset string, input io.Reader) (io.Reader, error) {
	enc, err := htmlindex.Get(charset)
	if err != nil {
		return nil, err
	}

	return enc.NewDecoder().Reader(input), nil
}

// NormaliseHeaderLineBreaks returns msg with every line break in the header block (up to the first
// empty line) as <CR><LF>, and whether anything had to be changed. The line breaks which are replaced
// are <LF> on its own, <CR> on its own, and <CR><CR><LF> (or more <CR>). The body is never changed.
//
// A <CR> on its own is a line break to some mail software, but part of the text to others (Go's
// parser, for one). A message can therefore have different headers depending on which reads it, for
// example a "Bcc" header which is hidden in the middle of another header's value. Normalising the
// message first means that every reader sees the same headers. The message is not copied if there
// is nothing to change.
func NormaliseHeaderLineBreaks(msg []byte) ([]byte, bool) {
	var out []byte // only made when something has to change
	last := 0      // msg[:last] has been copied to out
	lineLen := 0   // the length of the line being read

	for i := 0; i < len(msg); {
		c := msg[i]
		if c != '\r' && c != '\n' {
			lineLen++
			i++

			continue
		}

		// a line break: <CR>+<LF>, <LF> or a <CR> on its own
		end := i + 1
		compliant := false

		switch c {
		case '\n':
			// <LF> on its own
		default:
			j := i
			for j < len(msg) && msg[j] == '\r' {
				j++
			}
			if j < len(msg) && msg[j] == '\n' {
				end = j + 1
				compliant = j-i == 1
			}
		}

		if !compliant {
			if out == nil {
				out = make([]byte, 0, len(msg)+16)
			}
			out = append(out, msg[last:i]...)
			out = append(out, '\r', '\n')
			last = end
		}

		i = end

		if lineLen == 0 {
			// that was the empty line which ends the headers, the rest is the body
			break
		}
		lineLen = 0
	}

	if out == nil {
		return msg, false
	}

	return append(out, msg[last:]...), true
}

// headerRange is the position of a header in a message: from the start of its first line to just
// after its last line (including any folded lines and the final line break).
type headerRange struct {
	start, end int
}

// findHeaders returns the position of every header called name in the header block of msg, in
// order. The message is read line by line, with CRLF or LF line endings, up to the first blank
// line, so the positions are exactly where the header is, rather than where its text happens to
// first appear in the message.
func findHeaders(msg []byte, name string) []headerRange {
	var found []headerRange

	prefix := name + ":"
	current := -1 // index in found of the header being read, which any folded lines belong to

	for pos := 0; pos < len(msg); {
		next := len(msg)
		lineEnd := len(msg)
		if i := bytes.IndexByte(msg[pos:], '\n'); i >= 0 {
			lineEnd = pos + i
			next = lineEnd + 1
		}

		line := bytes.TrimSuffix(msg[pos:lineEnd], []byte("\r"))
		if len(line) == 0 {
			// the end of the headers
			break
		}

		switch {
		case line[0] == ' ' || line[0] == '\t':
			// a folded line belongs to the header before it
			if current >= 0 {
				found[current].end = next
			}
		case len(line) >= len(prefix) && strings.EqualFold(string(line[:len(prefix)]), prefix):
			found = append(found, headerRange{start: pos, end: next})
			current = len(found) - 1
		default:
			current = -1
		}

		pos = next
	}

	return found
}

// replaceHeaders returns msg with the first of the headers replaced by replacement (which may be
// empty to remove it), and any others removed.
func replaceHeaders(msg []byte, headers []headerRange, replacement []byte) []byte {
	out := make([]byte, 0, len(msg)+len(replacement))
	pos := 0

	for i, h := range headers {
		out = append(out, msg[pos:h.start]...)
		if i == 0 {
			out = append(out, replacement...)
		}
		pos = h.end
	}

	return append(out, msg[pos:]...)
}

// hasHeader returns whether the parsed headers have a header called name, even if its value is empty.
func hasHeader(h mail.Header, name string) bool {
	_, ok := h[textproto.CanonicalMIMEHeaderKey(name)]

	return ok
}

// RemoveMessageHeaders scans a message for headers, and removes them if found. Every instance of
// each header is removed, including the ones which are folded over several lines. The line breaks of
// the header block are made <CR><LF> first (see NormaliseHeaderLineBreaks), as are those of the
// other functions which change headers.
//
// It returns an error rather than the original message if a header is found by the parser but
// cannot be located in the raw message.
func RemoveMessageHeaders(msg []byte, headers []string) ([]byte, error) {
	msg, _ = NormaliseHeaderLineBreaks(msg)

	m, err := mail.ReadMessage(bytes.NewReader(msg))
	if err != nil {
		return nil, err
	}

	for _, name := range headers {
		if !hasHeader(m.Header, name) {
			continue
		}

		found := findHeaders(msg, name)
		if len(found) == 0 {
			// mail.ReadMessage saw the header but the raw pass could not locate
			// it. Fail closed rather than silently returning the original message.
			return nil, fmt.Errorf("%s header parsed but not located in raw message", name)
		}

		logger.Log().Debugf("[relay] removed %d %s header(s)", len(found), name)
		msg = replaceHeaders(msg, found, nil)
	}

	return msg, nil
}

// SetMessageHeader scans a message for a header and updates its value if found.
// It does not consider multiple instances of the same header (only the first is updated).
// If not found it will add the header to the beginning of the message.
func SetMessageHeader(msg []byte, header, value string) ([]byte, error) {
	msg, _ = NormaliseHeaderLineBreaks(msg)

	m, err := mail.ReadMessage(bytes.NewReader(msg))
	if err != nil {
		return nil, err
	}

	if hasHeader(m.Header, header) {
		found := findHeaders(msg, header)
		if len(found) == 0 {
			// mail.ReadMessage saw the header but the raw pass could not locate
			// it. Fail closed rather than silently returning the original message.
			return nil, fmt.Errorf("%s header parsed but not located in raw message", header)
		}

		// only the first is replaced
		return replaceHeaders(msg, found[:1], []byte(header+": "+value+"\r\n")), nil
	}

	// no header, so add one to beginning
	return append([]byte(header+": "+value+"\r\n"), msg...), nil
}

// OverrideFromHeader scans a message for the From header and replaces it with a different email address.
// If the message has more than one From header the others are removed, so that the address is the only one.
func OverrideFromHeader(msg []byte, address string) ([]byte, error) {
	msg, _ = NormaliseHeaderLineBreaks(msg)

	m, err := mail.ReadMessage(bytes.NewReader(msg))
	if err != nil {
		return nil, err
	}

	if !hasHeader(m.Header, "From") {
		// no From header, so add one
		msg = append([]byte("From: "+address+"\r\n"), msg...)
		logger.Log().Debugf("[relay] Added From email: %s", address)

		return msg, nil
	}

	found := findHeaders(msg, "From")
	if len(found) == 0 {
		// mail.ReadMessage saw the From header but the raw pass could not
		// locate it. Fail closed rather than silently returning the original
		// message, which would leak an attacker-controlled From through the
		// relay while the envelope carries the operator's address.
		return nil, errors.New("From header parsed but not located in raw message")
	}

	// the value is as parsed, so folded lines are joined
	originalFrom := m.Header.Get("From")

	replacement := "From: " + address + "\r\n"

	if from, err := mail.ParseAddress(originalFrom); err == nil {
		originalFrom = from.Address
		// replace the from email, but keep the original name
		from.Address = address
		replacement = "From: " + from.String() + "\r\n"
	}
	// otherwise there is no address which can be kept, so the whole header is replaced

	msg = replaceHeaders(msg, found, []byte(replacement))

	// insert the original From header as X-Original-From
	if originalFrom != "" {
		msg = append([]byte("X-Original-From: "+originalFrom+"\r\n"), msg...)
	}

	logger.Log().Debugf("[relay] Replaced From email address with %s", address)

	return msg, nil
}

// PrefixSubject prepends prefix, followed by a single space, to the message Subject.
// The existing Subject is decoded (RFC 2047) and the result re-encoded if it
// contains non-ASCII characters. If there is no Subject, one is added containing
// only the prefix. The prefix is space-trimmed, and an empty prefix is a no-op.
func PrefixSubject(msg []byte, prefix string) ([]byte, error) {
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		return msg, nil
	}
	if strings.ContainsAny(prefix, "\r\n") {
		return nil, errors.New("subject prefix must not contain line breaks")
	}

	msg, _ = NormaliseHeaderLineBreaks(msg)

	m, err := mail.ReadMessage(bytes.NewReader(msg))
	if err != nil {
		return nil, err
	}

	subject := m.Header.Get("Subject")
	if decoded, err := new(mime.WordDecoder).DecodeHeader(subject); err == nil {
		subject = decoded
	}
	subject = strings.TrimSpace(subject)

	value := prefix
	if subject != "" {
		value += " " + subject
	}

	// returns the string unchanged if it is plain ASCII
	return SetMessageHeader(msg, "Subject", mime.QEncoding.Encode("utf-8", value))
}
