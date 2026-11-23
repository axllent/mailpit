// Package spamassassin will return results from either a SpamAssassin server or
// Postmark's public API depending on configuration
package spamassassin

import (
	"errors"
	"math"
	"strings"

	"github.com/axllent/mailpit/internal/spamassassin/postmark"
	"github.com/axllent/mailpit/internal/spamassassin/spamc"
)

var (
	// Service to use, either "<host>:<ip>" for self-hosted SpamAssassin or "postmark"
	service string

	// spamScore is the score at which a message is determined to be spam when the service does not
	// say (Postmark does not). SpamAssassin's own required score is used for that.
	spamScore = 5.0

	// Timeout in seconds
	timeout = 8
)

// Result is a SpamAssassin result
//
// openapi:model SpamAssassinResponse
type Result struct {
	// Whether the message is spam or not
	IsSpam bool
	// If populated will return an error string
	Error string
	// Total spam score based on triggered rules
	Score float64
	// Score at which a message is treated as spam: the required score of the SpamAssassin server
	// (5 for Postmark)
	Threshold float64
	// Spam rules triggered
	Rules []Rule
}

// Rule struct
//
// openapi:model Rule
type Rule struct {
	// Spam rule score
	Score float64
	// SpamAssassin rule name
	Name string
	// SpamAssassin rule description
	Description string
}

// SetService defines which service should be used.
func SetService(s string) {
	switch s {
	case "postmark":
		service = "postmark"
	default:
		service = s
	}
}

// client returns the spamd client for the configured service.
func client() *spamc.Client {
	if path, ok := strings.CutPrefix(service, "unix:"); ok {
		return spamc.NewUnix(path, timeout)
	}

	return spamc.NewTCP(service, timeout)
}

// Ping returns whether a service is active or not
func Ping() error {
	if service == "postmark" {
		return nil
	}

	return client().Ping()
}

// Check will return a Result
func Check(msg []byte) (Result, error) {
	r := Result{Score: 0}

	if service == "" {
		return r, errors.New("no SpamAssassin service defined")
	}

	if service == "postmark" {
		res, err := postmark.Check(msg, timeout)
		if err != nil {
			r.Error = err.Error()
			return r, nil
		}

		return fromPostmark(res), nil
	} else {
		res, err := client().Report(msg)
		if err != nil {
			r.Error = err.Error()
			return r, nil
		}
		// the server decides what is spam, by the required score it is configured with
		r.IsSpam = res.Spam
		r.Threshold = round2dp(res.Threshold)
		r.Score = round1dm(res.Score)
		r.Rules = []Rule{}
		for _, sr := range res.Rules {
			rule := Rule{}
			if value, err := spamc.ParseScore(sr.Points); err == nil {
				rule.Score = round1dm(value)
			}
			rule.Name = sr.Name
			rule.Description = sr.Description
			r.Rules = append(r.Rules, rule)
		}
	}

	return r, nil
}

// fromPostmark returns the result from Postmark's response. Postmark does not return the score it
// treats as spam, so it is the usual 5.
func fromPostmark(res postmark.Response) Result {
	r := Result{Threshold: spamScore, Error: res.Message}

	if res.Score != "" || res.Success {
		score, err := spamc.ParseScore(res.Score)
		if err != nil {
			r.Error = "Postmark returned an invalid score"
			return r
		}
		r.Score = round1dm(score)
		r.IsSpam = score >= spamScore
	}

	if r.Error == "" && !res.Success {
		r.Error = "Postmark did not return a result"
	}

	for _, pr := range res.Rules {
		rule := Rule{}
		if value, err := spamc.ParseScore(pr.Score); err == nil {
			rule.Score = round1dm(value)
		}
		rule.Name = pr.Name
		rule.Description = pr.Description
		r.Rules = append(r.Rules, rule)
	}

	return r
}

// round2dp rounds to the nearest two decimal places, which a required score can have (eg: 4.25)
func round2dp(n float64) float64 {
	return math.Round(n*100) / 100
}

// Round down to one decimal place
func round1dm(n float64) float64 {
	return math.Floor(n*10) / 10
}
