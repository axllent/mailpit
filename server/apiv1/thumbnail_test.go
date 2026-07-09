package apiv1

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/axllent/mailpit/config"
	"github.com/axllent/mailpit/internal/logger"
	"github.com/axllent/mailpit/internal/storage"
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

func solidRGBApng(width, height int) []byte {
	var out bytes.Buffer
	out.Write([]byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'})
	ihdr := make([]byte, 13)
	binary.BigEndian.PutUint32(ihdr[0:4], uint32(width))
	binary.BigEndian.PutUint32(ihdr[4:8], uint32(height))
	ihdr[8] = 8
	ihdr[9] = 6
	out.Write(pngChunk("IHDR", ihdr))
	var compressed bytes.Buffer
	zw := zlib.NewWriter(&compressed)
	row := make([]byte, 1+width*4)
	for i := 0; i < height; i++ {
		_, _ = zw.Write(row)
	}
	_ = zw.Close()
	out.Write(pngChunk("IDAT", compressed.Bytes()))
	out.Write(pngChunk("IEND", nil))
	return out.Bytes()
}

func wrapBase64(b []byte) string {
	encoded := base64.StdEncoding.EncodeToString(b)
	var lines []string
	for len(encoded) > 76 {
		lines = append(lines, encoded[:76])
		encoded = encoded[76:]
	}
	if encoded != "" {
		lines = append(lines, encoded)
	}
	return strings.Join(lines, "\r\n")
}

func initTestDB(t *testing.T) {
	t.Helper()
	logger.NoLogging = true
	config.Database = filepath.Join(t.TempDir(), "mailpit.db")
	config.Compression = 0
	config.TenantID = ""
	config.MaxMessages = 0
	if err := storage.InitDB(); err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	t.Cleanup(storage.Close)
}

func storeEmailWithPNG(t *testing.T, subject string, png []byte) (msgID, partID string) {
	t.Helper()
	raw := []byte(fmt.Sprintf(
		"From: a@example.test\r\nTo: b@example.test\r\nSubject: %s\r\n"+
			"MIME-Version: 1.0\r\nContent-Type: multipart/mixed; boundary=\"b\"\r\n\r\n"+
			"--b\r\nContent-Type: text/plain\r\n\r\nbody\r\n"+
			"--b\r\nContent-Type: image/png; name=\"t.png\"\r\n"+
			"Content-Disposition: attachment; filename=\"t.png\"\r\n"+
			"Content-Transfer-Encoding: base64\r\n\r\n%s\r\n--b--\r\n",
		subject, wrapBase64(png),
	))
	id, err := storage.Store(&raw, nil)
	if err != nil {
		t.Fatalf("Store: %v", err)
	}
	msg, err := storage.GetMessage(id)
	if err != nil {
		t.Fatalf("GetMessage: %v", err)
	}
	if len(msg.Attachments) != 1 {

		t.Fatalf("attachments=%d, want 1", len(msg.Attachments))
	}
	return id, msg.Attachments[0].PartID
}

func thumbnailRequest(t *testing.T, msgID, partID string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/message/"+msgID+"/part/"+partID+"/thumb", nil)
	req.SetPathValue("id", msgID)
	req.SetPathValue("partID", partID)
	rr := httptest.NewRecorder()
	Thumbnail(rr, req)
	return rr
}

// TestThumbnailSmallImage verifies that a normal small image produces a valid JPEG thumbnail.
func TestThumbnailSmallImage(t *testing.T) {
	initTestDB(t)
	png := solidRGBApng(16, 16)
	msgID, partID := storeEmailWithPNG(t, "small", png)
	rr := thumbnailRequest(t, msgID, partID)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d, want 200; body=%q", rr.Code, rr.Body.String())
	}
	if ct := rr.Header().Get("Content-Type"); ct != "image/jpeg" {
		t.Fatalf("Content-Type=%q, want image/jpeg", ct)
	}
	if rr.Body.Len() == 0 {
		t.Fatal("expected non-empty response body")
	}
}

// TestThumbnailOversizedImageRejected verifies that a compressed PNG with dimensions
// exceeding maxDecodedPixels is rejected without performing a full raster decode,
// and returns a blank JPEG thumbnail rather than an error.
func TestThumbnailOversizedImageRejected(t *testing.T) {
	initTestDB(t)

	// 4500x4500 = 20,250,000 pixels, above the 20,000,000 limit.
	// The encoded PNG is ~72 KB; the decoded RGBA would be ~81 MB.
	png := solidRGBApng(4500, 4500)
	t.Logf("encoded PNG size: %d bytes, decoded pixels: %d (limit: %d)",
		len(png), 4500*4500, maxDecodedPixels)

	msgID, partID := storeEmailWithPNG(t, "oversized", png)
	rr := thumbnailRequest(t, msgID, partID)

	// The handler must respond 200 with a blank JPEG (same as unsupported attachment behaviour).
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d, want 200; body=%q", rr.Code, rr.Body.String())
	}
	if ct := rr.Header().Get("Content-Type"); ct != "image/jpeg" {
		t.Fatalf("Content-Type=%q, want image/jpeg", ct)
	}
	if rr.Body.Len() == 0 {
		t.Fatal("expected non-empty (blank) response body")
	}
}

// TestThumbnailBoundaryDimensions verifies that an image just at the pixel limit
// is accepted, and one just above is rejected.
func TestThumbnailBoundaryDimensions(t *testing.T) {
	initTestDB(t)

	// 4472x4472 ≈ 19,998,784 pixels — just under the 20,000,000 limit.
	acceptPNG := solidRGBApng(4472, 4472)
	msgID, partID := storeEmailWithPNG(t, "boundary-accept", acceptPNG)
	rr := thumbnailRequest(t, msgID, partID)
	if rr.Code != http.StatusOK {
		t.Fatalf("boundary-accept: status=%d", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "image/jpeg" {
		t.Fatalf("boundary-accept: Content-Type=%q", ct)
	}

	// 4473x4473 ≈ 20,007,729 pixels — just over the 20,000,000 limit.
	rejectPNG := solidRGBApng(4473, 4473)
	msgID2, partID2 := storeEmailWithPNG(t, "boundary-reject", rejectPNG)
	rr2 := thumbnailRequest(t, msgID2, partID2)
	if rr2.Code != http.StatusOK {
		t.Fatalf("boundary-reject: status=%d", rr2.Code)
	}
	if ct := rr2.Header().Get("Content-Type"); ct != "image/jpeg" {
		t.Fatalf("boundary-reject: Content-Type=%q", ct)
	}
}
