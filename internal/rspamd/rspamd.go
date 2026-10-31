// Package rspamd returns results from an Rspamd server using its HTTP API
// See https://docs.rspamd.com/developers/protocol/ for details.
package rspamd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

const (
	// maxResponseSize is the maximum size of a response that will be read from Rspamd
	maxResponseSize = 10 << 20

	// timeout for requests to Rspamd
	timeout = 8 * time.Second
)

var (
	// base URL of the Rspamd server, without a trailing slash
	baseURL string

	// optional controller password
	password string
)

// Result is a Rspamd result
//
// swagger:model RspamdResponse
type Result struct {
	// Whether the message is spam or not
	IsSpam bool
	// If populated will return an error string
	Error string
	// Total spam score based on triggered symbols
	Score float64
	// Score at which Rspamd will start to treat a message as spam
	Threshold float64
	// Action Rspamd recommends, eg: "no action", "greylist", "add header", "reject"
	Action string
	// Rspamd symbols triggered
	Rules []Rule
}

// Rule struct
type Rule struct {
	// Symbol score
	Score float64
	// Rspamd symbol name
	Name string
	// Rspamd symbol description
	Description string
}

// rspamdResponse is the relevant subset of the /checkv2 response
type rspamdResponse struct {
	Score         float64            `json:"score"`
	RequiredScore float64            `json:"required_score"`
	Action        string             `json:"action"`
	Thresholds    map[string]float64 `json:"thresholds"`
	Symbols       map[string]struct {
		Name        string  `json:"name"`
		Score       float64 `json:"score"`
		Description string  `json:"description"`
	} `json:"symbols"`
	Error string `json:"error"`
}

// SetService sets the Rspamd server, either as a URL or "<host>:<port>" (http is assumed).
// An optional password may be included as URL userinfo, eg: https://:password@host:11334
// (special characters in the password must be percent-encoded). The returned string is the
// server address without the password, suitable for logging.
func SetService(server string) (string, error) {
	server = strings.TrimSpace(server)
	if !strings.HasPrefix(server, "http://") && !strings.HasPrefix(server, "https://") {
		server = "http://" + server
	}

	u, err := url.Parse(server)
	if err != nil {
		// do not include the original error, as it contains the full URL (and password)
		return "", errors.New("invalid Rspamd address")
	}
	if u.Host == "" {
		return "", errors.New("invalid Rspamd address: missing host")
	}

	password = ""
	if u.User != nil {
		// Rspamd has no username, so also accept "https://password@host"
		if pass, ok := u.User.Password(); ok {
			password = pass
		} else {
			password = u.User.Username()
		}
	}
	u.User = nil
	u.Path = strings.TrimRight(u.Path, "/")
	baseURL = u.String()

	return baseURL, nil
}

// Ping returns whether the Rspamd server is reachable
func Ping() error {
	body, status, err := do(http.MethodGet, "/ping", nil)
	if err != nil {
		return err
	}
	if status != http.StatusOK || !strings.HasPrefix(strings.TrimSpace(string(body)), "pong") {
		return fmt.Errorf("unexpected response from Rspamd (HTTP %d)", status)
	}

	return nil
}

// Check will return a Result. Errors communicating with Rspamd are returned in Result.Error.
func Check(msg []byte) (Result, error) {
	r := Result{}

	if baseURL == "" {
		return r, errors.New("no Rspamd service defined")
	}

	body, status, err := do(http.MethodPost, "/checkv2", msg)
	if err != nil {
		r.Error = err.Error()
		return r, nil
	}

	var res rspamdResponse
	if err := json.Unmarshal(body, &res); err != nil {
		if status != http.StatusOK {
			r.Error = fmt.Sprintf("Rspamd returned HTTP %d", status)
		} else {
			r.Error = "invalid response from Rspamd: " + err.Error()
		}
		return r, nil
	}
	if status != http.StatusOK || res.Error != "" {
		r.Error = res.Error
		if r.Error == "" {
			r.Error = fmt.Sprintf("Rspamd returned HTTP %d", status)
		}
		return r, nil
	}

	r.Score = round2dp(res.Score)
	r.Action = res.Action
	r.Threshold = res.RequiredScore
	// Rspamd treats a message as spam from the "add header" action upwards
	for _, k := range []string{"add header", "reject"} {
		if v := res.Thresholds[k]; v > 0 {
			r.Threshold = v
			break
		}
	}
	r.Threshold = round2dp(r.Threshold)

	switch res.Action {
	case "reject", "add header", "rewrite subject":
		r.IsSpam = true
	case "":
		r.IsSpam = r.Threshold > 0 && res.Score >= r.Threshold
	}

	r.Rules = []Rule{}
	for key, s := range res.Symbols {
		name := s.Name
		if name == "" {
			name = key
		}
		r.Rules = append(r.Rules, Rule{Score: round2dp(s.Score), Name: name, Description: s.Description})
	}
	// highest impact first, then alphabetical for a stable order
	sort.Slice(r.Rules, func(i, j int) bool {
		a, b := r.Rules[i], r.Rules[j]
		if a.Score != b.Score {
			return a.Score > b.Score
		}
		return a.Name < b.Name
	})

	return r, nil
}

// do performs a request against the Rspamd server
func do(method, path string, data []byte) ([]byte, int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, method, baseURL+path, bytes.NewReader(data))
	if err != nil {
		return nil, 0, err
	}
	if password != "" {
		req.Header.Set("Password", password)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize))
	if err != nil {
		return nil, 0, err
	}

	return body, resp.StatusCode, nil
}

// Round to two decimal places
func round2dp(n float64) float64 {
	return math.Round(n*100) / 100
}
