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
// It will only remove a single instance of any header, and is intended to remove
// Bcc & Message-Id.
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
				logger.Log().Debugf("[release] removing %s header", hdr)
				msg = bytes.Replace(msg, hdr, []byte(""), 1)
			}
		}
	}

	return msg, nil
}
