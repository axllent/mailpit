// Package html2text is a simple library to convert HTML to plain text
package html2text

import (
	"bytes"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/net/html"
)

var (
	// block-level closing tags which get a space inserted before a directly following tag,
	// eg: </p><p> becomes </p> <p>
	spaceTags = map[string]bool{
		"div": true, "p": true, "td": true, "th": true,
		"h1": true, "h2": true, "h3": true, "h4": true, "h5": true, "h6": true,
		"ul": true, "ol": true, "li": true, "address": true, "article": true, "aside": true,
		"blockquote": true, "dl": true, "dt": true, "footer": true, "header": true, "hr": true,
		"main": true, "nav": true, "pre": true, "table": true, "thead": true, "tfoot": true, "video": true,
	}
	skip = make(map[string]bool)
)

func init() {
	skip["script"] = true
	skip["title"] = true
	skip["head"] = true
	skip["link"] = true
	skip["meta"] = true
	skip["style"] = true
	skip["noscript"] = true
}

// Strip will convert a HTML string to plain text
func Strip(h string, includeLinks bool) (string, error) {
	plain, withLinks, err := StripBoth(h)
	if includeLinks {
		return withLinks, err
	}

	return plain, err
}

// StripBoth converts a HTML string to plain text, returning both the text without
// and with links, while only parsing the HTML once
func StripBoth(h string) (plain, withLinks string, err error) {
	plain, withLinks, _, err = stripBoth(h)

	return plain, withLinks, err
}

// StripTrimmedAndFull returns the same as Strip(strings.TrimSpace(h), false) and
// Strip(h, true), as used for message snippets and search text respectively.
// Surrounding whitespace only changes the text without links in rare cases (an
// incomplete trailing tag, or a text node which matches a skipped tag name, eg: "link"),
// so the HTML is normally only parsed once.
func StripTrimmedAndFull(h string) (trimmedPlain, withLinks string, err error) {
	plain, withLinks, skipNameText, err := stripBoth(h)
	t := strings.TrimSpace(h)
	if err != nil || t == h || (strings.HasSuffix(t, ">") && !skipNameText) {
		return plain, withLinks, err
	}

	trimmedPlain, err = Strip(t, false)

	return trimmedPlain, withLinks, err
}

// stripBoth does the work for StripBoth, also returning whether any text node
// (ignoring surrounding whitespace) matched a skipped tag name
func stripBoth(h string) (plain, withLinks string, skipNameText bool, err error) {
	h = spaceBlockTags(h)
	h = replaceBr(h)
	h = spaceImg(h)

	doc, err := html.Parse(strings.NewReader(h))
	if err != nil {
		return "", "", false, fmt.Errorf("html2text: parsing HTML: %w", err)
	}

	var plainBuf, linksBuf bytes.Buffer
	extract(doc, &plainBuf, &linksBuf, &skipNameText)

	return clean(plainBuf.String()), clean(linksBuf.String()), skipNameText, nil
}

// spaceBlockTags inserts a space between a closing block-level tag and a directly following
// tag (case-insensitive), eg: "</p><p>" becomes "</p> <p>". The "<" of the following tag is
// consumed by a match, so it cannot start the next match, eg: "</p></div><" only spaces the "</p>".
func spaceBlockTags(h string) string {
	var b strings.Builder
	last := 0 // h[:last] has been handled
	i := 0
	for {
		j := strings.Index(h[i:], "</")
		if j < 0 {
			break
		}
		start := i + j
		i = start + 2
		// the longest tag name is 10 bytes, allow for multi-byte case folds (eg: U+017F, U+212A).
		// Only look for the closing ">" within that window, otherwise many "</" without a
		// nearby ">" would make this quadratic.
		w := h[i:]
		if len(w) > 33 {
			w = w[:33]
		}
		k := strings.IndexByte(w, '>')
		if k < 1 || i+k+1 >= len(h) || h[i+k+1] != '<' || !isSpaceTag(h[i:i+k]) {
			continue
		}
		end := i + k + 2 // after the following "<"
		if b.Cap() == 0 {
			b.Grow(len(h) + 64)
		}
		b.WriteString(h[last : end-1])
		b.WriteString(" <")
		last = end
		i = end
	}
	if last == 0 {
		return h
	}
	b.WriteString(h[last:])

	return b.String()
}

func isSpaceTag(name string) bool {
	if spaceTags[name] {
		return true
	}
	if spaceTags[strings.ToLower(name)] {
		return true
	}
	if isASCII(name) {
		return false
	}
	// Unicode simple case folding (as used by the (?i) regexp flag), eg: "ſ" matches "s"
	for t := range spaceTags {
		if strings.EqualFold(t, name) {
			return true
		}
	}

	return false
}

func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= utf8.RuneSelf {
			return false
		}
	}

	return true
}

// replaceBr replaces "<br>" and "<br />" (case-insensitive) with a space
func replaceBr(h string) string {
	var b strings.Builder
	last := 0
	i := 0
	for {
		j := strings.IndexByte(h[i:], '<')
		if j < 0 {
			break
		}
		start := i + j
		i = start + 1
		if len(h)-start < 4 || h[start+1]|0x20 != 'b' || h[start+2]|0x20 != 'r' {
			continue
		}
		var end int
		switch {
		case strings.HasPrefix(h[start+3:], " />"):
			end = start + 6
		case h[start+3] == '>':
			end = start + 4
		default:
			continue
		}
		if b.Cap() == 0 {
			b.Grow(len(h))
		}
		b.WriteString(h[last:start])
		b.WriteByte(' ')
		last = end
		i = end
	}
	if last == 0 {
		return h
	}
	b.WriteString(h[last:])

	return b.String()
}

// spaceImg inserts a space before every "<img" (case-insensitive)
func spaceImg(h string) string {
	var b strings.Builder
	last := 0
	i := 0
	for {
		j := strings.IndexByte(h[i:], '<')
		if j < 0 {
			break
		}
		start := i + j
		i = start + 1
		if len(h)-start < 4 || h[start+1]|0x20 != 'i' || h[start+2]|0x20 != 'm' || h[start+3]|0x20 != 'g' {
			continue
		}
		if b.Cap() == 0 {
			b.Grow(len(h) + 64)
		}
		b.WriteString(h[last:start])
		b.WriteByte(' ')
		last = start
		i = start + 4
	}
	if last == 0 && b.Cap() == 0 {
		return h
	}
	b.WriteString(h[last:])

	return b.String()
}

func extract(node *html.Node, plain, withLinks *bytes.Buffer, skipNameText *bool) {
	if node.Type == html.TextNode {
		data := node.Data
		if data != "" {
			plain.WriteString(data)
			withLinks.WriteString(data)
		}
	}
	for c := node.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.TextNode && skip[strings.TrimSpace(c.Data)] {
			*skipNameText = true
		}
		if _, shouldSkip := skip[c.Data]; !shouldSkip {
			extract(c, plain, withLinks, skipNameText)
			if c.Data == "a" {
				for _, a := range c.Attr {
					if a.Key == "href" && strings.HasPrefix(strings.ToLower(a.Val), "http") {
						withLinks.WriteString(" " + a.Val + " ")
					}
				}
			}
		}
	}
}

// clean replaces \uFEFF and non-printable characters with a space, collapses
// consecutive spaces and trims the result
func clean(text string) string {
	var b strings.Builder
	b.Grow(len(text))
	space := true // drop leading spaces
	for _, r := range text {
		// replace \uFEFF with space, see https://github.com/golang/go/issues/42274#issuecomment-1017258184
		if r == '\uFEFF' || !unicode.IsPrint(r) {
			r = ' '
		}
		if r == ' ' {
			if !space {
				b.WriteByte(' ')
				space = true
			}
			continue
		}
		space = false
		if r < utf8.RuneSelf {
			b.WriteByte(byte(r))
		} else {
			b.WriteRune(r)
		}
	}

	return strings.TrimSuffix(b.String(), " ")
}
