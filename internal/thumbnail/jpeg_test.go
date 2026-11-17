package thumbnail

import (
	"bytes"
	"context"
	"image"
	"image/jpeg"
	"testing"
)

// segment builds a JPEG marker segment.
func segment(marker byte, payload ...byte) []byte {
	n := len(payload) + 2

	return append([]byte{0xFF, marker, byte(n >> 8), byte(n)}, payload...)
}

// frameHeader builds an SOF segment for a w x h image with one component.
func frameHeader(marker byte, w, h int) []byte {
	return segment(marker, 8, byte(h>>8), byte(h), byte(w>>8), byte(w), 1, 1, 0x11, 0)
}

func joinBytes(parts ...[]byte) []byte {
	return bytes.Join(parts, nil)
}

func TestIsProgressiveJPEG(t *testing.T) {
	soi := []byte{0xFF, 0xD8}
	sos := segment(jpegSOS, 1, 1, 0, 0, 63, 0)
	app1 := segment(jpegAPP1, []byte("Exif\x00\x00II*\x00")...)

	tests := []struct {
		name string
		data []byte
		want bool
	}{
		{"control: baseline SOF0", joinBytes(soi, frameHeader(0xC0, 10, 10), sos), false},
		{"control: extended sequential SOF1", joinBytes(soi, frameHeader(0xC1, 10, 10), sos), false},
		{"progressive SOF2", joinBytes(soi, frameHeader(0xC2, 10, 10), sos), true},
		// frame headers which the decoder does not support are refused by it, so the answer does not matter, and
		// the safe one is given
		{"differential progressive SOF6", joinBytes(soi, frameHeader(0xC6, 10, 10), sos), true},
		{"progressive arithmetic SOF10", joinBytes(soi, frameHeader(0xCA, 10, 10), sos), true},
		{"differential progressive arithmetic SOF14", joinBytes(soi, frameHeader(0xCE, 10, 10), sos), true},
		{"sequential arithmetic SOF9", joinBytes(soi, frameHeader(0xC9, 10, 10), sos), true},
		{"after other segments", joinBytes(soi, app1, segment(0xDB, make([]byte, 65)...), segment(jpegDHT, make([]byte, 17)...), frameHeader(0xC2, 10, 10), sos), true},
		{"control: baseline after other segments", joinBytes(soi, app1, segment(jpegCOM, 'h', 'i'), segment(0xDB, make([]byte, 65)...), segment(jpegDRI, 0, 0), frameHeader(0xC0, 10, 10), sos), false},
		{"JPG and DAC are not frame headers (and the decoder refuses them)", joinBytes(soi, segment(jpegJPG, 0, 0), segment(jpegDAC, 0, 0), frameHeader(0xC0, 10, 10), sos), true},
		{"fill bytes and restart markers", joinBytes(soi, []byte{0xFF, 0xFF, 0xFF, 0xD3}, frameHeader(0xC2, 10, 10), sos), true},
		{"control: baseline after fill bytes and restart markers", joinBytes(soi, []byte{0xFF, 0xFF, 0xFF, 0xD3}, frameHeader(0xC0, 10, 10), sos), false},
		{"TEM, which the decoder refuses", joinBytes(soi, []byte{0xFF, 0x01}, frameHeader(0xC0, 10, 10), sos), true},
		{"frame header only after the image data starts", joinBytes(soi, sos, frameHeader(0xC0, 10, 10)), true},
		{"end of image before a frame header", joinBytes(soi, []byte{0xFF, 0xD9}, frameHeader(0xC0, 10, 10)), true},

		// what the decoder skips, which once made this answer false for a progressive JPEG
		{"bytes before a marker", joinBytes(soi, []byte{0x00}, frameHeader(0xC2, 10, 10), sos), true},
		{"bytes between segments", joinBytes(soi, app1, []byte{0x12, 0x34}, frameHeader(0xC2, 10, 10), sos), true},
		{"0xFF 0x00", joinBytes(soi, []byte{0xFF, 0x00}, frameHeader(0xC2, 10, 10), sos), true},
		// controls: the same for a baseline JPEG, which is still found to be baseline
		{"control: baseline after bytes before a marker", joinBytes(soi, []byte{0x00}, frameHeader(0xC0, 10, 10), sos), false},
		{"control: baseline after bytes between segments", joinBytes(soi, app1, []byte{0x12, 0x34}, frameHeader(0xC0, 10, 10), sos), false},
		{"control: baseline after 0xFF 0x00", joinBytes(soi, []byte{0xFF, 0x00}, frameHeader(0xC0, 10, 10), sos), false},

		// a frame header inside another segment is not read, as the decoder does not read it
		{"progressive frame header inside a segment", joinBytes(soi, segment(jpegAPP1, frameHeader(0xC2, 10, 10)...), frameHeader(0xC0, 10, 10), sos), false},
		{"baseline frame header inside a segment", joinBytes(soi, segment(jpegAPP1, frameHeader(0xC0, 10, 10)...), frameHeader(0xC2, 10, 10), sos), true},

		// what can not be followed is treated as progressive
		{"empty", nil, true},
		{"not a JPEG", []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\x0d"), true},
		{"SOI only", soi, true},
		{"progressive frame header cut short", joinBytes(soi, frameHeader(0xC2, 10, 10)[:6]), true},
		{"other segment cut short", joinBytes(soi, segment(0xDB, make([]byte, 65)...)[:20]), true},
		{"segment length below 2", joinBytes(soi, []byte{0xFF, 0xDB, 0x00, 0x01}, frameHeader(0xC0, 10, 10)), true},
		{"segment longer than the data", joinBytes(soi, []byte{0xFF, 0xDB, 0xFF, 0xFF}, frameHeader(0xC0, 10, 10)), true},
		{"length cut short", joinBytes(soi, []byte{0xFF, 0xDB, 0x00}), true},
		{"fill bytes to the end", joinBytes(soi, []byte{0xFF, 0xFF, 0xFF}), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isProgressiveJPEG(tt.data); got != tt.want {
				t.Errorf("isProgressiveJPEG = %v, want %v", got, tt.want)
			}
		})
	}

	// control: a real JPEG from the encoder, which is baseline
	var jb bytes.Buffer
	if err := jpeg.Encode(&jb, testPattern(20, 20, image.Point{}), nil); err != nil {
		t.Fatal(err)
	}
	if isProgressiveJPEG(jb.Bytes()) {
		t.Error("a baseline JPEG from the encoder is reported as progressive")
	}
}

// patchFrame changes the type and size of a JPEG's frame header.
func patchFrame(tb testing.TB, data []byte, marker byte, w, h int) []byte {
	tb.Helper()

	out := append([]byte{}, data...)
	for i := 2; i+9 < len(out); {
		if out[i] != 0xFF {
			tb.Fatal("no frame header found")
		}
		if out[i+1] == jpegSOF0 {
			out[i+1] = marker
			out[i+5], out[i+6] = byte(h>>8), byte(h)
			out[i+7], out[i+8] = byte(w>>8), byte(w)

			return out
		}
		i += 2 + int(out[i+2])<<8 + int(out[i+3])
	}
	tb.Fatal("no frame header found")

	return nil
}

func grayJPEG(tb testing.TB, w, h int) []byte {
	tb.Helper()

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, image.NewGray(image.Rect(0, 0, w, h)), nil); err != nil {
		tb.Fatal(err)
	}

	return buf.Bytes()
}

// Progressive JPEGs need about ten times the memory of baseline ones to decode, so they have a
// lower pixel limit. The exploit cases are over it, and the controls are the same size without
// being progressive, or progressive within the limit.
func TestProgressiveJPEGLimit(t *testing.T) {
	// 16 megapixels: over the progressive limit, within the general one
	large := grayJPEG(t, 4000, 4000)
	// exactly at the progressive limit
	atLimit := grayJPEG(t, 5000, 2500)

	if got := int64(5000 * 2500); got != maxProgressivePixels {
		t.Fatalf("the test sizes assume a limit of %d, not %d", got, maxProgressivePixels)
	}

	rejected := func(data []byte) bool {
		img, err := safeDecode(data)

		return img == nil && err == nil
	}

	// control: a baseline JPEG of the same size is decoded
	img, err := safeDecode(large)
	if err != nil || img == nil {
		t.Fatalf("a 16 MP baseline JPEG was not decoded: %v", err)
	}

	// exploit: the same image as a progressive one is refused, without being decoded
	if !rejected(patchFrame(t, large, jpegSOF2, 4000, 4000)) {
		t.Error("a 16 MP progressive JPEG was not rejected")
	}

	// control: progressive within the limit is not refused (it is then decoded, or is an error
	// here as the image data is not really progressive)
	if rejected(patchFrame(t, atLimit, jpegSOF2, 5000, 2500)) {
		t.Error("a progressive JPEG exactly at the limit was rejected")
	}
	if rejected(patchFrame(t, atLimit, jpegSOF2, 3000, 3000)) {
		t.Error("a 9 MP progressive JPEG was rejected")
	}

	// exploit: one row over the limit
	if !rejected(patchFrame(t, atLimit, jpegSOF2, 5000, 2501)) {
		t.Error("a progressive JPEG one row over the limit was not rejected")
	}

	// control: the same size as a baseline frame is not refused for being progressive
	if rejected(patchFrame(t, atLimit, jpegSOF0, 5000, 2501)) {
		t.Error("a baseline JPEG over the progressive limit was rejected")
	}

	// the general limit still applies to everything
	if !rejected(patchFrame(t, atLimit, jpegSOF0, 5000, 5000)) {
		t.Error("a 25 MP baseline JPEG was not rejected")
	}

	// and a rejected image gives a blank thumbnail, not an error
	out, err := Generate(context.Background(), patchFrame(t, large, jpegSOF2, 4000, 4000), 180, 120)
	if err != nil {
		t.Fatal(err)
	}
	checkThumbnail(t, out)
}

// insertAfterSOI returns data with extra inserted straight after the start of image marker.
func insertAfterSOI(data, extra []byte) []byte {
	return joinBytes(data[:2], extra, data[2:])
}

// A progressive JPEG with bytes the decoder skips (a byte which is not a marker, or 0xFF 0x00) is still held to the
// lower limit. The check once stopped at such bytes and said the image was not progressive, so a single byte let a
// progressive JPEG of up to the general limit be decoded. The decoder reads these images as it does without the bytes,
// which shows that they are real images and not just data it refuses. A baseline JPEG with the same bytes (the
// control) is decoded.
func TestProgressiveJPEGLimitWithSkippedBytes(t *testing.T) {
	baseline := grayJPEG(t, 4000, 4000)
	progressive := patchFrame(t, baseline, jpegSOF2, 4000, 4000)

	for name, extra := range map[string][]byte{
		"a byte which is not a marker": {0x00},
		"several bytes":                {0x12, 0x34, 0x56},
		"0xFF 0x00":                    {0xFF, 0x00},
	} {
		t.Run(name, func(t *testing.T) {
			data := insertAfterSOI(progressive, extra)

			cfg, err := jpeg.DecodeConfig(bytes.NewReader(data))
			if err != nil || cfg.Width != 4000 || cfg.Height != 4000 {
				t.Fatalf("the decoder does not read the image: %v (%dx%d)", err, cfg.Width, cfg.Height)
			}

			if img, err := safeDecode(data); img != nil || err != nil {
				t.Errorf("a 16 MP progressive JPEG was not rejected: %v", err)
			}

			// control
			img, err := safeDecode(insertAfterSOI(baseline, extra))
			if err != nil || img == nil {
				t.Errorf("a 16 MP baseline JPEG was not decoded: %v", err)
			}
		})
	}
}

// The scan never panics, and only reports a JPEG as not progressive when it has a sequential frame header which the
// decoder reads (it is then decoded as such, or refused).
func FuzzIsProgressiveJPEG(f *testing.F) {
	soi := []byte{0xFF, 0xD8}
	f.Add(joinBytes(soi, frameHeader(0xC2, 10, 10)))
	f.Add(joinBytes(soi, frameHeader(0xC0, 10, 10)))
	f.Add(joinBytes(soi, []byte{0x00, 0xFF, 0x00}, frameHeader(0xC2, 10, 10)))
	f.Add(joinBytes(soi, segment(jpegAPP1, []byte("Exif\x00\x00II*\x00")...), frameHeader(0xCA, 1, 1)))
	f.Add([]byte{0xFF, 0xD8, 0xFF, 0xFF, 0xFF, 0x01, 0xFF, 0xC2})
	f.Add([]byte{})

	f.Fuzz(func(t *testing.T, data []byte) {
		if isProgressiveJPEG(data) {
			return
		}

		if len(data) < 4 || data[0] != 0xFF || data[1] != 0xD8 {
			t.Fatal("not progressive without a JPEG header")
		}
		if !bytes.Contains(data, []byte{0xFF, jpegSOF0}) && !bytes.Contains(data, []byte{0xFF, jpegSOF1}) {
			t.Fatal("not progressive without a sequential frame marker")
		}
	})
}
