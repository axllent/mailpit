// Package webhook will optionally call a preconfigured endpoint
package webhook

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"

	"github.com/axllent/mailpit/config"
	"github.com/axllent/mailpit/internal/logger"
	"golang.org/x/time/rate"
)

var (
	// RateLimit is the minimum number of seconds between requests
	RateLimit = 1

	rl rate.Sometimes

	rateLimiterSet bool
)

// Send will post the MessageSummary to a webhook (if configured)
func Send(msg interface{}) {
	if config.WebhookURL == "" {
		return
	}

	if !rateLimiterSet {
		if RateLimit > 0 {
			rl = rate.Sometimes{Interval: time.Duration(RateLimit) * time.Second}
		} else {
			// run 1000 per second - ie: do not limit
			rl = rate.Sometimes{First: 1000, Interval: time.Second}
		}
		rateLimiterSet = true
	}

	go func() {
		rl.Do(func() {
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

			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				logger.Log().Errorf("[webhook] error sending data: %s", err.Error())
				return
			}

			if resp.StatusCode < 200 || resp.StatusCode > 299 {
				logger.Log().Warnf("[webhook] %s returned a %d status", config.WebhookURL, resp.StatusCode)
				return
			}

			defer resp.Body.Close()
		})
	}()
}
