// Package handlers contains a specific handlers
package handlers

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/axllent/mailpit/config"
	"github.com/axllent/mailpit/internal/logger"
)

var linkRe = regexp.MustCompile(`(?i)^https?:\/\/`)

// ProxyHandler is used to proxy assets for printing
func ProxyHandler(w http.ResponseWriter, r *http.Request) {
	uri := strings.TrimSpace(r.URL.Query().Get("url"))
	if uri == "" {
		logger.Log().Warn("[proxy] URL missing")
		httpError(w, "Error: URL missing")
		return
	}

	if !linkRe.MatchString(uri) {
		logger.Log().Warnf("[proxy] invalid URL %s", uri)
		httpError(w, "Error: invalid URL")
		return
	}

	tr := &http.Transport{}

	if config.AllowUntrustedTLS {
		tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} // #nosec
	}

	client := &http.Client{
		Transport: tr,
		Timeout:   10 * time.Second,
	}

	req, err := http.NewRequest("GET", uri, nil)
	if err != nil {
		logger.Log().Warnf("[proxy] %s", err.Error())
		httpError(w, err.Error())
		return
	}

	// use requesting useragent
	req.Header.Set("User-Agent", r.UserAgent())

	resp, err := client.Do(req)
	if err != nil {
		logger.Log().Warnf("[proxy] %s", err.Error())
		httpError(w, err.Error())
		return
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Log().Warnf("[proxy] %s", err.Error())
		httpError(w, err.Error())
		return
	}

	// relay common headers
	if resp.Header.Get("content-type") != "" {
		w.Header().Set("content-type", resp.Header.Get("content-type"))
	}
	if resp.Header.Get("last-modified") != "" {
		w.Header().Set("last-modified", resp.Header.Get("last-modified"))
	}
	if resp.Header.Get("content-disposition") != "" {
		w.Header().Set("content-disposition", resp.Header.Get("content-disposition"))
	}
	if resp.Header.Get("cache-control") != "" {
		w.Header().Set("cache-control", resp.Header.Get("cache-control"))
	}

	// replace url() values with proxy address, eg: fonts & images
	if strings.HasPrefix(resp.Header.Get("content-type"), "text/css") {
		var re = regexp.MustCompile(`(?mi)(url\((\'|\")?([^\)\'\"]+)(\'|\")?\))`)
		body = re.ReplaceAllFunc(body, func(s []byte) []byte {
			parts := re.FindStringSubmatch(string(s))

			// don't resolve inline `data:..`
			if strings.HasPrefix(parts[3], "data:") {
				return []byte(parts[3])
			}

			address, err := absoluteURL(parts[3], uri)
			if err != nil {
				logger.Log().Errorf("[proxy] %s", err.Error())
				return []byte(parts[3])
			}

			return []byte("url(" + parts[2] + config.Webroot + "proxy?url=" + url.QueryEscape(address) + parts[4] + ")")
		})
	}

	logger.Log().Debugf("[proxy] %s (%d)", uri, resp.StatusCode)

	// relay status code - WriteHeader must come after Header.Set()
	w.WriteHeader(resp.StatusCode)

	if _, err := w.Write(body); err != nil {
		logger.Log().Warnf("[proxy] %s", err.Error())
	}
}

// AbsoluteURL will return a full URL regardless whether it is relative or absolute
func absoluteURL(link, baseURL string) (string, error) {
	// scheme relative links, eg <script src="//example.com/script.js">
	if len(link) > 1 && link[0:2] == "//" {
		base, err := url.Parse(baseURL)
		if err != nil {
			return link, err
		}
		link = base.Scheme + ":" + link
	}

	u, err := url.Parse(link)
	if err != nil {
		return link, err
	}

	// remove hashes
	u.Fragment = ""

	base, err := url.Parse(baseURL)
	if err != nil {
		return link, err
	}

	result := base.ResolveReference(u)

	// ensure link is HTTP(S)
	if result.Scheme != "http" && result.Scheme != "https" {
		return link, fmt.Errorf("Invalid URL: %s", result.String())
	}

	return result.String(), nil
}

// HTTPError returns a basic error message (400 response)
func httpError(w http.ResponseWriter, msg string) {
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("Content-Security-Policy", config.ContentSecurityPolicy)
	w.WriteHeader(http.StatusBadRequest)
	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprint(w, msg)
}
