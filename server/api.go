package server

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/axllent/mailpit/data"
	"github.com/axllent/mailpit/server/websockets"
	"github.com/axllent/mailpit/storage"
	"github.com/gorilla/mux"
)

type messagesResult struct {
	Total int            `json:"total"`
	Count int            `json:"count"`
	Start int            `json:"start"`
	Items []data.Summary `json:"items"`
}

// Return a list of available mailboxes
func apiListMailboxes(w http.ResponseWriter, _ *http.Request) {
	res, err := storage.ListMailboxes()
	if err != nil {
		httpError(w, err.Error())
		return
	}

	bytes, _ := json.Marshal(res)
	w.Header().Add("Content-Type", "application/json")
	w.Write(bytes)
}

func apiListMailbox(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	mailbox := vars["mailbox"]

	if !storage.MailboxExists(mailbox) {
		fourOFour(w)
		return
	}

	start, limit := getStartLimit(r)

	messages, err := storage.List(mailbox, start, limit)
	if err != nil {
		httpError(w, err.Error())
		return
	}

	total, err := storage.Count(mailbox)
	if err != nil {
		httpError(w, err.Error())
		return
	}

	var res messagesResult

	res.Start = start
	res.Items = messages
	res.Count = len(res.Items)
	res.Total = total

	bytes, _ := json.Marshal(res)
	w.Header().Add("Content-Type", "application/json")
	w.Write(bytes)
}

func apiSearchMailbox(w http.ResponseWriter, r *http.Request) {
	search := strings.TrimSpace(r.URL.Query().Get("query"))
	if search == "" {
		fourOFour(w)
		return
	}

	vars := mux.Vars(r)
	mailbox := vars["mailbox"]

	if !storage.MailboxExists(mailbox) {
		fourOFour(w)
		return
	}

	// we will only return up to 200 results
	start := 0
	limit := 200

	messages, err := storage.Search(mailbox, search, start, limit)
	if err != nil {
		httpError(w, err.Error())
		return
	}

	total, err := storage.Count(mailbox)
	if err != nil {
		httpError(w, err.Error())
		return
	}

	// total := limit
	// count := len(messages)
	// if total > count {
	// 	total = count
	// }

	var res messagesResult

	res.Start = start
	res.Items = messages
	res.Count = len(messages)
	res.Total = total

	bytes, _ := json.Marshal(res)
	w.Header().Add("Content-Type", "application/json")
	w.Write(bytes)
}

// Open a message
func apiOpenMessage(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	mailbox := vars["mailbox"]
	id := vars["id"]

	msg, err := storage.GetMessage(mailbox, id)
	if err != nil {
		httpError(w, err.Error())
		return
	}

	bytes, _ := json.Marshal(msg)
	w.Header().Add("Content-Type", "application/json")
	w.Write(bytes)
}

// Download/view an attachment
func apiDownloadAttachment(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	mailbox := vars["mailbox"]
	id := vars["id"]
	partID := vars["partID"]

	a, err := storage.GetAttachmentPart(mailbox, id, partID)
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
	w.Write(a.Content)
}

// View the full email source as plain text
func apiDownloadSource(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	mailbox := vars["mailbox"]
	id := vars["id"]

	dl := r.FormValue("dl")

	data, err := storage.GetMessageRaw(mailbox, id)
	if err != nil {
		httpError(w, err.Error())
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	if dl == "1" {
		w.Header().Set("Content-Disposition", "attachment; filename=\""+id+".eml\"")
	}
	w.Write(data)
}

// Delete all messages in the mailbox
func apiDeleteAll(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	mailbox := vars["mailbox"]

	err := storage.DeleteAllMessages(mailbox)
	if err != nil {
		httpError(w, err.Error())
		return
	}

	w.Header().Add("Content-Type", "text/plain")
	w.Write([]byte("ok"))
}

// Delete a single message
func apiDeleteOne(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	mailbox := vars["mailbox"]
	id := vars["id"]

	err := storage.DeleteOneMessage(mailbox, id)
	if err != nil {
		httpError(w, err.Error())
		return
	}

	w.Header().Add("Content-Type", "text/plain")
	w.Write([]byte("ok"))
}

// Mark single message as unread
func apiUnreadOne(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	mailbox := vars["mailbox"]
	id := vars["id"]

	err := storage.UnreadMessage(mailbox, id)
	if err != nil {
		httpError(w, err.Error())
		return
	}

	w.Header().Add("Content-Type", "text/plain")
	w.Write([]byte("ok"))
}

// Websocket to broadcast changes
func apiWebsocket(w http.ResponseWriter, r *http.Request) {
	websockets.ServeWs(websockets.MessageHub, w, r)
}
