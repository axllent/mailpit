// Package apiv1 handles all the API responses
package apiv1

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/mail"
	"strconv"
	"strings"

	"github.com/axllent/mailpit/config"
	"github.com/axllent/mailpit/server/smtpd"
	"github.com/axllent/mailpit/storage"
	"github.com/axllent/mailpit/utils/htmlcheck"
	"github.com/axllent/mailpit/utils/linkcheck"
	"github.com/axllent/mailpit/utils/logger"
	"github.com/axllent/mailpit/utils/tools"
	"github.com/gorilla/mux"
	uuid "github.com/satori/go.uuid"
)

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
	res.Count = len(messages) // legacy - now undocumented in API specs
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
	// Returns the latest messages matching a search.
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

	messages, results, err := storage.Search(search, start, limit)
	if err != nil {
		httpError(w, err.Error())
		return
	}

	stats := storage.StatsGet()

	var res MessagesSummary

	res.Start = start
	res.Messages = messages
	res.Count = results // legacy - now undocumented in API specs
	res.Total = stats.Total
	res.MessagesCount = results
	res.Unread = stats.Unread
	res.Tags = stats.Tags

	bytes, _ := json.Marshal(res)
	w.Header().Add("Content-Type", "application/json")
	_, _ = w.Write(bytes)
}

// GetMessage (method: GET) returns the Message as JSON
func GetMessage(w http.ResponseWriter, r *http.Request) {
	// swagger:route GET /api/v1/message/{ID} message Message
	//
	// # Get message summary
	//
	// Returns the summary of a message, marking the message as read.
	//
	//	Produces:
	//	- application/json
	//
	//	Schemes: http, https
	//
	//	Parameters:
	//	  + name: ID
	//	    in: path
	//	    description: Database ID
	//	    required: true
	//	    type: string
	//
	//	Responses:
	//		200: Message
	//		default: ErrorResponse

	vars := mux.Vars(r)

	id := vars["id"]

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
	//	    description: Database ID
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
	//	Produces:
	//	- application/json
	//
	//	Schemes: http, https
	//
	//	Parameters:
	//	  + name: ID
	//	    in: path
	//	    description: Database ID
	//	    required: true
	//	    type: string
	//
	//	Responses:
	//	  200: MessageHeaders
	//	  default: ErrorResponse

	vars := mux.Vars(r)

	id := vars["id"]

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

	bytes, _ := json.Marshal(m.Header)

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
	//	Produces:
	//	- text/plain
	//
	//	Schemes: http, https
	//
	//	Parameters:
	//	  + name: ID
	//	    in: path
	//	    description: Database ID
	//	    required: true
	//	    type: string
	//
	//	Responses:
	//		200: TextResponse
	//		default: ErrorResponse

	vars := mux.Vars(r)

	id := vars["id"]

	dl := r.FormValue("dl")

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
	// swagger:route DELETE /api/v1/messages messages Delete
	//
	// # Delete messages
	//
	// If no IDs are provided then all messages are deleted.
	//
	//	Consumes:
	//	- application/json
	//
	//	Produces:
	//	- text/plain
	//
	//	Schemes: http, https
	//
	//	Parameters:
	//	  + name: ids
	//	    in: body
	//	    description: Database IDs to delete
	//	    required: false
	//	    type: DeleteRequest
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
		for _, id := range data.IDs {
			if err := storage.DeleteOneMessage(id); err != nil {
				httpError(w, err.Error())
				return
			}
		}
	}

	w.Header().Add("Content-Type", "text/plain")
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
	//	Parameters:
	//	  + name: ids
	//	    in: body
	//	    description: Database IDs to update
	//	    required: false
	//	    type: SetReadStatusRequest
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

// SetTags (method: PUT) will set the tags for all provided IDs
func SetTags(w http.ResponseWriter, r *http.Request) {
	// swagger:route PUT /api/v1/tags tags SetTags
	//
	// # Set message tags
	//
	// To remove all tags from a message, pass an empty tags array.
	//
	//	Consumes:
	//	- application/json
	//
	//	Produces:
	//	- text/plain
	//
	//	Schemes: http, https
	//
	//	Parameters:
	//	  + name: ids
	//	    in: body
	//	    description: Database IDs to update
	//	    required: true
	//	    type: SetTagsRequest
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
			if err := storage.SetTags(id, data.Tags); err != nil {
				httpError(w, err.Error())
				return
			}
		}
	}

	w.Header().Add("Content-Type", "text/plain")
	_, _ = w.Write([]byte("ok"))
}

// ReleaseMessage (method: POST) will release a message via a pre-configured external SMTP server.
// If no IDs are provided then all messages are updated.
func ReleaseMessage(w http.ResponseWriter, r *http.Request) {
	// swagger:route POST /api/v1/message/{ID}/release message Release
	//
	// # Release message
	//
	// Release a message via a pre-configured external SMTP server..
	//
	//	Consumes:
	//	- application/json
	//
	//	Produces:
	//	- text/plain
	//
	//	Schemes: http, https
	//
	//	Parameters:
	//	  + name: ID
	//	    in: path
	//	    description: Database ID
	//	    required: true
	//	    type: string
	//		+ name: to
	//	    in: body
	//	    description: Array of email addresses to release message to
	//	    required: true
	//	    type: ReleaseMessageRequest
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

	data := releaseMessageRequest{}

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

		if config.SMTPRelayConfig.RecipientAllowlistRegexp != nil && !config.SMTPRelayConfig.RecipientAllowlistRegexp.MatchString(address.Address) {
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

	msg, err = tools.RemoveMessageHeaders(msg, []string{"Bcc", "Message-Id"})
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

	// generate unique ID
	uid := uuid.NewV4().String() + "@mailpit"
	// add unique ID
	msg = append([]byte("Message-Id: <"+uid+">\r\n"), msg...)

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
	// swagger:route GET /api/v1/message/{ID}/html-check Other HTMLCheckResponse
	//
	// # HTML check (beta)
	//
	// Returns the summary of the message HTML checker.
	//
	// NOTE: This feature is currently in beta and is documented for reference only.
	// Please do not integrate with it (yet) as there may be changes.
	//
	//	Produces:
	//	- application/json
	//
	//	Schemes: http, https
	//
	//	Parameters:
	//	  + name: ID
	//	    in: path
	//	    description: Database ID
	//	    required: true
	//	    type: string
	//
	//	Responses:
	//		200: HTMLCheckResponse
	//		default: ErrorResponse

	vars := mux.Vars(r)
	id := vars["id"]

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
	// swagger:route GET /api/v1/message/{ID}/link-check Other LinkCheckResponse
	//
	// # Link check (beta)
	//
	// Returns the summary of the message Link checker.
	//
	// NOTE: This feature is currently in beta and is documented for reference only.
	// Please do not integrate with it (yet) as there may be changes.
	//
	//	Produces:
	//	- application/json
	//
	//	Schemes: http, https
	//
	//	Parameters:
	//	  + name: ID
	//	    in: path
	//	    description: Database ID
	//	    required: true
	//	    type: string
	//	  + name: follow
	//	    in: query
	//	    description: Follow redirects
	//	    required: false
	//	    type: boolean
	//	    default: false
	//
	//	Responses:
	//		200: LinkCheckResponse
	//		default: ErrorResponse

	vars := mux.Vars(r)
	id := vars["id"]

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
func GetOptions(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "text/plain")
	_, _ = w.Write([]byte(""))
}
