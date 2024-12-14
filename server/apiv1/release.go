package apiv1

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/mail"
	"strings"
	"time"

	"github.com/axllent/mailpit/config"
	"github.com/axllent/mailpit/internal/logger"
	"github.com/axllent/mailpit/internal/smtpd"
	"github.com/axllent/mailpit/internal/storage"
	"github.com/axllent/mailpit/internal/tools"
	"github.com/gorilla/mux"
	"github.com/lithammer/shortuuid/v4"
)

// swagger:parameters ReleaseMessageParams
type releaseMessageParams struct {
	// Message database ID
	//
	// in: path
	// description: Message database ID
	// required: true
	ID string

	// in: body
	Body struct {
		// Array of email addresses to relay the message to
		//
		// required: true
		// example: ["user1@example.com", "user2@example.com"]
		To []string
	}
}

// ReleaseMessage (method: POST) will release a message via a pre-configured external SMTP server.
func ReleaseMessage(w http.ResponseWriter, r *http.Request) {
	// swagger:route POST /api/v1/message/{ID}/release message ReleaseMessageParams
	//
	// # Release message
	//
	// Release a message via a pre-configured external SMTP server. This is only enabled if message relaying has been configured.
	//
	// The ID can be set to `latest` to reference the latest message.
	//
	//	Consumes:
	//	  - application/json
	//
	//	Produces:
	//	  - text/plain
	//
	//	Schemes: http, https
	//
	//	Responses:
	//	  200: OKResponse
	//    400: ErrorResponse
	//    404: NotFoundResponse

	if config.DemoMode {
		httpError(w, "this functionality has been disabled for demonstration purposes")
		return
	}

	vars := mux.Vars(r)

	id := vars["id"]

	msg, err := storage.GetMessageRaw(id)
	if err != nil {
		fourOFour(w)
		return
	}

	decoder := json.NewDecoder(r.Body)

	var data struct {
		To []string
	}

	if err := decoder.Decode(&data); err != nil {
		httpError(w, err.Error())
		return
	}

	blocked := []string{}
	notAllowed := []string{}

	for _, to := range data.To {
		address, err := mail.ParseAddress(to)

		if err != nil {
			httpError(w, "Invalid email address: "+to)
			return
		}

		if config.SMTPRelayConfig.AllowedRecipientsRegexp != nil && !config.SMTPRelayConfig.AllowedRecipientsRegexp.MatchString(address.Address) {
			notAllowed = append(notAllowed, to)
			continue
		}

		if config.SMTPRelayConfig.BlockedRecipientsRegexp != nil && config.SMTPRelayConfig.BlockedRecipientsRegexp.MatchString(address.Address) {
			blocked = append(blocked, to)
			continue
		}
	}

	if len(notAllowed) > 0 {
		addr := tools.Plural(len(notAllowed), "Address", "Addresses")
		httpError(w, "Failed: "+addr+" do not match the allowlist: "+strings.Join(notAllowed, ", "))
		return
	}

	if len(blocked) > 0 {
		addr := tools.Plural(len(blocked), "Address", "Addresses")
		httpError(w, "Failed: "+addr+" found on blocklist: "+strings.Join(blocked, ", "))
		return
	}

	if len(data.To) == 0 {
		httpError(w, "No valid addresses found")
		return
	}

	reader := bytes.NewReader(msg)
	m, err := mail.ReadMessage(reader)
	if err != nil {
		httpError(w, err.Error())
		return
	}

	fromAddresses, err := m.Header.AddressList("From")
	if err != nil {
		httpError(w, err.Error())
		return
	}

	if len(fromAddresses) == 0 {
		httpError(w, "No From header found")
		return
	}

	from := fromAddresses[0].Address

	// if sender is used, then change from to the sender
	if senders, err := m.Header.AddressList("Sender"); err == nil {
		from = senders[0].Address
	}

	msg, err = tools.RemoveMessageHeaders(msg, []string{"Bcc"})
	if err != nil {
		httpError(w, err.Error())
		return
	}

	// set the Return-Path and SMTP from
	if config.SMTPRelayConfig.ReturnPath != "" {
		if m.Header.Get("Return-Path") != "<"+config.SMTPRelayConfig.ReturnPath+">" {
			msg, err = tools.RemoveMessageHeaders(msg, []string{"Return-Path"})
			if err != nil {
				httpError(w, err.Error())
				return
			}
			msg = append([]byte("Return-Path: <"+config.SMTPRelayConfig.ReturnPath+">\r\n"), msg...)
		}

		from = config.SMTPRelayConfig.ReturnPath
	}

	// update message date
	msg, err = tools.UpdateMessageHeader(msg, "Date", time.Now().Format(time.RFC1123Z))
	if err != nil {
		httpError(w, err.Error())
		return
	}

	// generate unique ID
	uid := shortuuid.New() + "@mailpit"
	// update Message-Id with unique ID
	msg, err = tools.UpdateMessageHeader(msg, "Message-Id", "<"+uid+">")
	if err != nil {
		httpError(w, err.Error())
		return
	}

	if err := smtpd.Relay(from, data.To, msg); err != nil {
		logger.Log().Errorf("[smtp] error sending message: %s", err.Error())
		httpError(w, "SMTP error: "+err.Error())
		return
	}

	w.Header().Add("Content-Type", "text/plain")
	_, _ = w.Write([]byte("ok"))
}
