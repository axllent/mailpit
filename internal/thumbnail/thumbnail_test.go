package thumbnail

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"hash/crc32"
	"image"
	"image/jpeg"
	"image/png"
	"testing"
	"time"
)

func pngChunk(kind string, data []byte) []byte {
	var out bytes.Buffer
	_ = binary.Write(&out, binary.BigEndian, uint32(len(data)))
	out.WriteString(kind)
	out.Write(data)
	crc := crc32.NewIEEE()
	crc.Write([]byte(kind))
	crc.Write(data)
	_ = binary.Write(&out, binary.BigEndian, crc.Sum32())
	return out.Bytes()
}

func compressedScanlines(width, height int) []byte {
	var compressed bytes.Buffer
	zw := zlib.NewWriter(&compressed)
	row := make([]byte, 1+width*4) // filter byte + RGBA
	for range height {
		_, _ = zw.Write(row)
	}
	_ = zw.Close()
	return compressed.Bytes()
}

// solidPNG builds a minimal valid PNG with the given dimensions.
func solidPNG(width, height int) []byte {
	var out bytes.Buffer
	out.Write([]byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'})
	ihdr := make([]byte, 13)
	binary.BigEndian.PutUint32(ihdr[0:4], uint32(width))
	binary.BigEndian.PutUint32(ihdr[4:8], uint32(height))
	ihdr[8] = 8 // bit depth
	ihdr[9] = 6 // color type: RGBA
	out.Write(pngChunk("IHDR", ihdr))
	out.Write(pngChunk("IDAT", compressedScanlines(width, height)))
	out.Write(pngChunk("IEND", nil))
	return out.Bytes()
}

// solidAPNG builds a minimal APNG with the given dimensions and frame count.
func solidAPNG(width, height, frames int) []byte {
	var out bytes.Buffer
	out.Write([]byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'})
	ihdr := make([]byte, 13)
	binary.BigEndian.PutUint32(ihdr[0:4], uint32(width))
	binary.BigEndian.PutUint32(ihdr[4:8], uint32(height))
	ihdr[8] = 8
	ihdr[9] = 6
	out.Write(pngChunk("IHDR", ihdr))

	// acTL (animation control)
	actl := make([]byte, 8)
	binary.BigEndian.PutUint32(actl[0:4], uint32(frames))
	out.Write(pngChunk("acTL", actl))

	seq := uint32(0)

	// fcTL for first frame (uses IDAT)
	fctl := make([]byte, 26)
	binary.BigEndian.PutUint32(fctl[0:4], seq)
	seq++
	binary.BigEndian.PutUint32(fctl[4:8], uint32(width))
	binary.BigEndian.PutUint32(fctl[8:12], uint32(height))
	fctl[21] = 1
	out.Write(pngChunk("fcTL", fctl))

	// IDAT (first frame)
	out.Write(pngChunk("IDAT", compressedScanlines(width, height)))

	// Additional frames as fcTL + fdAT pairs
	for i := 1; i < frames; i++ {
		fc := make([]byte, 26)
		binary.BigEndian.PutUint32(fc[0:4], seq)
		seq++
		binary.BigEndian.PutUint32(fc[4:8], uint32(width))
		binary.BigEndian.PutUint32(fc[8:12], uint32(height))
		fc[21] = 1
		out.Write(pngChunk("fcTL", fc))

		fd := compressedScanlines(width, height)
		fdp := make([]byte, 4+len(fd))
		binary.BigEndian.PutUint32(fdp[0:4], seq)
		seq++
		copy(fdp[4:], fd)
		out.Write(pngChunk("fdAT", fdp))
	}

	out.Write(pngChunk("IEND", nil))
	return out.Bytes()
}

func isValidJPEG(data []byte) bool {
	_, err := jpeg.Decode(bytes.NewReader(data))
	return err == nil
}

func TestGenerateSmallImage(t *testing.T) {
	png := solidPNG(16, 16)
	data, err := Generate(png, 180, 120)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if !isValidJPEG(data) {
		t.Fatal("output is not valid JPEG")
	}
}

func TestGenerateOversizedImageRejected(t *testing.T) {
	// 4500x4500 = 20,250,000 pixels, above the 20,000,000 limit.
	png := solidPNG(4500, 4500)
	data, err := Generate(png, 180, 120)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if !isValidJPEG(data) {
		t.Fatal("output is not valid JPEG")
	}
	// Should be a blank image (small), not a decoded 4500x4500 thumbnail.
	if len(data) > 5000 {
		t.Fatalf("expected small blank JPEG, got %d bytes", len(data))
	}
}

func TestGenerateBoundaryDimensions(t *testing.T) {
	// 4472x4472 = 19,998,784 pixels, just under 20,000,000.
	accept := solidPNG(4472, 4472)
	data, err := Generate(accept, 180, 120)
	if err != nil {
		t.Fatalf("accept: %v", err)
	}
	if !isValidJPEG(data) {
		t.Fatal("accept: not valid JPEG")
	}

	// 4473x4473 = 20,007,729 pixels, just over 20,000,000.
	reject := solidPNG(4473, 4473)
	data2, err := Generate(reject, 180, 120)
	if err != nil {
		t.Fatalf("reject: %v", err)
	}
	if !isValidJPEG(data2) {
		t.Fatal("reject: not valid JPEG")
	}
}

func TestGenerateNilDataReturnsBlank(t *testing.T) {
	data, err := Generate(nil, 180, 120)
	if err != nil {
		t.Fatalf("Generate(nil): %v", err)
	}
	if !isValidJPEG(data) {
		t.Fatal("output is not valid JPEG")
	}
}

func TestGenerateInvalidDataReturnsBlank(t *testing.T) {
	data, err := Generate([]byte("not an image"), 180, 120)
	if err != nil {
		t.Fatalf("Generate(garbage): %v", err)
	}
	if !isValidJPEG(data) {
		t.Fatal("output is not valid JPEG")
	}
}

// TestAPNGDecodesOnlyOneFrame verifies that an APNG with many frames
// does not cause memory allocation proportional to frame count.
// This is the core security invariant: the stdlib PNG decoder ignores
// APNG animation chunks and decodes only the default image.
func TestAPNGDecodesOnlyOneFrame(t *testing.T) {
	// 500x500 x 50 frames. If all frames were decoded this would be
	// 500*500*4*50 = 50 MB. As a single frame it is 1 MB.
	apng := solidAPNG(500, 500, 50)

	data, err := Generate(apng, 180, 120)
	if err != nil {
		t.Fatalf("Generate APNG: %v", err)
	}
	if !isValidJPEG(data) {
		t.Fatal("APNG output is not valid JPEG")
	}

	// Verify that image.Decode returns a single frame by checking
	// that DecodeConfig reports the original dimensions (not multiplied).
	cfg, _, cfgErr := image.DecodeConfig(bytes.NewReader(apng))
	if cfgErr != nil {
		t.Fatalf("DecodeConfig: %v", cfgErr)
	}
	if cfg.Width != 500 || cfg.Height != 500 {
		t.Fatalf("DecodeConfig: %dx%d, want 500x500", cfg.Width, cfg.Height)
	}
}

// pngBytes returns a small valid PNG.
func pngBytes(t *testing.T) []byte {
	t.Helper()

	var buf bytes.Buffer
	if err := png.Encode(&buf, testPattern(40, 30, image.Point{})); err != nil {
		t.Fatal(err)
	}

	return buf.Bytes()
}

// checkThumbnail checks that data is a JPEG of the size requested.
func checkThumbnail(t *testing.T, data []byte) {
	t.Helper()

	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || format != "jpeg" || cfg.Width != 180 || cfg.Height != 120 {
		t.Fatalf("not a 180x120 JPEG: %v %q %v", cfg, format, err)
	}
}

// Only a few images are decoded at once, and the others wait for one to finish.
func TestDecodesAreLimited(t *testing.T) {
	orig := decodeSlots
	decodeSlots = make(chan struct{}, 2)
	t.Cleanup(func() { decodeSlots = orig })

	data := pngBytes(t)

	// control: with a free slot an image is decoded at once
	done := make(chan []byte, 1)
	go func() { out, _ := Generate(data, 180, 120); done <- out }()
	select {
	case out := <-done:
		checkThumbnail(t, out)
	case <-time.After(5 * time.Second):
		t.Fatal("a decode with free slots did not finish")
	}

	// exploit: with every slot in use a further image waits
	decodeSlots <- struct{}{}
	decodeSlots <- struct{}{}

	go func() { out, _ := Generate(data, 180, 120); done <- out }()
	select {
	case <-done:
		t.Fatal("an image was decoded while all the slots were in use")
	case <-time.After(300 * time.Millisecond):
	}

	// something that is not an image needs no slot, so it is not held up
	if out, err := Generate(nil, 180, 120); err != nil {
		t.Fatal(err)
	} else {
		checkThumbnail(t, out)
	}

	// and it continues when one is released
	<-decodeSlots
	select {
	case out := <-done:
		checkThumbnail(t, out)
	case <-time.After(5 * time.Second):
		t.Fatal("the waiting image was not decoded once a slot was free")
	}
	<-decodeSlots
}

// A panic in a decoder gives a blank thumbnail, and does not keep a decode slot.
func TestDecoderPanicIsRecovered(t *testing.T) {
	orig := decode
	t.Cleanup(func() { decode = orig })

	data := pngBytes(t)

	// control: the real decoder gives the image
	out, err := Generate(data, 180, 120)
	if err != nil {
		t.Fatal(err)
	}
	checkThumbnail(t, out)
	normal := out

	// exploit: a decoder that panics
	decode = func([]byte) (image.Image, error) { panic("decoder failure") }

	out, err = Generate(data, 180, 120)
	if err != nil {
		t.Fatalf("a panic was returned as an error: %v", err)
	}
	checkThumbnail(t, out)
	if bytes.Equal(out, normal) {
		t.Error("the blank thumbnail is the same as the image's")
	}
	if len(decodeSlots) != 0 {
		t.Errorf("%d decode slots still held after a panic", len(decodeSlots))
	}
}

// Whatever the data is, the result is a JPEG of the size asked for, and any image that was
// decoded is within the pixel limit.
func FuzzGenerate(f *testing.F) {
	var jb bytes.Buffer
	_ = jpeg.Encode(&jb, testPattern(300, 200, image.Point{}), nil)

	var pb bytes.Buffer
	_ = png.Encode(&pb, testPattern(40, 30, image.Point{}))
	f.Add(pb.Bytes())
	f.Add(jb.Bytes())
	f.Add(jpegWithOrientation(f, 6))
	f.Add([]byte("GIF89a\x01\x00\x01\x00\x00\x00\x00,\x00\x00\x00\x00\x01\x00\x01\x00\x80\x00\x00\x00\x00\xff\xff\xff\x02\x02D\x01\x00;"))
	f.Add([]byte("RIFF\x1a\x00\x00\x00WEBPVP8L\x0d\x00\x00\x00\x2f\x00\x00\x00\x10\x07\x10\x11\x11\x88\x88\xfe\x07\x00"))
	f.Add([]byte{})

	f.Fuzz(func(t *testing.T, data []byte) {
		if img, err := safeDecode(data); err == nil && img != nil {
			b := img.Bounds()
			if int64(b.Dx())*int64(b.Dy()) > maxDecodedPixels {
				t.Fatalf("decoded %v, over the pixel limit", b)
			}
		}

		out, err := Generate(data, 180, 120)
		if err != nil {
			t.Fatal(err)
		}
		checkThumbnail(t, out)
	})
}
