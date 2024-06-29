package apiv1

import (
	"encoding/json"
	"net/http"

	"github.com/axllent/mailpit/internal/storage"
	"github.com/axllent/mailpit/server/websockets"
	"github.com/gorilla/mux"
)

// GetAllTags (method: GET) will get all tags currently in use
func GetAllTags(w http.ResponseWriter, _ *http.Request) {
	// swagger:route GET /api/v1/tags tags GetAllTags
	//
	// # Get all current tags
	//
	// Returns a JSON array of all unique message tags.
	//
	//	Produces:
	//	- application/json
	//
	//	Schemes: http, https
	//
	//	Responses:
	//		200: ArrayResponse
	//		default: ErrorResponse

	w.Header().Add("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(storage.GetAllTags()); err != nil {
		httpError(w, err.Error())
	}
}

// SetMessageTags (method: PUT) will set the tags for all provided IDs
func SetMessageTags(w http.ResponseWriter, r *http.Request) {
	// swagger:route PUT /api/v1/tags tags SetTags
	//
	// # Set message tags
	//
	// This will overwrite any existing tags for selected message database IDs. To remove all tags from a message, pass an empty tags array.
	//
	//	Consumes:
	//	- application/json
	//
	//	Produces:
	//	- text/plain
	//
	//	Schemes: http, https
	//
	//	Responses:
	//		200: OKResponse
	//		default: ErrorResponse

	decoder := json.NewDecoder(r.Body)

	var data struct {
		Tags []string
		IDs  []string
	}

	err := decoder.Decode(&data)
	if err != nil {
		httpError(w, err.Error())
		return
	}

	ids := data.IDs

	if len(ids) > 0 {
		for _, id := range ids {
			if _, err := storage.SetMessageTags(id, data.Tags); err != nil {
				httpError(w, err.Error())
				return
			}
		}
	}

	w.Header().Add("Content-Type", "text/plain")
	_, _ = w.Write([]byte("ok"))
}

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
