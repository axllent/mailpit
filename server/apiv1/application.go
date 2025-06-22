package apiv1

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/axllent/mailpit/config"
	"github.com/axllent/mailpit/internal/smtpd/chaos"
	"github.com/axllent/mailpit/internal/stats"
)

// AppInfo returns some basic details about the running app including the latest release (unless disabled).
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

// WebUIConfig returns configuration settings for the web UI.
func WebUIConfig(w http.ResponseWriter, _ *http.Request) {
	// swagger:route GET /api/v1/webui application WebUIConfigurationResponse
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

	conf := webUIConfigurationResponse{}

	conf.Body.Label = config.Label
	conf.Body.MessageRelay.Enabled = config.ReleaseEnabled
	if config.ReleaseEnabled {
		conf.Body.MessageRelay.SMTPServer = fmt.Sprintf("%s:%d", config.SMTPRelayConfig.Host, config.SMTPRelayConfig.Port)
		conf.Body.MessageRelay.ReturnPath = config.SMTPRelayConfig.ReturnPath
		conf.Body.MessageRelay.AllowedRecipients = config.SMTPRelayConfig.AllowedRecipients
		conf.Body.MessageRelay.BlockedRecipients = config.SMTPRelayConfig.BlockedRecipients
		conf.Body.MessageRelay.OverrideFrom = config.SMTPRelayConfig.OverrideFrom
		conf.Body.MessageRelay.PreserveMessageIDs = config.SMTPRelayConfig.PreserveMessageIDs

		// DEPRECATED 2024/03/12
		conf.Body.MessageRelay.RecipientAllowlist = config.SMTPRelayConfig.AllowedRecipients
	}

	conf.Body.SpamAssassin = config.EnableSpamAssassin != ""
	conf.Body.ChaosEnabled = chaos.Enabled
	conf.Body.DuplicatesIgnored = config.IgnoreDuplicateIDs
	conf.Body.HideDeleteAllButton = config.HideDeleteAllButton

	w.Header().Add("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(conf.Body); err != nil {
		httpError(w, err.Error())
	}
}
