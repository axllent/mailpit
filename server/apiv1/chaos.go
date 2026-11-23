package apiv1

import (
	"encoding/json"
	"net/http"

	"github.com/axllent/mailpit/internal/smtpd/chaos"
)

// GetChaos returns the current Chaos triggers
func GetChaos(w http.ResponseWriter, _ *http.Request) {
	if !chaos.Enabled {
		httpError(w, "Chaos is not enabled")
		return
	}

	conf := chaos.Get()

	w.Header().Add("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(conf); err != nil {
		httpError(w, err.Error())
	}
}

// SetChaos sets the Chaos configuration.
func SetChaos(w http.ResponseWriter, r *http.Request) {
	if !chaos.Enabled {
		httpError(w, "Chaos is not enabled")
		return
	}

	data := chaos.Triggers{}

	decoder := json.NewDecoder(r.Body)

	err := decoder.Decode(&data)
	if err != nil {
		httpError(w, err.Error())
		return
	}

	if err := chaos.SetFromStruct(data); err != nil {
		httpError(w, err.Error())
		return
	}

	conf := chaos.Get()

	w.Header().Add("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(conf); err != nil {
		httpError(w, err.Error())
	}
}
