// Package apiv1 handles all the API responses
package apiv1

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/araddon/dateparse"
	"github.com/axllent/mailpit/config"
	"github.com/axllent/mailpit/internal/logger"
)

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

// httpJSONError returns a basic error message (400 response) in JSON format
func httpJSONError(w http.ResponseWriter, msg string) {
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("Content-Security-Policy", config.ContentSecurityPolicy)
	w.WriteHeader(http.StatusBadRequest)
	e := JSONErrorMessage{
		Error: msg,
	}

	w.Header().Add("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(e); err != nil {
		httpError(w, err.Error())
	}
}

// Get the start and limit based on query params. Defaults to 0, 50
func getStartLimit(req *http.Request) (start int, beforeTS int64, limit int) {
	start = 0
	limit = 50
	beforeTS = 0 // timestamp

	s := req.URL.Query().Get("start")
	if n, err := strconv.Atoi(s); err == nil && n > 0 {
		start = n
	}

	l := req.URL.Query().Get("limit")
	if n, err := strconv.Atoi(l); err == nil && n > 0 {
		limit = n
	}

	b := req.URL.Query().Get("before")
	if b != "" {
		t, err := dateparse.ParseLocal(b)
		if err != nil {
			logger.Log().Warnf("ignoring invalid before: date \"%s\"", b)
		} else {
			beforeTS = t.UnixMilli()
		}
	}

	return start, beforeTS, limit
}

// GetOptions returns a blank response
func GetOptions(w http.ResponseWriter, _ *http.Request) {

	w.Header().Set("Content-Type", "text/plain")
	_, _ = w.Write([]byte(""))
}
