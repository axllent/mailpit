package cmd

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"github.com/axllent/mailpit/config"
	"github.com/spf13/cobra"
)

var (
	useHTTPS        bool
	readyzWait      bool
	readyzTimeout   time.Duration
	readyzPollEvery = time.Second
)

// readyzCmd represents the healthcheck command
var readyzCmd = &cobra.Command{
	Use:   "readyz",
	Short: "Run a healthcheck to test if Mailpit is running",
	Long: `This command connects to the /readyz endpoint of a running Mailpit server
and exits with a status of 0 if the connection is successful, else with a 
status 1 if unhealthy.

If running within Docker, it should automatically detect environment
settings to determine the HTTP bind interface & port.
`,
	Run: func(_ *cobra.Command, _ []string) {
		webroot := strings.TrimRight(path.Join("/", config.Webroot, "/"), "/") + "/"
		proto := "http"
		if useHTTPS {
			proto = "https"
		}

		uri := fmt.Sprintf("%s://%s%sreadyz", proto, config.HTTPListen, webroot)

		conf := &http.Transport{
			IdleConnTimeout:       time.Second * 5,
			ExpectContinueTimeout: time.Second * 5,
			TLSHandshakeTimeout:   time.Second * 5,
			// do not verify TLS if this instance is using HTTPS as we connect using IP
			// so won't be the same as the cert
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // #nosec
		}
		client := &http.Client{Transport: conf}

		if readyzWait {
			if err := waitForReady(client, uri, readyzTimeout); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			return
		}

		if err := checkReady(client, uri); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	},
}

func checkReady(client *http.Client, uri string) error {
	res, err := client.Get(uri)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: %s", res.Status)
	}

	return nil
}

func waitForReady(client *http.Client, uri string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for {
		if err := checkReady(client, uri); err == nil {
			return nil
		}

		if time.Now().After(deadline) {
			return fmt.Errorf("timed out after %s waiting for Mailpit to become ready", timeout)
		}

		time.Sleep(readyzPollEvery)
	}
}

func init() {
	rootCmd.AddCommand(readyzCmd)

	if len(os.Getenv("MP_UI_BIND_ADDR")) > 0 {
		config.HTTPListen = os.Getenv("MP_UI_BIND_ADDR")
	}

	if len(os.Getenv("MP_WEBROOT")) > 0 {
		config.Webroot = os.Getenv("MP_WEBROOT")
	}

	config.UITLSCert = os.Getenv("MP_UI_TLS_CERT")

	if config.UITLSCert != "" {
		useHTTPS = true
	}

	readyzCmd.Flags().StringVarP(&config.HTTPListen, "listen", "l", config.HTTPListen, "Set the HTTP bind interface & port")
	readyzCmd.Flags().StringVar(&config.Webroot, "webroot", config.Webroot, "Set the webroot for web UI & API")
	readyzCmd.Flags().BoolVar(&useHTTPS, "https", useHTTPS, "Connect via HTTPS (ignores HTTPS validation)")
	readyzCmd.Flags().BoolVar(&readyzWait, "wait", readyzWait, "Wait until Mailpit is ready instead of checking once")
	readyzCmd.Flags().DurationVar(&readyzTimeout, "timeout", 30*time.Second, "Maximum time to wait when --wait is set")
}
