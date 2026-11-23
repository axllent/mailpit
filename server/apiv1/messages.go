package apiv1

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/axllent/mailpit/internal/storage"
	"github.com/axllent/mailpit/internal/tools"
)

// SetReadStatusRequest is the request body for setting the read status of messages
//
// openapi:model SetReadStatusRequest
type SetReadStatusRequest struct {
	// Read status
	//
	// required: false
	// default: false
	// example: true
	Read bool

	// Optional array of message database IDs
	//
	// required: false
	// default: []
	// example: ["4oRBnPtCXgAqZniRhzLNmS", "hXayS6wnCgNnt6aFTvmOF6"]
	IDs []string

	// Optional messages matching a search
	//
	// required: false
	// example: tag:backups
	Search string
}

// DeleteMessagesRequest is the request body for deleting messages
//
// openapi:model DeleteMessagesRequest
type DeleteMessagesRequest struct {
	// Array of message database IDs
	//
	// required: false
	// example: ["4oRBnPtCXgAqZniRhzLNmS", "hXayS6wnCgNnt6aFTvmOF6"]
	IDs []string
}

// ForceRequestBody is a marker which makes the OpenAPI generator document the body of this DELETE request.
func (DeleteMessagesRequest) ForceRequestBody() {}

// MessagesSummary is a summary of a list of messages
//
// openapi:model MessagesSummary
type MessagesSummary struct {
	// Total number of messages in mailbox
	// example: 42
	Total uint64 `json:"total"`

	// Total number of unread messages in mailbox
	// example: 10
	Unread uint64 `json:"unread"`

	// Legacy - now undocumented in API specs but left for backwards compatibility.
	// Removed from API documentation 2023-07-12
	// openapi:ignore
	Count uint64 `json:"count"`

	// Total number of messages matching current query
	// example: 12
	MessagesCount uint64 `json:"messages_count"`

	// Total number of unread messages matching current query
	// example: 2
	MessagesUnreadCount uint64 `json:"messages_unread"`

	// Pagination offset
	// example: 0
	Start int `json:"start"`

	// All current tags
	// example: ["Tag 1", "Tag 2"]
	Tags []string `json:"tags"`

	// Messages summary
	Messages []storage.MessageSummary `json:"messages"`
}

// GetMessages returns a paginated list of messages as JSON
func GetMessages(w http.ResponseWriter, r *http.Request) {
	start, beforeTS, limit := getStartLimit(r)

	messages, err := storage.List(start, beforeTS, limit)
	if err != nil {
		httpError(w, err.Error())
		return
	}

	stats := storage.StatsGet()

	var res MessagesSummary

	res.Start = start
	res.Messages = messages
	res.Count = uint64(len(messages)) // legacy - now undocumented in API specs
	res.Total = stats.Total
	res.Unread = stats.Unread
	res.Tags = stats.Tags
	res.MessagesCount = stats.Total
	res.MessagesUnreadCount = stats.Unread

	w.Header().Add("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(res); err != nil {
		httpError(w, err.Error())
	}
}

// SetReadStatus (method: PUT) will update the status to Read/Unread for all provided IDs.
func SetReadStatus(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)

	var data SetReadStatusRequest

	err := decoder.Decode(&data)
	if err != nil {
		httpError(w, err.Error())
		return
	}

	ids := data.IDs
	search := data.Search

	if len(ids) > 0 && search != "" {
		httpError(w, "You may specify either IDs or a search query, not both")
		return
	}

	if search != "" {
		err := storage.SetSearchReadStatus(search, r.URL.Query().Get("tz"), data.Read)
		if err != nil {
			httpError(w, err.Error())
			return
		}
	} else if len(ids) == 0 {
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
			if err := storage.MarkRead(ids); err != nil {
				httpError(w, err.Error())
				return
			}
		} else {
			if err := storage.MarkUnread(ids); err != nil {
				httpError(w, err.Error())
				return
			}
		}
	}

	w.Header().Add("Content-Type", "text/plain")
	_, _ = w.Write([]byte("ok"))
}

// DeleteMessages (method: DELETE) deletes all messages matching IDS.
func DeleteMessages(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var data DeleteMessagesRequest
	// an empty body (io.EOF) is a valid request to delete all messages
	if err := decoder.Decode(&data); err != nil && !errors.Is(err, io.EOF) {
		httpError(w, err.Error())
		return
	}

	if len(data.IDs) == 0 {
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

	w.Header().Add("Content-Type", "text/plain")
	_, _ = w.Write([]byte("ok"))
}

// Search returns the latest messages as JSON
func Search(w http.ResponseWriter, r *http.Request) {
	search := strings.TrimSpace(r.URL.Query().Get("query"))
	if search == "" {
		httpError(w, "Error: no search query")
		return
	}

	start, beforeTS, limit := getStartLimit(r)

	messages, results, err := storage.Search(search, r.URL.Query().Get("tz"), start, beforeTS, limit)
	if err != nil {
		httpError(w, err.Error())
		return
	}

	stats := storage.StatsGet()

	var res MessagesSummary

	res.Start = start
	res.Messages = messages
	res.Count = tools.SafeUint64(len(messages)) // legacy - now undocumented in API specs
	res.Total = stats.Total                     // total messages in mailbox
	res.MessagesCount = tools.SafeUint64(results)
	res.Unread = stats.Unread
	res.Tags = stats.Tags

	unread, err := storage.SearchUnreadCount(search, r.URL.Query().Get("tz"), beforeTS)
	if err != nil {
		httpError(w, err.Error())
		return
	}

	res.MessagesUnreadCount = tools.SafeUint64(unread)

	w.Header().Add("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(res); err != nil {
		httpError(w, err.Error())
	}
}

// DeleteSearch will delete all messages matching a search
func DeleteSearch(w http.ResponseWriter, r *http.Request) {
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
