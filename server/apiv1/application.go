package apiv1

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/axllent/mailpit/config"
	"github.com/axllent/mailpit/internal/smtpd/chaos"
	"github.com/axllent/mailpit/internal/stats"
)

// Application information
// swagger:response AppInfoResponse
type appInfoResponse struct {
	// Application information
	//
	// in: body
	Body stats.AppInformation
}

// AppInfo returns some basic details about the running app, and latest release.
func AppInfo(w http.ResponseWriter, _ *http.Request) {
	// swagger:route GET /api/v1/info application AppInformation
	//
	// # Get application information
	//
	// Returns basic runtime information, message totals and latest release version.
	//
	//	Produces:
	//	- application/json
	//
	//	Schemes: http, https
	//
	//	Responses:
	//		200: AppInfoResponse
	//		400: ErrorResponse

	w.Header().Add("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(stats.Load()); err != nil {
		httpError(w, err.Error())
	}
}

// Response includes global web UI settings
//
// swagger:model WebUIConfiguration
type webUIConfiguration struct {
	// Optional label to identify this Mailpit instance
	Label string
	// Message Relay information
	MessageRelay struct {
		// Whether message relaying (release) is enabled
		Enabled bool
		// The configured SMTP server address
		SMTPServer string
		// Enforced Return-Path (if set) for relay bounces
		ReturnPath string
		// Only allow relaying to these recipients (regex)
		AllowedRecipients string
		// Block relaying to these recipients (regex)
		BlockedRecipients string
		// Overrides the "From" address for all relayed messages
		OverrideFrom string
		// DEPRECATED 2024/03/12
		// swagger:ignore
		RecipientAllowlist string
	}

	// Whether SpamAssassin is enabled
	SpamAssassin bool

	// Whether Chaos support is enabled at runtime
	ChaosEnabled bool

	// Whether messages with duplicate IDs are ignored
	DuplicatesIgnored bool
}

// Web UI configuration response
// swagger:response WebUIConfigurationResponse
type webUIConfigurationResponse struct {
	// Web UI configuration settings
	//
	// in: body
	Body webUIConfiguration
}

// WebUIConfig returns configuration settings for the web UI.
func WebUIConfig(w http.ResponseWriter, _ *http.Request) {
	// swagger:route GET /api/v1/webui application WebUIConfiguration
	//
	// # Get web UI configuration
	//
	// Returns configuration settings for the web UI.
	// Intended for web UI only!
	//
	//	Produces:
	//	 - application/json
	//
	//	Schemes: http, https
	//
	//	Responses:
	//	  200: WebUIConfigurationResponse
	//	  400: ErrorResponse

	conf := webUIConfiguration{}

	conf.Label = config.Label
	conf.MessageRelay.Enabled = config.ReleaseEnabled
	if config.ReleaseEnabled {
		conf.MessageRelay.SMTPServer = fmt.Sprintf("%s:%d", config.SMTPRelayConfig.Host, config.SMTPRelayConfig.Port)
		conf.MessageRelay.ReturnPath = config.SMTPRelayConfig.ReturnPath
		conf.MessageRelay.AllowedRecipients = config.SMTPRelayConfig.AllowedRecipients
		conf.MessageRelay.BlockedRecipients = config.SMTPRelayConfig.BlockedRecipients
		conf.MessageRelay.OverrideFrom = config.SMTPRelayConfig.OverrideFrom
		// DEPRECATED 2024/03/12
		conf.MessageRelay.RecipientAllowlist = config.SMTPRelayConfig.AllowedRecipients
	}

	conf.SpamAssassin = config.EnableSpamAssassin != ""
	conf.ChaosEnabled = chaos.Enabled
	conf.DuplicatesIgnored = config.IgnoreDuplicateIDs

	w.Header().Add("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(conf); err != nil {
		httpError(w, err.Error())
	}
}
