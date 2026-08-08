// Package licenses provides embedded license information
package licenses

import (
	_ "embed"
	"regexp"
	"sort"
	"strings"
)

// Mailpit is set by main.go with the embedded MIT license text
var Mailpit string

//go:embed third-party.txt
var thirdPartyRaw string

// LicenseEntry represents a single software license
type LicenseEntry struct {
	// Package or project name
	Name string `json:"Name"`
	// SPDX license identifier
	License string `json:"License"`
	// Full license text
	Text string `json:"Text"`
}

// All returns all licenses with Mailpit first, followed by third-party licenses
func All() []LicenseEntry {
	entries := []LicenseEntry{{
		Name:    "Mailpit",
		License: "MIT",
		Text:    strings.TrimSpace(Mailpit),
	}}
	tp := parseThirdParty()
	sort.Slice(tp, func(i, j int) bool {
		return strings.ToLower(tp[i].Name) < strings.ToLower(tp[j].Name)
	})
	entries = append(entries, tp...)
	return entries
}

var headerRe = regexp.MustCompile(`(?m)^## (.+?) \((.+?)\)\s*$`)

func parseThirdParty() []LicenseEntry {
	matches := headerRe.FindAllStringSubmatchIndex(thirdPartyRaw, -1)
	if len(matches) == 0 {
		return nil
	}

	entries := make([]LicenseEntry, 0, len(matches))
	for i, match := range matches {
		name := thirdPartyRaw[match[2]:match[3]]
		licenseName := thirdPartyRaw[match[4]:match[5]]

		textStart := match[1]
		var textEnd int
		if i+1 < len(matches) {
			textEnd = matches[i+1][0]
		} else {
			textEnd = len(thirdPartyRaw)
		}

		text := strings.TrimSpace(thirdPartyRaw[textStart:textEnd])
		entries = append(entries, LicenseEntry{
			Name:    name,
			License: licenseName,
			Text:    text,
		})
	}
	return entries
}
