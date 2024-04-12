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
	useHTTPS bool
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
	Run: func(cmd *cobra.Command, args []string) {
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
			// do not verify TLS in case this instance is using HTTPS
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // #nosec
		}
		client := &http.Client{Transport: conf}

		res, err := client.Get(uri)
		if err != nil || res.StatusCode != 200 {
			os.Exit(1)
		}
	},
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
}
