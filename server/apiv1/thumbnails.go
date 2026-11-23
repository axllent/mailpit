package apiv1

import (
	"context"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/axllent/mailpit/internal/storage"
	"github.com/axllent/mailpit/internal/thumbnail"
)

var (
	thumbWidth  = 180
	thumbHeight = 120
)

// Thumbnail returns a thumbnail image for an attachment (images only)
func Thumbnail(w http.ResponseWriter, r *http.Request) {
	id, ok := resolveMessageID(w, r, r.PathValue("id"))
	if !ok {
		return
	}
	partID := r.PathValue("partID")

	a, err := storage.GetAttachmentPart(id, partID)
	if err != nil {
		httpError(w, err.Error())
		return
	}

	fileName := a.FileName
	if fileName == "" {
		fileName = a.ContentID
	}

	if !strings.HasPrefix(a.ContentType, "image/") {
		writeBlank(w, fileName)
		return
	}

	data, err := thumbnail.Generate(r.Context(), a.Content, thumbWidth, thumbHeight)
	if err != nil {
		if r.Context().Err() != nil {
			// the client has gone, so there is nobody to send anything to
			return
		}

		writeBlank(w, fileName)
		return
	}

	w.Header().Add("Content-Type", "image/jpeg")
	w.Header().Set("Content-Disposition", "filename=\""+url.PathEscape(jpegName(fileName))+"\"")
	_, _ = w.Write(data) // #nosec G705 -- JPEG bytes from thumbnail.Generate, served as image/jpeg
}

// writeBlank returns a blank thumbnail when the attachment is not a
// supported image or cannot be decoded.
func writeBlank(w http.ResponseWriter, fileName string) {
	data, _ := thumbnail.Generate(context.Background(), nil, thumbWidth, thumbHeight)
	w.Header().Add("Content-Type", "image/jpeg")
	w.Header().Set("Content-Disposition", "filename=\""+url.PathEscape(jpegName(fileName))+"\"")
	_, _ = w.Write(data) // #nosec G705 -- blank JPEG from thumbnail.Generate, served as image/jpeg
}

// jpegName replaces the extension of the source filename with .jpg to
// match the JPEG bytes we always emit. Empty input becomes "thumb.jpg".
func jpegName(name string) string {
	name = strings.TrimSuffix(name, path.Ext(name))
	if name == "" {
		return "thumb.jpg"
	}
	return name + ".jpg"
}
