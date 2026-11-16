package apiv1

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/axllent/mailpit/internal/storage"
)

// A From address with a line break in it was stored in the Return-Path header as it was given, which let it add
// headers to the message (or, with an empty line, end the header block, so that the real headers were the body).
// It is refused now, as an address of the recipients is. The controls (a plain address, and a quoted local part) are
// stored with their Return-Path as they always were.
func TestSendFromAddressLineBreaks(t *testing.T) {
	initTestDB(t)

	post := func(from string) (*httptest.ResponseRecorder, string) {
		body, _ := json.Marshal(map[string]any{
			"From": map[string]string{"Email": from}, "To": []map[string]string{{"Email": "c@d.test"}},
			"Subject": "from line breaks", "Text": "t",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/send", strings.NewReader(string(body)))
		req.RemoteAddr = "127.0.0.1:1"
		w := httptest.NewRecorder()
		SendMessageHandler(w, req)

		var res struct{ ID string }
		_ = json.Unmarshal(w.Body.Bytes(), &res)

		return w, res.ID
	}

	for _, from := range []string{
		"a@b.test\r\nX-Injected: proof",
		"a@b.test\nX-Injected: proof",
		"a@b.test\r\n\r\nfake body",
		"a@b.test\rX-Injected: proof",
		"",
		"not an address",
	} {
		w, id := post(from)
		if w.Code == http.StatusOK || id != "" {
			t.Errorf("From %q was accepted: status %d, body %q", from, w.Code, w.Body.String())
		}
	}

	for from, returnPath := range map[string]string{
		"a@b.test":          "Return-Path: <a@b.test>",
		`"odd user"@b.test`: `Return-Path: <"odd user"@b.test>`,
	} {
		w, id := post(from)
		if w.Code != http.StatusOK || id == "" {
			t.Fatalf("control From %q was refused: status %d, body %q", from, w.Code, w.Body.String())
		}

		raw, err := storage.GetMessageRaw(id)
		if err != nil {
			t.Fatalf("control From %q: %v", from, err)
		}
		if !strings.Contains(string(raw), returnPath+"\r\n") {
			t.Errorf("control From %q: the message has no %q:\n%s", from, returnPath, raw)
		}
		if strings.Contains(string(raw), "X-Injected") {
			t.Errorf("control From %q: the message has an injected header:\n%s", from, raw)
		}
	}
}
