package apiv1

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/axllent/mailpit/internal/storage"
)

// swagger:parameters GetMessagesParams
type getMessagesParams struct {
	// Pagination offset
	//
	// in: query
	// name: start
	// required: false
	// default: 0
	// type: integer
	Start int `json:"start"`

	// Limit number of results
	//
	// in: query
	// name: limit
	// required: false
	// default: 50
	// type: integer
	Limit int `json:"limit"`
}

// Summary of messages
// swagger:response MessagesSummaryResponse
type messagesSummaryResponse struct {
	// The messages summary
	// in: body
	Body MessagesSummary
}

// MessagesSummary is a summary of a list of messages
type MessagesSummary struct {
	// Total number of messages in mailbox
	Total float64 `json:"total"`

	// Total number of unread messages in mailbox
	Unread float64 `json:"unread"`

	// Legacy - now undocumented in API specs but left for backwards compatibility.
	// Removed from API documentation 2023-07-12
	// swagger:ignore
	Count float64 `json:"count"`

	// Total number of messages matching current query
	MessagesCount float64 `json:"messages_count"`

	// Pagination offset
	Start int `json:"start"`

	// All current tags
	Tags []string `json:"tags"`

	// Messages summary
	// in: body
	Messages []storage.MessageSummary `json:"messages"`
}

// GetMessages returns a paginated list of messages as JSON
func GetMessages(w http.ResponseWriter, r *http.Request) {
	// swagger:route GET /api/v1/messages messages GetMessagesParams
	//
	// # List messages
	//
	// Returns messages from the mailbox ordered from newest to oldest.
	//
	//	Produces:
	//	  - application/json
	//
	//	Schemes: http, https
	//
	//	Responses:
	//	  200: MessagesSummaryResponse
	//    400: ErrorResponse

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
	res.Count = float64(len(messages)) // legacy - now undocumented in API specs
	res.Total = stats.Total
	res.Unread = stats.Unread
	res.Tags = stats.Tags
	res.MessagesCount = stats.Total

	w.Header().Add("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(res); err != nil {
		httpError(w, err.Error())
	}
}

// swagger:parameters SetReadStatusParams
type setReadStatusParams struct {
	// in: body
	Body struct {
		// Read status
		//
		// required: false
		// default: false
		// example: true
		Read bool

		// Array of message database IDs
		//
		// required: false
		// example: ["4oRBnPtCXgAqZniRhzLNmS", "hXayS6wnCgNnt6aFTvmOF6"]
		IDs []string
	}
}

// SetReadStatus (method: PUT) will update the status to Read/Unread for all provided IDs
// If no IDs are provided then all messages are updated.
func SetReadStatus(w http.ResponseWriter, r *http.Request) {
	// swagger:route PUT /api/v1/messages messages SetReadStatusParams
	//
	// # Set read status
	//
	// If no IDs are provided then all messages are updated.
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

// swagger:parameters DeleteMessagesParams
type deleteMessagesParams struct {
	// Delete request
	// in: body
	Body struct {
		// Array of message database IDs
		//
		// required: false
		// example: ["4oRBnPtCXgAqZniRhzLNmS", "hXayS6wnCgNnt6aFTvmOF6"]
		IDs []string
	}
}

// DeleteMessages (method: DELETE) deletes all messages matching IDS.
func DeleteMessages(w http.ResponseWriter, r *http.Request) {
	// swagger:route DELETE /api/v1/messages messages DeleteMessagesParams
	//
	// # Delete messages
	//
	// Delete individual or all messages. If no IDs are provided then all messages are deleted.
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

	w.Header().Add("Content-Type", "text/plain")
	_, _ = w.Write([]byte("ok"))
}

// swagger:parameters SearchParams
type searchParams struct {
	// Search query
	//
	// in: query
	// required: true
	// type: string
	Query string `json:"query"`

	// Pagination offset
	//
	// in: query
	// required: false
	// type integer
	Start string `json:"start"`

	// Limit results
	//
	// in: query
	// required: false
	// type integer
	Limit string `json:"limit"`

	// [Timezone identifier](https://en.wikipedia.org/wiki/List_of_tz_database_time_zones) used only for `before:` & `after:` searches (eg: "Pacific/Auckland").
	//
	// in: query
	// required: false
	// type string
	TZ string `json:"tz"`
}

// Search returns the latest messages as JSON
func Search(w http.ResponseWriter, r *http.Request) {
	// swagger:route GET /api/v1/search messages SearchParams
	//
	// # Search messages
	//
	// Returns messages matching [a search](https://mailpit.axllent.org/docs/usage/search-filters/), sorted by received date (descending).
	//
	//	Produces:
	//	  - application/json
	//
	//	Schemes: http, https
	//
	//	Responses:
	//	  200: MessagesSummaryResponse
	//    400: ErrorResponse

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
	res.Count = float64(len(messages)) // legacy - now undocumented in API specs
	res.Total = stats.Total            // total messages in mailbox
	res.MessagesCount = float64(results)
	res.Unread = stats.Unread
	res.Tags = stats.Tags

	w.Header().Add("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(res); err != nil {
		httpError(w, err.Error())
	}
}

// swagger:parameters DeleteSearchParams
type deleteSearchParams struct {
	// Search query
	//
	// in: query
	// required: true
	// type: string
	Query string `json:"query"`

	// [Timezone identifier](https://en.wikipedia.org/wiki/List_of_tz_database_time_zones) used only for `before:` & `after:` searches (eg: "Pacific/Auckland").
	//
	// in: query
	// required: false
	// type string
	TZ string `json:"tz"`
}

// DeleteSearch will delete all messages matching a search
func DeleteSearch(w http.ResponseWriter, r *http.Request) {
	// swagger:route DELETE /api/v1/search messages DeleteSearchParams
	//
	// # Delete messages by search
	//
	// Delete all messages matching [a search](https://mailpit.axllent.org/docs/usage/search-filters/).
	//
	//	Produces:
	//	  - application/json
	//
	//	Schemes: http, https
	//
	//	Responses:
	//	  200: OKResponse
	//    400: ErrorResponse

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
