// Package healthcheck probes a running Mailpit instance's /readyz endpoint.
package healthcheck

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"path"
	"strings"
	"time"
)

// PollInterval is the delay between polls in Wait. Exported as a variable so
// tests can shorten it.
var PollInterval = time.Second

// CheckTimeout is how long a single probe may take, including reading the
// response. A server which accepts the connection but never answers would
// otherwise hold the probe (and Wait, whatever its timeout) for ever. It is a
// variable for the tests.
var CheckTimeout = 5 * time.Second

// URI builds the readyz URL from a listen address, webroot, and TLS flag.
func URI(listen, webroot string, https bool) string {
	proto := "http"
	if https {
		proto = "https"
	}
	root := strings.TrimRight(path.Join("/", webroot, "/"), "/") + "/"
	return fmt.Sprintf("%s://%s%sreadyz", proto, listen, root)
}

// NewClient returns an HTTP client suitable for probing a Mailpit readyz
// endpoint. TLS verification is disabled because probes typically connect via
// IP, which won't match the server certificate.
func NewClient() *http.Client {
	return &http.Client{Transport: &http.Transport{
		IdleConnTimeout:       time.Second * 5,
		ExpectContinueTimeout: time.Second * 5,
		TLSHandshakeTimeout:   time.Second * 5,
		TLSClientConfig:       &tls.Config{InsecureSkipVerify: true}, // #nosec
	}}
}

// Check makes a single readiness probe, of up to CheckTimeout. Returns nil if
// the server responds with 200 OK.
func Check(client *http.Client, uri string) error {
	return check(context.Background(), client, uri)
}

// check makes a single readiness probe, of up to CheckTimeout and no longer
// than ctx allows.
func check(ctx context.Context, client *http.Client, uri string) error {
	ctx, cancel := context.WithTimeout(ctx, CheckTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, uri, nil)
	if err != nil {
		return err
	}

	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: %s", res.Status)
	}

	return nil
}

// Wait polls uri until Check succeeds or timeout elapses. No probe runs past
// the timeout.
func Wait(client *http.Client, uri string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	ctx, cancel := context.WithDeadline(context.Background(), deadline)
	defer cancel()

	for {
		if err := check(ctx, client, uri); err == nil {
			return nil
		}

		if time.Now().After(deadline) {
			return fmt.Errorf("timed out after %s waiting for Mailpit to become ready", timeout)
		}

		time.Sleep(PollInterval)
	}
}
