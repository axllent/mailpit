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
	"github.com/axllent/mailpit/storage"
	"github.com/gorilla/mux"
)

// GetMessages returns a paginated list of messages as JSON
func GetMessages(w http.ResponseWriter, r *http.Request) {
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
	res.Count = len(messages)
	res.Total = stats.Total
	res.Unread = stats.Unread
	res.Tags = stats.Tags

	bytes, _ := json.Marshal(res)
	w.Header().Add("Content-Type", "application/json")
	_, _ = w.Write(bytes)
}

// Search returns up to 200 of the latest messages as JSON
func Search(w http.ResponseWriter, r *http.Request) {
	search := strings.TrimSpace(r.URL.Query().Get("query"))
	if search == "" {
		fourOFour(w)
		return
	}

	start, limit := getStartLimit(r)

	messages, err := storage.Search(search, start, limit)
	if err != nil {
		httpError(w, err.Error())
		return
	}

	stats := storage.StatsGet()

	var res MessagesSummary

	res.Start = 0
	res.Messages = messages
	res.Count = len(messages)
	res.Total = stats.Total
	res.Unread = stats.Unread
	res.Tags = stats.Tags

	bytes, _ := json.Marshal(res)
	w.Header().Add("Content-Type", "application/json")
	_, _ = w.Write(bytes)
}

// GetMessage (method: GET) returns the *data.Message as JSON
func GetMessage(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	id := vars["id"]

	msg, err := storage.GetMessage(id)
	if err != nil {
		httpError(w, "Message not found")
		return
	}

	bytes, _ := json.Marshal(msg)
	w.Header().Add("Content-Type", "application/json")
	_, _ = w.Write(bytes)
}

// DownloadAttachment (method: GET) returns the attachment data
func DownloadAttachment(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	id := vars["id"]
	partID := vars["partID"]

	a, err := storage.GetAttachmentPart(id, partID)
	if err != nil {
		httpError(w, err.Error())
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

// Headers (method: GET) returns the message headers as JSON
func Headers(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	id := vars["id"]

	data, err := storage.GetMessageRaw(id)
	if err != nil {
		httpError(w, err.Error())
		return
	}

	reader := bytes.NewReader(data)
	m, err := mail.ReadMessage(reader)
	if err != nil {
		httpError(w, err.Error())
		return
	}

	headers := m.Header
	bytes, _ := json.Marshal(headers)

	w.Header().Add("Content-Type", "application/json")
	_, _ = w.Write(bytes)
}

// DownloadRaw (method: GET) returns the full email source as plain text
func DownloadRaw(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	id := vars["id"]

	dl := r.FormValue("dl")

	data, err := storage.GetMessageRaw(id)
	if err != nil {
		httpError(w, err.Error())
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	if dl == "1" {
		w.Header().Set("Content-Disposition", "attachment; filename=\""+id+".eml\"")
	}
	_, _ = w.Write(data)
}

// DeleteMessages (method: DELETE) deletes all messages matching IDS.
// If no IDs are provided then all messages are deleted.
func DeleteMessages(w http.ResponseWriter, r *http.Request) {
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
func SetReadStatus(w http.ResponseWriter, r *http.Request) {
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
