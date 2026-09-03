// Package tools provides various methods for various things
package tools

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"net/mail"
	"regexp"
	"strings"

	"github.com/axllent/mailpit/internal/logger"
)

// RemoveMessageHeaders scans a message for headers, if found them removes them.
// It will only remove a single instance of any given message header.
func RemoveMessageHeaders(msg []byte, headers []string) ([]byte, error) {
	reader := bytes.NewReader(msg)
	m, err := mail.ReadMessage(reader)
	if err != nil {
		return nil, err
	}

	reBlank := regexp.MustCompile(`^\s+`)

	for _, name := range headers {
		if m.Header.Get(name) == "" {
			continue
		}

		// case-insensitive
		reHdr := regexp.MustCompile(`(?i)^` + regexp.QuoteMeta(name+":"))

		// bound the scanner to the header block so long body content and
		// oversized header lines can't silently truncate the scan.
		headerBlockEnd := bytes.Index(msg, []byte("\r\n\r\n"))
		if headerBlockEnd < 0 {
			headerBlockEnd = len(msg)
		}
		headerBlock := msg[:headerBlockEnd]

		scanner := bufio.NewScanner(bytes.NewReader(headerBlock))
		scanner.Buffer(make([]byte, 0, 64*1024), len(headerBlock)+1)
		found := false
		hdr := []byte("")
		for scanner.Scan() {
			line := scanner.Bytes()
			if !found && reHdr.Match(line) {
				// add the first line starting with <header>:
				hdr = append(hdr, line...)
				hdr = append(hdr, []byte("\r\n")...)
				found = true
			} else if found && reBlank.Match(line) {
				// add any following lines starting with a whitespace (tab or space)
				hdr = append(hdr, line...)
				hdr = append(hdr, []byte("\r\n")...)
			} else if found {
				// stop scanning, we have the full <header>
				break
			}
		}
		if err := scanner.Err(); err != nil {
			return nil, fmt.Errorf("scanning message for %s header: %w", name, err)
		}

		if len(hdr) == 0 {
			// mail.ReadMessage saw the header but the raw pass could not locate
			// it. Fail closed rather than silently returning the original message.
			return nil, fmt.Errorf("%s header parsed but not located in raw message", name)
		}

		logger.Log().Debugf("[relay] removed %s header", hdr)
		msg = bytes.Replace(msg, hdr, []byte(""), 1)
	}

	return msg, nil
}

// SetMessageHeader scans a message for a header and updates its value if found.
// It does not consider multiple instances of the same header.
// If not found it will add the header to the beginning of the message.
func SetMessageHeader(msg []byte, header, value string) ([]byte, error) {
	reader := bytes.NewReader(msg)
	m, err := mail.ReadMessage(reader)
	if err != nil {
		return nil, err
	}

	if m.Header.Get(header) != "" {
		reBlank := regexp.MustCompile(`^\s+`)
		reHdr := regexp.MustCompile(`(?i)^` + regexp.QuoteMeta(header+":"))

		// bound the scanner to the header block so long body content and
		// oversized header lines can't silently truncate the scan.
		headerBlockEnd := bytes.Index(msg, []byte("\r\n\r\n"))
		if headerBlockEnd < 0 {
			headerBlockEnd = len(msg)
		}
		headerBlock := msg[:headerBlockEnd]

		scanner := bufio.NewScanner(bytes.NewReader(headerBlock))
		scanner.Buffer(make([]byte, 0, 64*1024), len(headerBlock)+1)
		found := false
		hdr := []byte("")
		for scanner.Scan() {
			line := scanner.Bytes()
			if !found && reHdr.Match(line) {
				// add the first line starting with <header>:
				hdr = append(hdr, line...)
				hdr = append(hdr, []byte("\r\n")...)
				found = true
			} else if found && reBlank.Match(line) {
				// add any following lines starting with a whitespace (tab or space)
				hdr = append(hdr, line...)
				hdr = append(hdr, []byte("\r\n")...)
			} else if found {
				// stop scanning, we have the full <header>
				break
			}
		}
		if err := scanner.Err(); err != nil {
			return nil, fmt.Errorf("scanning message for %s header: %w", header, err)
		}

		if len(hdr) == 0 {
			// mail.ReadMessage saw the header but the raw pass could not locate
			// it. Fail closed rather than silently returning the original message.
			return nil, fmt.Errorf("%s header parsed but not located in raw message", header)
		}

		return bytes.Replace(msg, hdr, []byte(header+": "+value+"\r\n"), 1), nil
	}

	// no header, so add one to beginning
	return append([]byte(header+": "+value+"\r\n"), msg...), nil
}

// OverrideFromHeader scans a message for the From header and replaces it with a different email address.
func OverrideFromHeader(msg []byte, address string) ([]byte, error) {
	reader := bytes.NewReader(msg)
	m, err := mail.ReadMessage(reader)
	if err != nil {
		return nil, err
	}

	if m.Header.Get("From") != "" {
		reBlank := regexp.MustCompile(`^\s+`)
		reHdr := regexp.MustCompile(`(?i)^` + regexp.QuoteMeta("From:"))

		// bound the scanner to the header block so long body content and
		// oversized header lines can't silently truncate the scan.
		headerBlockEnd := bytes.Index(msg, []byte("\r\n\r\n"))
		if headerBlockEnd < 0 {
			headerBlockEnd = len(msg)
		}
		headerBlock := msg[:headerBlockEnd]

		scanner := bufio.NewScanner(bytes.NewReader(headerBlock))
		scanner.Buffer(make([]byte, 0, 64*1024), len(headerBlock)+1)
		found := false
		hdr := []byte("")
		for scanner.Scan() {
			line := scanner.Bytes()
			if !found && reHdr.Match(line) {
				// add the first line starting with <header>:
				hdr = append(hdr, line...)
				hdr = append(hdr, []byte("\r\n")...)
				found = true
			} else if found && reBlank.Match(line) {
				// add any following lines starting with a whitespace (tab or space)
				hdr = append(hdr, line...)
				hdr = append(hdr, []byte("\r\n")...)
			} else if found {
				// stop scanning, we have the full <header>
				break
			}
		}
		if err := scanner.Err(); err != nil {
			return nil, fmt.Errorf("scanning message for From header: %w", err)
		}

		if len(hdr) == 0 {
			// mail.ReadMessage saw the From header but the raw pass could not
			// locate it. Fail closed rather than silently returning the original
			// message, which would leak an attacker-controlled From through the
			// relay while the envelope carries the operator's address.
			return nil, errors.New("From header parsed but not located in raw message")
		}

		originalFrom := strings.TrimRight(string(hdr[5:]), "\r\n")

		from, err := mail.ParseAddress(originalFrom)
		if err != nil {
			// error parsing the from address, so just replace the whole line
			msg = bytes.Replace(msg, hdr, []byte("From: "+address+"\r\n"), 1)
		} else {
			originalFrom = from.Address
			// replace the from email, but keep the original name
			from.Address = address
			msg = bytes.Replace(msg, hdr, []byte("From: "+from.String()+"\r\n"), 1)
		}

		// insert the original From header as X-Original-From
		msg = append([]byte("X-Original-From: "+originalFrom+"\r\n"), msg...)

		logger.Log().Debugf("[relay] Replaced From email address with %s", address)
	} else {
		// no From header, so add one
		msg = append([]byte("From: "+address+"\r\n"), msg...)
		logger.Log().Debugf("[relay] Added From email: %s", address)
	}

	return msg, nil
}
