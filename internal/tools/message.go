// Package tools provides various methods for various things
package tools

import (
	"bufio"
	"bytes"
	"net/mail"
	"regexp"

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

	for _, hdr := range headers {
		// case-insensitive
		reHdr := regexp.MustCompile(`(?i)^` + regexp.QuoteMeta(hdr+":"))

		// header := []byte(hdr + ":")
		if m.Header.Get(hdr) != "" {
			scanner := bufio.NewScanner(bytes.NewReader(msg))
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

			if len(hdr) > 0 {
				logger.Log().Debugf("[release] removed %s header", hdr)
				msg = bytes.Replace(msg, hdr, []byte(""), 1)
			}
		}
	}

	return msg, nil
}

// UpdateMessageHeader scans a message for a header and updates its value if found.
func UpdateMessageHeader(msg []byte, header, value string) ([]byte, error) {
	reader := bytes.NewReader(msg)
	m, err := mail.ReadMessage(reader)
	if err != nil {
		return nil, err
	}

	if m.Header.Get(header) != "" {
		reBlank := regexp.MustCompile(`^\s+`)
		reHdr := regexp.MustCompile(`(?i)^` + regexp.QuoteMeta(header+":"))

		scanner := bufio.NewScanner(bytes.NewReader(msg))
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

		if len(hdr) > 0 {
			logger.Log().Debugf("[release] replaced %s header", hdr)
			msg = bytes.Replace(msg, hdr, []byte(header+": "+value+"\r\n"), 1)
		}
	}

	return msg, nil
}
