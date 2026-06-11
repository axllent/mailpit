// Package healthcheck probes a running Mailpit instance's /readyz endpoint.
package healthcheck

import (
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

// Check makes a single readiness probe. Returns nil if the server responds
// with 200 OK.
func Check(client *http.Client, uri string) error {
	res, err := client.Get(uri)
	if err != nil {
		return err
	}
	defer func() { _ = res.Body.Close() }()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: %s", res.Status)
	}

	return nil
}

// Wait polls uri until Check succeeds or timeout elapses.
func Wait(client *http.Client, uri string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for {
		if err := Check(client, uri); err == nil {
			return nil
		}

		if time.Now().After(deadline) {
			return fmt.Errorf("timed out after %s waiting for Mailpit to become ready", timeout)
		}

		time.Sleep(PollInterval)
	}
}
