package tools

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

// ListUnsubscribeParser will attempt to parse a `List-Unsubscribe` header and return
// a slide of addresses (mail & URLs)
func ListUnsubscribeParser(v string) ([]string, error) {
	var results = []string{}
	var re = regexp.MustCompile(`(?mU)<(.*)>`)
	var reJoins = regexp.MustCompile(`(?imUs)>(.*)<`)
	var reValidJoinChars = regexp.MustCompile(`(?imUs)^(\s+)?,(\s+)?$`)
	var reWrapper = regexp.MustCompile(`(?imUs)^<(.*)>$`)
	var reMailTo = regexp.MustCompile(`^mailto:[a-zA-Z0-9]`)
	var reHTTP = regexp.MustCompile(`^(?i)https?://[a-zA-Z0-9]`)
	var reSpaces = regexp.MustCompile(`\s`)
	var reComments = regexp.MustCompile(`(?mUs)\(.*\)`)
	var hasMailTo bool
	var hasHTTP bool

	v = strings.TrimSpace(v)

	comments := reComments.FindAllStringSubmatch(v, -1)
	for _, c := range comments {
		// strip comments
		v = strings.Replace(v, c[0], "", -1)
		v = strings.TrimSpace(v)
	}

	if !re.MatchString(v) {
		return results, fmt.Errorf("\"%s\" no valid unsubscribe links found", v)
	}

	errors := []string{}

	if !reWrapper.MatchString(v) {
		return results, fmt.Errorf("\"%s\" should be enclosed in <>", v)
	}

	matches := re.FindAllStringSubmatch(v, -1)

	if len(matches) > 2 {
		errors = append(errors, fmt.Sprintf("\"%s\" should include a maximum of one email and one HTTP link", v))
	} else {
		splits := reJoins.FindAllStringSubmatch(v, -1)
		for _, g := range splits {
			if !reValidJoinChars.MatchString(g[1]) {
				return results, fmt.Errorf("\"%s\" <> should be split with a comma and optional spaces", v)
			}
		}

		for _, m := range matches {
			r := m[1]
			if reSpaces.MatchString(r) {
				errors = append(errors, fmt.Sprintf("\"%s\" should not contain spaces", r))
				continue
			}

			if reMailTo.MatchString(r) {
				if hasMailTo {
					errors = append(errors, fmt.Sprintf("\"%s\" should only contain one mailto:", r))
					continue
				}

				hasMailTo = true
			} else if reHTTP.MatchString(r) {
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
