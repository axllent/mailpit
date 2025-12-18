// Package apiv1 provides the API v1 endpoints for Mailpit.
//
// These structs are for the purpose of defining swagger HTTP responses in go-swagger
// in order to generate a spec file. They are lowercased to avoid exporting them as public types.
//
//nolint:unused
package apiv1

import (
	"github.com/axllent/mailpit/internal/smtpd/chaos"
	"github.com/axllent/mailpit/internal/stats"
)

// Binary data response which inherits the attachment's content type.
// swagger:response BinaryResponse
type binaryResponse string

// Plain text response
// swagger:response TextResponse
type textResponse string

// HTML response
// swagger:response HTMLResponse
type htmlResponse string

// Server error will return with a 400 status code
// with the error message in the body
// swagger:response ErrorResponse
type errorResponse string

// Not found error will return a 404 status code
// swagger:response NotFoundResponse
type notFoundResponse string

// Plain text "ok" response
// swagger:response OKResponse
type okResponse string

// Plain JSON array response
// swagger:response ArrayResponse
type arrayResponse []string

// JSON error response
// swagger:response JSONErrorResponse
type jsonErrorResponse struct {
	// A JSON-encoded error response
	//
	// in: body
	Body struct {
		// Error message
		// example: invalid format
		Error string
	}
}

// Web UI configuration response
// swagger:response WebUIConfigurationResponse
type webUIConfigurationResponse struct {
	// Web UI configuration settings
	//
	// in: body
	Body struct {
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
			// Default "To" address for releasing messages (pre-populates the release form)
			DefaultReleaseTo string
			// Overrides the "From" address for all relayed messages
			OverrideFrom string
			// Preserve the original Message-IDs when relaying messages
			PreserveMessageIDs bool

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

		// Whether the delete button should be hidden
		HideDeleteAllButton bool
	}
}

// Application information
// swagger:response AppInfoResponse
type appInfoResponse struct {
	// Application information
	//
	// in: body
	Body stats.AppInformation
}

// Response for the Chaos triggers configuration
// swagger:response ChaosResponse
type chaosResponse struct {
	// The current Chaos triggers
	//
	// in: body
	Body chaos.Triggers
}

// Message headers
// swagger:model MessageHeadersResponse
type messageHeadersResponse map[string][]string

// Summary of messages
// swagger:response MessagesSummaryResponse
type messagesSummaryResponse struct {
	// The messages summary
	// in: body
	Body MessagesSummary
}

// Confirmation message for HTTP send API
// swagger:response SendMessageResponse
type sendMessageResponse struct {
	// Response for sending messages via the HTTP API
	//
	// in: body
	Body struct {
		// Database ID
		// example: iAfZVVe2UQfNSG5BAjgYwa
		ID string
	}
}
