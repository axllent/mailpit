package htmlcheck

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/axllent/mailpit/internal/tools"
	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
)

// RunTests will run all tests on an HTML string
func RunTests(html string) (Response, error) {
	s := Response{}
	s.Warnings = []Warning{}
	if platforms, err := Platforms(); err == nil {
		s.Platforms = platforms
	}

	s.Total = Total{}

	// crude way to determine whether the HTML contains a <body> structure
	// or whether it's just plain HTML content
	re := regexp.MustCompile(`(?im)</body>`)
	nodeMatch := "body *, script"
	if re.MatchString(html) {
		nodeMatch = "*:not(html):not(head):not(meta), script"
	}
	reader := strings.NewReader(html)
	// Load the HTML document
	doc, err := goquery.NewDocumentFromReader(reader)
	if err != nil {
		return s, err
	}
	// calculate the number of nodes in HTML
	s.Total.Nodes = len(doc.Find(nodeMatch).Nodes)

	if err := loadJSONData(); err != nil {
		return s, err
	}

	// HTML tests
	htmlResults, totalTests, err := runHTMLTests(html)
	if err != nil {
		return s, err
	}

	s.Total.Tests = s.Total.Tests + totalTests

	// add html test totals
	s.Warnings = append(s.Warnings, htmlResults...)

	// CSS tests
	cssResults, totalTests, err := runCSSTests(html)
	if err != nil {
		return s, err
	}

	s.Total.Tests = s.Total.Tests + totalTests

	// add css test totals
	s.Warnings = append(s.Warnings, cssResults...)

	// calculate total score
	var partial, unsupported float32
	partial = 0
	unsupported = 0

	for _, w := range s.Warnings {
		if w.Score.Found == 0 {
			continue
		}

		// supported is calculated by subtracting partial and unsupported from 100%
		if w.Score.Partial > 0 {
			weighted := w.Score.Partial * float32(w.Score.Found) / float32(s.Total.Nodes)
			if weighted > partial {
				partial = weighted
			}
		}
		if w.Score.Unsupported > 0 {
			weighted := w.Score.Unsupported * float32(w.Score.Found) / float32(s.Total.Nodes)
			if weighted > unsupported {
				unsupported = weighted
			}
		}
	}

	s.Total.Supported = 100 - partial - unsupported
	s.Total.Partial = partial
	s.Total.Unsupported = unsupported

	// sort slice to get lowest scores first
	sort.Slice(s.Warnings, func(i, j int) bool {
		return (s.Warnings[i].Score.Unsupported+s.Warnings[i].Score.Partial)*float32(s.Warnings[i].Score.Found)/float32(s.Total.Nodes) >
			(s.Warnings[j].Score.Unsupported+s.Warnings[j].Score.Partial)*float32(s.Warnings[j].Score.Found)/float32(s.Total.Nodes)
	})

	return s, nil
}

// Test returns a test
func (c CanIEmail) getTest(k string) (Warning, error) {
	warning := Warning{}
	exists := false
	found := JSONResult{}
	for _, r := range cie.Data {
		if r.Slug == k {
			found = r
			exists = true
			break
		}
	}

	if !exists {
		return warning, fmt.Errorf("%s does not exist", k)
	}

	warning.Slug = found.Slug
	warning.Title = found.Title
	warning.Description = mdToHTML(found.Description)
	warning.Category = found.Category
	warning.URL = found.URL
	warning.Tags = found.Tags
	// warning.Keywords = found.Keywords
	// warning.Notes = found.Notes
	warning.NotesByNumber = make(map[string]string, len(found.NotesByNumber))
	for nr, note := range found.NotesByNumber {
		warning.NotesByNumber[nr] = mdToHTML(note)
	}
	warning.Results = []Result{}

	var y, n, p float32

	for family, stats := range found.Stats {
		if len(LimitFamilies) != 0 && !tools.InArray(family, LimitFamilies) {
			continue
		}

		for platform, clients := range stats.(map[string]interface{}) {
			if len(LimitPlatforms) != 0 && !tools.InArray(platform, LimitPlatforms) {
				continue
			}
			for version, support := range clients.(map[string]interface{}) {
				s := Result{}
				s.Name = fmt.Sprintf("%s %s (%s)", c.NiceNames.Family[family], c.NiceNames.Platform[platform], version)
				s.Family = family
				s.Platform = platform
				s.Version = version

				if support == "y" {
					y++
					s.Support = "yes"
				} else if support == "n" {
					n++
					s.Support = "no"
				} else {
					p++
					s.Support = "partial"

					noteIDS := noteMatch.FindStringSubmatch(fmt.Sprintf("%s", support))

					for _, id := range noteIDS {
						s.NoteNumber = id
					}
				}

				warning.Results = append(warning.Results, s)
			}
		}

	}

	total := y + n + p
	warning.Score.Supported = y / total * 100
	warning.Score.Unsupported = n / total * 100
	warning.Score.Partial = p / total * 100

	return warning, nil
}

// Convert markdown to HTML, stripping <p> & </p>
func mdToHTML(str string) string {
	md := []byte(str)
	// create markdown parser with extensions
	extensions := parser.CommonExtensions | parser.AutoHeadingIDs | parser.NoEmptyLineBeforeBlock
	// extensions := parser.NoExtensions
	p := parser.NewWithExtensions(extensions)
	doc := p.Parse(md)

	// create HTML renderer with extensions
	htmlFlags := html.CommonFlags | html.HrefTargetBlank
	opts := html.RendererOptions{Flags: htmlFlags}
	renderer := html.NewRenderer(opts)

	return strings.TrimSuffix(strings.TrimPrefix(strings.TrimSpace(string(markdown.Render(doc, renderer))), "<p>"), "</p>")
}
