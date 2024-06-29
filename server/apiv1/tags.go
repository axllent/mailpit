package apiv1

import (
	"encoding/json"
	"net/http"

	"github.com/axllent/mailpit/internal/storage"
	"github.com/axllent/mailpit/server/websockets"
	"github.com/gorilla/mux"
)

// RenameTag (method: PUT) used to rename a tag
func RenameTag(w http.ResponseWriter, r *http.Request) {
	// swagger:route PUT /api/v1/tags/{tag} tags RenameTag
	//
	// # Rename a tag
	//
	// Renames a tag.
	//
	//	Produces:
	//	- text/plain
	//
	//	Schemes: http, https
	//
	//	Parameters:
	//	  + name: tag
	//	    in: path
	//	    description: The url-encoded tag name to rename
	//	    required: true
	//	    type: string
	//
	//	Responses:
	//		200: OKResponse
	//		default: ErrorResponse

	vars := mux.Vars(r)

	tag := vars["tag"]

	decoder := json.NewDecoder(r.Body)

	var data struct {
		Name string
	}

	err := decoder.Decode(&data)
	if err != nil {
		httpError(w, err.Error())
		return
	}

	if err := storage.RenameTag(tag, data.Name); err != nil {
		httpError(w, err.Error())
		return
	}

	websockets.Broadcast("prune", nil)

	w.Header().Add("Content-Type", "text/plain")
	_, _ = w.Write([]byte("ok"))
}

// DeleteTag (method: DELETE) used to delete a tag
func DeleteTag(w http.ResponseWriter, r *http.Request) {
	// swagger:route DELETE /api/v1/tags/{tag} tags DeleteTag
	//
	// # Delete a tag
	//
	// Deletes a tag. This will not delete any messages with this tag.
	//
	//	Produces:
	//	- text/plain
	//
	//	Schemes: http, https
	//
	//	Parameters:
	//	  + name: tag
	//	    in: path
	//	    description: The url-encoded tag name to delete
	//	    required: true
	//	    type: string
	//
	//	Responses:
	//		200: OKResponse
	//		default: ErrorResponse

	vars := mux.Vars(r)

	tag := vars["tag"]

	if err := storage.DeleteTag(tag); err != nil {
		httpError(w, err.Error())
		return
	}

	websockets.Broadcast("prune", nil)

	w.Header().Add("Content-Type", "text/plain")
	_, _ = w.Write([]byte("ok"))
}
