package linkcheck

import (
	"crypto/tls"
	"net/http"
	"regexp"
	"sync"
	"time"

	"github.com/axllent/mailpit/config"
	"github.com/axllent/mailpit/internal/logger"
)

func getHTTPStatuses(links []string, followRedirects bool) []Link {
	// allow 5 threads
	threads := make(chan int, 5)

	results := make(map[string]Link, len(links))
	resultsMutex := sync.RWMutex{}

	output := []Link{}

	var wg sync.WaitGroup

	for _, l := range links {
		wg.Add(1)
		go func(link string, w *sync.WaitGroup) {
			threads <- 1 // will block if MAX threads
			defer w.Done()

			code, err := doHead(link, followRedirects)
			l := Link{}
			l.URL = link
			if err != nil {
				l.StatusCode = 0
				l.Status = httpErrorSummary(err)
			} else {
				l.StatusCode = code
				l.Status = http.StatusText(code)
			}
			resultsMutex.Lock()
			results[link] = l
			resultsMutex.Unlock()

			<-threads // remove from threads
		}(l, &wg)
	}

	wg.Wait()

	for _, l := range results {
		output = append(output, l)
	}

	return output
}

// Do a HEAD request to return HTTP status code
func doHead(link string, followRedirects bool) (int, error) {

	timeout := time.Duration(10 * time.Second)

	tr := &http.Transport{}

	if config.AllowUntrustedTLS {
		tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} // #nosec
	}

	client := http.Client{
		Timeout:   timeout,
		Transport: tr,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if followRedirects {
				return nil
			}
			return http.ErrUseLastResponse
		},
	}

	req, err := http.NewRequest("HEAD", link, nil)
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
