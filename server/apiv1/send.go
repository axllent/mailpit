package apiv1

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/mail"
	"strings"

	"github.com/axllent/mailpit/config"
	"github.com/axllent/mailpit/internal/smtpd"
	"github.com/axllent/mailpit/internal/tools"
	"github.com/jhillyerd/enmime"
)

// swagger:parameters SendMessageParams
type sendMessageParams struct {
	// in: body
	Body *SendRequest
}

// SendRequest to send a message via HTTP
// swagger:model SendRequest
type SendRequest struct {
	// "From" recipient
	// required: true
	From struct {
		// Optional name
		// example: John Doe
		Name string
		// Email address
		// example: john@example.com
		// required: true
		Email string
	}

	// "To" recipients
	To []struct {
		// Optional name
		// example: Jane Doe
		Name string
		// Email address
		// example: jane@example.com
		// required: true
		Email string
	}

	// Cc recipients
	Cc []struct {
		// Optional name
		// example: Manager
		Name string
		// Email address
		// example: manager@example.com
		// required: true
		Email string
	}

	// Bcc recipients email addresses only
	// example: ["jack@example.com"]
	Bcc []string

	// Optional Reply-To recipients
	ReplyTo []struct {
		// Optional name
		// example: Secretary
		Name string
		// Email address
		// example: secretary@example.com
		// required: true
		Email string
	}

	// Subject
	// example: Mailpit message via the HTTP API
	Subject string

	// Message body (text)
	// example: Mailpit is awesome!
	Text string

	// Message body (HTML)
	// example: <div style="text-align:center"><p style="font-family: arial; font-size: 24px;">Mailpit is <b>awesome</b>!</p><p><img src="cid:mailpit-logo" /></p></div>
	HTML string

	// Attachments
	Attachments []struct {
		// Base64-encoded string of the file content
		// required: true
		// example: iVBORw0KGgoAAAANSUhEUgAAAEEAAAA8CAMAAAAOlSdoAAAACXBIWXMAAAHrAAAB6wGM2bZBAAAAS1BMVEVHcEwRfnUkZ2gAt4UsSF8At4UtSV4At4YsSV4At4YsSV8At4YsSV4At4YsSV4sSV4At4YsSV4At4YtSV4At4YsSV4At4YtSV8At4YsUWYNAAAAGHRSTlMAAwoXGiktRE5dbnd7kpOlr7zJ0d3h8PD8PCSRAAACWUlEQVR42pXT4ZaqIBSG4W9rhqQYocG+/ys9Y0Z0Br+x3j8zaxUPewFh65K+7yrIMeIY4MT3wPfEJCidKXEMnLaVkxDiELiMz4WEOAZSFghxBIypCOlKiAMgXfIqTnBgSm8CIQ6BImxEUxEckClVQiHGj4Ba4AQHikAIClwTE9KtIghAhUJwoLkmLnCiAHJLRKgIMsEtVUKbBUIwoAg2C4QgQBE6l4VCnApBgSKYLLApCnCa0+96AEMW2BQcmC+Pr3nfp7o5Exy49gIADcIqUELGfeA+bp93LmAJp8QJoEcN3C7NY3sbVANixMyI0nku20/n5/ZRf3KI2k6JEDWQtxcbdGuAqu3TAXG+/799Oyyas1B1MnMiA+XyxHp9q0PUKGPiRAau1fZbLRZV09wZcT8/gHk8QQAxXn8VgaDqcUmU6O/r28nbVwXAqca2mRNtPAF5+zoP2MeN9Fy4NgC6RfcbgE7XITBRYTtOE3U3C2DVff7pk+PkUxgAbvtnPXJaD6DxulMLwOhPS/M3MQkgg1ZFrIXnmfaZoOfpKiFgzeZD/WuKqQEGrfJYkyWf6vlG3xUgTuscnkNkQsb599q124kdpMUjCa/XARHs1gZymVtGt3wLkiFv8rUgTxitYCex5EVGec0Y9VmoDTFBSQte2TfXGXlf7hbdaUM9Sk7fisEN9qfBBTK+FZcvM9fQSdkl2vj4W2oX/bRogO3XasiNH7R0eW7fgRM834ImTg+Lg6BEnx4vz81rhr+MYPBBQg1v8GndEOrthxaCTxNAOut8WKLGZQl+MPz88Q9tAO/hVuSeqQAAAABJRU5ErkJggg==
		Content string
		// Filename
		// required: true
		// example: mailpit.png
		Filename string
		// Optional Content Type for the the attachment.
		// If this field is not set (or empty) then the content type is automatically detected.
		// required: false
		// example: image/png
		ContentType string
		// Optional Content-ID (`cid`) for attachment.
		// If this field is set then the file is attached inline.
		// required: false
		// example: mailpit-logo
		ContentID string
	}

	// Mailpit tags
	// example: ["Tag 1","Tag 2"]
	Tags []string

	// Optional headers in {"key":"value"} format
	// example: {"X-IP":"1.2.3.4"}
	Headers map[string]string
}

// JSONErrorMessage struct
type JSONErrorMessage struct {
	// Error message
	// example: invalid format
	Error string
}

// Confirmation message for HTTP send API
// swagger:response sendMessageResponse
type sendMessageResponse struct {
	// Response for sending messages via the HTTP API
	//
	// in: body
	Body SendMessageConfirmation
}

// SendMessageConfirmation struct
type SendMessageConfirmation struct {
	// Database ID
	// example: iAfZVVe2UQfNSG5BAjgYwa
	ID string
}

// SendMessageHandler handles HTTP requests to send a new message
func SendMessageHandler(w http.ResponseWriter, r *http.Request) {
	// swagger:route POST /api/v1/send message SendMessageParams
	//
	// # Send a message
	//
	// Send a message via the HTTP API.
	//
	//	Consumes:
	//	  - application/json
	//
	//	Produces:
	//	  - application/json
	//
	//	Schemes: http, https
	//
	//	Responses:
	//	  200: sendMessageResponse
	//	  400: jsonErrorResponse

	if config.DemoMode {
		httpJSONError(w, "this functionality has been disabled for demonstration purposes")
		return
	}

	decoder := json.NewDecoder(r.Body)

	data := SendRequest{}

	if err := decoder.Decode(&data); err != nil {
		httpJSONError(w, err.Error())
		return
	}

	id, err := data.Send(r.RemoteAddr)

	if err != nil {
		httpJSONError(w, err.Error())
		return
	}

	w.Header().Add("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(SendMessageConfirmation{ID: id}); err != nil {
		httpError(w, err.Error())
	}
}

// Send will validate the message structure and attempt to send to Mailpit.
// It returns a sending summary or an error.
func (d SendRequest) Send(remoteAddr string) (string, error) {
	ip, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		return "", fmt.Errorf("error parsing request RemoteAddr: %s", err.Error())
	}

	ipAddr := &net.IPAddr{IP: net.ParseIP(ip)}

	addresses := []string{}

	msg := enmime.Builder().
		From(d.From.Name, d.From.Email).
		Subject(d.Subject).
		Text([]byte(d.Text))

	if d.HTML != "" {
		msg = msg.HTML([]byte(d.HTML))
	}

	if len(d.To) > 0 {
		for _, a := range d.To {
			if _, err := mail.ParseAddress(a.Email); err == nil {
				msg = msg.To(a.Name, a.Email)
				addresses = append(addresses, a.Email)
			} else {
				return "", fmt.Errorf("invalid To address: %s", a.Email)
			}
		}
	}

	if len(d.Cc) > 0 {
		for _, a := range d.Cc {
			if _, err := mail.ParseAddress(a.Email); err == nil {
				msg = msg.CC(a.Name, a.Email)
				addresses = append(addresses, a.Email)
			} else {
				return "", fmt.Errorf("invalid Cc address: %s", a.Email)
			}
		}
	}

	if len(d.Bcc) > 0 {
		for _, e := range d.Bcc {
			if _, err := mail.ParseAddress(e); err == nil {
				msg = msg.BCC("", e)
				addresses = append(addresses, e)
			} else {
				return "", fmt.Errorf("invalid Bcc address: %s", e)
			}
		}
	}

	if len(d.ReplyTo) > 0 {
		for _, a := range d.ReplyTo {
			if _, err := mail.ParseAddress(a.Email); err == nil {
				msg = msg.ReplyTo(a.Name, a.Email)
			} else {
				return "", fmt.Errorf("invalid Reply-To address: %s", a.Email)
			}
		}
	}

	restrictedHeaders := []string{"To", "From", "Cc", "Bcc", "Reply-To", "Date", "Subject", "Content-Type", "Mime-Version"}

	if len(d.Tags) > 0 {
		msg = msg.Header("X-Tags", strings.Join(d.Tags, ", "))
		restrictedHeaders = append(restrictedHeaders, "X-Tags")
	}

	if len(d.Headers) > 0 {
		for k, v := range d.Headers {
			// check header isn't in "restricted" headers
			if tools.InArray(k, restrictedHeaders) {
				return "", fmt.Errorf("cannot overwrite header: \"%s\"", k)
			}
			msg = msg.Header(k, v)
		}
	}

	if len(d.Attachments) > 0 {
		for _, a := range d.Attachments {
			// workaround: split string because JS readAsDataURL() returns the base64 string
			// with the mime type prefix eg: data:image/png;base64,<base64String>
			parts := strings.Split(a.Content, ",")
			content := parts[len(parts)-1]
			b, err := base64.StdEncoding.DecodeString(content)
			if err != nil {
				return "", fmt.Errorf("error decoding base64 attachment \"%s\": %s", a.Filename, err.Error())
			}
			contentType := http.DetectContentType(b)
			if a.ContentType != "" {
				contentType = a.ContentType
			}
			if a.ContentID != "" {
				msg = msg.AddInline(b, contentType, a.Filename, a.ContentID)
			} else {
				msg = msg.AddAttachment(b, contentType, a.Filename)
			}
		}
	}

	part, err := msg.Build()
	if err != nil {
		return "", fmt.Errorf("error building message: %s", err.Error())
	}

	var buff bytes.Buffer

	if err := part.Encode(io.Writer(&buff)); err != nil {
		return "", fmt.Errorf("error building message: %s", err.Error())
	}

	return smtpd.SaveToDatabase(ipAddr, d.From.Email, addresses, buff.Bytes())
}
