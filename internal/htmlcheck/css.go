package htmlcheck

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/axllent/mailpit/config"
	"github.com/axllent/mailpit/internal/logger"
	"github.com/axllent/mailpit/internal/tools"
	"github.com/vanng822/go-premailer/premailer"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// Go cannot calculate any rendered CSS attributes, so we merge all styles
// into the HTML and detect elements with styles containing the keywords.
func runCSSTests(html string) ([]Warning, int, error) {
	results := []Warning{}
	totalTests := 0

	inlined, err := inlineRemoteCSS(html)
	if err != nil {
		inlined = html
	}

	// merge all CSS inline
	merged, err := mergeInlineCSS(inlined)
	if err != nil {
		merged = inlined
	}

	reader := strings.NewReader(merged)

	// Load the HTML document
	doc, err := goquery.NewDocumentFromReader(reader)
	if err != nil {
		return results, totalTests, err
	}

	for key, test := range cssInlineTests {
		totalTests++
		found := len(doc.Find(test).Nodes)
		if found > 0 {
			result, err := cie.getTest(key)
			if err != nil {
				return results, totalTests, err
			}
			result.Score.Found = found
			results = append(results, result)
		}
	}

	// get a list of all generated styles from all nodes
	allNodeStyles := []string{}
	for _, n := range doc.Find("*[style]").Nodes {
		style, err := tools.GetHTMLAttributeVal(n, "style")
		if err == nil {
			allNodeStyles = append(allNodeStyles, style)
		}
	}

	for key, re := range cssRegexpUnitTests {
		totalTests++
		result, err := cie.getTest(key)
		if err != nil {
			return results, totalTests, err
		}

		found := 0
		// loop through all styles to count total
		for _, styles := range allNodeStyles {
			found = found + len(re.FindAllString(styles, -1))
		}

		if found > 0 {
			result.Score.Found = found
			results = append(results, result)
		}
	}

	// get all inline CSS block data
	reader = strings.NewReader(inlined)

	// Load the HTML document
	doc, _ = goquery.NewDocumentFromReader(reader)

	cssCode := ""
	for _, n := range doc.Find("style").Nodes {
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			cssCode = cssCode + c.Data
		}
	}

	for key, re := range cssRegexpTests {
		totalTests++
		result, err := cie.getTest(key)
		if err != nil {
			return results, totalTests, err
		}

		found := len(re.FindAllString(cssCode, -1))
		if found > 0 {
			result.Score.Found = found
			results = append(results, result)
		}
	}

	return results, totalTests, nil
}

// MergeInlineCSS merges header CSS into element attributes
func mergeInlineCSS(html string) (string, error) {
	options := premailer.NewOptions()
	// options.RemoveClasses = true
	// options.CssToAttributes = false
	options.KeepBangImportant = true
	pre, err := premailer.NewPremailerFromString(html, options)
	if err != nil {
		return "", err
	}

	return pre.Transform()
}

// InlineRemoteCSS searches the HTML for linked stylesheets, downloads the content, and
// inserts new <style> blocks into the head, unless BlockRemoteCSSAndFonts is set
func inlineRemoteCSS(h string) (string, error) {
	reader := strings.NewReader(h)

	// Load the HTML document
	doc, err := goquery.NewDocumentFromReader(reader)
	if err != nil {
		return h, err
	}

	remoteCSS := doc.Find("link[rel=\"stylesheet\"]").Nodes
	for _, link := range remoteCSS {
		attributes := link.Attr
		for _, a := range attributes {
			if a.Key == "href" {
				if !isURL(a.Val) {
					// skip invalid URL
					continue
				}

				if config.BlockRemoteCSSAndFonts {
					logger.Log().Debugf("[html-check] skip testing remote CSS content: %s (--block-remote-css-and-fonts)", a.Val)
					return h, nil
				}

				resp, err := downloadToBytes(a.Val)
				if err != nil {
					logger.Log().Warnf("[html-check] failed to download %s", a.Val)
					continue
				}

				// create new <style> block and insert downloaded CSS
				styleBlock := &html.Node{
					Type:     html.ElementNode,
					Data:     "style",
					DataAtom: atom.Style,
				}
				styleBlock.AppendChild(&html.Node{
					Type: html.TextNode,
					Data: string(resp),
				})

				link.Parent.AppendChild(styleBlock)
			}
		}
	}

	newDoc, err := doc.Html()
	if err != nil {
		logger.Log().Warnf("[html-check] failed to download %s", err.Error())
		return h, err
	}

	return newDoc, nil
}

// DownloadToBytes returns a []byte slice from a URL
func downloadToBytes(url string) ([]byte, error) {
	client := http.Client{
		Timeout: 5 * time.Second,
	}

	// Get the link response data
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		err := fmt.Errorf("Error downloading %s", url)
		return nil, err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

// Test if str is a URL
func isURL(str string) bool {
	u, err := url.Parse(str)
	return err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Host != ""
}
