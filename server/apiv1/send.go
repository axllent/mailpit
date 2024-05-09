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

	"github.com/axllent/mailpit/internal/tools"
	"github.com/axllent/mailpit/server/smtpd"
	"github.com/jhillyerd/enmime"
)

// swagger:parameters SendMessage
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
	// example: This is the text body
	Text string

	// Message body (HTML)
	// example: <p style="font-family: arial">Mailpit is <b>awesome</b>!</p>
	HTML string

	// Attachments
	Attachments []struct {
		// Base64-encoded string of the file content
		// required: true
		// example: VGhpcyBpcyBhIHBsYWluIHRleHQgYXR0YWNobWVudA==
		Content string
		// Filename
		// required: true
		// example: AttachedFile.txt
		Filename string
	}

	// Mailpit tags
	// example: ["Tag 1","Tag 2"]
	Tags []string

	// Optional headers in {"key":"value"} format
	// example: {"X-IP":"1.2.3.4"}
	Headers map[string]string
}

// SendMessageConfirmation struct
type SendMessageConfirmation struct {
	// Database ID
	// example: iAfZVVe2UQFNSG5BAjgYwa
	ID string
}

// JSONErrorMessage struct
type JSONErrorMessage struct {
	// Error message
	// example: invalid format
	Error string
}

// SendMessageHandler handles HTTP requests to send a new message
func SendMessageHandler(w http.ResponseWriter, r *http.Request) {
	// swagger:route POST /api/v1/send message SendMessage
	//
	// # Send a message
	//
	// Send a message via the HTTP API.
	//
	//	Consumes:
	//	- application/json
	//
	//	Produces:
	//	- application/json
	//
	//	Schemes: http, https
	//
	//	Responses:
	//		200: sendMessageResponse
	//		default: jsonErrorResponse

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

			mimeType := http.DetectContentType(b)
			msg = msg.AddAttachment(b, mimeType, a.Filename)
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

	return smtpd.Store(ipAddr, d.From.Email, addresses, buff.Bytes())
}
