package apiv1

import (
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
	// swagger:route GET /api/v1/message/{ID}/part/{PartID}/thumb message ThumbnailParams
	//
	// # Get an attachment image thumbnail
	//
	// This will return a cropped 180x120 JPEG thumbnail of an image attachment.
	// If the image is smaller than 180x120 then the image is padded. If the attachment is not an image then a blank image is returned.
	//
	// The ID can be set to `latest` to return the latest message.
	//
	//	Produces:
	//	  - image/jpeg
	//
	//	Schemes: http, https
	//
	//	Responses:
	//	  200: BinaryResponse
	//    400: ErrorResponse

	id := r.PathValue("id")
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

	data, err := thumbnail.Generate(a.Content, thumbWidth, thumbHeight)
	if err != nil {
		writeBlank(w, fileName)
		return
	}

	w.Header().Add("Content-Type", "image/jpeg")
	w.Header().Set("Content-Disposition", "filename=\""+url.PathEscape(jpegName(fileName))+"\"")
	_, _ = w.Write(data)
}

// writeBlank returns a blank thumbnail when the attachment is not a
// supported image or cannot be decoded.
func writeBlank(w http.ResponseWriter, fileName string) {
	data, _ := thumbnail.Generate(nil, thumbWidth, thumbHeight)
	w.Header().Add("Content-Type", "image/jpeg")
	w.Header().Set("Content-Disposition", "filename=\""+url.PathEscape(jpegName(fileName))+"\"")
	_, _ = w.Write(data)
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
