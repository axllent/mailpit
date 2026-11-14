package tools

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

// maxListUnsubscribeLength is the longest List-Unsubscribe value that is parsed. Real ones are a
// few hundred bytes (an address and a link), but the header is only limited by the size of the
// message, so a limit keeps the cost of viewing a message small whatever it contains.
const maxListUnsubscribeLength = 16 << 10

var (
	luLinkRe      = regexp.MustCompile(`(?mU)<(.*)>`)
	luJoinsRe     = regexp.MustCompile(`(?imUs)>(.*)<`)
	luJoinCharsRe = regexp.MustCompile(`(?imUs)^(\s+)?,(\s+)?$`)
	luWrapperRe   = regexp.MustCompile(`(?imUs)^<(.*)>$`)
	luMailToRe    = regexp.MustCompile(`^mailto:[a-zA-Z0-9]`)
	luHTTPRe      = regexp.MustCompile(`^(?i)https?://[a-zA-Z0-9]`)
	luSpacesRe    = regexp.MustCompile(`\s`)
	luCommentsRe  = regexp.MustCompile(`(?mUs)\(.*\)`)
)

// stripListUnsubscribeComments removes the (comments) from a List-Unsubscribe value in a single
// pass. It used to remove each comment found with a search of the whole value, which takes time
// in proportion to the number of comments multiplied by the length (48 seconds for 80,000 comments).
func stripListUnsubscribeComments(v string) string {
	return strings.TrimSpace(luCommentsRe.ReplaceAllString(v, ""))
}

// ListUnsubscribeParser will attempt to parse a `List-Unsubscribe` header and return
// a slide of addresses (mail & URLs)
func ListUnsubscribeParser(v string) ([]string, error) {
	var results = []string{}
	var hasMailTo bool
	var hasHTTP bool

	v = strings.TrimSpace(v)

	if len(v) > maxListUnsubscribeLength {
		return results, fmt.Errorf("the value is too long to be parsed (%d bytes, the maximum is %d)", len(v), maxListUnsubscribeLength)
	}

	v = stripListUnsubscribeComments(v)

	if !luLinkRe.MatchString(v) {
		return results, fmt.Errorf("\"%s\" no valid unsubscribe links found", v)
	}

	errors := []string{}

	if !luWrapperRe.MatchString(v) {
		return results, fmt.Errorf("\"%s\" should be enclosed in <>", v)
	}

	matches := luLinkRe.FindAllStringSubmatch(v, -1)

	if len(matches) > 2 {
		errors = append(errors, fmt.Sprintf("\"%s\" should include a maximum of one email and one HTTP link", v))
	} else {
		splits := luJoinsRe.FindAllStringSubmatch(v, -1)
		for _, g := range splits {
			if !luJoinCharsRe.MatchString(g[1]) {
				return results, fmt.Errorf("\"%s\" <> should be split with a comma and optional spaces", v)
			}
		}

		for _, m := range matches {
			r := m[1]
			if luSpacesRe.MatchString(r) {
				errors = append(errors, fmt.Sprintf("\"%s\" should not contain spaces", r))
				continue
			}

			if luMailToRe.MatchString(r) {
				if hasMailTo {
					errors = append(errors, fmt.Sprintf("\"%s\" should only contain one mailto:", r))
					continue
				}

				hasMailTo = true
			} else if luHTTPRe.MatchString(r) {
				if hasHTTP {
					errors = append(errors, fmt.Sprintf("\"%s\" should only contain one HTTP link", r))
					continue
				}

				hasHTTP = true

			} else {
				errors = append(errors, fmt.Sprintf("\"%s\" should start with either http(s):// or mailto:", r))
				continue
			}

			_, err := url.ParseRequestURI(r)
			if err != nil {
				errors = append(errors, err.Error())
				continue
			}

			results = append(results, r)
		}
	}

	var err error
	if len(errors) > 0 {
		err = fmt.Errorf("%s", strings.Join(errors, ", "))
	}

	return results, err
}
