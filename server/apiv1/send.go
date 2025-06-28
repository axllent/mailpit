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
	"github.com/jhillyerd/enmime/v2"
)

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
	//	  200: SendMessageResponse
	//	  400: JSONErrorResponse

	if config.DemoMode {
		httpJSONError(w, "this functionality has been disabled for demonstration purposes")
		return
	}

	decoder := json.NewDecoder(r.Body)

	data := sendMessageParams{}

	if err := decoder.Decode(&data.Body); err != nil {
		httpJSONError(w, err.Error())
		return
	}

	var httpAuthUser *string
	if user, _, ok := r.BasicAuth(); ok {
		httpAuthUser = &user
	}

	id, err := data.Send(r.RemoteAddr, httpAuthUser)

	if err != nil {
		httpJSONError(w, err.Error())
		return
	}

	w.Header().Add("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(struct{ ID string }{ID: id}); err != nil {
		httpError(w, err.Error())
	}
}

// Send will validate the message structure and attempt to send to Mailpit.
// It returns a sending summary or an error.
func (d sendMessageParams) Send(remoteAddr string, httpAuthUser *string) (string, error) {
	ip, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		return "", fmt.Errorf("error parsing request RemoteAddr: %s", err.Error())
	}

	ipAddr := &net.IPAddr{IP: net.ParseIP(ip)}

	addresses := []string{}

	msg := enmime.Builder().
		From(d.Body.From.Name, d.Body.From.Email).
		Subject(d.Body.Subject).
		Text([]byte(d.Body.Text))

	if d.Body.HTML != "" {
		msg = msg.HTML([]byte(d.Body.HTML))
	}

	if len(d.Body.To) > 0 {
		for _, a := range d.Body.To {
			if _, err := mail.ParseAddress(a.Email); err == nil {
				msg = msg.To(a.Name, a.Email)
				addresses = append(addresses, a.Email)
			} else {
				return "", fmt.Errorf("invalid To address: %s", a.Email)
			}
		}
	}

	if len(d.Body.Cc) > 0 {
		for _, a := range d.Body.Cc {
			if _, err := mail.ParseAddress(a.Email); err == nil {
				msg = msg.CC(a.Name, a.Email)
				addresses = append(addresses, a.Email)
			} else {
				return "", fmt.Errorf("invalid Cc address: %s", a.Email)
			}
		}
	}

	if len(d.Body.Bcc) > 0 {
		for _, e := range d.Body.Bcc {
			if _, err := mail.ParseAddress(e); err == nil {
				msg = msg.BCC("", e)
				addresses = append(addresses, e)
			} else {
				return "", fmt.Errorf("invalid Bcc address: %s", e)
			}
		}
	}

	if len(d.Body.ReplyTo) > 0 {
		for _, a := range d.Body.ReplyTo {
			if _, err := mail.ParseAddress(a.Email); err == nil {
				msg = msg.ReplyTo(a.Name, a.Email)
			} else {
				return "", fmt.Errorf("invalid Reply-To address: %s", a.Email)
			}
		}
	}

	restrictedHeaders := []string{"To", "From", "Cc", "Bcc", "Reply-To", "Date", "Subject", "Content-Type", "Mime-Version"}

	if len(d.Body.Tags) > 0 {
		msg = msg.Header("X-Tags", strings.Join(d.Body.Tags, ", "))
		restrictedHeaders = append(restrictedHeaders, "X-Tags")
	}

	if len(d.Body.Headers) > 0 {
		for k, v := range d.Body.Headers {
			// check header isn't in "restricted" headers
			if tools.InArray(k, restrictedHeaders) {
				return "", fmt.Errorf("cannot overwrite header: \"%s\"", k)
			}
			msg = msg.Header(k, v)
		}
	}

	if len(d.Body.Attachments) > 0 {
		for _, a := range d.Body.Attachments {
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

	return smtpd.SaveToDatabase(ipAddr, d.Body.From.Email, addresses, buff.Bytes(), httpAuthUser)
}
