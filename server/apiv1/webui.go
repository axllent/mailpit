package apiv1

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/axllent/mailpit/config"
)

// Response includes global web UI settings
//
// swagger:model WebUIConfiguration
type webUIConfiguration struct {
	// Message Relay information
	MessageRelay struct {
		// Whether message relaying (release) is enabled
		Enabled bool
		// The configured SMTP server address
		SMTPServer string
		// Enforced Return-Path (if set) for relay bounces
		ReturnPath string
		// Allowlist of accepted recipients
		RecipientAllowlist string
	}
}

// WebUIConfig returns configuration settings for the web UI.
func WebUIConfig(w http.ResponseWriter, r *http.Request) {
	// swagger:route GET /api/v1/webui application WebUIConfiguration
	//
	// # Get web UI configuration
	//
	// Returns configuration settings for the web UI.
	//
	//	Produces:
	//	- application/json
	//
	//	Schemes: http, https
	//
	//	Responses:
	//		200: WebUIConfigurationResponse
	//		default: ErrorResponse
	conf := webUIConfiguration{}

	conf.MessageRelay.Enabled = config.ReleaseEnabled
	if config.ReleaseEnabled {
		conf.MessageRelay.SMTPServer = fmt.Sprintf("%s:%d", config.SMTPRelayConfig.Host, config.SMTPRelayConfig.Port)
		conf.MessageRelay.ReturnPath = config.SMTPRelayConfig.ReturnPath
		conf.MessageRelay.RecipientAllowlist = config.SMTPRelayConfig.RecipientAllowlist
	}

	bytes, _ := json.Marshal(conf)

	w.Header().Add("Content-Type", "application/json")
	_, _ = w.Write(bytes)
}
