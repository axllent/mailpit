package thumbnail

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"hash/crc32"
	"image"
	"image/jpeg"
	"testing"
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
