// Package apiv1 handles all the API responses
package apiv1

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/mail"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/axllent/mailpit/config"
	"github.com/axllent/mailpit/internal/htmlcheck"
	"github.com/axllent/mailpit/internal/linkcheck"
	"github.com/axllent/mailpit/internal/logger"
	"github.com/axllent/mailpit/internal/spamassassin"
	"github.com/axllent/mailpit/internal/storage"
	"github.com/axllent/mailpit/internal/tools"
	"github.com/axllent/mailpit/server/smtpd"
	"github.com/gorilla/mux"
	"github.com/lithammer/shortuuid/v4"
)

// SendMessage sends a new message
func SendMessage(w http.ResponseWriter, r *http.Request) {
	// swagger:route POST /api/v1/messages message SendMessage
	//
	// # Send message
	//
	// Sends a message to the mailbox
	//
	//	Consumes:
	//	- application/json
	//
	//	Produces:
	//	- text/plain
	//
	//	Schemes: http, https
	//
	//	Responses:
	//		200: OKResponse
	//		default: ErrorResponse

	decoder := json.NewDecoder(r.Body)

	data := sendMessageRequestBody{}

	if err := decoder.Decode(&data); err != nil {
		httpError(w, err.Error())
		return
	}

	fromEmail := data.From.Email
	if fromEmail == "" {
		httpError(w, "No valid sender address found")
		return
	}
	_, err := mail.ParseAddress(fromEmail)
	if err != nil {
		httpError(w, "Invalid sender email address: "+fromEmail)
		return
	}
	from := fromEmail
	if data.From.Name != "" {
		from = fmt.Sprintf("%s <%s>", data.From.Name, data.From.Email)
	}

	if len(data.To) == 0 || data.To[0].Email == "" {
		httpError(w, "No valid recipient addresses found")
		return
	}
	toEmails := []string{}
	toRecipients := []string{}
	for _, to := range data.To {
		if _, err := mail.ParseAddress(to.Email); err != nil {
			httpError(w, "Invalid 'to' recipient email address: "+to.Email)
			return
		}
		toEmails = append(toEmails, to.Email)
		if to.Name != "" {
			toRecipients = append(toRecipients, fmt.Sprintf("%s <%s>", to.Name, to.Email))
		} else {
			toRecipients = append(toRecipients, to.Email)
		}
	}

	ccRecipients := []string{}
	for _, cc := range data.CC {
		if _, err := mail.ParseAddress(cc.Email); err != nil {
			httpError(w, "Invalid 'cc' email address: "+cc.Email)
			return
		}
		if cc.Name != "" {
			ccRecipients = append(ccRecipients, fmt.Sprintf("%s <%s>", cc.Name, cc.Email))
		} else {
			ccRecipients = append(ccRecipients, cc.Email)
		}
	}

	bccRecipients := []string{}
	for _, bcc := range data.BCC {
		if _, err := mail.ParseAddress(bcc.Email); err != nil {
			httpError(w, "Invalid 'bcc' email address: "+bcc.Email)
			return
		}
		if bcc.Name != "" {
			bccRecipients = append(bccRecipients, fmt.Sprintf("%s <%s>", bcc.Name, bcc.Email))
		} else {
			bccRecipients = append(bccRecipients, bcc.Email)
		}
	}

	replyToRecipients := []string{}
	for _, replyTo := range data.ReplyTo {
		if _, err := mail.ParseAddress(replyTo.Email); err != nil {
			httpError(w, "Invalid 'replyTo' email address: "+replyTo.Email)
			return
		}
		if replyTo.Name != "" {
			replyToRecipients = append(replyToRecipients, fmt.Sprintf("%s <%s>", replyTo.Name, replyTo.Email))
		} else {
			replyToRecipients = append(replyToRecipients, replyTo.Email)
		}
	}

	if data.Subject == "" {
		httpError(w, "No valid subject found")
		return
	}

	if data.HTML == "" {
		httpError(w, "No valid HTML body found")
		return
	}

	if data.Text == "" {
		httpError(w, "No valid text body found")
		return
	}

	clientIP, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		fmt.Fprintf(w, "Error parsing request RemoteAddr: %s", err)
		return
	}

	templateData := map[string]interface{}{
		"From":        from,
		"To":          strings.Join(toRecipients, ", "),
		"CC":          strings.Join(ccRecipients, ", "),
		"BCC":         strings.Join(bccRecipients, ", "),
		"ReplyTo":     strings.Join(replyToRecipients, ", "),
		"Tags":        data.Tags,
		"Subject":     data.Subject,
		"Headers":     data.Headers,
		"Text":        data.Text,
		"HTML":        data.HTML,
		"Attachments": data.Attachments,
	}
	messageTemplate, err := template.New("message").Parse(
		"" +
			"Content-Type: multipart/mixed; boundary=boundary\r\n" +
			"{{range $key, $value := .Headers}}{{$key}}: {{$value}}\r\n{{end}}" +
			"X-Tags:{{range $index, $tag := .Tags}} {{$tag}}\r\n{{end}}" +
			"From: {{.From}}\r\n" +
			"To: {{.To}}\r\n" +
			"{{if .CC}}Cc: {{.CC}}\r\n{{end}}" +
			"{{if .BCC}}Bcc: {{.BCC}}\r\n{{end}}" +
			"{{if .ReplyTo}}Reply-To: {{.ReplyTo}}\r\n{{end}}" +
			"Subject: {{.Subject}}\r\n" +
			"\r\n" +
			"{{if .Text }}--boundary\r\n" +
			"Content-Type: text/plain; charset=\"UTF-8\"\r\n\r\n" +
			"{{.Text}}" +
			"\r\n{{ end }}" +
			"{{if .HTML }}--boundary\r\n" +
			"Content-Type: text/html; charset=\"UTF-8\"\r\n\r\n" +
			"{{.HTML}}" +
			"\r\n{{ end}}" +
			"{{range .Attachments}}" +
			"--boundary\r\n" +
			"Content-Type: application/octet-stream\r\n" +
			"Content-Disposition: attachment; filename=\"{{.Filename}}\"\r\n" +
			"Content-Transfer-Encoding: base64\r\n\r\n" +
			"{{.Content}}\r\n" +
			"{{end}}" +
			"--boundary--\r\n",
	)
	if err != nil {
		fmt.Fprintf(w, "Error initialising email body template: %s", err)
		return
	}
	message := bytes.NewBuffer(nil)
	messageTemplate.Execute(message, templateData)
	msg := message.Bytes()

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

	if err := smtpd.MailHandler(&net.IPAddr{IP: net.ParseIP(clientIP)}, fromEmail, toEmails, msg); err != nil {
		logger.Log().Errorf("[smtp] error sending message: %s", err.Error())
		httpError(w, "SMTP error: "+err.Error())
		return
	}

	w.Header().Add("Content-Type", "text/plain")
	_, _ = w.Write([]byte("ok"))
}

// GetMessages returns a paginated list of messages as JSON
func GetMessages(w http.ResponseWriter, r *http.Request) {
	// swagger:route GET /api/v1/messages messages GetMessages
	//
	// # List messages
	//
	// Returns messages from the mailbox ordered from newest to oldest.
	//
	//	Produces:
	//	- application/json
	//
	//	Schemes: http, https
	//
	//	Parameters:
	//	  + name: start
	//	    in: query
	//	    description: Pagination offset
	//	    required: false
	//	    type: integer
	//	    default: 0
	//	  + name: limit
	//	    in: query
	//	    description: Limit results
	//	    required: false
	//	    type: integer
	//	    default: 50
	//
	//	Responses:
	//		200: MessagesSummaryResponse
	//		default: ErrorResponse
	start, limit := getStartLimit(r)

	messages, err := storage.List(start, limit)
	if err != nil {
		httpError(w, err.Error())
		return
	}

	stats := storage.StatsGet()

	var res MessagesSummary

	res.Start = start
	res.Messages = messages
	res.Count = float64(len(messages)) // legacy - now undocumented in API specs
	res.Total = stats.Total
	res.Unread = stats.Unread
	res.Tags = stats.Tags
	res.MessagesCount = stats.Total

	bytes, _ := json.Marshal(res)
	w.Header().Add("Content-Type", "application/json")
	_, _ = w.Write(bytes)
}

// Search returns the latest messages as JSON
func Search(w http.ResponseWriter, r *http.Request) {
	// swagger:route GET /api/v1/search messages MessagesSummary
	//
	// # Search messages
	//
	// Returns messages matching [a search](https://mailpit.axllent.org/docs/usage/search-filters/), sorted by received date (descending).
	//
	//	Produces:
	//	- application/json
	//
	//	Schemes: http, https
	//
	//	Parameters:
	//	  + name: query
	//	    in: query
	//	    description: Search query
	//	    required: true
	//	    type: string
	//	  + name: start
	//	    in: query
	//	    description: Pagination offset
	//	    required: false
	//	    type: integer
	//	    default: 0
	//	  + name: limit
	//	    in: query
	//	    description: Limit results
	//	    required: false
	//	    type: integer
	//	    default: 50
	//	  + name: tz
	//	    in: query
	//	    description: [Timezone identifier](https://en.wikipedia.org/wiki/List_of_tz_database_time_zones) used specifically for `before:` & `after:` searches (eg: "Pacific/Auckland").
	//	    required: false
	//	    type: string
	//
	//	Responses:
	//		200: MessagesSummaryResponse
	//		default: ErrorResponse
	search := strings.TrimSpace(r.URL.Query().Get("query"))
	if search == "" {
		httpError(w, "Error: no search query")
		return
	}

	start, limit := getStartLimit(r)

	messages, results, err := storage.Search(search, r.URL.Query().Get("tz"), start, limit)
	if err != nil {
		httpError(w, err.Error())
		return
	}

	stats := storage.StatsGet()

	var res MessagesSummary

	res.Start = start
	res.Messages = messages
	res.Count = float64(len(messages)) // legacy - now undocumented in API specs
	res.Total = stats.Total            // total messages in mailbox
	res.MessagesCount = float64(results)
	res.Unread = stats.Unread
	res.Tags = stats.Tags

	bytes, _ := json.Marshal(res)
	w.Header().Add("Content-Type", "application/json")
	_, _ = w.Write(bytes)
}

// DeleteSearch will delete all messages matching a search
func DeleteSearch(w http.ResponseWriter, r *http.Request) {
	// swagger:route DELETE /api/v1/search messages DeleteSearch
	//
	// # Delete messages by search
	//
	// Delete all messages matching [a search](https://mailpit.axllent.org/docs/usage/search-filters/).
	//
	//	Produces:
	//	- application/json
	//
	//	Schemes: http, https
	//
	//	Parameters:
	//	  + name: query
	//	    in: query
	//	    description: Search query
	//	    required: true
	//	    type: string
	//	  + name: tz
	//	    in: query
	//	    description: [Timezone identifier](https://en.wikipedia.org/wiki/List_of_tz_database_time_zones) used specifically for `before:` & `after:` searches (eg: "Pacific/Auckland").
	//	    required: false
	//	    type: string
	//
	//	Responses:
	//		200: OKResponse
	//		default: ErrorResponse
	search := strings.TrimSpace(r.URL.Query().Get("query"))
	if search == "" {
		httpError(w, "Error: no search query")
		return
	}

	if err := storage.DeleteSearch(search, r.URL.Query().Get("tz")); err != nil {
		httpError(w, err.Error())
		return
	}

	w.Header().Add("Content-Type", "text/plain")
	_, _ = w.Write([]byte("ok"))
}

// GetMessage (method: GET) returns the Message as JSON
func GetMessage(w http.ResponseWriter, r *http.Request) {
	// swagger:route GET /api/v1/message/{ID} message Message
	//
	// # Get message summary
	//
	// Returns the summary of a message, marking the message as read.
	//
	// The ID can be set to `latest` to return the latest message.
	//
	//	Produces:
	//	- application/json
	//
	//	Schemes: http, https
	//
	//	Parameters:
	//	  + name: ID
	//	    in: path
	//	    description: Message database ID or "latest"
	//	    required: true
	//	    type: string
	//
	//	Responses:
	//		200: Message
	//		default: ErrorResponse

	vars := mux.Vars(r)

	id := vars["id"]

	if id == "latest" {
		var err error
		id, err = storage.LatestID(r)
		if err != nil {
			w.WriteHeader(404)
			fmt.Fprint(w, err.Error())
			return
		}
	}

	msg, err := storage.GetMessage(id)
	if err != nil {
		fourOFour(w)
		return
	}

	bytes, _ := json.Marshal(msg)
	w.Header().Add("Content-Type", "application/json")
	_, _ = w.Write(bytes)
}

// DownloadAttachment (method: GET) returns the attachment data
func DownloadAttachment(w http.ResponseWriter, r *http.Request) {
	// swagger:route GET /api/v1/message/{ID}/part/{PartID} message Attachment
	//
	// # Get message attachment
	//
	// This will return the attachment part using the appropriate Content-Type.
	//
	//	Produces:
	//	- application/*
	//	- image/*
	//	- text/*
	//
	//	Schemes: http, https
	//
	//	Parameters:
	//	  + name: ID
	//	    in: path
	//	    description: Message database ID
	//	    required: true
	//	    type: string
	//	  + name: PartID
	//	    in: path
	//	    description: Attachment part ID
	//	    required: true
	//	    type: string
	//
	//	Responses:
	//		200: BinaryResponse
	//		default: ErrorResponse

	vars := mux.Vars(r)

	id := vars["id"]
	partID := vars["partID"]

	a, err := storage.GetAttachmentPart(id, partID)
	if err != nil {
		fourOFour(w)
		return
	}
	fileName := a.FileName
	if fileName == "" {
		fileName = a.ContentID
	}

	w.Header().Add("Content-Type", a.ContentType)
	w.Header().Set("Content-Disposition", "filename=\""+fileName+"\"")
	_, _ = w.Write(a.Content)
}

// GetHeaders (method: GET) returns the message headers as JSON
func GetHeaders(w http.ResponseWriter, r *http.Request) {
	// swagger:route GET /api/v1/message/{ID}/headers message Headers
	//
	// # Get message headers
	//
	// Returns the message headers as an array.
	//
	// The ID can be set to `latest` to return the latest message headers.
	//
	//	Produces:
	//	- application/json
	//
	//	Schemes: http, https
	//
	//	Parameters:
	//	  + name: ID
	//	    in: path
	//	    description: Message database ID or "latest"
	//	    required: true
	//	    type: string
	//
	//	Responses:
	//	  200: MessageHeaders
	//	  default: ErrorResponse

	vars := mux.Vars(r)

	id := vars["id"]

	if id == "latest" {
		var err error
		id, err = storage.LatestID(r)
		if err != nil {
			w.WriteHeader(404)
			fmt.Fprint(w, err.Error())
			return
		}
	}

	data, err := storage.GetMessageRaw(id)
	if err != nil {
		fourOFour(w)
		return
	}

	reader := bytes.NewReader(data)
	m, err := mail.ReadMessage(reader)
	if err != nil {
		httpError(w, err.Error())
		return
	}

	bytes, err := json.Marshal(m.Header)
	if err != nil {
		httpError(w, err.Error())
		return
	}

	w.Header().Add("Content-Type", "application/json")
	_, _ = w.Write(bytes)
}

// DownloadRaw (method: GET) returns the full email source as plain text
func DownloadRaw(w http.ResponseWriter, r *http.Request) {
	// swagger:route GET /api/v1/message/{ID}/raw message Raw
	//
	// # Get message source
	//
	// Returns the full email source as plain text.
	//
	// The ID can be set to `latest` to return the latest message source.
	//
	//	Produces:
	//	- text/plain
	//
	//	Schemes: http, https
	//
	//	Parameters:
	//	  + name: ID
	//	    in: path
	//	    description: Message database ID or "latest"
	//	    required: true
	//	    type: string
	//
	//	Responses:
	//		200: TextResponse
	//		default: ErrorResponse

	vars := mux.Vars(r)

	id := vars["id"]
	dl := r.FormValue("dl")

	if id == "latest" {
		var err error
		id, err = storage.LatestID(r)
		if err != nil {
			w.WriteHeader(404)
			fmt.Fprint(w, err.Error())
			return
		}
	}

	data, err := storage.GetMessageRaw(id)
	if err != nil {
		fourOFour(w)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	if dl == "1" {
		w.Header().Set("Content-Disposition", "attachment; filename=\""+id+".eml\"")
	}
	_, _ = w.Write(data)
}

// DeleteMessages (method: DELETE) deletes all messages matching IDS.
func DeleteMessages(w http.ResponseWriter, r *http.Request) {
	// swagger:route DELETE /api/v1/messages messages DeleteMessages
	//
	// # Delete messages
	//
	// Delete individual or all messages. If no IDs are provided then all messages are deleted.
	//
	//	Consumes:
	//	- application/json
	//
	//	Produces:
	//	- text/plain
	//
	//	Schemes: http, https
	//
	//	Responses:
	//		200: OKResponse
	//		default: ErrorResponse

	decoder := json.NewDecoder(r.Body)
	var data struct {
		IDs []string
	}
	err := decoder.Decode(&data)
	if err != nil || len(data.IDs) == 0 {
		if err := storage.DeleteAllMessages(); err != nil {
			httpError(w, err.Error())
			return
		}
	} else {
		if err := storage.DeleteMessages(data.IDs); err != nil {
			httpError(w, err.Error())
			return
		}
	}

	w.Header().Add("Content-Type", "application/plain")
	_, _ = w.Write([]byte("ok"))
}

// SetReadStatus (method: PUT) will update the status to Read/Unread for all provided IDs
// If no IDs are provided then all messages are updated.
func SetReadStatus(w http.ResponseWriter, r *http.Request) {
	// swagger:route PUT /api/v1/messages messages SetReadStatus
	//
	// # Set read status
	//
	// If no IDs are provided then all messages are updated.
	//
	//	Consumes:
	//	- application/json
	//
	//	Produces:
	//	- text/plain
	//
	//	Schemes: http, https
	//
	//	Responses:
	//		200: OKResponse
	//		default: ErrorResponse

	decoder := json.NewDecoder(r.Body)

	var data struct {
		Read bool
		IDs  []string
	}

	err := decoder.Decode(&data)
	if err != nil {
		httpError(w, err.Error())
		return
	}

	ids := data.IDs

	if len(ids) == 0 {
		if data.Read {
			err := storage.MarkAllRead()
			if err != nil {
				httpError(w, err.Error())
				return
			}
		} else {
			err := storage.MarkAllUnread()
			if err != nil {
				httpError(w, err.Error())
				return
			}
		}
	} else {
		if data.Read {
			for _, id := range ids {
				if err := storage.MarkRead(id); err != nil {
					httpError(w, err.Error())
					return
				}
			}
		} else {
			for _, id := range ids {
				if err := storage.MarkUnread(id); err != nil {
					httpError(w, err.Error())
					return
				}
			}
		}
	}

	w.Header().Add("Content-Type", "text/plain")
	_, _ = w.Write([]byte("ok"))
}

// GetAllTags (method: GET) will get all tags currently in use
func GetAllTags(w http.ResponseWriter, _ *http.Request) {
	// swagger:route GET /api/v1/tags tags GetAllTags
	//
	// # Get all current tags
	//
	// Returns a JSON array of all unique message tags.
	//
	//	Produces:
	//	- application/json
	//
	//	Schemes: http, https
	//
	//	Responses:
	//		200: ArrayResponse
	//		default: ErrorResponse

	tags := storage.GetAllTags()

	data, err := json.Marshal(tags)
	if err != nil {
		httpError(w, err.Error())
		return
	}

	w.Header().Add("Content-Type", "application/json")
	_, _ = w.Write(data)
}

// SetMessageTags (method: PUT) will set the tags for all provided IDs
func SetMessageTags(w http.ResponseWriter, r *http.Request) {
	// swagger:route PUT /api/v1/tags tags SetTags
	//
	// # Set message tags
	//
	// This will overwrite any existing tags for selected message database IDs. To remove all tags from a message, pass an empty tags array.
	//
	//	Consumes:
	//	- application/json
	//
	//	Produces:
	//	- text/plain
	//
	//	Schemes: http, https
	//
	//	Responses:
	//		200: OKResponse
	//		default: ErrorResponse

	decoder := json.NewDecoder(r.Body)

	var data struct {
		Tags []string
		IDs  []string
	}

	err := decoder.Decode(&data)
	if err != nil {
		httpError(w, err.Error())
		return
	}

	ids := data.IDs

	if len(ids) > 0 {
		for _, id := range ids {
			if err := storage.SetMessageTags(id, data.Tags); err != nil {
				httpError(w, err.Error())
				return
			}
		}
	}

	w.Header().Add("Content-Type", "text/plain")
	_, _ = w.Write([]byte("ok"))
}

// ReleaseMessage (method: POST) will release a message via a pre-configured external SMTP server.
func ReleaseMessage(w http.ResponseWriter, r *http.Request) {
	// swagger:route POST /api/v1/message/{ID}/release message ReleaseMessage
	//
	// # Release message
	//
	// Release a message via a pre-configured external SMTP server. This is only enabled if message relaying has been configured.
	//
	//	Consumes:
	//	- application/json
	//
	//	Produces:
	//	- text/plain
	//
	//	Schemes: http, https
	//
	//	Responses:
	//		200: OKResponse
	//		default: ErrorResponse

	vars := mux.Vars(r)

	id := vars["id"]

	msg, err := storage.GetMessageRaw(id)
	if err != nil {
		fourOFour(w)
		return
	}

	decoder := json.NewDecoder(r.Body)

	data := releaseMessageRequestBody{}

	if err := decoder.Decode(&data); err != nil {
		httpError(w, err.Error())
		return
	}

	tos := data.To
	if len(tos) == 0 {
		httpError(w, "No valid addresses found")
		return
	}

	for _, to := range tos {
		address, err := mail.ParseAddress(to)

		if err != nil {
			httpError(w, "Invalid email address: "+to)
			return
		}

		if config.SMTPRelayConfig.AllowedRecipientsRegexp != nil && !config.SMTPRelayConfig.AllowedRecipientsRegexp.MatchString(address.Address) {
			httpError(w, "Mail address does not match allowlist: "+to)
			return
		}
	}

	reader := bytes.NewReader(msg)
	m, err := mail.ReadMessage(reader)
	if err != nil {
		httpError(w, err.Error())
		return
	}

	froms, err := m.Header.AddressList("From")
	if err != nil {
		httpError(w, err.Error())
		return
	}

	from := froms[0].Address

	// if sender is used, then change from to the sender
	if senders, err := m.Header.AddressList("Sender"); err == nil {
		from = senders[0].Address
	}

	msg, err = tools.RemoveMessageHeaders(msg, []string{"Bcc"})
	if err != nil {
		httpError(w, err.Error())
		return
	}

	// set the Return-Path and SMTP mfrom
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

	if err := smtpd.Send(from, tos, msg); err != nil {
		logger.Log().Errorf("[smtp] error sending message: %s", err.Error())
		httpError(w, "SMTP error: "+err.Error())
		return
	}

	w.Header().Add("Content-Type", "text/plain")
	_, _ = w.Write([]byte("ok"))
}

// HTMLCheck returns a summary of the HTML client support
func HTMLCheck(w http.ResponseWriter, r *http.Request) {
	// swagger:route GET /api/v1/message/{ID}/html-check Other HTMLCheck
	//
	// # HTML check (beta)
	//
	// Returns the summary of the message HTML checker.
	//
	//	Produces:
	//	- application/json
	//
	//	Schemes: http, https
	//
	//	Responses:
	//		200: HTMLCheckResponse
	//		default: ErrorResponse

	vars := mux.Vars(r)
	id := vars["id"]

	if id == "latest" {
		var err error
		id, err = storage.LatestID(r)
		if err != nil {
			w.WriteHeader(404)
			fmt.Fprint(w, err.Error())
			return
		}
	}

	msg, err := storage.GetMessage(id)
	if err != nil {
		fourOFour(w)
		return
	}

	if msg.HTML == "" {
		httpError(w, "message does not contain HTML")
		return
	}

	checks, err := htmlcheck.RunTests(msg.HTML)
	if err != nil {
		httpError(w, err.Error())
		return
	}

	bytes, _ := json.Marshal(checks)
	w.Header().Add("Content-Type", "application/json")
	_, _ = w.Write(bytes)
}

// LinkCheck returns a summary of links in the email
func LinkCheck(w http.ResponseWriter, r *http.Request) {
	// swagger:route GET /api/v1/message/{ID}/link-check Other LinkCheck
	//
	// # Link check (beta)
	//
	// Returns the summary of the message Link checker.
	//
	//	Produces:
	//	- application/json
	//
	//	Schemes: http, https
	//
	//	Responses:
	//		200: LinkCheckResponse
	//		default: ErrorResponse

	vars := mux.Vars(r)
	id := vars["id"]

	if id == "latest" {
		var err error
		id, err = storage.LatestID(r)
		if err != nil {
			w.WriteHeader(404)
			fmt.Fprint(w, err.Error())
			return
		}
	}

	msg, err := storage.GetMessage(id)
	if err != nil {
		fourOFour(w)
		return
	}

	f := r.URL.Query().Get("follow")
	followRedirects := f == "true" || f == "1"

	summary, err := linkcheck.RunTests(msg, followRedirects)
	if err != nil {
		httpError(w, err.Error())
		return
	}

	bytes, _ := json.Marshal(summary)
	w.Header().Add("Content-Type", "application/json")
	_, _ = w.Write(bytes)
}

// SpamAssassinCheck returns a summary of SpamAssassin results (if enabled)
func SpamAssassinCheck(w http.ResponseWriter, r *http.Request) {
	// swagger:route GET /api/v1/message/{ID}/sa-check Other SpamAssassinCheck
	//
	// # SpamAssassin check (beta)
	//
	// Returns the SpamAssassin (if enabled) summary of the message.
	//
	// NOTE: This feature is currently in beta and is documented for reference only.
	// Please do not integrate with it (yet) as there may be changes.
	//
	//	Produces:
	//	- application/json
	//
	//	Schemes: http, https
	//
	//	Responses:
	//		200: SpamAssassinResponse
	//		default: ErrorResponse

	vars := mux.Vars(r)
	id := vars["id"]

	if id == "latest" {
		var err error
		id, err = storage.LatestID(r)
		if err != nil {
			w.WriteHeader(404)
			fmt.Fprint(w, err.Error())
			return
		}
	}

	msg, err := storage.GetMessageRaw(id)
	if err != nil {
		fourOFour(w)
		return
	}

	summary, err := spamassassin.Check(msg)
	if err != nil {
		httpError(w, err.Error())
		return
	}

	bytes, _ := json.Marshal(summary)
	w.Header().Add("Content-Type", "application/json")
	_, _ = w.Write(bytes)
}

// FourOFour returns a basic 404 message
func fourOFour(w http.ResponseWriter) {
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("Content-Security-Policy", config.ContentSecurityPolicy)
	w.WriteHeader(http.StatusNotFound)
	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprint(w, "404 page not found")
}

// HTTPError returns a basic error message (400 response)
func httpError(w http.ResponseWriter, msg string) {
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("Content-Security-Policy", config.ContentSecurityPolicy)
	w.WriteHeader(http.StatusBadRequest)
	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprint(w, msg)
}

// Get the start and limit based on query params. Defaults to 0, 50
func getStartLimit(req *http.Request) (start int, limit int) {
	start = 0
	limit = 50

	s := req.URL.Query().Get("start")
	if n, err := strconv.Atoi(s); err == nil && n > 0 {
		start = n
	}

	l := req.URL.Query().Get("limit")
	if n, err := strconv.Atoi(l); err == nil && n > 0 {
		limit = n
	}

	return start, limit
}

// GetOptions returns a blank response
func GetOptions(w http.ResponseWriter, _ *http.Request) {

	w.Header().Set("Content-Type", "text/plain")
	_, _ = w.Write([]byte(""))
}
