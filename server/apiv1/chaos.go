package apiv1

import (
	"encoding/json"
	"net/http"

	"github.com/axllent/mailpit/internal/smtpd/chaos"
)

// ChaosTriggers is the Chaos configuration
//
// swagger:model Triggers
type ChaosTriggers chaos.Triggers

// Response for the Chaos triggers configuration
// swagger:response ChaosResponse
type chaosResponse struct {
	// The current Chaos triggers
	//
	// in: body
	Body ChaosTriggers
}

// GetChaos returns the current Chaos triggers
func GetChaos(w http.ResponseWriter, _ *http.Request) {
	// swagger:route GET /api/v1/chaos testing getChaos
	//
	// # Get Chaos triggers
	//
	// Returns the current Chaos triggers configuration.
	// This API route will return an error if Chaos is not enabled at runtime.
	//
	//	Produces:
	//	  - application/json
	//
	//	Schemes: http, https
	//
	//	Responses:
	//	  200: ChaosResponse
	//	  400: ErrorResponse

	if !chaos.Enabled {
		httpError(w, "Chaos is not enabled")
		return
	}

	conf := chaos.Config

	w.Header().Add("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(conf); err != nil {
		httpError(w, err.Error())
	}
}

// swagger:parameters setChaosParams
type setChaosParams struct {
	// in: body
	Body ChaosTriggers
}

// SetChaos sets the Chaos configuration.
func SetChaos(w http.ResponseWriter, r *http.Request) {
	// swagger:route PUT /api/v1/chaos testing setChaosParams
	//
	// # Set Chaos triggers
	//
	// Set the Chaos triggers configuration and return the updated values.
	// This API route will return an error if Chaos is not enabled at runtime.
	//
	// If any triggers are omitted from the request, then those are reset to their
	// default values with a 0% probability (ie: disabled).
	// Setting a blank `{}` will reset all triggers to their default values.
	//
	//	Consumes:
	//	  - application/json
	//
	//	Produces:
	//	  - application/json
	//
	//	Schemes: http, https
	//
	//	Responses:
	//	  200: ChaosResponse
	//	  400: ErrorResponse

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

	conf := chaos.Config

	w.Header().Add("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(conf); err != nil {
		httpError(w, err.Error())
	}
}
