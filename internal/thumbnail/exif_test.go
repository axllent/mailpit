package thumbnail

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/jpeg"
	"testing"
	"time"
)

// tiffBlock builds a TIFF block whose IFDs each hold one orientation entry,
// chained via the given next-IFD offsets (IFD n starts at 8 + n*18).
func tiffBlock(order binary.ByteOrder, orientation uint16, nexts ...uint32) []byte {
	var b bytes.Buffer
	if order == binary.LittleEndian {
		b.WriteString("II")
	} else {
		b.WriteString("MM")
	}
	_ = binary.Write(&b, order, uint16(42))
	_ = binary.Write(&b, order, uint32(8))
	for _, next := range nexts {
		_ = binary.Write(&b, order, uint16(1))
		_ = binary.Write(&b, order, uint16(exifOrientationTag))
		_ = binary.Write(&b, order, uint16(tiffTypeShort))
		_ = binary.Write(&b, order, uint32(1))
		_ = binary.Write(&b, order, orientation)
		_ = binary.Write(&b, order, uint16(0))
		_ = binary.Write(&b, order, next)
	}
	return b.Bytes()
}

// jpegWithExif inserts an APP1 Exif segment into a valid 1x1 JPEG.
func jpegWithExif(tb testing.TB, tiff []byte) []byte {
	var jb bytes.Buffer
	if err := jpeg.Encode(&jb, image.NewRGBA(image.Rect(0, 0, 1, 1)), nil); err != nil {
		tb.Fatal(err)
	}
	raw := jb.Bytes()
	payload := append([]byte("Exif\x00\x00"), tiff...)

	var out bytes.Buffer
	out.Write(raw[:2])
	out.Write([]byte{0xFF, 0xE1})
	_ = binary.Write(&out, binary.BigEndian, uint16(len(payload)+2))
	out.Write(payload)
	out.Write(raw[2:])
	return out.Bytes()
}

func TestReadOrientation(t *testing.T) {
	for _, order := range []binary.ByteOrder{binary.LittleEndian, binary.BigEndian} {
		for o := uint16(1); o <= 8; o++ {
			data := jpegWithExif(t, tiffBlock(order, o, 0))
			if got := readOrientation(data); got != int(o) {
				t.Errorf("%v orientation %d: got %d", order, o, got)
			}
		}
	}

	invalid := map[string][]byte{
		"nil":          nil,
		"not jpeg":     []byte("hello world"),
		"out of range": jpegWithExif(t, tiffBlock(binary.LittleEndian, 9, 0)),
		"zero":         jpegWithExif(t, tiffBlock(binary.LittleEndian, 0, 0)),
		"bad magic":    jpegWithExif(t, append([]byte("II\x2b\x00\x08\x00\x00\x00"), 0, 0)),
		"truncated":    jpegWithExif(t, tiffBlock(binary.LittleEndian, 6, 0)[:14]),
	}
	for name, data := range invalid {
		if got := readOrientation(data); got != 0 {
			t.Errorf("%s: expected 0, got %d", name, got)
		}
	}
}

// Exploit case: IFD0 -> IFD1 -> IFD0 chain (crashed goexif with an
// unbounded loop). Matched control: same shape with a terminated chain.
func TestReadOrientationIFDCycle(t *testing.T) {
	control := jpegWithExif(t, tiffBlock(binary.LittleEndian, 6, 26, 0))
	cycle := jpegWithExif(t, tiffBlock(binary.LittleEndian, 6, 26, 8))

	for name, data := range map[string][]byte{"control": control, "cycle": cycle} {
		done := make(chan int, 1)
		go func() { done <- readOrientation(data) }()
		select {
		case got := <-done:
			if got != 6 {
				t.Errorf("%s: expected orientation 6, got %d", name, got)
			}
		case <-time.After(2 * time.Second):
			t.Fatalf("%s: readOrientation did not return", name)
		}
	}

	// full pipeline
	for name, data := range map[string][]byte{"control": control, "cycle": cycle} {
		done := make(chan struct{})
		go func() { _, _ = Generate(data, 180, 120); close(done) }()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Fatalf("%s: Generate did not return", name)
		}
	}
}

// ifdEntry builds a raw 12-byte IFD entry.
func ifdEntry(order binary.ByteOrder, tag, typ uint16, count uint32, value uint16) []byte {
	var b bytes.Buffer
	_ = binary.Write(&b, order, tag)
	_ = binary.Write(&b, order, typ)
	_ = binary.Write(&b, order, count)
	_ = binary.Write(&b, order, value)
	_ = binary.Write(&b, order, uint16(0))
	return b.Bytes()
}

// tiffIFD0 builds a little-endian TIFF block with a single IFD0 holding the
// given entries. declared is the entry count written in the IFD header, which
// can differ from len(entries) to simulate overrunning counts.
func tiffIFD0(declared uint16, entries ...[]byte) []byte {
	le := binary.LittleEndian
	var b bytes.Buffer
	b.WriteString("II")
	_ = binary.Write(&b, le, uint16(42))
	_ = binary.Write(&b, le, uint32(8))
	_ = binary.Write(&b, le, declared)
	for _, e := range entries {
		b.Write(e)
	}
	_ = binary.Write(&b, le, uint32(0))
	return b.Bytes()
}

// app1 wraps a payload in an APP1 segment with a correct length.
func app1(payload []byte) []byte {
	seg := []byte{0xFF, 0xE1, 0, 0}
	binary.BigEndian.PutUint16(seg[2:], uint16(len(payload)+2))
	return append(seg, payload...)
}

func exifApp1(tiff []byte) []byte {
	return app1(append([]byte("Exif\x00\x00"), tiff...))
}

// jpegWithSegments inserts raw bytes directly after the SOI of a valid 1x1 JPEG.
func jpegWithSegments(tb testing.TB, segs ...[]byte) []byte {
	var jb bytes.Buffer
	if err := jpeg.Encode(&jb, image.NewRGBA(image.Rect(0, 0, 1, 1)), nil); err != nil {
		tb.Fatal(err)
	}
	raw := jb.Bytes()
	out := append([]byte{}, raw[:2]...)
	for _, s := range segs {
		out = append(out, s...)
	}
	return append(out, raw[2:]...)
}

// TestReadOrientationEdgeCases pairs every malformed input with a matched
// control of the same shape that is valid, so a refactor can't make both
// behave identically.
func TestReadOrientationEdgeCases(t *testing.T) {
	le := binary.LittleEndian
	orient := func(v uint16) []byte { return ifdEntry(le, exifOrientationTag, tiffTypeShort, 1, v) }
	other := ifdEntry(le, 0x010F, 2, 1, 0) // Make, irrelevant
	valid := tiffIFD0(1, orient(6))

	withIFDOffset := func(off uint32) []byte {
		b := append([]byte{}, valid...)
		le.PutUint32(b[4:], off)
		return b
	}

	// straddle returns a block with the given IFD offset where the byte at
	// 8 is 0x01. At offset 8 that is a valid count of 1 (little-endian) and
	// the entry at 10 is read. At offset 7 the count would overlap the
	// header, and only an explicit check stops the entry at 9 being read.
	straddle := func(off uint32) []byte {
		b := []byte("II\x2a\x00\x00\x00\x00\x00")
		le.PutUint32(b[4:], off)
		if off == 8 {
			b = append(b, 0x01, 0x00)
		} else {
			b = append(b, 0x01)
		}
		b = append(b, orient(6)...)
		return append(b, 0, 0, 0, 0)
	}

	tests := []struct {
		name string
		data []byte
		want int
	}{
		// IFD offset
		{"control: valid", jpegWithSegments(t, exifApp1(valid)), 6},
		{"ifd offset 0 (overlaps header)", jpegWithSegments(t, exifApp1(withIFDOffset(0))), 0},
		{"ifd offset 1", jpegWithSegments(t, exifApp1(withIFDOffset(1))), 0},
		{"ifd offset 7", jpegWithSegments(t, exifApp1(withIFDOffset(7))), 0},
		{"control: ifd offset 8", jpegWithSegments(t, exifApp1(withIFDOffset(8))), 6},
		{"ifd count straddles header", jpegWithSegments(t, exifApp1(straddle(7))), 0},
		{"control: same bytes, ifd after header", jpegWithSegments(t, exifApp1(straddle(8))), 6},
		{"ifd offset max uint32", jpegWithSegments(t, exifApp1(withIFDOffset(0xFFFFFFFF))), 0},
		{"ifd offset past segment", jpegWithSegments(t, exifApp1(withIFDOffset(uint32(len(valid))))), 0},
		{"ifd offset leaves 1 byte", jpegWithSegments(t, exifApp1(withIFDOffset(uint32(len(valid)-1)))), 0},

		// Entry counts and positions
		{"control: orientation second", jpegWithSegments(t, exifApp1(tiffIFD0(2, other, orient(3)))), 3},
		{"count overruns, orientation first", jpegWithSegments(t, exifApp1(tiffIFD0(0xFFFF, orient(6)))), 6},
		{"count overruns, no orientation", jpegWithSegments(t, exifApp1(tiffIFD0(0xFFFF, other))), 0},
		{"count zero", jpegWithSegments(t, exifApp1(tiffIFD0(0, orient(6)))), 0},
		{"control: entry fits exactly", jpegWithSegments(t, exifApp1(tiffIFD0(1, orient(8))[:8+2+12])), 8},
		{"entry truncated mid-value", jpegWithSegments(t, exifApp1(tiffIFD0(1, orient(8))[:8+2+10])), 0},

		// Linear-walk paths at their limits
		{"maximal APP1, orientation last", jpegWithSegments(t, maximalExif(8)), 8},
		{"long fill and RST run before APP1", jpegWithSegments(t, fillRun(1<<20), exifApp1(valid)), 6},
		{"control: long fill run, no Exif", jpegWithSegments(t, fillRun(1<<20)), 0},

		// Entry encoding
		{"wrong type LONG", jpegWithSegments(t, exifApp1(tiffIFD0(1, ifdEntry(le, exifOrientationTag, 4, 1, 6)))), 0},
		{"count 2", jpegWithSegments(t, exifApp1(tiffIFD0(1, ifdEntry(le, exifOrientationTag, tiffTypeShort, 2, 6)))), 0},
		{"malformed first wins over valid duplicate", jpegWithSegments(t, exifApp1(tiffIFD0(2, orient(9), orient(6)))), 0},

		// JPEG segment walking
		{"control: no fill byte", jpegWithSegments(t, exifApp1(valid)), 6},
		{"fill byte before APP1", jpegWithSegments(t, []byte{0xFF}, exifApp1(valid)), 6},
		{"RST marker before APP1", jpegWithSegments(t, []byte{0xFF, 0xD0}, exifApp1(valid)), 6},
		{"control: XMP only", jpegWithSegments(t, app1([]byte("http://ns.adobe.com/xap/1.0/\x00xmp"))), 0},
		{"XMP before Exif", jpegWithSegments(t, app1([]byte("http://ns.adobe.com/xap/1.0/\x00xmp")), exifApp1(valid)), 6},
		{"first Exif segment wins", jpegWithSegments(t, exifApp1(valid), exifApp1(tiffIFD0(1, orient(3)))), 6},
		{"bogus first Exif segment, no fallthrough", jpegWithSegments(t, exifApp1([]byte("XX*\x00\x08\x00\x00\x00")), exifApp1(valid)), 0},
		{"Exif after SOS ignored", jpegWithSegments(t, []byte{0xFF, 0xDA, 0x00, 0x02}, exifApp1(valid)), 0},
		{"segment length below 2", jpegWithSegments(t, []byte{0xFF, 0xE0, 0x00, 0x01}, exifApp1(valid)), 0},
		{"segment length overruns data", jpegWithSegments(t, []byte{0xFF, 0xE0, 0xFF, 0xFF}, exifApp1(valid)), 0},
		{"control: sized APPn before Exif", jpegWithSegments(t, []byte{0xFF, 0xE0, 0x00, 0x04, 0, 0}, exifApp1(valid)), 6},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := readOrientation(tt.data); got != tt.want {
				t.Errorf("got %d, want %d", got, tt.want)
			}
		})
	}
}

// maximalExif returns an APP1 Exif segment at the 64 KiB limit: IFD0 declares
// 0xFFFF entries, is packed with irrelevant ones, and holds the orientation
// entry last so that the reader has to walk the whole segment to find it.
func maximalExif(orientation uint16) []byte {
	le := binary.LittleEndian
	// 8 header + 2 count + n*12 entries + 4 next offset + 6 Exif header + 2 length <= 65535
	const entries = 5459
	list := make([][]byte, 0, entries)
	for range entries - 1 {
		list = append(list, ifdEntry(le, 0x010F, 2, 1, 0))
	}
	list = append(list, ifdEntry(le, exifOrientationTag, tiffTypeShort, 1, orientation))
	return exifApp1(tiffIFD0(0xFFFF, list...))
}

// fillRun returns n alternating fill bytes and RSTn markers, all of which the
// segment walker must skip one step at a time.
func fillRun(n int) []byte {
	b := make([]byte, 0, n)
	for len(b) < n {
		b = append(b, 0xFF, 0xFF, 0xD0)
	}
	return b
}

// exifSegmentEnd returns the offset in data at which the Exif TIFF payload
// ends, or -1 if there is no Exif segment. It relies on findExifSegment
// returning a sub-slice of data.
func exifSegmentEnd(data []byte) int {
	seg := findExifSegment(data)
	if seg == nil {
		return -1
	}
	return cap(data) - cap(seg) + len(seg)
}

func FuzzReadOrientation(f *testing.F) {
	le := binary.LittleEndian
	f.Add(jpegWithExif(f, tiffBlock(le, 6, 26, 8)))
	f.Add(jpegWithExif(f, tiffBlock(binary.BigEndian, 3, 0)))
	f.Add(jpegWithSegments(f, []byte{0xFF}, exifApp1(tiffIFD0(0xFFFF, ifdEntry(le, exifOrientationTag, tiffTypeShort, 1, 8)))))
	f.Add(jpegWithSegments(f, app1([]byte("http://ns.adobe.com/xap/1.0/\x00")), exifApp1(tiffBlock(le, 2, 0))))
	f.Add(jpegWithSegments(f, []byte{0xFF, 0xD0, 0xFF, 0xE0, 0x00, 0x01}))
	f.Add(jpegWithSegments(f, maximalExif(8)))
	f.Add(jpegWithSegments(f, fillRun(1<<16), exifApp1(tiffBlock(le, 6, 0))))

	f.Fuzz(func(t *testing.T, data []byte) {
		o := readOrientation(data)
		if o < 0 || o > 8 {
			t.Fatalf("orientation out of range: %d", o)
		}

		// The result must depend only on the Exif segment: dropping
		// everything after it, or appending junk, must not change it.
		if end := exifSegmentEnd(data); end >= 0 {
			if got := readOrientation(data[:end]); got != o {
				t.Fatalf("result changed when trailing data removed: %d != %d", got, o)
			}
			junk := append(append([]byte{}, data...), 0xFF, 0xE1, 0xFF, 0xFF, 0x00)
			if got := readOrientation(junk); got != o {
				t.Fatalf("result changed when trailing data appended: %d != %d", got, o)
			}
		} else if o != 0 {
			t.Fatalf("orientation %d returned without an Exif segment", o)
		}
	})
}
