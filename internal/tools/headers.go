// Package tools provides various methods for various things
package tools

import (
	"bufio"
	"bytes"
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
				logger.Log().Debugf("[relay] removed %s header", hdr)
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
			logger.Log().Debugf("[relay] replaced %s header", hdr)
			msg = bytes.Replace(msg, hdr, []byte(header+": "+value+"\r\n"), 1)
		}
	}

	return msg, nil
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
		}
	} else {
		// no From header, so add one
		msg = append([]byte("From: "+address+"\r\n"), msg...)
		logger.Log().Debugf("[relay] Added From email: %s", address)
	}

	return msg, nil
}
