package handlers

import "net/http"

// Healthz is a liveness probe
func HealthzHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}
