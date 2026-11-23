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
	w.Header().Add("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(stats.Load(true)); err != nil {
		httpError(w, err.Error())
	}
}

// WebUIMessageRelay is the message relay (release) configuration for the web UI
//
// openapi:model WebUIMessageRelay
type WebUIMessageRelay struct {
	// Whether message relaying (release) is enabled
	// example: true
	Enabled bool
	// The configured SMTP server address
	// example: smtp.example.com:587
	SMTPServer string
	// Enforced Return-Path (if set) for relay bounces
	// example: bounce@example.com
	ReturnPath string
	// Only allow relaying to these recipients (regex)
	// example: .*@example.com
	AllowedRecipients string
	// Block relaying to these recipients (regex)
	// example: .*@blocked.com
	BlockedRecipients string
	// Overrides the "From" address for all relayed messages
	// example: no-reply@example.com
	OverrideFrom string
	// Preserve the original Message-IDs when relaying messages
	// example: true
	PreserveMessageIDs bool

	// DEPRECATED 2024/03/12
	// openapi:ignore
	RecipientAllowlist string
}

// WebUIConfiguration returns the Web UI configuration settings
//
// openapi:model WebUIConfiguration
type WebUIConfiguration struct {
	// Optional label to identify this Mailpit instance
	// example: Local Mailpit dev
	Label string

	// Message Relay information
	MessageRelay WebUIMessageRelay

	// Whether SpamAssassin is enabled
	// example: false
	SpamAssassin bool

	// Whether Rspamd is enabled
	// example: true
	Rspamd bool

	// Whether Chaos support is enabled at runtime
	// example: false
	ChaosEnabled bool

	// Whether messages with duplicate IDs are ignored
	// example: true
	DuplicatesIgnored bool

	// Whether the delete button should be hidden
	// example: false
	HideDeleteAllButton bool
}

// WebUIConfig returns configuration settings for the web UI.
func WebUIConfig(w http.ResponseWriter, _ *http.Request) {
	conf := WebUIConfiguration{}

	conf.Label = config.Label
	conf.MessageRelay.Enabled = config.ReleaseEnabled
	if config.ReleaseEnabled {
		conf.MessageRelay.SMTPServer = fmt.Sprintf("%s:%d", config.SMTPRelayConfig.Host, config.SMTPRelayConfig.Port)
		conf.MessageRelay.ReturnPath = config.SMTPRelayConfig.ReturnPath
		conf.MessageRelay.AllowedRecipients = config.SMTPRelayConfig.AllowedRecipients
		conf.MessageRelay.BlockedRecipients = config.SMTPRelayConfig.BlockedRecipients
		conf.MessageRelay.OverrideFrom = config.SMTPRelayConfig.OverrideFrom
		conf.MessageRelay.PreserveMessageIDs = config.SMTPRelayConfig.PreserveMessageIDs

		// DEPRECATED 2024/03/12
		conf.MessageRelay.RecipientAllowlist = config.SMTPRelayConfig.AllowedRecipients
	}

	conf.SpamAssassin = config.EnableSpamAssassin != ""
	conf.Rspamd = config.EnableRspamd != ""
	conf.ChaosEnabled = chaos.Enabled
	conf.DuplicatesIgnored = config.IgnoreDuplicateIDs
	conf.HideDeleteAllButton = config.HideDeleteAllButton

	w.Header().Add("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(conf); err != nil {
		httpError(w, err.Error())
	}
}
