package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/axllent/mailpit/config"
	"github.com/axllent/mailpit/internal/healthcheck"
	"github.com/spf13/cobra"
)

var (
	useHTTPS      bool
	readyzWait    bool
	readyzTimeout time.Duration
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
		uri := healthcheck.URI(config.HTTPListen, config.Webroot, useHTTPS)
		client := healthcheck.NewClient()

		var err error
		if readyzWait {
			err = healthcheck.Wait(client, uri, readyzTimeout)
		} else {
			err = healthcheck.Check(client, uri)
		}

		if err != nil {
			fmt.Fprintln(os.Stderr, err)
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
	readyzCmd.Flags().BoolVar(&readyzWait, "wait", readyzWait, "Wait until Mailpit is ready instead of checking once")
	readyzCmd.Flags().DurationVar(&readyzTimeout, "timeout", 30*time.Second, "Maximum time to wait when --wait is set")
}
