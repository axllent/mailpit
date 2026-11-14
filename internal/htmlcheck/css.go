package htmlcheck

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
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
func runCSSTests(ctx context.Context, html string) ([]Warning, int, error) {
	results := []Warning{}
	totalTests := 0

	inlined, err := inlineRemoteCSS(ctx, html)
	if err != nil {
		// the request was cancelled (e.g. the client disconnected), so abort
		if ctx.Err() != nil {
			return results, totalTests, ctx.Err()
		}
		inlined = html
	}

	// parse the HTML once to collect the CSS code and estimate the cost of merging it
	inlinedDoc, err := goquery.NewDocumentFromReader(strings.NewReader(inlined))
	if err != nil {
		return results, totalTests, err
	}

	cssCode := collectStyleCSS(inlinedDoc)

	// merge all CSS inline, unless the document is so large that doing so would take
	// an unreasonable amount of CPU
	merged := inlined
	if elements := len(inlinedDoc.Find("*").Nodes); mergeWithinBudget(elements, len(cssCode)) {
		if m, err := mergeInlineCSS(inlined); err == nil {
			merged = m
		}
	} else {
		logger.Log().Warnf("[html-check] skipping merging of CSS into elements, the HTML is too large (%d elements, %d bytes of CSS)", elements, len(cssCode))
	}

	reader := strings.NewReader(merged)

	// Load the HTML document
	doc, err := goquery.NewDocumentFromReader(reader)
	if err != nil {
		return results, totalTests, err
	}

	inlineStyleResults := testInlineStyles(doc)
	totalTests = totalTests + len(cssInlineRegexTests) + len(styleInlineAttributes)
	for key, count := range inlineStyleResults {
		result, err := cie.getTest(key)
		if err == nil {
			result.Score.Found = count
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

// maxMergeWork is the maximum value of (number of elements * bytes of CSS) for which the
// CSS is merged into the elements. Merging matches every rule against every element, so
// its cost grows with both and is otherwise only bounded by the message size.
// At this value merging takes in the order of seconds, while real emails are well below it.
const maxMergeWork int64 = 1_000_000_000

// mergeWithinBudget returns true if merging the CSS into the elements is within the work limit.
func mergeWithinBudget(elements, cssBytes int) bool {
	return int64(elements)*int64(cssBytes) <= maxMergeWork
}

// collectStyleCSS returns the content of all <style> elements concatenated.
func collectStyleCSS(doc *goquery.Document) string {
	var sb strings.Builder

	for _, n := range doc.Find("style").Nodes {
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			sb.WriteString(c.Data)
		}
	}

	return sb.String()
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

const (
	// maxRemoteCSSBytes is the total number of bytes of remote CSS downloaded per HTML check.
	maxRemoteCSSBytes int64 = 5 * 1024 * 1024

	// maxRemoteCSSFiles is the maximum number of remote stylesheets requested per HTML check.
	maxRemoteCSSFiles = 10
)

// InlineRemoteCSS searches the HTML for linked stylesheets, downloads the content, and
// inserts new <style> blocks into the head, unless BlockRemoteCSSAndFonts is set.
// The number of stylesheets requested and the total bytes downloaded are both bounded,
// whichever limit is reached first stops any further downloads.
func inlineRemoteCSS(ctx context.Context, h string) (string, error) {
	return inlineRemoteCSSWithLimits(ctx, h, maxRemoteCSSFiles, maxRemoteCSSBytes)
}

func inlineRemoteCSSWithLimits(ctx context.Context, h string, maxFiles int, maxBytes int64) (string, error) {
	reader := strings.NewReader(h)

	// Load the HTML document
	doc, err := goquery.NewDocumentFromReader(reader)
	if err != nil {
		return h, err
	}

	seen := make(map[string]struct{})
	requested := 0
	remaining := maxBytes
	limited := false

	remoteCSS := doc.Find("link[rel=\"stylesheet\"]").Nodes
	for _, link := range remoteCSS {
		attributes := link.Attr
		for _, a := range attributes {
			if a.Key == "href" {
				if config.BlockRemoteCSSAndFonts {
					logger.Log().Debugf("[html-check] skip testing remote CSS content: %s (--block-remote-css-and-fonts)", a.Val)
					return h, nil
				}

				if !isValidURL(a.Val) {
					// skip invalid URL
					logger.Log().Warnf("[html-check] ignoring unsupported stylesheet URL: %s", a.Val)
					continue
				}

				if _, ok := seen[a.Val]; ok {
					continue
				}
				seen[a.Val] = struct{}{}

				if err := ctx.Err(); err != nil {
					return h, err
				}

				if requested >= maxFiles || remaining <= 0 {
					limited = true
					continue
				}
				requested++

				resp, err := downloadCSSToBytes(ctx, a.Val, remaining)
				if err != nil {
					logger.Log().Warnf("[html-check] %s", err.Error())
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

				remaining -= int64(len(resp))

				link.Parent.AppendChild(styleBlock)
			}
		}
	}

	if limited {
		logger.Log().Warnf("[html-check] remote CSS limit reached (max %d files or %d bytes), skipping remaining stylesheets", maxFiles, maxBytes)
	}

	newDoc, err := doc.Html()
	if err != nil {
		logger.Log().Warnf("[html-check] failed to download %s", err.Error())
		return h, err
	}

	return newDoc, nil
}

// DownloadCSSToBytes returns a []byte slice from a URL.
// It requires the HTTP response code to be 200 and the content-type to be text/css.
// It will download a maximum of `limit` bytes.
func downloadCSSToBytes(ctx context.Context, url string, limit int64) ([]byte, error) {
	client := safeHTTPClient()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Mailpit HTML Checker/"+config.Version)

	// Get the link response data
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 200 {
		err := fmt.Errorf("error downloading %s", url)
		return nil, err
	}

	ct := strings.ToLower(resp.Header.Get("content-type"))
	if !strings.Contains(ct, "text/css") {
		err := fmt.Errorf("invalid CSS content-type from %s: \"%s\" (expected \"text/css\")", url, ct)
		return nil, err
	}

	limitedReader := &io.LimitedReader{R: resp.Body, N: limit}

	body, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, err
	}

	return body, nil
}

// Test if the string is a supported URL.
// The URL must have the "http" or "https" scheme, and must not contain any login info (http://user:pass@<host>).
func isValidURL(str string) bool {
	u, err := url.Parse(str)

	return err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Hostname() != "" && u.User.String() == ""
}

// Test the HTML for inline CSS styles and styling attributes
func testInlineStyles(doc *goquery.Document) map[string]int {
	matches := make(map[string]int)

	// find all elements containing a style attribute
	styles := doc.Find("[style]").Nodes
	for _, s := range styles {
		style, err := tools.GetHTMLAttributeVal(s, "style")
		if err != nil {
			continue
		}

		for id, test := range cssInlineRegexTests {
			if test.MatchString(style) {
				if _, ok := matches[id]; !ok {
					matches[id] = 0
				}
				matches[id]++
			}
		}
	}

	// find all elements containing styleInlineAttributes
	for id, test := range styleInlineAttributes {
		a := doc.Find(test).Nodes
		if len(a) > 0 {
			if _, ok := matches[id]; !ok {
				matches[id] = 0
			}
			matches[id]++
		}
	}

	return matches
}

func safeHTTPClient() *http.Client {
	dialer := &net.Dialer{
		Timeout:   5 * time.Second,
		KeepAlive: 30 * time.Second,
	}

	tr := &http.Transport{
		Proxy:                 nil, // avoid env proxy surprises unless you explicitly want it
		DialContext:           safeDialContext(dialer),
		TLSHandshakeTimeout:   5 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		IdleConnTimeout:       30 * time.Second,
		MaxIdleConns:          50,
	}

	client := &http.Client{
		Transport: tr,
		Timeout:   15 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// re-validate every redirect hop.
			if len(via) >= 3 {
				return errors.New("too many redirects")
			}
			if !isValidURL(req.URL.String()) {
				return errors.New("invalid redirect URL")
			}

			return nil
		},
	}

	return client
}

// safeDialContext is the same pattern as linkcheck/status.go::safeDialContext
// — copy the function (or factor a shared helper into internal/tools/net.go).
func safeDialContext(dialer *net.Dialer) func(ctx context.Context, network, address string) (net.Conn, error) {
	return func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, err
		}

		ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
		if err != nil {
			return nil, err
		}

		if !config.AllowInternalHTTPRequests {
			for _, ip := range ips {
				if tools.IsInternalIP(ip.IP) {
					return nil, fmt.Errorf("blocked request to %s (%s): private/reserved address", host, ip)
				}
			}
		}

		return dialer.DialContext(ctx, network, net.JoinHostPort(ips[0].IP.String(), port))
	}
}
