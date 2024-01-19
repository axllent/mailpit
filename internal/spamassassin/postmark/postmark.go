// Package postmark uses the free https://spamcheck.postmarkapp.com/
// See https://spamcheck.postmarkapp.com/doc/ for more details.
package postmark

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"
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
	}

	resp, err := client.Post("https://spamcheck.postmarkapp.com/filter", "application/json",
		bytes.NewBuffer(data))

	if err != nil {
		return r, err
	}

	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&r)

	// remove trailing line spaces for all lines in report
	re := regexp.MustCompile("\r?\n")
	lines := re.Split(r.Report, -1)
	reportLines := []string{}
	for _, l := range lines {
		line := strings.TrimRight(l, " ")
		reportLines = append(reportLines, line)
	}
	reportRaw := strings.Join(reportLines, "\n")

	// join description lines to make a single line per rule
	re2 := regexp.MustCompile("\n                                ")
	report := re2.ReplaceAllString(reportRaw, "")
	for i, rule := range r.Rules {
		// populate rule name
		r.Rules[i].Name = nameFromReport(rule.Score, rule.Description, report)
	}

	return r, err
}

// Extract the name of the test from the report as Postmark does not include this in the JSON reports
func nameFromReport(score, description, report string) string {
	score = regexp.QuoteMeta(score)
	description = regexp.QuoteMeta(description)
	str := fmt.Sprintf("%s\\s+([A-Z0-9\\_]+)\\s+%s", score, description)
	re := regexp.MustCompile(str)

	matches := re.FindAllStringSubmatch(report, 1)
	if len(matches) > 0 && len(matches[0]) == 2 {
		return strings.TrimSpace(matches[0][1])
	}

	return ""
}
