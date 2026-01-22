package apiv1

import (
	"bytes"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/axllent/mailpit/config"
	"github.com/axllent/mailpit/internal/htmlcheck"
	"github.com/axllent/mailpit/internal/linkcheck"
	"github.com/axllent/mailpit/internal/spamassassin"
	"github.com/axllent/mailpit/internal/storage"
	"github.com/gorilla/mux"
	"github.com/jhillyerd/enmime/v2"
)

// HTMLCheck returns a summary of the HTML client support
func HTMLCheck(w http.ResponseWriter, r *http.Request) {
	// swagger:route GET /api/v1/message/{ID}/html-check other HTMLCheckParams
	//
	// # HTML check
	//
	// Returns the summary of the message HTML checker.
	//
	// The ID can be set to `latest` to return the latest message.
	//
	//	Produces:
	//	  - application/json
	//
	//	Schemes: http, https
	//
	//	Responses:
	//	  200: HTMLCheckResponse
	//    400: ErrorResponse
	//    404: NotFoundResponse

	vars := mux.Vars(r)
	id := vars["id"]

	if id == "latest" {
		var err error
		id, err = storage.LatestID(r)
		if err != nil {
			fourOFour(w)
			return
		}
	}

	raw, err := storage.GetMessageRaw(id)
	if err != nil {
		fourOFour(w)
		return
	}

	e := bytes.NewReader(raw)

	parser := enmime.NewParser(enmime.DisableCharacterDetection(true))

	msg, err := parser.ReadEnvelope(e)
	if err != nil {
		httpError(w, err.Error())
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

	w.Header().Add("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(checks); err != nil {
		httpError(w, err.Error())
	}
}

// LinkCheck returns a summary of links in the email
func LinkCheck(w http.ResponseWriter, r *http.Request) {
	// swagger:route GET /api/v1/message/{ID}/link-check other LinkCheckParams
	//
	// # Link check
	//
	// Returns the summary of the message Link checker.
	//
	// The ID can be set to `latest` to return the latest message.
	//
	//	Produces:
	//	  - application/json
	//
	//	Schemes: http, https
	//
	//	Responses:
	//	  200: LinkCheckResponse
	//    400: ErrorResponse
	//    404: NotFoundResponse

	if config.DemoMode {
		httpError(w, "this functionality has been disabled for demonstration purposes")
		return
	}

	vars := mux.Vars(r)
	id := vars["id"]

	if id == "latest" {
		var err error
		id, err = storage.LatestID(r)
		if err != nil {
			fourOFour(w)
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

	w.Header().Add("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(summary); err != nil {
		httpError(w, err.Error())
	}
}

// SpamAssassinCheck returns a summary of SpamAssassin results (if enabled)
func SpamAssassinCheck(w http.ResponseWriter, r *http.Request) {
	// swagger:route GET /api/v1/message/{ID}/sa-check other SpamAssassinCheckParams
	//
	// # SpamAssassin check
	//
	// Returns the SpamAssassin summary (if enabled) of the message.
	//
	// The ID can be set to `latest` to return the latest message.
	//
	//	Produces:
	//	  - application/json
	//
	//	Schemes: http, https
	//
	//	Responses:
	//	  200: SpamAssassinResponse
	//    400: ErrorResponse
	//    404: NotFoundResponse

	vars := mux.Vars(r)
	id := vars["id"]

	if id == "latest" {
		var err error
		id, err = storage.LatestID(r)
		if err != nil {
			w.WriteHeader(404)
			_, _ = fmt.Fprint(w, err.Error())
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

	w.Header().Add("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(summary); err != nil {
		httpError(w, err.Error())
	}
}

// GetLinks returns all links extracted from the email
func GetLinks(w http.ResponseWriter, r *http.Request) {
	// swagger:route GET /api/v1/message/{ID}/links other GetLinksParams
	//
	// # Get message links
	//
	// Returns all unique links extracted from the message HTML and text.
	//
	// The ID can be set to `latest` to return the latest message.
	//
	//	Produces:
	//	  - application/json
	//
	//	Schemes: http, https
	//
	//	Responses:
	//	  200: LinksResponse
	//    400: ErrorResponse
	//    404: NotFoundResponse

	vars := mux.Vars(r)
	id := vars["id"]

	if id == "latest" {
		var err error
		id, err = storage.LatestID(r)
		if err != nil {
			fourOFour(w)
			return
		}
	}

	msg, err := storage.GetMessage(id)
	if err != nil {
		fourOFour(w)
		return
	}

	links := linkcheck.ExtractLinks(msg)

	response := linkcheck.LinksResponse{
		Total: len(links),
		Links: links,
	}

	w.Header().Add("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		httpError(w, err.Error())
	}
}

// AttachmentDetail represents detailed attachment information including hashes
type AttachmentDetail struct {
	// Part ID
	PartID string `json:"PartID"`
	// File name
	FileName string `json:"FileName"`
	// Content type
	ContentType string `json:"ContentType"`
	// Content ID
	ContentID string `json:"ContentID"`
	// File size in bytes
	Size uint64 `json:"Size"`
	// MD5 hash
	MD5 string `json:"MD5"`
	// SHA1 hash
	SHA1 string `json:"SHA1"`
	// SHA256 hash
	SHA256 string `json:"SHA256"`
}

// AttachmentDetailsResponse contains all attachment details
type AttachmentDetailsResponse struct {
	// Attachments
	Attachments []AttachmentDetail `json:"Attachments"`
	// Inline attachments
	Inline []AttachmentDetail `json:"Inline"`
}

// GetAttachmentDetails returns detailed information about all attachments including hashes
func GetAttachmentDetails(w http.ResponseWriter, r *http.Request) {
	// swagger:route GET /api/v1/message/{ID}/attachments other GetAttachmentDetailsParams
	//
	// # Get attachment details
	//
	// Returns detailed information about all attachments including file hashes.
	//
	// The ID can be set to `latest` to return the latest message.
	//
	//	Produces:
	//	  - application/json
	//
	//	Schemes: http, https
	//
	//	Responses:
	//	  200: AttachmentDetailsResponse
	//    400: ErrorResponse
	//    404: NotFoundResponse

	vars := mux.Vars(r)
	id := vars["id"]

	if id == "latest" {
		var err error
		id, err = storage.LatestID(r)
		if err != nil {
			fourOFour(w)
			return
		}
	}

	raw, err := storage.GetMessageRaw(id)
	if err != nil {
		fourOFour(w)
		return
	}

	e := bytes.NewReader(raw)

	parser := enmime.NewParser(enmime.DisableCharacterDetection(true))

	env, err := parser.ReadEnvelope(e)
	if err != nil {
		httpError(w, err.Error())
		return
	}

	response := AttachmentDetailsResponse{
		Attachments: []AttachmentDetail{},
		Inline:      []AttachmentDetail{},
	}

	// Process regular attachments
	for _, a := range env.Attachments {
		if a.FileName != "" || a.ContentID != "" {
			detail := createAttachmentDetail(a)
			response.Attachments = append(response.Attachments, detail)
		}
	}

	// Process inline attachments
	for _, a := range env.Inlines {
		if a.FileName != "" || a.ContentID != "" {
			detail := createAttachmentDetail(a)
			response.Inline = append(response.Inline, detail)
		}
	}

	// Process other parts
	for _, a := range env.OtherParts {
		if a.FileName != "" || a.ContentID != "" {
			detail := createAttachmentDetail(a)
			response.Inline = append(response.Inline, detail)
		}
	}

	w.Header().Add("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		httpError(w, err.Error())
	}
}

// createAttachmentDetail creates an AttachmentDetail from an enmime.Part
func createAttachmentDetail(a *enmime.Part) AttachmentDetail {
	md5Hash := md5.Sum(a.Content)
	sha1Hash := sha1.Sum(a.Content)
	sha256Hash := sha256.Sum256(a.Content)

	fileName := a.FileName
	if fileName == "" {
		fileName = a.ContentID
	}

	return AttachmentDetail{
		PartID:      a.PartID,
		FileName:    fileName,
		ContentType: a.ContentType,
		ContentID:   a.ContentID,
		Size:        uint64(len(a.Content)),
		MD5:         hex.EncodeToString(md5Hash[:]),
		SHA1:        hex.EncodeToString(sha1Hash[:]),
		SHA256:      hex.EncodeToString(sha256Hash[:]),
	}
}
