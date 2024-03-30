package handlers

import (
	"net/http"
	"sync/atomic"

	"github.com/axllent/mailpit/internal/storage"
)

// ReadyzHandler is a ready probe that signals k8s to be able to retrieve traffic
func ReadyzHandler(isReady *atomic.Value) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		if isReady == nil || !isReady.Load().(bool) || storage.Ping() != nil {
			http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}
