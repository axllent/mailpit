package tools

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

const (
	// maxEmailActions is the maximum number of actions returned for a message
	maxEmailActions = 10
	// maxEmailActionsJSON is the largest JSON-LD block that is parsed
	maxEmailActionsJSON = 64 << 10
	// maxEmailActionsDepth is the deepest nesting that is searched for actions
	maxEmailActionsDepth = 8
)

// EmailAction is an email action (quick action button, as rendered by Gmail and
// other clients that support it) found in the schema.org JSON-LD markup of an HTML message
//
// swagger:model EmailAction
type EmailAction struct {
	// schema.org action type, eg: ConfirmAction
	Type string
	// Button label
	Name string
	// Action target URL (if any)
	URL string
	// Description (if set)
	Description string
	// Validation errors that would prevent mail clients from showing the action
	Errors []string
}

// emailActionsHintRe is a cheap check, so HTML without JSON-LD is never parsed
var emailActionsHintRe = regexp.MustCompile(`(?i)ld\+json`)

// schemaName removes the schema.org prefix from a type, eg: http://schema.org/ViewAction
func schemaName(s string) string {
	return strings.TrimPrefix(strings.TrimPrefix(s, "http://schema.org/"), "https://schema.org/")
}

// emailActionTypes are the action types in the schema.org email markup, as documented by Gmail
// https://developers.google.com/workspace/gmail/markup/actions/actions-overview
var emailActionTypes = map[string]bool{
	"ConfirmAction": true,
	"SaveAction":    true,
	"RsvpAction":    true,
	"ReviewAction":  true,
	"TrackAction":   true,
	"UpdateAction":  true,
	"ViewAction":    true,
	"GoToAction":    true,
}

// ParseEmailActions extracts the email actions from the JSON-LD blocks in an HTML message.
// It never fails: invalid blocks are reported as an action with errors.
func ParseEmailActions(html string) []EmailAction {
	results := []EmailAction{}
	if !emailActionsHintRe.MatchString(html) {
		return results
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return results
	}

	doc.Find("script").Each(func(_ int, s *goquery.Selection) {
		if len(results) >= maxEmailActions {
			return
		}
		if t, _ := s.Attr("type"); !strings.EqualFold(strings.TrimSpace(t), "application/ld+json") {
			return
		}
		raw := strings.TrimSpace(s.Text())
		if len(raw) > maxEmailActionsJSON {
			results = append(results, EmailAction{Errors: []string{"JSON-LD block is too large"}})
			return
		}
		var data any
		if err := json.Unmarshal([]byte(raw), &data); err != nil {
			results = append(results, EmailAction{Errors: []string{"invalid JSON-LD: " + err.Error()}})
			return
		}
		collectEmailActions(data, false, 0, &results)
	})

	if len(results) > maxEmailActions {
		results = results[:maxEmailActions]
	}

	return results
}

// collectEmailActions walks a decoded JSON-LD value looking for `potentialAction`
func collectEmailActions(v any, onEmail bool, depth int, out *[]EmailAction) {
	if depth > maxEmailActionsDepth {
		return
	}
	switch n := v.(type) {
	case []any:
		for _, i := range n {
			collectEmailActions(i, onEmail, depth+1, out)
		}
	case map[string]any:
		if g, ok := n["@graph"]; ok {
			collectEmailActions(g, onEmail, depth+1, out)
		}
		isEmail := jsonLDHasType(n, "EmailMessage")
		if pa, ok := n["potentialAction"]; ok {
			for _, a := range toSlice(pa) {
				if m, ok := a.(map[string]any); ok && len(*out) < maxEmailActions {
					*out = append(*out, buildEmailAction(m, isEmail))
				}
			}
		}
	}
}

func toSlice(v any) []any {
	if s, ok := v.([]any); ok {
		return s
	}
	return []any{v}
}

func jsonLDHasType(m map[string]any, t string) bool {
	for _, v := range toSlice(m["@type"]) {
		if s, ok := v.(string); ok && schemaName(s) == t {
			return true
		}
	}
	return false
}

func jsonLDString(m map[string]any, k string) string {
	s, _ := m[k].(string)
	return strings.TrimSpace(s)
}

func buildEmailAction(m map[string]any, onEmail bool) EmailAction {
	a := EmailAction{Errors: []string{}}
	for _, t := range toSlice(m["@type"]) {
		if s, ok := t.(string); ok {
			a.Type = schemaName(s)
			break
		}
	}
	a.Name = jsonLDString(m, "name")
	a.Description = jsonLDString(m, "description")

	// the target can be a string, an EntryPoint / HttpActionHandler or a list of them
	for _, t := range toSlice(m["target"]) {
		switch tv := t.(type) {
		case string:
			a.URL = strings.TrimSpace(tv)
		case map[string]any:
			a.URL = jsonLDString(tv, "url")
			if a.URL == "" {
				a.URL = jsonLDString(tv, "urlTemplate")
			}
		}
		if a.URL != "" {
			break
		}
	}
	if a.URL == "" {
		a.URL = jsonLDString(m, "url")
	}

	if !onEmail {
		a.Errors = append(a.Errors, "action is not attached to an EmailMessage")
	}
	if a.Type == "" {
		a.Errors = append(a.Errors, "missing @type")
	} else if !emailActionTypes[a.Type] {
		a.Errors = append(a.Errors, fmt.Sprintf("action type %q is not a supported email action", a.Type))
	}
	if a.Name == "" {
		a.Errors = append(a.Errors, "missing name (button label)")
	}
	if a.URL == "" {
		a.Errors = append(a.Errors, "missing target URL")
	} else if u, err := url.Parse(a.URL); err != nil || u.Scheme != "https" {
		a.Errors = append(a.Errors, "target URL must be HTTPS")
	}

	return a
}
