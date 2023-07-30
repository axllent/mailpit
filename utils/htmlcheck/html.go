package htmlcheck

import (
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/html"
)

// HTML tests
func runHTMLTests(html string) ([]Warning, int, error) {
	results := []Warning{}
	totalTests := 0

	reader := strings.NewReader(html)

	// Load the HTML document
	doc, err := goquery.NewDocumentFromReader(reader)
	if err != nil {
		return results, totalTests, err
	}

	// Almost all <script> is bad
	scripts := len(doc.Find("script:not([type=\"application/ld+json\"])").Nodes)
	if scripts > 0 {
		var result = Warning{}
		result.Title = "<script> element"
		result.Slug = "html-script"
		result.Category = "html"
		result.Description = "JavaScript is not supported in any email client."
		result.Tags = []string{}
		result.Results = []Result{}
		result.NotesByNumber = map[string]string{}
		result.Score.Found = scripts
		result.Score.Supported = 0
		result.Score.Partial = 0
		result.Score.Unsupported = 100
		results = append(results, result)
		totalTests++
	}

	for key, test := range htmlTests {
		totalTests++
		if test == "body" {
			re := regexp.MustCompile(`(?im)</body>`)
			if re.MatchString(html) {
				result, err := cie.getTest(key)
				if err != nil {
					return results, totalTests, err
				}

				result.Score.Found = 1
				results = append(results, result)
			}
		} else if len(doc.Find(test).Nodes) > 0 {
			result, err := cie.getTest(key)
			if err != nil {
				return results, totalTests, err
			}
			totalTests++

			result.Score.Found = len(doc.Find(test).Nodes)

			results = append(results, result)
		}
	}

	// find all images
	images := doc.Find("img[src]").Nodes
	imageResults := make(map[string]int)
	totalTests = totalTests + len(imageRegexpTests)

	for _, image := range images {
		src, err := getHTMLAttributeVal(image, "src")
		if err != nil {
			continue
		}
		for key, test := range imageRegexpTests {
			if test.MatchString(src) {
				matches, exists := imageResults[key]
				if exists {
					imageResults[key] = matches + 1
				} else {
					imageResults[key] = 1
				}

			}
		}
	}

	for key, found := range imageResults {
		result, err := cie.getTest(key)
		if err != nil {
			return results, totalTests, err
		}
		result.Score.Found = found
		results = append(results, result)
	}

	return results, totalTests, nil
}

func getHTMLAttributeVal(e *html.Node, key string) (string, error) {
	for _, a := range e.Attr {
		if a.Key == key {
			return a.Val, nil
		}
	}

	return "", nil
}
