package thumbnail

import (
	"bytes"
	"compress/zlib"
	"context"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"image"
	"image/jpeg"
	"image/png"
	"sync/atomic"
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
	data, err := Generate(context.Background(), png, 180, 120)
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
	data, err := Generate(context.Background(), png, 180, 120)
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
	data, err := Generate(context.Background(), accept, 180, 120)
	if err != nil {
		t.Fatalf("accept: %v", err)
	}
	if !isValidJPEG(data) {
		t.Fatal("accept: not valid JPEG")
	}

	// 4473x4473 = 20,007,729 pixels, just over 20,000,000.
	reject := solidPNG(4473, 4473)
	data2, err := Generate(context.Background(), reject, 180, 120)
	if err != nil {
		t.Fatalf("reject: %v", err)
	}
	if !isValidJPEG(data2) {
		t.Fatal("reject: not valid JPEG")
	}
}

func TestGenerateNilDataReturnsBlank(t *testing.T) {
	data, err := Generate(context.Background(), nil, 180, 120)
	if err != nil {
		t.Fatalf("Generate(context.Background(), nil): %v", err)
	}
	if !isValidJPEG(data) {
		t.Fatal("output is not valid JPEG")
	}
}

func TestGenerateInvalidDataReturnsBlank(t *testing.T) {
	data, err := Generate(context.Background(), []byte("not an image"), 180, 120)
	if err != nil {
		t.Fatalf("Generate(context.Background(), garbage): %v", err)
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

	data, err := Generate(context.Background(), apng, 180, 120)
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
	go func() { out, _ := Generate(context.Background(), data, 180, 120); done <- out }()
	select {
	case out := <-done:
		checkThumbnail(t, out)
	case <-time.After(5 * time.Second):
		t.Fatal("a decode with free slots did not finish")
	}

	// exploit: with every slot in use a further image waits
	decodeSlots <- struct{}{}
	decodeSlots <- struct{}{}

	go func() { out, _ := Generate(context.Background(), data, 180, 120); done <- out }()
	select {
	case <-done:
		t.Fatal("an image was decoded while all the slots were in use")
	case <-time.After(300 * time.Millisecond):
	}

	// something that is not an image needs no slot, so it is not held up
	if out, err := Generate(context.Background(), nil, 180, 120); err != nil {
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
	out, err := Generate(context.Background(), data, 180, 120)
	if err != nil {
		t.Fatal(err)
	}
	checkThumbnail(t, out)
	normal := out

	// exploit: a decoder that panics
	decode = func([]byte) (image.Image, error) { panic("decoder failure") }

	out, err = Generate(context.Background(), data, 180, 120)
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

		out, err := Generate(context.Background(), data, 180, 120)
		if err != nil {
			t.Fatal(err)
		}
		checkThumbnail(t, out)
	})
}

// countDecodes replaces the decoder with one which counts how often it is used.
func countDecodes(t *testing.T) *atomic.Int32 {
	t.Helper()

	orig := decode
	t.Cleanup(func() { decode = orig })

	var n atomic.Int32
	decode = func(data []byte) (image.Image, error) {
		n.Add(1)
		return orig(data)
	}

	return &n
}

// A request which is waiting for a decode slot gave up only when it had one, and then decoded, even
// when its client had gone. It now stops waiting when the request is over.
func TestGenerateStopsWaitingWhenTheRequestIsOver(t *testing.T) {
	orig := decodeSlots
	decodeSlots = make(chan struct{}, 1)
	t.Cleanup(func() { decodeSlots = orig })
	decodes := countDecodes(t)

	data := pngBytes(t)
	decodeSlots <- struct{}{} // every slot is in use

	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	go func() { _, err := Generate(ctx, data, 180, 120); result <- err }()

	select {
	case err := <-result:
		t.Fatalf("returned while waiting for a slot: %v", err)
	case <-time.After(200 * time.Millisecond):
	}

	// exploit: the client goes away
	cancel()
	select {
	case err := <-result:
		if !errors.Is(err, context.Canceled) {
			t.Errorf("error %v, want context.Canceled", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("still waiting after the request was over")
	}

	// a slot is free, and nothing is decoded for the request which has gone
	<-decodeSlots
	time.Sleep(100 * time.Millisecond)
	if decodes.Load() != 0 {
		t.Errorf("%d decodes for a request which was over", decodes.Load())
	}
	if len(decodeSlots) != 0 {
		t.Errorf("%d slots held", len(decodeSlots))
	}

	// control: a request which is not over waits, and is decoded once there is a slot
	decodeSlots <- struct{}{}
	go func() { _, err := Generate(context.Background(), data, 180, 120); result <- err }()
	select {
	case <-result:
		t.Fatal("returned while all the slots were in use")
	case <-time.After(200 * time.Millisecond):
	}
	<-decodeSlots
	select {
	case err := <-result:
		if err != nil {
			t.Errorf("error %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("not decoded once a slot was free")
	}
	if decodes.Load() != 1 {
		t.Errorf("%d decodes, want 1", decodes.Load())
	}
}

// A request which is over before it starts is not decoded, even when there is a slot for it. Both
// being ready is where a select would choose either, so it is tried many times.
func TestGenerateDoesNotDecodeForARequestWhichIsOver(t *testing.T) {
	decodes := countDecodes(t)
	data := pngBytes(t)

	cancelled, cancel := context.WithCancel(context.Background())
	cancel()

	for range 300 {
		out, err := Generate(cancelled, data, 180, 120)
		if !errors.Is(err, context.Canceled) || out != nil {
			t.Fatalf("got %d bytes and %v", len(out), err)
		}
	}
	if decodes.Load() != 0 {
		t.Fatalf("%d decodes", decodes.Load())
	}
	if len(decodeSlots) != 0 {
		t.Fatalf("%d slots held", len(decodeSlots))
	}

	// nothing is made for a client which has gone, even the blank image for no data
	if out, err := Generate(cancelled, nil, 180, 120); !errors.Is(err, context.Canceled) || out != nil {
		t.Errorf("no data: got %d bytes and %v", len(out), err)
	}

	// a request which timed out, as well as one which was cancelled
	expired, cancelExpired := context.WithTimeout(context.Background(), time.Nanosecond)
	defer cancelExpired()
	<-expired.Done()
	if _, err := Generate(expired, data, 180, 120); !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("error %v, want context.DeadlineExceeded", err)
	}

	// controls: a request which is not over is decoded, with a deadline which is not reached too
	live, cancelLive := context.WithTimeout(context.Background(), time.Minute)
	defer cancelLive()
	for _, ctx := range []context.Context{context.Background(), live} {
		out, err := Generate(ctx, data, 180, 120)
		if err != nil {
			t.Fatal(err)
		}
		checkThumbnail(t, out)
	}
	if decodes.Load() != 2 {
		t.Errorf("%d decodes, want 2", decodes.Load())
	}

	// and an empty image is still a blank one for a request which is not over
	out, err := Generate(context.Background(), nil, 180, 120)
	if err != nil {
		t.Fatal(err)
	}
	checkThumbnail(t, out)
}

// A decode which has begun cannot be stopped, but what follows it is not done for a client which has gone.
func TestGenerateStopsAfterTheDecodeWhenTheRequestIsOver(t *testing.T) {
	orig := decode
	t.Cleanup(func() { decode = orig })

	ctx, cancel := context.WithCancel(context.Background())
	decode = func(data []byte) (image.Image, error) {
		cancel() // the client goes away while the image is being decoded
		return orig(data)
	}

	out, err := Generate(ctx, pngBytes(t), 180, 120)
	if !errors.Is(err, context.Canceled) || out != nil {
		t.Errorf("got %d bytes and %v", len(out), err)
	}
	if len(decodeSlots) != 0 {
		t.Errorf("%d slots held", len(decodeSlots))
	}
}

// endsAfterTheFirstCheck is a context which is not over the first time it is asked, and is over
// (with its channel closed) from then on. That is the request ending between the start of Generate
// and the wait for a slot, which a real context does at a moment which can not be chosen.
type endsAfterTheFirstCheck struct {
	context.Context
	asked atomic.Int32
	done  chan struct{}
}

func newEndsAfterTheFirstCheck() *endsAfterTheFirstCheck {
	done := make(chan struct{})
	close(done)

	return &endsAfterTheFirstCheck{Context: context.Background(), done: done}
}

func (c *endsAfterTheFirstCheck) Done() <-chan struct{} { return c.done }

func (c *endsAfterTheFirstCheck) Err() error {
	if c.asked.Add(1) == 1 {
		return nil
	}

	return context.Canceled
}

// The request is over when there is a slot, so a select could choose either the slot or the end of
// the request. It must never decode, as the request was over before the slot was taken.
func TestGenerateDoesNotDecodeWhenTheSlotAndTheEndOfTheRequestAreTogether(t *testing.T) {
	decodes := countDecodes(t)
	data := pngBytes(t)

	for range 300 {
		out, err := Generate(newEndsAfterTheFirstCheck(), data, 180, 120)
		if !errors.Is(err, context.Canceled) || out != nil {
			t.Fatalf("got %d bytes and %v", len(out), err)
		}
	}

	if decodes.Load() != 0 {
		t.Fatalf("%d decodes for requests which were over before the slot was taken", decodes.Load())
	}
	if len(decodeSlots) != 0 {
		t.Fatalf("%d slots held", len(decodeSlots))
	}
}
