package linkcheck

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/axllent/mailpit/config"
	"github.com/axllent/mailpit/internal/logger"
	"github.com/axllent/mailpit/internal/tools"
)

func getHTTPStatuses(ctx context.Context, links []string, followRedirects bool) []Link {
	results := make([]Link, len(links))
	var wg sync.WaitGroup
	var warnedDomains sync.Map

	for i, l := range links {
		if cached, ok := cachedLink(l); ok {
			results[i] = cached
			continue
		}

		wg.Add(1)
		go func(idx int, link string) {
			defer wg.Done()

			domain := registeredDomain(link)
			release, err := acquireDomainSlot(ctx, domain, &warnedDomains)
			if err != nil {
				results[idx] = Link{URL: link, StatusCode: 0, Status: httpErrorSummary(err)}
				return
			}
			defer release()

			code, err := doHead(ctx, link, followRedirects)
			l := Link{URL: link}
			if err != nil {
				l.StatusCode = 0
				l.Status = httpErrorSummary(err)
				if strings.Contains(l.Status, "private/reserved address") {
					l.Status = "Blocked private/reserved address"
					l.StatusCode = 451
				}
			} else {
				l.StatusCode = code
				l.Status = http.StatusText(code)
			}
			results[idx] = l
			storeLink(link, l)
		}(i, l)
	}

	wg.Wait()
	return results
}

// Do a HEAD request to return HTTP status code
func doHead(ctx context.Context, link string, followRedirects bool) (int, error) {
	if !tools.IsValidLinkURL(link) {
		return 0, fmt.Errorf("invalid URL: %s", link)
	}

	dialer := &net.Dialer{
		Timeout:   10 * time.Second,
		KeepAlive: 30 * time.Second,
	}

	tr := &http.Transport{
		DialContext: safeDialContext(dialer),
	}

	if config.AllowUntrustedTLS {
		// user has explicitly allowed untrusted TLS, so we will not verify it for link checks
		tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} // #nosec
	}

	client := http.Client{
		Timeout:   10 * time.Second,
		Transport: tr,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 3 {
				return errors.New("too many redirects")
			}
			if !followRedirects {
				return http.ErrUseLastResponse
			}
			if !tools.IsValidLinkURL(req.URL.String()) {
				return fmt.Errorf("blocked redirect to invalid URL: %s", req.URL)
			}
			return nil
		},
	}

	req, err := http.NewRequestWithContext(ctx, "HEAD", link, nil)
	if err != nil {
		logger.Log().Errorf("[link-check] %s", err.Error())
		return 0, err
	}

	req.Header.Set("User-Agent", "Mailpit/"+config.Version)

	res, err := client.Do(req)
	if err != nil {
		if res != nil {
			return res.StatusCode, err
		}

		return 0, err
	}

	return res.StatusCode, nil
}

// HTTP errors include a lot more info that just the actual error, so this
// tries to take the final part of it, eg: `no such host`
func httpErrorSummary(err error) string {
	var re = regexp.MustCompile(`.*: (.*)$`)

	e := err.Error()
	if !re.MatchString(e) {
		return e
	}
	parts := re.FindAllStringSubmatch(e, -1)

	return parts[0][len(parts[0])-1]
}

// SafeDialContext is a custom dialer that checks if the resolved IP addresses are internal before allowing the connection.
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
					logger.Log().Warnf("[link-check] Blocked HEAD request to private/reserved address: %s (%s)", host, ip)
					return nil, fmt.Errorf("blocked request to %s (%s): private/reserved address", host, ip)
				}
			}
		}

		return dialer.DialContext(ctx, network, net.JoinHostPort(ips[0].IP.String(), port))
	}
}
