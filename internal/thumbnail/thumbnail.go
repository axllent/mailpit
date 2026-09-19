// Package thumbnail generates fixed-size JPEG thumbnails from image data.
//
// It uses only the Go standard library and golang.org/x/image for decoding
// and resizing. Animated formats (APNG, GIF, WebP) are decoded as a single
// frame by the stdlib decoders, which avoids unbounded memory allocation
// from multi-frame images.
package thumbnail

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"

	// Register GIF and PNG decoders
	_ "image/gif"
	_ "image/png"

	"github.com/axllent/mailpit/internal/logger"
	xdraw "golang.org/x/image/draw"

	// Register WebP decoder
	_ "golang.org/x/image/webp"
)

// maxDecodedPixels is the maximum number of decoded pixels (width * height)
// allowed before thumbnail generation. This guards against compressed images
// that declare large dimensions, which would allocate large rasters in memory.
// At 4 bytes per RGBA pixel, 20 MP = 80 MB per decode.
const maxDecodedPixels int64 = 20_000_000

// Generate produces a JPEG thumbnail of exactly width × height pixels
// from the given image data, on a white background.
//
// If the data is not a recognised image, exceeds the pixel budget, or
// cannot be decoded, a blank white JPEG of the requested dimensions is
// returned with no error. This matches the UI expectation that the
// endpoint always returns a displayable image.
func Generate(data []byte, width, height int) ([]byte, error) {
	img, err := safeDecode(data)
	if err != nil || img == nil {
		return encodeJPEG(whiteCanvas(width, height))
	}

	img = fixOrientation(data, img)

	return encodeJPEG(resize(img, width, height))
}

// safeDecode decodes a single image frame after checking that the
// declared dimensions fall within the pixel budget.
func safeDecode(data []byte) (image.Image, error) {
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	if int64(cfg.Width)*int64(cfg.Height) > maxDecodedPixels {
		logger.Log().Warnf("[thumbnail] rejected oversized image dimensions %dx%d (exceeds %d pixel limit)", cfg.Width, cfg.Height, maxDecodedPixels)
		return nil, nil
	}

	img, _, err := image.Decode(bytes.NewReader(data))
	return img, err
}

// resize returns an image of exactly dstW × dstH containing src.
// Sources at least as large as the target on both axes are scaled to
// cover and centre-cropped. Smaller sources are scaled down to fit
// (never upscaled) and centred on a white background.
func resize(src image.Image, dstW, dstH int) image.Image {
	srcW := src.Bounds().Dx()
	srcH := src.Bounds().Dy()

	if srcW >= dstW && srcH >= dstH {
		return cover(src, dstW, dstH)
	}
	return fit(src, dstW, dstH)
}

// cover scales src to fill dstW × dstH and centre-crops the excess.
func cover(src image.Image, dstW, dstH int) image.Image {
	srcW := src.Bounds().Dx()
	srcH := src.Bounds().Dy()

	scaleW := float64(dstW) / float64(srcW)
	scaleH := float64(dstH) / float64(srcH)
	scale := scaleW
	if scaleH > scaleW {
		scale = scaleH
	}

	newW := int(float64(srcW)*scale + 0.5)
	newH := int(float64(srcH)*scale + 0.5)

	tmp := image.NewRGBA(image.Rect(0, 0, newW, newH))
	xdraw.ApproxBiLinear.Scale(tmp, tmp.Bounds(), src, src.Bounds(), xdraw.Src, nil)

	dst := whiteCanvas(dstW, dstH)
	draw.Draw(dst, dst.Bounds(), tmp, image.Pt((newW-dstW)/2, (newH-dstH)/2), draw.Over)
	return dst
}

// fit scales src down to fit within dstW × dstH (never upscales) and
// places the result centred on a white dstW × dstH canvas.
func fit(src image.Image, dstW, dstH int) image.Image {
	srcW := src.Bounds().Dx()
	srcH := src.Bounds().Dy()

	scaleW := float64(dstW) / float64(srcW)
	scaleH := float64(dstH) / float64(srcH)
	scale := scaleW
	if scaleH < scaleW {
		scale = scaleH
	}
	if scale > 1 {
		scale = 1
	}

	newW := int(float64(srcW)*scale + 0.5)
	newH := int(float64(srcH)*scale + 0.5)
	if newW < 1 {
		newW = 1
	}
	if newH < 1 {
		newH = 1
	}

	dst := whiteCanvas(dstW, dstH)
	target := image.Rect((dstW-newW)/2, (dstH-newH)/2, (dstW-newW)/2+newW, (dstH-newH)/2+newH)
	if scale == 1 {
		draw.Draw(dst, target, src, src.Bounds().Min, draw.Over)
	} else {
		xdraw.ApproxBiLinear.Scale(dst, target, src, src.Bounds(), xdraw.Over, nil)
	}
	return dst
}

func whiteCanvas(w, h int) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(dst, dst.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)
	return dst
}

func encodeJPEG(img image.Image) ([]byte, error) {
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 70}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
