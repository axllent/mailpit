// Package spamassassin will return results from either a SpamAssassin server or
// Postmark's public API depending on configuration
package spamassassin

import (
	"errors"
	"math"
	"strconv"
	"strings"

	"github.com/axllent/mailpit/internal/spamassassin/postmark"
	"github.com/axllent/mailpit/internal/spamassassin/spamc"
)

var (
	// Service to use, either "<host>:<ip>" for self-hosted SpamAssassin or "postmark"
	service string

	// SpamScore is the score at which a message is determined to be spam
	spamScore = 5.0

	// Timeout in seconds
	timeout = 8
)

// Result is a SpamAssassin result
//
// swagger:model SpamAssassinResponse
type Result struct {
	// Whether the message is spam or not
	IsSpam bool
	// If populated will return an error string
	Error string
	// Total spam score based on triggered rules
	Score float64
	// Spam rules triggered
	Rules []Rule
}

// Rule struct
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

// SetTimeout defines the timeout
func SetTimeout(t int) {
	if t > 0 {
		timeout = t
	}
}

// Ping returns whether a service is active or not
func Ping() error {
	if service == "postmark" {
		return nil
	}

	var client *spamc.Client
	if strings.HasPrefix(service, "unix:") {
		client = spamc.NewUnix(strings.TrimLeft(service, "unix:"))
	} else {
		client = spamc.NewTCP(service, timeout)
	}

	return client.Ping()
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
		resFloat, err := strconv.ParseFloat(res.Score, 32)
		if err == nil {
			r.Score = round1dm(resFloat)
			r.IsSpam = resFloat >= spamScore
		}
		r.Error = res.Message
		for _, pr := range res.Rules {
			rule := Rule{}
			value, err := strconv.ParseFloat(pr.Score, 32)
			if err == nil {
				rule.Score = round1dm(value)
			}
			rule.Name = pr.Name
			rule.Description = pr.Description
			r.Rules = append(r.Rules, rule)
		}
	} else {
		var client *spamc.Client
		if strings.HasPrefix(service, "unix:") {
			client = spamc.NewUnix(strings.TrimLeft(service, "unix:"))
		} else {
			client = spamc.NewTCP(service, timeout)
		}

		res, err := client.Report(msg)
		if err != nil {
			r.Error = err.Error()
			return r, nil
		}
		r.IsSpam = res.Score >= spamScore
		r.Score = round1dm(res.Score)
		r.Rules = []Rule{}
		for _, sr := range res.Rules {
			rule := Rule{}
			value, err := strconv.ParseFloat(sr.Points, 32)
			if err == nil {
				rule.Score = round1dm(value)
			}
			rule.Name = sr.Name
			rule.Description = sr.Description
			r.Rules = append(r.Rules, rule)
		}
	}

	return r, nil
}

// Round to one decimal place
func round1dm(n float64) float64 {
	return math.Floor(n*10) / 10
}
