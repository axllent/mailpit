package apiv1

import (
	"encoding/json"
	"net/http"

	"github.com/axllent/mailpit/internal/storage"
	"github.com/axllent/mailpit/server/websockets"
)

// SetTagsRequest is the request body for setting message tags
//
// openapi:model SetTagsRequest
type SetTagsRequest struct {
	// Array of tag names to set
	//
	// required: true
	// example: ["Tag 1", "Tag 2"]
	Tags []string

	// Array of message database IDs
	//
	// required: true
	// example: ["4oRBnPtCXgAqZniRhzLNmS", "hXayS6wnCgNnt6aFTvmOF6"]
	IDs []string
}

// RenameTagRequest is the request body for renaming a tag
//
// openapi:model RenameTagRequest
type RenameTagRequest struct {
	// New name
	//
	// required: true
	// example: New name
	Name string
}

// GetAllTags (method: GET) will get all tags currently in use
func GetAllTags(w http.ResponseWriter, _ *http.Request) {
	w.Header().Add("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(storage.GetAllTags()); err != nil {
		httpError(w, err.Error())
	}
}

// SetMessageTags (method: PUT) will set the tags for all provided IDs
func SetMessageTags(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)

	var data SetTagsRequest

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
	tag := r.PathValue("tag")

	decoder := json.NewDecoder(r.Body)

	var data RenameTagRequest

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
	tag := r.PathValue("tag")

	if err := storage.DeleteTag(tag); err != nil {
		httpError(w, err.Error())
		return
	}

	websockets.Broadcast("prune", nil)

	w.Header().Add("Content-Type", "text/plain")
	_, _ = w.Write([]byte("ok"))
}
