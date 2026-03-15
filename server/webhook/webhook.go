// Package webhook will optionally call a preconfigured endpoint
package webhook

import (
	"bytes"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/axllent/mailpit/config"
	"github.com/axllent/mailpit/internal/logger"
	"golang.org/x/time/rate"
)

var (
	// RateLimit is the minimum number of seconds between requests.
	// Additional requests within this period will be ignored until
	// the time has elapsed.
	RateLimit = 1

	// Delay is the number of seconds to wait before sending each webhook request
	// This can allow for other processing to complete before the webhook is triggered.
	Delay = 0

	rl rate.Sometimes

	once sync.Once
)

// Send will post the MessageSummary to a webhook (if configured)
func Send(msg any) {
	if config.WebhookURL == "" {
		return
	}

	once.Do(func() {
		if RateLimit > 0 {
			rl = rate.Sometimes{Interval: time.Duration(RateLimit) * time.Second}
		} else {
			// allow every request
			rl = rate.Sometimes{Every: 1}
		}
	})

	rl.Do(func() {
		go func() {
			// apply delay if configured
			if Delay > 0 {
				time.Sleep(time.Duration(Delay) * time.Second)
			}

			b, err := json.Marshal(msg)
			if err != nil {
				logger.Log().Errorf("[webhook] invalid data: %s", err.Error())
				return
			}

			req, err := http.NewRequest("POST", config.WebhookURL, bytes.NewBuffer(b))
			if err != nil {
				logger.Log().Errorf("[webhook] error: %s", err.Error())
				return
			}

			req.Header.Set("User-Agent", "Mailpit/"+config.Version)
			req.Header.Set("Content-Type", "application/json")

			if config.Label != "" {
				req.Header.Set("Mailpit-Label", config.Label)
			}

			client := &http.Client{Timeout: 5 * time.Second}
			resp, err := client.Do(req)
			if err != nil {
				logger.Log().Errorf("[webhook] error sending data: %s", err.Error())
				return
			}
			defer func() { _ = resp.Body.Close() }()

			if resp.StatusCode < 200 || resp.StatusCode > 299 {
				logger.Log().Warnf("[webhook] %s returned a %d status", config.WebhookURL, resp.StatusCode)
				return
			}
		}()
	})
}
