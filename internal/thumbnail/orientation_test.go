package thumbnail

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/jpeg"
	"runtime"
	"testing"
)

// jpegWithOrientation returns a JPEG with an EXIF orientation tag.
func jpegWithOrientation(tb testing.TB, o int) []byte {
	tb.Helper()

	return jpegWithExif(tb, tiffBlock(binary.BigEndian, uint16(o), 0))
}

// orientedPoint returns where the pixel (x, y) of a w x h image ends up when the EXIF orientation
// o is applied, as described by the EXIF specification.
func orientedPoint(o, x, y, w, h int) (int, int) {
	switch o {
	case 2:
		return w - 1 - x, y
	case 3:
		return w - 1 - x, h - 1 - y
	case 4:
		return x, h - 1 - y
	case 5:
		return y, x
	case 6:
		return h - 1 - y, x
	case 7:
		return h - 1 - y, w - 1 - x
	case 8:
		return y, w - 1 - x
	default:
		return x, y
	}
}

// testPattern returns an asymmetric w x h image in which every pixel is different, with its top
// left corner at min.
func testPattern(w, h int, min image.Point) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(min.X, min.Y, min.X+w, min.Y+h))
	for y := range h {
		for x := range w {
			img.SetNRGBA(min.X+x, min.Y+y, color.NRGBA{R: uint8(x*40 + 10), G: uint8(y*60 + 10), B: uint8(x*5 + y*7), A: 255})
		}
	}

	return img
}

func TestFixOrientation(t *testing.T) {
	const w, h = 5, 3

	for _, min := range []image.Point{{0, 0}, {7, 11}} {
		src := testPattern(w, h, min)

		for o := 1; o <= 8; o++ {
			got := fixOrientation(jpegWithOrientation(t, o), src)

			wantW, wantH := w, h
			if o >= 5 {
				wantW, wantH = h, w
			}
			b := got.Bounds()
			if b.Dx() != wantW || b.Dy() != wantH {
				t.Fatalf("orientation %d (min %v): size %dx%d, want %dx%d", o, min, b.Dx(), b.Dy(), wantW, wantH)
			}

			for y := range h {
				for x := range w {
					dx, dy := orientedPoint(o, x, y, w, h)
					want := color.NRGBAModel.Convert(src.At(min.X+x, min.Y+y))
					have := color.NRGBAModel.Convert(got.At(b.Min.X+dx, b.Min.Y+dy))
					if want != have {
						t.Fatalf("orientation %d (min %v): pixel (%d,%d) moved to (%d,%d) is %v, want %v", o, min, x, y, dx, dy, have, want)
					}
				}
			}
		}
	}
}

// Without a valid orientation the image is returned as it is.
func TestFixOrientationUnchanged(t *testing.T) {
	src := testPattern(4, 2, image.Point{})

	for _, data := range [][]byte{nil, []byte("not an image"), jpegWithOrientation(t, 1), jpegWithOrientation(t, 0), jpegWithOrientation(t, 9)} {
		if got := fixOrientation(data, src); got != image.Image(src) {
			t.Errorf("%q: the image was changed", data)
		}
	}
}

// Pixels outside the image are transparent, so a scaler that asks for one does not panic.
func TestOrientedImageOutOfBounds(t *testing.T) {
	img := fixOrientation(jpegWithOrientation(t, 6), testPattern(5, 3, image.Point{3, 4}))

	for _, p := range []image.Point{{-1, 0}, {0, -1}, {3, 0}, {0, 5}, {100, 100}} {
		if _, _, _, a := img.At(p.X, p.Y).RGBA(); a != 0 {
			t.Errorf("At(%v) is not transparent", p)
		}
	}
}

// materialise returns a rotated copy of the image, as was made before the orientation became a view.
func materialise(src *image.NRGBA, o int) *image.NRGBA {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	dw, dh := w, h
	if o >= 5 {
		dw, dh = h, w
	}

	dst := image.NewNRGBA(image.Rect(0, 0, dw, dh))
	for y := range h {
		for x := range w {
			dx, dy := orientedPoint(o, x, y, w, h)
			dst.Set(dx, dy, src.At(b.Min.X+x, b.Min.Y+y))
		}
	}

	return dst
}

// The thumbnail of the view must be the thumbnail of the rotated copy, for images that are
// scaled down and cropped (large) and ones that are padded (small).
func TestOrientedThumbnailMatchesRotatedCopy(t *testing.T) {
	for _, size := range [][2]int{{400, 300}, {300, 400}, {100, 50}, {50, 100}} {
		src := testPattern(size[0], size[1], image.Point{})
		// make the pattern vary across the whole image, not just in a small range
		for y := range size[1] {
			for x := range size[0] {
				src.SetNRGBA(x, y, color.NRGBA{R: uint8(x * 255 / size[0]), G: uint8(y * 255 / size[1]), B: uint8((x ^ y) & 255), A: 255})
			}
		}

		for o := 2; o <= 8; o++ {
			want := resize(materialise(src, o), 180, 120).(*image.RGBA)
			got := resize(fixOrientation(jpegWithOrientation(t, o), src), 180, 120).(*image.RGBA)

			if want.Bounds() != got.Bounds() {
				t.Fatalf("%v orientation %d: bounds %v, want %v", size, o, got.Bounds(), want.Bounds())
			}

			maxDiff := 0
			for i := range want.Pix {
				d := int(want.Pix[i]) - int(got.Pix[i])
				if d < 0 {
					d = -d
				}
				maxDiff = max(maxDiff, d)
			}
			if maxDiff > 2 {
				t.Errorf("%v orientation %d: thumbnails differ by up to %d", size, o, maxDiff)
			}
		}
	}
}

// withOrientation inserts an EXIF orientation into an encoded JPEG.
func withOrientation(tb testing.TB, raw []byte, o int) []byte {
	tb.Helper()

	marker := jpegWithOrientation(tb, o)
	// the 1x1 test JPEG starts with the SOI marker and the APP1 segment, which is all that is needed
	segEnd := 2 + 2 + int(binary.BigEndian.Uint16(marker[4:6]))

	out := append([]byte{}, raw[:2]...)
	out = append(out, marker[2:segEnd]...)

	return append(out, raw[2:]...)
}

// A rotated image used to be copied at its full size before it was scaled down, which cost
// about 230 MB and 1.3 seconds more than the same image without a rotation, for a 20 megapixel JPEG.
func TestOrientationDoesNotCopyTheImage(t *testing.T) {
	// just inside the pixel limit
	const side = 4472

	var jb bytes.Buffer
	if err := jpeg.Encode(&jb, image.NewGray(image.Rect(0, 0, side, side)), nil); err != nil {
		t.Fatal(err)
	}

	alloc := func(data []byte) uint64 {
		runtime.GC()
		var m0, m1 runtime.MemStats
		runtime.ReadMemStats(&m0)
		if _, err := Generate(data, 180, 120); err != nil {
			t.Fatal(err)
		}
		runtime.ReadMemStats(&m1)

		return m1.TotalAlloc - m0.TotalAlloc
	}

	// control: no rotation, which only decodes
	plain := alloc(withOrientation(t, jb.Bytes(), 1))
	for _, o := range []int{2, 6, 8} {
		rotated := alloc(withOrientation(t, jb.Bytes(), o))
		t.Logf("orientation %d: %d MiB, without rotation %d MiB", o, rotated>>20, plain>>20)

		if rotated > plain+16<<20 {
			t.Errorf("orientation %d allocated %d MiB, against %d MiB for the same image without one", o, rotated>>20, plain>>20)
		}
	}
}
