// Package postmark uses the free https://spamcheck.postmarkapp.com/
// See https://spamcheck.postmarkapp.com/doc/ for more details.
package postmark

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// endpoint is the Postmark API. It is a variable so that the tests can use their own server.
var endpoint = "https://spamcheck.postmarkapp.com/filter"

// maxResponseBytes is the most that is read from Postmark. A response is a few kilobytes. It is a
// variable for the tests.
var maxResponseBytes int64 = 4 << 20

// maxDescriptionLength is the longest rule description that is looked up in the report. Real
// ones are one line.
const maxDescriptionLength = 1024

var (
	lineBreakRe    = regexp.MustCompile("\r?\n")
	continuationRe = regexp.MustCompile("\n                                ")
)

// Response struct
type Response struct {
	Success bool   `json:"success"`
	Message string `json:"message"` // for errors only
	Score   string `json:"score"`
	Rules   []Rule `json:"rules"`
	Report  string `json:"report"` // ignored
}

// Rule struct
type Rule struct {
	Score string `json:"score"`
	// Name not returned by postmark but rather extracted from description
	Name        string `json:"name"`
	Description string `json:"description"`
}

// Check will post the email data to Postmark
func Check(email []byte, timeout int) (Response, error) {
	r := Response{}
	// '{"email":"raw dump of email", "options":"short"}'
	var d struct {
		// The raw dump of the email to be filtered, including all headers.
		Email string `json:"email"`
		// Default "long". Must either be "long" for a full report of processing rules, or "short" for a score request.
		Options string `json:"options"`
	}

	d.Email = string(email)
	d.Options = "long"

	data, err := json.Marshal(d)
	if err != nil {
		return r, err
	}

	client := http.Client{
		Timeout: time.Duration(timeout) * time.Second,
		// the message must only go to Postmark, so a redirect is not followed
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	resp, err := client.Post(endpoint, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return r, err
	}

	defer func() { _ = resp.Body.Close() }()

	body := io.LimitReader(resp.Body, maxResponseBytes+1)
	counter := &countingReader{r: body}

	decodeErr := json.NewDecoder(counter).Decode(&r)

	// checked first, as a response cut off at the limit also fails to decode
	if counter.n > maxResponseBytes {
		return Response{}, fmt.Errorf("postmark response is larger than %d bytes", maxResponseBytes)
	}

	if decodeErr != nil {
		if resp.StatusCode != http.StatusOK {
			return r, fmt.Errorf("postmark responded with status %d", resp.StatusCode)
		}
		return r, decodeErr
	}

	if resp.StatusCode != http.StatusOK && r.Message == "" {
		return r, fmt.Errorf("postmark responded with status %d", resp.StatusCode)
	}

	// remove trailing line spaces for all lines in report
	lines := lineBreakRe.Split(r.Report, -1)
	reportLines := []string{}
	for _, l := range lines {
		line := strings.TrimRight(l, " ")
		reportLines = append(reportLines, line)
	}
	reportRaw := strings.Join(reportLines, "\n")

	// join description lines to make a single line per rule
	report := continuationRe.ReplaceAllString(reportRaw, "")
	for i, rule := range r.Rules {
		// populate rule name
		r.Rules[i].Name = nameFromReport(rule.Score, rule.Description, report)
	}

	return r, nil
}

// countingReader counts the bytes read.
type countingReader struct {
	r io.Reader
	n int64
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.n += int64(n)

	return n, err
}

// Extract the name of the test from the report as Postmark does not include this in the JSON reports
func nameFromReport(score, description, report string) string {
	// the values come from the response, so they are bounded and a pattern that does not compile
	// gives no name instead of stopping Mailpit
	if len(score) > maxDescriptionLength || len(description) > maxDescriptionLength {
		return ""
	}

	score = regexp.QuoteMeta(score)
	description = regexp.QuoteMeta(description)
	str := fmt.Sprintf("%s\\s+([A-Z0-9\\_]+)\\s+%s", score, description)
	re, err := regexp.Compile(str)
	if err != nil {
		return ""
	}

	matches := re.FindAllStringSubmatch(report, 1)
	if len(matches) > 0 && len(matches[0]) == 2 {
		return strings.TrimSpace(matches[0][1])
	}

	return ""
}
