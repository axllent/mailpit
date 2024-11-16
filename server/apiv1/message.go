package apiv1

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/mail"

	"github.com/axllent/mailpit/internal/storage"
	"github.com/gorilla/mux"
)

// swagger:parameters GetMessageParams
type getMessageParams struct {
	// Message database ID or "latest"
	//
	// in: path
	// required: true
	ID string
}

// GetMessage (method: GET) returns the Message as JSON
func GetMessage(w http.ResponseWriter, r *http.Request) {
	// swagger:route GET /api/v1/message/{ID} message GetMessageParams
	//
	// # Get message summary
	//
	// Returns the summary of a message, marking the message as read.
	//
	// The ID can be set to `latest` to return the latest message.
	//
	//	Produces:
	//	  - application/json
	//
	//	Schemes: http, https
	//
	//	Responses:
	//	  200: Message
	//    400: ErrorResponse
	//    404: NotFoundResponse

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

	w.Header().Add("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(msg); err != nil {
		httpError(w, err.Error())
	}
}

// swagger:parameters GetHeadersParams
type getHeadersParams struct {
	// Message database ID or "latest"
	//
	// in: path
	// required: true
	ID string
}

// Message headers
// swagger:model MessageHeadersResponse
type messageHeaders map[string][]string

// GetHeaders (method: GET) returns the message headers as JSON
func GetHeaders(w http.ResponseWriter, r *http.Request) {
	// swagger:route GET /api/v1/message/{ID}/headers message GetHeadersParams
	//
	// # Get message headers
	//
	// Returns the message headers as an array. Note that header keys are returned alphabetically.
	//
	// The ID can be set to `latest` to return the latest message headers.
	//
	//	Produces:
	//	  - application/json
	//
	//	Schemes: http, https
	//
	//	Responses:
	//	  200: MessageHeadersResponse
	//    400: ErrorResponse
	//    404: NotFoundResponse

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

	w.Header().Add("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(m.Header); err != nil {
		httpError(w, err.Error())
	}
}

// swagger:parameters AttachmentParams
type attachmentParams struct {
	// Message database ID or "latest"
	//
	// in: path
	// required: true
	ID string

	// Attachment part ID
	//
	// in: path
	// required: true
	PartID string
}

// DownloadAttachment (method: GET) returns the attachment data
func DownloadAttachment(w http.ResponseWriter, r *http.Request) {
	// swagger:route GET /api/v1/message/{ID}/part/{PartID} message AttachmentParams
	//
	// # Get message attachment
	//
	// This will return the attachment part using the appropriate Content-Type.
	//
	// The ID can be set to `latest` to reference the latest message.
	//
	//	Produces:
	//	  - application/*
	//	  - image/*
	//	  - text/*
	//
	//	Schemes: http, https
	//
	//	Responses:
	//	  200: BinaryResponse
	//    400: ErrorResponse
	//    404: NotFoundResponse

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

// swagger:parameters DownloadRawParams
type downloadRawParams struct {
	// Message database ID or "latest"
	//
	// in: path
	// required: true
	ID string
}

// DownloadRaw (method: GET) returns the full email source as plain text
func DownloadRaw(w http.ResponseWriter, r *http.Request) {
	// swagger:route GET /api/v1/message/{ID}/raw message DownloadRawParams
	//
	// # Get message source
	//
	// Returns the full email source as plain text.
	//
	// The ID can be set to `latest` to return the latest message source.
	//
	//	Produces:
	//	  - text/plain
	//
	//	Schemes: http, https
	//
	//	Responses:
	//	  200: TextResponse
	//    400: ErrorResponse
	//    404: NotFoundResponse

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
