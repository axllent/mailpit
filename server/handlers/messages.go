package handlers

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/axllent/mailpit/config"
	"github.com/axllent/mailpit/internal/storage"
)

// RedirectToLatestMessage (method: GET) redirects the web UI to the latest message
func RedirectToLatestMessage(w http.ResponseWriter, r *http.Request) {
	var messages []storage.MessageSummary
	var err error

	search := strings.TrimSpace(r.URL.Query().Get("query"))
	if search != "" {
		messages, _, err = storage.Search(search, "", 0, 0, 1)
		if err != nil {
			httpError(w, err.Error())
			return
		}
	} else {
		messages, err = storage.List(0, 0, 1)
		if err != nil {
			httpError(w, err.Error())
			return
		}
	}

	uri := config.Webroot

	if len(messages) == 1 {
		uri, err = url.JoinPath(uri, "/view/"+messages[0].ID)
		if err != nil {
			httpError(w, err.Error())
			return
		}
	}

	http.Redirect(w, r, uri, 302)
}
