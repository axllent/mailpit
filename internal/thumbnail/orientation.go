package thumbnail

import (
	"bytes"
	"github.com/rwcarlsen/goexif/exif"
	"image"
)

// fixOrientation reads the EXIF orientation tag from JPEG data and
// applies the corresponding transform. For non-JPEG data or data
// without an orientation tag the image is returned unchanged.
func fixOrientation(data []byte, img image.Image) image.Image {
	x, err := exif.Decode(bytes.NewReader(data))
	if err != nil {
		return img
	}

	tag, err := x.Get(exif.Orientation)
	if err != nil {
		return img
	}

	o, err := tag.Int(0)
	if err != nil || o < 1 || o > 8 {
		return img
	}

	switch o {
	case 2:
		return flipH(img)
	case 3:
		return rotate180(img)
	case 4:
		return flipV(img)
	case 5:
		return transpose(img)
	case 6:
		return rotate270(img)
	case 7:
		return transverse(img)
	case 8:
		return rotate90(img)
	default:
		return img
	}
}

// The eight EXIF orientation transforms below operate on generic
// image.Image values. Each allocates an NRGBA target and copies
// pixels with the appropriate coordinate mapping.

func flipH(src image.Image) image.Image {
	b := src.Bounds()
	dst := image.NewNRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			dst.Set(b.Max.X-1-x, y-b.Min.Y, src.At(x, y))
		}
	}
	return dst
}

func flipV(src image.Image) image.Image {
	b := src.Bounds()
	dst := image.NewNRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			dst.Set(x-b.Min.X, b.Max.Y-1-y, src.At(x, y))
		}
	}
	return dst
}

func rotate90(src image.Image) image.Image {
	b := src.Bounds()
	dst := image.NewNRGBA(image.Rect(0, 0, b.Dy(), b.Dx()))
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			dst.Set(y-b.Min.Y, b.Max.X-1-x, src.At(x, y))
		}
	}
	return dst
}

func rotate180(src image.Image) image.Image {
	b := src.Bounds()
	dst := image.NewNRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			dst.Set(b.Max.X-1-x, b.Max.Y-1-y, src.At(x, y))
		}
	}
	return dst
}

func rotate270(src image.Image) image.Image {
	b := src.Bounds()
	dst := image.NewNRGBA(image.Rect(0, 0, b.Dy(), b.Dx()))
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			dst.Set(b.Max.Y-1-y, x-b.Min.X, src.At(x, y))
		}
	}
	return dst
}

func transpose(src image.Image) image.Image {
	b := src.Bounds()
	dst := image.NewNRGBA(image.Rect(0, 0, b.Dy(), b.Dx()))
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			dst.Set(y-b.Min.Y, x-b.Min.X, src.At(x, y))
		}
	}
	return dst
}

func transverse(src image.Image) image.Image {
	b := src.Bounds()
	dst := image.NewNRGBA(image.Rect(0, 0, b.Dy(), b.Dx()))
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			dst.Set(b.Max.Y-1-y, b.Max.X-1-x, src.At(x, y))
		}
	}
	return dst
}
