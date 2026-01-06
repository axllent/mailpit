// Package handlers contains a specific handlers
package handlers

import (
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/axllent/mailpit/config"
	"github.com/axllent/mailpit/internal/logger"
	"github.com/axllent/mailpit/internal/storage"
	"github.com/axllent/mailpit/internal/tools"
)

var (
	linkRe = regexp.MustCompile(`(?i)^https?:\/\/`)

	urlRe = regexp.MustCompile(`(?mU)url\(('|")?(https?:\/\/[^)'"]+)('|")?\)`)

	assetsMutex sync.Mutex

	assets = map[string]MessageAssets{}
)

// MessageAssets represents assets linked in a message
type MessageAssets struct {
	ID string
	// Created timestamp so we can expire old entries
	Created time.Time
	// Assets found in the message
	Assets []string
}

func init() {
	// Start a goroutine to clean up old asset entries every minute
	go func() {
		for {
			time.Sleep(time.Minute)
			assetsMutex.Lock()
			now := time.Now()
			for id, entry := range assets {
				if now.Sub(entry.Created) > time.Minute {
					logger.Log().Debugf("[proxy] cleaning up assets for message %s", id)
					delete(assets, id)
				}
			}
			assetsMutex.Unlock()
		}
	}()
}

// ProxyHandler is used to proxy assets for printing.
// It accepts a base64-encoded message-id:url string as the `data` query parameter.
func ProxyHandler(w http.ResponseWriter, r *http.Request) {
	encoded := strings.TrimSpace(r.URL.Query().Get("data"))
	if encoded == "" {
		logger.Log().Warn("[proxy] Data missing")
		httpError(w, "Error: Data missing")
		return
	}

	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		logger.Log().Warnf("[proxy] Data parameter corrupted: %s", err.Error())
		httpError(w, "Error: invalid request")
		return
	}

	parts := strings.SplitN(string(decoded), ":", 2)
	if len(parts) != 2 {
		logger.Log().Warnf("[proxy] Invalid data parameter: %s", string(decoded))
		httpError(w, "Error: invalid request")
		return
	}

	id := parts[0]
	uri := parts[1]

	links, err := getAssets(id)
	if err != nil {
		httpError(w, "Error: invalid request")
		return
	}

	if !tools.InArray(uri, links) {
		logger.Log().Warnf("[proxy] URL %s not found in message %s", uri, id)
		httpError(w, "Error: invalid request")
		return
	}

	if !linkRe.MatchString(uri) {
		logger.Log().Warnf("[proxy] invalid request %s", uri)
		httpError(w, "Error: invalid request")
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
		httpError(w, "Error: invalid request")
		return
	}

	// use requesting useragent
	req.Header.Set("User-Agent", r.UserAgent())

	resp, err := client.Do(req)
	if err != nil {
		logger.Log().Warnf("[proxy] %s", err.Error())
		httpError(w, "Error: invalid request")
		return
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		logger.Log().Warnf("[proxy] received status code %d for %s", resp.StatusCode, uri)
		httpError(w, "Error: invalid request")
		return
	}

	ct := strings.ToLower(resp.Header.Get("content-type"))
	if !supportedProxyContentType(ct) {
		logger.Log().Warnf("[proxy] blocking unsupported content-type %s for %s", ct, uri)
		httpError(w, "Error: invalid request")
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Log().Warnf("[proxy] %s", err.Error())
		httpError(w, "Error: invalid request")
		return
	}

	// relay common headers
	w.Header().Set("content-type", ct)
	if resp.Header.Get("last-modified") != "" {
		w.Header().Set("last-modified", resp.Header.Get("last-modified"))
	}
	if resp.Header.Get("content-disposition") != "" {
		w.Header().Set("content-disposition", resp.Header.Get("content-disposition"))
	}
	if resp.Header.Get("cache-control") != "" {
		w.Header().Set("cache-control", resp.Header.Get("cache-control"))
	}

	// replace CSS url() values with proxy address, eg: fonts & images
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

			// store asset address against message ID
			if result, ok := assets[id]; ok {
				if !tools.InArray(address, result.Assets) {
					assetsMutex.Lock()
					result.Assets = append(result.Assets, address)
					assets[id] = result
					assetsMutex.Unlock()
				}
			}

			// encode with base64 to handle any special characters and group message ID with URL
			encoded := base64.StdEncoding.EncodeToString([]byte(id + ":" + address))

			return []byte("url(" + parts[2] + config.Webroot + "proxy?data=" + encoded + parts[4] + ")")
		})
	}

	logger.Log().Debugf("[proxy] %s (%d)", uri, resp.StatusCode)

	// relay status code - WriteHeader must come after Header.Set()
	w.WriteHeader(resp.StatusCode)

	if _, err := w.Write(body); err != nil {
		logger.Log().Warnf("[proxy] %s", err.Error())
	}
}

// GetAssets retrieves and parses the message to return linked assets.
// Linked CSS files are appended to the assets list via the ProxyHandler when proxying CSS files.
func getAssets(id string) ([]string, error) {
	assetsMutex.Lock()
	defer assetsMutex.Unlock()

	result, ok := assets[id]
	if ok {
		// return cached assets
		return result.Assets, nil
	}

	msg, err := storage.GetMessage(id)
	if err != nil {
		return nil, err
	}

	links := []string{}

	reader := strings.NewReader(msg.HTML)

	// load the HTML document
	doc, err := goquery.NewDocumentFromReader(reader)
	if err != nil {
		return nil, err
	}

	// css & font links
	doc.Find("link").Each(func(_ int, s *goquery.Selection) {
		if href, exists := s.Attr("href"); exists {
			if linkRe.MatchString(href) && !tools.InArray(href, links) {
				links = append(links, href)
			}
		}
	})

	// images
	doc.Find("img").Each(func(_ int, s *goquery.Selection) {
		if src, exists := s.Attr("src"); exists {
			if linkRe.MatchString(src) && !tools.InArray(src, links) {
				links = append(links, src)
			}
		}
	})

	// background="<>" links
	doc.Find("[background]").Each(func(_ int, s *goquery.Selection) {
		if bg, exists := s.Attr("background"); exists {
			if linkRe.MatchString(bg) && !tools.InArray(bg, links) {
				links = append(links, bg)
			}
		}
	})

	// url(<>) links in style blocks
	matches := urlRe.FindAllStringSubmatch(msg.HTML, -1)
	for _, match := range matches {
		if len(match) >= 3 {
			link := match[2]
			if linkRe.MatchString(link) && !tools.InArray(link, links) {
				links = append(links, link)
			}
		}
	}

	r := MessageAssets{}
	r.ID = id
	r.Created = time.Now()
	r.Assets = links
	assets[id] = r

	return links, nil
}

// AbsoluteURL will return a full URL regardless whether it is relative or absolute.
// This is used to replace relative CSS url(...) links when proxying.
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
		return link, fmt.Errorf("invalid URL: %s", result.String())
	}

	return result.String(), nil
}

// HTTPError returns a basic error message (400 response)
func httpError(w http.ResponseWriter, msg string) {
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("Content-Security-Policy", config.ContentSecurityPolicy)
	w.WriteHeader(http.StatusBadRequest)
	w.Header().Set("Content-Type", "text/plain")
	_, _ = fmt.Fprint(w, msg)
}

// SupportedProxyContentType checks if the content-type is supported for proxying.
// This is limited to fonts, images and css only.
func supportedProxyContentType(ct string) bool {
	ct = strings.ToLower(ct)

	types := []string{
		"font/otf",
		"font/ttf",
		"font/woff",
		"font/woff2",
		"image/apng",
		"image/avif",
		"image/bmp",
		"image/gif",
		"image/jpeg",
		"image/jpg",
		"image/png",
		"image/tiff",
		"image/svg+xml",
		"image/webp",
		"text/css",
	}

	for _, t := range types {
		if strings.HasPrefix(ct, t) {
			return true
		}
	}

	return false
}
