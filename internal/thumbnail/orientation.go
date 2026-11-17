package thumbnail

import (
	"image"
	"image/color"
)

// fixOrientation reads the EXIF orientation tag from JPEG data and
// returns the image as it should be displayed. For non-JPEG data or data
// without a valid orientation tag the image is returned unchanged.
func fixOrientation(data []byte, img image.Image) image.Image {
	o := readOrientation(data)
	if o < 2 || o > 8 {
		return img
	}

	return newOrientedImage(img, o)
}

// orientedImage is a view of an image with an EXIF orientation (2-8) applied. It reads the
// pixels of the source image as they are asked for, instead of making a rotated copy. A copy
// of a large image is costly (80 MB for 20 megapixels, and about a second), while the
// thumbnail only reads a few pixels per output pixel.
type orientedImage struct {
	src image.Image
	o   int
	// size of the source, and of the view (which has the sides swapped for orientations 5-8)
	srcW, srcH int
	w, h       int
}

func newOrientedImage(src image.Image, o int) *orientedImage {
	b := src.Bounds()
	v := &orientedImage{src: src, o: o, srcW: b.Dx(), srcH: b.Dy()}

	if o >= 5 {
		v.w, v.h = v.srcH, v.srcW
	} else {
		v.w, v.h = v.srcW, v.srcH
	}

	return v
}

// ColorModel implements image.Image.
func (v *orientedImage) ColorModel() color.Model {
	return v.src.ColorModel()
}

// Bounds implements image.Image. The view starts at (0, 0).
func (v *orientedImage) Bounds() image.Rectangle {
	return image.Rect(0, 0, v.w, v.h)
}

// At implements image.Image. It returns the source pixel that is displayed at (x, y), which is
// the inverse of the EXIF orientation transform. Points outside the view are transparent.
func (v *orientedImage) At(x, y int) color.Color {
	if x < 0 || y < 0 || x >= v.w || y >= v.h {
		return color.Transparent
	}

	var sx, sy int

	switch v.o {
	case 2: // flip horizontally
		sx, sy = v.srcW-1-x, y
	case 3: // rotate 180
		sx, sy = v.srcW-1-x, v.srcH-1-y
	case 4: // flip vertically
		sx, sy = x, v.srcH-1-y
	case 5: // transpose
		sx, sy = y, x
	case 6: // rotate 90 clockwise
		sx, sy = y, v.srcH-1-x
	case 7: // transverse
		sx, sy = v.srcW-1-y, v.srcH-1-x
	case 8: // rotate 90 anticlockwise
		sx, sy = v.srcW-1-y, x
	default:
		sx, sy = x, y
	}

	b := v.src.Bounds()

	return v.src.At(b.Min.X+sx, b.Min.Y+sy)
}
