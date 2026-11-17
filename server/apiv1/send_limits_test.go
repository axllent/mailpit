package apiv1

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/axllent/mailpit/config"
	"github.com/axllent/mailpit/internal/storage"
	"github.com/axllent/mailpit/internal/tools"
)

// sendLimits sets the maximum message size (in MiB), and the budget (in bytes) and how long a request waits for it,
// for the length of the test.
func sendLimits(t *testing.T, maxMiB int, budget int64, wait time.Duration) {
	t.Helper()

	origMax, origBudget, origWait := config.MaxMessageSize, sendBudget, sendWait
	config.MaxMessageSize, sendBudget, sendWait = maxMiB, tools.NewBudget(budget), wait
	t.Cleanup(func() { config.MaxMessageSize, sendBudget, sendWait = origMax, origBudget, origWait })
}

// budgetFree is whether the whole of the budget can be taken, which is so when nothing has it reserved.
func budgetFree(b *tools.Budget, size int64) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	release, err := b.Acquire(ctx, size)
	if err != nil {
		return false
	}
	release()

	return true
}

// sendJSON is the body of a request to send a message with a text of the size given (in lines, so that the message
// is not made larger by the way it is encoded), padded with white space to the length given (0 for none).
func sendJSON(textBytes, padTo int) []byte {
	text := strings.Repeat(strings.Repeat("x", 60)+"\n", textBytes/61+1)[:textBytes]
	body, _ := json.Marshal(map[string]any{
		"From": map[string]string{"Email": "a@b.test"}, "To": []map[string]string{{"Email": "c@d.test"}},
		"Subject": "s", "Text": text,
	})
	if len(body) < padTo {
		body = append(body, bytes.Repeat([]byte(" "), padTo-len(body))...)
	}

	return body
}

func sendRequest(body io.Reader, contentLength int64) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/send", body)
	req.RemoteAddr = "127.0.0.1:1"
	req.ContentLength = contentLength

	return req
}

func doSend(req *http.Request) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	SendMessageHandler(w, req)

	return w
}

func errorOf(w *httptest.ResponseRecorder) string {
	var res struct{ Error string }
	_ = json.Unmarshal(w.Body.Bytes(), &res)

	return res.Error
}

// failingReader fails the test if it is read.
type failingReader struct{ t *testing.T }

func (f failingReader) Read([]byte) (int, error) {
	f.t.Error("the body of the request was read")
	return 0, io.EOF
}

// gatedReader gives its data once the gate is opened.
type gatedReader struct {
	r    io.Reader
	gate chan struct{}
}

func (g *gatedReader) Read(p []byte) (int, error) {
	<-g.gate
	return g.r.Read(p)
}

// A request which says that it is larger than the maximum message size is refused without any of it being read, as
// it was not (the limit was only applied as it was read). One which is not larger (the control) is sent.
func TestSendRefusesALargeRequestWithoutReadingIt(t *testing.T) {
	initTestDB(t)
	sendLimits(t, 1, 50<<20, time.Second)

	limit := int64(1 << 20)

	for _, n := range []int64{limit + 1, 2 * limit, 100 * limit} {
		w := doSend(sendRequest(failingReader{t}, n))
		if w.Code != http.StatusRequestEntityTooLarge || !strings.Contains(errorOf(w), "maximum message size of 1 MiB") {
			t.Fatalf("%d bytes: status %d: %s", n, w.Code, w.Body.String())
		}
		if w.Header().Get("Connection") != "close" || !strings.HasPrefix(w.Header().Get("Content-Type"), "application/json") {
			t.Fatalf("headers %v", w.Header())
		}
		// nothing was reserved for it
		if !budgetFree(sendBudget, 50<<20) {
			t.Fatal("a request which was refused has the budget")
		}
	}

	// control: a request of the size it can be is sent
	body := sendJSON(900<<10, 0)
	if w := doSend(sendRequest(bytes.NewReader(body), int64(len(body)))); w.Code != http.StatusOK {
		t.Fatalf("a request of %d bytes: status %d: %s", len(body), w.Code, w.Body.String())
	}
}

// A request which does not say how large it is (a chunked one) is read up to the limit, and refused when it is more.
func TestSendRefusesALargeChunkedRequest(t *testing.T) {
	initTestDB(t)
	sendLimits(t, 1, 50<<20, time.Second)

	large := sendJSON(1100<<10, 0)
	w := doSend(sendRequest(bytes.NewReader(large), -1))
	if w.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("%d bytes: status %d: %s", len(large), w.Code, w.Body.String())
	}

	// control: a small one is sent
	small := sendJSON(1000, 0)
	if w := doSend(sendRequest(bytes.NewReader(small), -1)); w.Code != http.StatusOK {
		t.Fatalf("status %d: %s", w.Code, w.Body.String())
	}

	// and one which says it is smaller than it is is not read beyond the limit either
	w = doSend(sendRequest(bytes.NewReader(large), 1000))
	if w.Code == http.StatusOK {
		t.Fatal("a request which was larger than it said was sent")
	}
}

// The message which is made is no larger than the maximum message size either, as the SMTP server does not accept
// one: a request which is below the limit makes one which is over it, as the attachments are base64 in it as well
// (with line breaks). Below it (the control) it is sent.
func TestSendRefusesAMessageWhichIsTooLargeOnceMade(t *testing.T) {
	initTestDB(t)
	sendLimits(t, 1, 50<<20, time.Second)

	send := func(attachment int) *httptest.ResponseRecorder {
		body, _ := json.Marshal(map[string]any{
			"From": map[string]string{"Email": "a@b.test"}, "To": []map[string]string{{"Email": "c@d.test"}},
			"Subject": "s", "Text": "t",
			"Attachments": []map[string]string{{"Filename": "a.bin", "Content": base64.StdEncoding.EncodeToString(make([]byte, attachment))}},
		})
		w := doSend(sendRequest(bytes.NewReader(body), int64(len(body))))
		if config.MaxMessageSize > 0 && w.Code != http.StatusRequestEntityTooLarge && len(body) > 1<<20 {
			t.Fatalf("a request of %d bytes was not refused", len(body))
		}

		return w
	}

	before := storage.CountTotal()

	// a request of 1,048,3xx bytes (below 1 MiB), which is a message of about 1,075,000
	w := send(3 * (1<<20 - 500) / 4)
	if w.Code != http.StatusRequestEntityTooLarge || !strings.Contains(errorOf(w), "larger than the maximum message size of 1 MiB") {
		t.Fatalf("status %d: %s", w.Code, w.Body.String())
	}
	if storage.CountTotal() != before {
		t.Fatal("a message which was too large was stored")
	}

	// control: below it
	if w := send(600 << 10); w.Code != http.StatusOK {
		t.Fatalf("status %d: %s", w.Code, w.Body.String())
	}
	if storage.CountTotal() != before+1 {
		t.Fatal("the message was not stored")
	}

	// and no limit at all is no limit
	config.MaxMessageSize = 0
	if w := send(2 << 20); w.Code != http.StatusOK {
		t.Fatalf("without a limit: status %d: %s", w.Code, w.Body.String())
	}
}

// Only so many are handled at once: a request which does not fit waits, and is told to try again if it still does not
// fit after that. Once the one which is being handled is done there is room (the control).
func TestSendBudgetMakesRequestsWait(t *testing.T) {
	initTestDB(t)
	sendLimits(t, 1, 1<<20, 200*time.Millisecond)

	// A is half of the budget, and is being read (which does not finish until it is allowed to)
	bodyA := sendJSON(1000, 512<<10)
	a := &gatedReader{r: bytes.NewReader(bodyA), gate: make(chan struct{})}
	doneA := make(chan *httptest.ResponseRecorder, 1)
	go func() { doneA <- doSend(sendRequest(a, int64(len(bodyA)))) }()
	time.Sleep(100 * time.Millisecond)

	// B does not fit with it: it waits, and does not have its body read
	start := time.Now()
	w := doSend(sendRequest(failingReader{t}, 600<<10))
	if w.Code != http.StatusServiceUnavailable || w.Header().Get("Retry-After") != "0" {
		t.Fatalf("status %d (%v): %s", w.Code, w.Header(), w.Body.String())
	}
	if took := time.Since(start); took < 150*time.Millisecond || took > 2*time.Second {
		t.Fatalf("it waited %v, want about 200ms", took)
	}

	// control: one which does fit with it is let in at once
	small := sendJSON(1000, 100<<10)
	if w := doSend(sendRequest(bytes.NewReader(small), int64(len(small)))); w.Code != http.StatusOK {
		t.Fatalf("one which fits: status %d: %s", w.Code, w.Body.String())
	}

	// A is finished, and what it had is given back
	close(a.gate)
	if w := <-doneA; w.Code != http.StatusOK {
		t.Fatalf("A: status %d: %s", w.Code, w.Body.String())
	}
	if !budgetFree(sendBudget, 1<<20) {
		t.Fatal("the budget is still reserved when nothing is being sent")
	}

	// and B is let in now
	bodyB := sendJSON(1000, 600<<10)
	if w := doSend(sendRequest(bytes.NewReader(bodyB), int64(len(bodyB)))); w.Code != http.StatusOK {
		t.Fatalf("B once there was room: status %d: %s", w.Code, w.Body.String())
	}
}

// A request is waiting for its turn is let in when there is room, in the order they came, without being told to try
// again.
func TestSendBudgetLetsWaitingRequestsIn(t *testing.T) {
	initTestDB(t)
	sendLimits(t, 1, 1<<20, 5*time.Second)

	bodyA := sendJSON(1000, 800<<10)
	a := &gatedReader{r: bytes.NewReader(bodyA), gate: make(chan struct{})}
	doneA := make(chan *httptest.ResponseRecorder, 1)
	go func() { doneA <- doSend(sendRequest(a, int64(len(bodyA)))) }()
	time.Sleep(100 * time.Millisecond)

	bodyB := sendJSON(1000, 800<<10)
	doneB := make(chan *httptest.ResponseRecorder, 1)
	go func() { doneB <- doSend(sendRequest(bytes.NewReader(bodyB), int64(len(bodyB)))) }()

	select {
	case <-doneB:
		t.Fatal("B was handled while A had the budget")
	case <-time.After(300 * time.Millisecond):
	}

	close(a.gate)
	for name, ch := range map[string]chan *httptest.ResponseRecorder{"A": doneA, "B": doneB} {
		select {
		case w := <-ch:
			if w.Code != http.StatusOK {
				t.Fatalf("%s: status %d: %s", name, w.Code, w.Body.String())
			}
		case <-time.After(10 * time.Second):
			t.Fatalf("%s was not handled", name)
		}
	}
}

// A request which is larger than the budget (when the maximum message size is) is the only one: it has all of it.
func TestSendBudgetForARequestLargerThanIt(t *testing.T) {
	initTestDB(t)
	sendLimits(t, 4, 1<<20, 200*time.Millisecond)

	big := sendJSON(2<<20, 0)
	a := &gatedReader{r: bytes.NewReader(big), gate: make(chan struct{})}
	doneA := make(chan *httptest.ResponseRecorder, 1)
	go func() { doneA <- doSend(sendRequest(a, int64(len(big)))) }()
	time.Sleep(100 * time.Millisecond)

	// nothing else fits, however small
	if w := doSend(sendRequest(failingReader{t}, 1000)); w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status %d while a request larger than the budget was handled", w.Code)
	}

	close(a.gate)
	if w := <-doneA; w.Code != http.StatusOK {
		t.Fatalf("status %d: %s", w.Code, w.Body.String())
	}

	// control: and when it is done they do
	small := sendJSON(1000, 0)
	if w := doSend(sendRequest(bytes.NewReader(small), int64(len(small)))); w.Code != http.StatusOK {
		t.Fatalf("status %d: %s", w.Code, w.Body.String())
	}
}

// A request which does not say how large it is has the maximum message size reserved for it, or all of the budget if there
// is no maximum.
func TestSendBudgetForARequestOfUnknownSize(t *testing.T) {
	for name, tc := range map[string]struct {
		maxMiB int
		fits   int64 // what is let in with it
	}{
		"a maximum message size": {1, 1},
		"no maximum":             {0, 1},
	} {
		t.Run(name, func(t *testing.T) {
			initTestDB(t)
			sendLimits(t, tc.maxMiB, 2<<20, 200*time.Millisecond)

			body := sendJSON(1000, 0)
			a := &gatedReader{r: bytes.NewReader(body), gate: make(chan struct{})}
			doneA := make(chan *httptest.ResponseRecorder, 1)
			go func() { doneA <- doSend(sendRequest(a, -1)) }()
			time.Sleep(100 * time.Millisecond)

			// with a maximum of 1 MiB there is 1 MiB left, and with none there is nothing
			fits := doSend(sendRequest(bytes.NewReader(body), int64(len(body))))
			want := http.StatusServiceUnavailable
			if tc.maxMiB == 1 {
				want = http.StatusOK
			}
			if fits.Code != want {
				t.Fatalf("status %d for a small one, want %d", fits.Code, want)
			}
			if tc.maxMiB == 1 {
				// but not another of the size it can be
				if w := doSend(sendRequest(failingReader{t}, 1<<20+1)); w.Code != http.StatusRequestEntityTooLarge {
					t.Fatalf("status %d", w.Code)
				}
				if w := doSend(sendRequest(failingReader{t}, 1100<<10)); w.Code != http.StatusRequestEntityTooLarge {
					t.Fatalf("status %d", w.Code)
				}
			}

			close(a.gate)
			if w := <-doneA; w.Code != http.StatusOK {
				t.Fatalf("status %d: %s", w.Code, w.Body.String())
			}
		})
	}
}

// What was reserved is given back whatever way the request ends.
func TestSendBudgetIsGivenBack(t *testing.T) {
	initTestDB(t)
	sendLimits(t, 1, 1<<20, 200*time.Millisecond)

	good := sendJSON(1000, 0)
	tooLarge, _ := json.Marshal(map[string]any{
		"From": map[string]string{"Email": "a@b.test"}, "To": []map[string]string{{"Email": "c@d.test"}}, "Subject": "s", "Text": "t",
		"Attachments": []map[string]string{{"Filename": "a.bin", "Content": base64.StdEncoding.EncodeToString(make([]byte, 786000))}},
	})

	for name, tc := range map[string]struct {
		body   io.Reader
		length int64
		code   int
	}{
		"sent":                         {bytes.NewReader(good), int64(len(good)), http.StatusOK},
		"not JSON":                     {strings.NewReader(strings.Repeat("x", 5000)), 5000, http.StatusBadRequest},
		"empty":                        {strings.NewReader(""), 0, http.StatusBadRequest},
		"the body ends early":          {bytes.NewReader(good[:100]), int64(len(good)), http.StatusBadRequest},
		"the body can not be read":     {io.MultiReader(bytes.NewReader(good[:100]), errReader{}), int64(len(good)), http.StatusBadRequest},
		"an address which is invalid":  {strings.NewReader(`{"From":{"Email":"a@b.test"},"To":[{"Email":"nope"}]}`), 58, http.StatusBadRequest},
		"a message which is too large": {bytes.NewReader(tooLarge), int64(len(tooLarge)), http.StatusRequestEntityTooLarge},
		"chunked":                      {bytes.NewReader(good), -1, http.StatusOK},
		"chunked and too large":        {bytes.NewReader(sendJSON(1100<<10, 0)), -1, http.StatusRequestEntityTooLarge},
	} {
		w := doSend(sendRequest(tc.body, tc.length))
		if w.Code != tc.code {
			t.Errorf("%s: status %d, want %d: %s", name, w.Code, tc.code, w.Body.String())
		}
		if !budgetFree(sendBudget, 1<<20) {
			t.Errorf("%s: the budget is still reserved", name)
		}
	}

	// a request whose client has gone while it waits does not have it either (and nothing is sent to the client)
	a := &gatedReader{r: bytes.NewReader(good), gate: make(chan struct{})}
	doneA := make(chan struct{})
	go func() { doSend(sendRequest(a, 1<<20)); close(doneA) }()
	time.Sleep(100 * time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	req := sendRequest(failingReader{t}, 1<<10).WithContext(ctx)
	cancelled := make(chan *httptest.ResponseRecorder, 1)
	go func() { cancelled <- doSend(req) }()
	time.Sleep(50 * time.Millisecond)
	cancel()
	if w := <-cancelled; w.Body.Len() != 0 {
		t.Fatalf("something was sent to a client which had gone: %d %s", w.Code, w.Body.String())
	}

	close(a.gate)
	<-doneA
	if !budgetFree(sendBudget, 1<<20) {
		t.Fatal("the budget is still reserved")
	}
}

type errReader struct{}

func (errReader) Read([]byte) (int, error) { return 0, errors.New("connection reset") }

// The body is decoded as it always was: the same accepted, the same errors, whatever the body is.
func TestDecodeSendBodyIsAsItWas(t *testing.T) {
	legacy := func(body []byte) (sendMessageParams, error) {
		data := sendMessageParams{}
		err := json.NewDecoder(bytes.NewReader(body)).Decode(&data.Body)

		return data, err
	}

	for name, body := range map[string]string{
		"valid":                     `{"From":{"Email":"a@b.test","Name":"A"},"To":[{"Email":"c@d.test"}],"Subject":"s","Text":"t","Tags":["a","b"]}`,
		"white space after":         `{"Subject":"s"}` + " \n\t ",
		"white space before":        " \n" + `{"Subject":"s"}`,
		"more after the value":      `{"Subject":"s"} garbage`,
		"another value after":       `{"Subject":"s"}{"Subject":"other"}`,
		"empty":                     ``,
		"white space only":          ` `,
		"truncated":                 `{"Subject":"s"`,
		"truncated in a string":     `{"Subject":"s`,
		"not JSON":                  `hello`,
		"an array":                  `[1,2]`,
		"null":                      `null`,
		"a number":                  `42`,
		"the wrong type":            `{"From":"a@b.test"}`,
		"the wrong type, later":     `{"Subject":"s","To":"nope","Text":"t"}`,
		"unknown fields":            `{"Nope":1,"Subject":"s"}`,
		"duplicate keys":            `{"Subject":"a","Subject":"b"}`,
		"a bad escape":              `{"Subject":"\q"}`,
		"a control character":       "{\"Subject\":\"a\nb\"}",
		"invalid UTF-8":             "{\"Subject\":\"\xff\xfe\"}",
		"a number which is too big": `{"Attachments":[{"Filename":1e999}]}`,
	} {
		want, wantErr := legacy([]byte(body))

		var got sendMessageParams
		gotErr := decodeSendBody([]byte(body), &got)

		if (gotErr == nil) != (wantErr == nil) || (gotErr != nil && gotErr.Error() != wantErr.Error()) {
			t.Errorf("%s: error %v, was %v", name, gotErr, wantErr)
			continue
		}
		if gotErr == nil {
			a, _ := json.Marshal(got.Body)
			b, _ := json.Marshal(want.Body)
			if !bytes.Equal(a, b) {
				t.Errorf("%s: decoded to %s, was %s", name, a, b)
			}
		}
	}
}

// A body which says how large it is is read into a slice of that size and no more (growing one leaves a copy of what
// it was each time it does), and any other is read as it comes.
func TestReadSendBody(t *testing.T) {
	data := bytes.Repeat([]byte("0123456789"), 1000)

	body, err := readSendBody(sendRequest(bytes.NewReader(data), int64(len(data))))
	if err != nil || !bytes.Equal(body, data) || cap(body) != len(data) {
		t.Fatalf("%v, %d bytes with room for %d", err, len(body), cap(body))
	}

	// control: any which does not say, or which says what is more than is made room for, is read to the end
	for name, length := range map[string]int64{"chunked": -1, "too large to make room for": sendBudgetBytes + 1} {
		body, err = readSendBody(sendRequest(bytes.NewReader(data), length))
		if err != nil || !bytes.Equal(body, data) {
			t.Fatalf("%s: %v, %d bytes", name, err, len(body))
		}
	}

	// one which ends before it says is an error, as it was
	if _, err = readSendBody(sendRequest(bytes.NewReader(data[:100]), int64(len(data)))); !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("a body which ends early: %v", err)
	}
	if body, err = readSendBody(sendRequest(strings.NewReader(""), 0)); err != nil || len(body) != 0 {
		t.Fatalf("an empty body: %v, %d bytes", err, len(body))
	}
}

// What is sent is what is stored: the text, the HTML, every attachment (with the content as it was, with the prefix
// of a data URL or without) and the rest.
func TestSendStoresTheMessage(t *testing.T) {
	initTestDB(t)

	png := []byte("\x89PNG\r\n\x1a\n" + strings.Repeat("png data ", 500))
	pdf := []byte("%PDF-1.4\n" + strings.Repeat("pdf data ", 700))
	inline := []byte("GIF89a" + strings.Repeat("gif data ", 100))

	body, _ := json.Marshal(map[string]any{
		"From":    map[string]string{"Name": "Sender", "Email": "from@example.test"},
		"To":      []map[string]string{{"Name": "To", "Email": "to@example.test"}},
		"Cc":      []map[string]string{{"Email": "cc@example.test"}},
		"Bcc":     []string{"bcc@example.test"},
		"ReplyTo": []map[string]string{{"Email": "reply@example.test"}},
		"Subject": "A subject",
		"Text":    "The text",
		"HTML":    `<p>The HTML <img src="cid:logo"></p>`,
		"Tags":    []string{"one", "two"},
		"Headers": map[string]string{"X-Custom": "value"},
		"Attachments": []map[string]string{
			{"Filename": "image.png", "Content": "data:image/png;base64," + base64.StdEncoding.EncodeToString(png)},
			{"Filename": "doc.pdf", "ContentType": "application/pdf", "Content": base64.StdEncoding.EncodeToString(pdf)},
			{"Filename": "logo.gif", "ContentID": "logo", "Content": base64.StdEncoding.EncodeToString(inline)},
		},
	})

	w := doSend(sendRequest(bytes.NewReader(body), int64(len(body))))
	var res struct{ ID string }
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil || w.Code != http.StatusOK || res.ID == "" {
		t.Fatalf("status %d: %s", w.Code, w.Body.String())
	}

	msg, err := storage.GetMessage(res.ID)
	if err != nil {
		t.Fatal(err)
	}
	if msg.Subject != "A subject" || strings.TrimSpace(msg.Text) != "The text" || !strings.Contains(msg.HTML, "The HTML") {
		t.Fatalf("subject %q, text %q, html %q", msg.Subject, msg.Text, msg.HTML)
	}
	if len(msg.Attachments) != 2 || len(msg.Inline) != 1 {
		t.Fatalf("%d attachments, %d inline", len(msg.Attachments), len(msg.Inline))
	}
	if len(msg.Tags) != 2 || len(msg.Cc) != 1 || len(msg.Bcc) != 1 || len(msg.ReplyTo) != 1 {
		t.Fatalf("tags %v, cc %v, bcc %v, reply-to %v", msg.Tags, msg.Cc, msg.Bcc, msg.ReplyTo)
	}

	for filename, want := range map[string][]byte{"image.png": png, "doc.pdf": pdf, "logo.gif": inline} {
		var found bool
		for _, a := range append(append([]storage.Attachment{}, msg.Attachments...), msg.Inline...) {
			if a.FileName != filename {
				continue
			}
			found = true
			part, err := storage.GetAttachmentPart(res.ID, a.PartID)
			if err != nil || !bytes.Equal(part.Content, want) {
				t.Fatalf("%s: %v, %d bytes, want %d", filename, err, len(part.Content), len(want))
			}
		}
		if !found {
			t.Fatalf("%s was not stored", filename)
		}
	}

	raw, _ := storage.GetMessageRaw(res.ID)
	if !strings.Contains(string(raw), "X-Custom: value") {
		t.Fatalf("the header was not stored: %s", raw[:400])
	}
}

// A request which is not valid is an error and nothing is stored (the controls for the above).
func TestSendRefusesWhatIsNotValid(t *testing.T) {
	initTestDB(t)
	before := storage.CountTotal()

	for name, body := range map[string]string{
		"not JSON":              `nope`,
		"an address":            `{"From":{"Email":"a@b.test"},"To":[{"Email":"nope"}]}`,
		"a header":              `{"From":{"Email":"a@b.test"},"To":[{"Email":"c@d.test"}],"Headers":{"Bcc":"x@y.test"}}`,
		"an attachment":         `{"From":{"Email":"a@b.test"},"To":[{"Email":"c@d.test"}],"Attachments":[{"Filename":"a","Content":"!!!"}]}`,
		"an attachment, a data": `{"From":{"Email":"a@b.test"},"To":[{"Email":"c@d.test"}],"Attachments":[{"Filename":"a","Content":"data:x;base64,!!!"}]}`,
	} {
		if w := doSend(sendRequest(strings.NewReader(body), int64(len(body)))); w.Code != http.StatusBadRequest {
			t.Errorf("%s: status %d: %s", name, w.Code, w.Body.String())
		}
	}

	if storage.CountTotal() != before {
		t.Fatal("something was stored")
	}
}

// Sends at the same time use what the budget allows, not the sum of them (8 of 20 MiB took 1283 MiB, with the
// requests read, decoded and made into a message all at once).
func TestSendMemoryIsBounded(t *testing.T) {
	if testing.Short() || raceEnabled {
		t.Skip("measures memory and time, which the race detector changes")
	}
	initTestDB(t)
	sendLimits(t, 50, sendBudgetBytes, time.Minute)

	const n = 8
	body := sendJSON(0, 0)
	raw := make([]byte, 15<<20)
	for i := range raw {
		raw[i] = byte(i * 7)
	}
	body, _ = json.Marshal(map[string]any{
		"From": map[string]string{"Email": "a@b.test"}, "To": []map[string]string{{"Email": "c@d.test"}}, "Subject": "s", "Text": "t",
		"Attachments": []map[string]string{{"Filename": "a.bin", "Content": base64.StdEncoding.EncodeToString(raw)}},
	})
	raw = nil

	var peak, base uint64
	var mu sync.Mutex
	stop := make(chan struct{})
	sample := func() {
		var m memStats
		m.read()
		mu.Lock()
		peak = max(peak, m.heapInUse)
		mu.Unlock()
	}
	var m memStats
	m.gc()
	m.read()
	base = m.heapInUse

	go func() {
		for {
			select {
			case <-stop:
				return
			default:
				sample()
				time.Sleep(5 * time.Millisecond)
			}
		}
	}()

	var wg sync.WaitGroup
	accepted := make(chan int, n)
	for range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w := doSend(sendRequest(bytes.NewReader(body), int64(len(body))))
			accepted <- w.Code
		}()
	}
	wg.Wait()
	close(stop)
	close(accepted)

	for code := range accepted {
		if code != http.StatusOK {
			t.Fatalf("status %d", code)
		}
	}

	mu.Lock()
	defer mu.Unlock()
	t.Logf("%d sends of %d MiB: peak heap in use %d MiB (base %d MiB; the bodies alone are %d MiB)", n, len(body)>>20, peak>>20, base>>20, n*len(body)>>20)

	// it was 1283 MiB, and is a few times what the budget allows (50 MiB of requests, and the message of each made from
	// them), not what all of them need
	if peak>>20 > 640 {
		t.Fatalf("peak heap in use was %d MiB", peak>>20)
	}
}

// zeroReader is a body which is far larger than the limit (so it is as good as one which never ends, but a failure of
// the limit is a failed test and not the end of the test run), and counts what has been read of it.
type zeroReader struct{ read int64 }

func (z *zeroReader) Read(p []byte) (int, error) {
	const size = 64 << 20
	if z.read >= size {
		return 0, io.EOF
	}

	p = p[:min(int64(len(p)), size-z.read)]
	clear(p)
	z.read += int64(len(p))

	return len(p), nil
}

// A request which does not say how large it is, and is far larger than it can be, is read as far as the limit and no
// further.
func TestSendStopsReadingAChunkedRequestAtTheLimit(t *testing.T) {
	initTestDB(t)
	sendLimits(t, 1, 50<<20, time.Second)

	z := &zeroReader{}
	w := doSend(sendRequest(z, -1))

	if w.Code != http.StatusBadRequest && w.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status %d", w.Code)
	}
	if z.read > 1<<20+(256<<10) {
		t.Fatalf("%d bytes were read of a request which never ends (the limit is %d)", z.read, 1<<20)
	}
	if !budgetFree(sendBudget, 50<<20) {
		t.Fatal("the budget is still reserved")
	}
}
