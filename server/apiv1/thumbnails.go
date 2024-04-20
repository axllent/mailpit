package apiv1

import (
	"bufio"
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"net/http"
	"strings"

	"github.com/axllent/mailpit/internal/logger"
	"github.com/axllent/mailpit/internal/storage"
	"github.com/gorilla/mux"
	"github.com/jhillyerd/enmime"
	"github.com/kovidgoyal/imaging"
)

var (
	thumbWidth  = 180
	thumbHeight = 120
)

// Thumbnail returns a thumbnail image for an attachment (images only)
func Thumbnail(w http.ResponseWriter, r *http.Request) {
	// swagger:route GET /api/v1/message/{ID}/part/{PartID}/thumb message Thumbnail
	//
	// # Get an attachment image thumbnail
	//
	// This will return a cropped 180x120 JPEG thumbnail of an image attachment.
	// If the image is smaller than 180x120 then the image is padded. If the attachment is not an image then a blank image is returned.
	//
	//	Produces:
	//	- image/jpeg
	//
	//	Schemes: http, https
	//
	//	Parameters:
	//	  + name: ID
	//	    in: path
	//	    description: Database ID
	//	    required: true
	//	    type: string
	//	  + name: PartID
	//	    in: path
	//	    description: Attachment part ID
	//	    required: true
	//	    type: string
	//
	//	Responses:
	//		200: BinaryResponse
	//		default: ErrorResponse
	vars := mux.Vars(r)

	id := vars["id"]
	partID := vars["partID"]

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
		blankImage(a, w)
		return
	}

	buf := bytes.NewBuffer(a.Content)

	img, err := imaging.Decode(buf, imaging.AutoOrientation(true))
	if err != nil {
		// it's not an image, return default
		logger.Log().Warnf("[image] %s", err.Error())
		blankImage(a, w)
		return
	}

	var b bytes.Buffer
	foo := bufio.NewWriter(&b)

	var dstImageFill *image.NRGBA

	if img.Bounds().Dx() < thumbWidth || img.Bounds().Dy() < thumbHeight {
		dstImageFill = imaging.Fit(img, thumbWidth, thumbHeight, imaging.Lanczos)
	} else {
		dstImageFill = imaging.Fill(img, thumbWidth, thumbHeight, imaging.Center, imaging.Lanczos)
	}
	// create white image and paste image over the top
	// preventing black backgrounds for transparent GIF/PNG images
	dst := imaging.New(thumbWidth, thumbHeight, color.White)
	// paste the original over the top
	dst = imaging.OverlayCenter(dst, dstImageFill, 1.0)

	if err := jpeg.Encode(foo, dst, &jpeg.Options{Quality: 70}); err != nil {
		logger.Log().Warnf("[image] %s", err.Error())
		blankImage(a, w)
		return
	}

	w.Header().Add("Content-Type", "image/jpeg")
	w.Header().Set("Content-Disposition", "filename=\""+fileName+"\"")
	_, _ = w.Write(b.Bytes())
}

// Return a blank image instead of an error when file or image not supported
func blankImage(a *enmime.Part, w http.ResponseWriter) {
	rect := image.Rect(0, 0, thumbWidth, thumbHeight)
	img := image.NewRGBA(rect)
	background := color.RGBA{255, 255, 255, 255}
	draw.Draw(img, img.Bounds(), &image.Uniform{background}, image.Point{}, draw.Src)
	var b bytes.Buffer
	foo := bufio.NewWriter(&b)
	dstImageFill := imaging.Fill(img, thumbWidth, thumbHeight, imaging.Center, imaging.Lanczos)

	if err := jpeg.Encode(foo, dstImageFill, &jpeg.Options{Quality: 70}); err != nil {
		logger.Log().Warnf("[image] %s", err.Error())
	}

	fileName := a.FileName
	if fileName == "" {
		fileName = a.ContentID
	}

	w.Header().Add("Content-Type", "image/jpeg")
	w.Header().Set("Content-Disposition", "filename=\""+fileName+"\"")
	_, _ = w.Write(b.Bytes())
}
