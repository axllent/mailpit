package apiv1

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/mail"
	"strconv"
	"strings"
	"time"

	"github.com/axllent/mailpit/config"
	"github.com/axllent/mailpit/internal/smtpd"
	"github.com/axllent/mailpit/internal/tools"
	"github.com/jhillyerd/enmime/v2"
)

// sendBudgetBytes is how many bytes of the requests to send a message are handled at the same time. Each is counted
// as the size of its body, which is reserved before it is read, but a message takes several times that in memory
// while it is made (the JSON, the attachments decoded, and the message itself encoded again), so this is a limit on
// memory of a few times as much. A request which needs more than all of it (when the maximum message size is more)
// waits until none are being handled, and is then the only one.
const sendBudgetBytes = 50 << 20

var (
	// sendBudget is shared by every request to send a message. It is a variable for the tests.
	sendBudget = tools.NewBudget(sendBudgetBytes)

	// sendWait is how long a request waits for its turn before it is told to try again later. It is a variable for
	// the tests.
	sendWait = 5 * time.Second
)

// messageTooLargeError is returned for a message which is larger than the maximum message size once it is made.
type messageTooLargeError struct {
	size, limit int64
}

func (e messageTooLargeError) Error() string {
	return fmt.Sprintf("the message is %d bytes, which is larger than the maximum message size of %d MiB", e.size, e.limit>>20)
}

// SendMessageHandler handles HTTP requests to send a new message
func SendMessageHandler(w http.ResponseWriter, r *http.Request) {
	if config.DemoMode {
		httpJSONError(w, "this functionality has been disabled for demonstration purposes")
		return
	}

	// the same limit as for a message sent to the SMTP server (0 is no limit)
	limit := int64(config.MaxMessageSize) * 1024 * 1024

	// A request which says it is larger is refused without reading any of it.
	if limit > 0 && r.ContentLength > limit {
		w.Header().Set("Connection", "close")
		httpJSONErrorStatus(w, http.StatusRequestEntityTooLarge,
			fmt.Sprintf("the request is %d bytes, which is larger than the maximum message size of %d MiB", r.ContentLength, config.MaxMessageSize))
		return
	}

	// only so many are handled at once, as each takes a lot of memory
	release, ok := acquireSendBudget(w, r, limit)
	if !ok {
		return
	}
	defer release()

	// One which does not say how large it is (or says what is not so) is refused once it is.
	if limit > 0 {
		r.Body = http.MaxBytesReader(w, r.Body, limit)
	}

	data, status, err := readSendRequest(r)
	if err != nil {
		httpJSONErrorStatus(w, status, err.Error())
		return
	}

	var httpAuthUser *string
	if user, _, ok := r.BasicAuth(); ok {
		httpAuthUser = &user
	}

	id, err := data.Send(r.RemoteAddr, httpAuthUser)

	if err != nil {
		if tooLarge, ok := errors.AsType[messageTooLargeError](err); ok {
			httpJSONErrorStatus(w, http.StatusRequestEntityTooLarge, tooLarge.Error())
			return
		}
		httpJSONError(w, err.Error())
		return
	}

	w.Header().Add("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(struct{ ID string }{ID: id}); err != nil {
		httpError(w, err.Error())
	}
}

// acquireSendBudget reserves what the request needs of the budget, which is the size of its body (the maximum message
// size for one which does not say, and all of the budget if there is none), waiting for it for a while. It writes the
// response, and returns false, if there is no room for it after that. The reservation is given back by the function
// which is returned.
func acquireSendBudget(w http.ResponseWriter, r *http.Request, limit int64) (func(), bool) {
	n := r.ContentLength
	if n < 0 {
		if n = limit; n == 0 {
			n = sendBudgetBytes
		}
	}

	ctx, cancel := context.WithTimeout(r.Context(), sendWait)
	defer cancel()

	release, err := sendBudget.Acquire(ctx, n)
	if err != nil {
		// the client has gone, so there is nobody to send anything to
		if r.Context().Err() != nil {
			return nil, false
		}

		w.Header().Set("Retry-After", strconv.Itoa(int(sendWait/time.Second)))
		httpJSONErrorStatus(w, http.StatusServiceUnavailable, "too many messages are being sent, please try again shortly")

		return nil, false
	}

	return release, true
}

// SendAddress is an email address with an optional name
//
// openapi:model SendAddress
type SendAddress struct {
	// Optional name
	// example: John Doe
	Name string
	// Email address
	// example: john@example.com
	// required: true
	Email string
}

// SendAttachment is a message attachment for the send API
//
// openapi:model SendAttachment
type SendAttachment struct {
	// Base64-encoded string of the file content
	// required: true
	// example: iVBORw0KGgoAAAANSUhEUgAAAEEAAAA8CAMAAAAOlSdoAAAACXBIWXMAAAHrAAAB6wGM2bZBAAAAS1BMVEVHcEwRfnUkZ2gAt4UsSF8At4UtSV4At4YsSV4At4YsSV8At4YsSV4At4YsSV4sSV4At4YsSV4At4YtSV4At4YsSV4At4YtSV8At4YsUWYNAAAAGHRSTlMAAwoXGiktRE5dbnd7kpOlr7zJ0d3h8PD8PCSRAAACWUlEQVR42pXT4ZaqIBSG4W9rhqQYocG+/ys9Y0Z0Br+x3j8zaxUPewFh65K+7yrIMeIY4MT3wPfEJCidKXEMnLaVkxDiELiMz4WEOAZSFghxBIypCOlKiAMgXfIqTnBgSm8CIQ6BImxEUxEckClVQiHGj4Ba4AQHikAIClwTE9KtIghAhUJwoLkmLnCiAHJLRKgIMsEtVUKbBUIwoAg2C4QgQBE6l4VCnApBgSKYLLApCnCa0+96AEMW2BQcmC+Pr3nfp7o5Exy49gIADcIqUELGfeA+bp93LmAJp8QJoEcN3C7NY3sbVANixMyI0nku20/n5/ZRf3KI2k6JEDWQtxcbdGuAqu3TAXG+/799Oyyas1B1MnMiA+XyxHp9q0PUKGPiRAau1fZbLRZV09wZcT8/gHk8QQAxXn8VgaDqcUmU6O/r28nbVwXAqca2mRNtPAF5+zoP2MeN9Fy4NgC6RfcbgE7XITBRYTtOE3U3C2DVff7pk+PkUxgAbvtnPXJaD6DxulMLwOhPS/M3MQkgg1ZFrIXnmfaZoOfpKiFgzeZD/WuKqQEGrfJYkyWf6vlG3xUgTuscnkNkQsb599q124kdpMUjCa/XARHs1gZymVtGt3wLkiFv8rUgTxitYCex5EVGec0Y9VmoDTFBSQte2TfXGXlf7hbdaUM9Sk7fisEN9qfBBTK+FZcvM9fQSdkl2vj4W2oX/bRogO3XasiNH7R0eW7fgRM834ImTg+Lg6BEnx4vz81rhr+MYPBBQg1v8GndEOrthxaCTxNAOut8WKLGZQl+MPz88Q9tAO/hVuSeqQAAAABJRU5ErkJggg==
	Content string
	// Filename
	// required: true
	// example: mailpit.png
	Filename string
	// Optional Content Type for the the attachment.
	// If this field is not set (or empty) then the content type is automatically detected.
	// required: false
	// example: image/png
	ContentType string
	// Optional Content-ID (`cid`) for attachment.
	// If this field is set then the file is attached inline.
	// required: false
	// example: mailpit-logo
	ContentID string
}

// SendRequest is the request body for the send API
//
// openapi:model SendRequest
type SendRequest struct {
	// "From" recipient
	// required: true
	From SendAddress

	// "To" recipients
	// example: [{"Name":"Jane Doe","Email":"jane@example.com"}]
	To []SendAddress

	// Cc recipients
	// example: [{"Name":"Manager","Email":"manager@example.com"}]
	Cc []SendAddress

	// Bcc recipients email addresses only
	// example: ["jack@example.com"]
	Bcc []string

	// Optional Reply-To recipients
	// example: [{"Name":"Secretary","Email":"secretary@example.com"}]
	ReplyTo []SendAddress

	// Subject
	// example: Mailpit message via the HTTP API
	Subject string

	// Message body (text)
	// example: Mailpit is awesome!
	Text string

	// Message body (HTML)
	// example: <div style="text-align:center"><p style="font-family: arial; font-size: 24px;">Mailpit is <b>awesome</b>!</p><p><img src="cid:mailpit-logo" /></p></div>
	HTML string

	// Attachments
	Attachments []SendAttachment

	// Mailpit tags
	// example: ["Tag 1","Tag 2"]
	Tags []string

	// Optional headers in {"key":"value"} format
	// example: {"X-IP":"1.2.3.4"}
	Headers map[string]string
}

// sendMessageParams wraps the decoded send API request
type sendMessageParams struct {
	Body SendRequest
}

// readSendRequest reads and decodes the body of the request. It returns the status to respond with if that is not
// possible, which is 413 if it is larger than the maximum message size (and was not known to be).
func readSendRequest(r *http.Request) (data sendMessageParams, status int, err error) {
	// Body is read before it is decoded so that MaxBytesReader errors are returned directly.
	// In Go 1.26+, json.Decoder wraps reader errors in *json.SyntaxError, which
	// prevents errors.As from finding *http.MaxBytesError to return a 413.
	body, err := readSendBody(r)
	if err != nil {
		if _, ok := errors.AsType[*http.MaxBytesError](err); ok {
			return data, http.StatusRequestEntityTooLarge, err
		}

		return data, http.StatusBadRequest, err
	}

	if err := decodeSendBody(body, &data); err != nil {
		return data, http.StatusBadRequest, err
	}

	// the body, which is as large as everything which was decoded from it, is not kept: it is not used after this
	return data, http.StatusOK, nil
}

// readSendBody reads the body of the request. One which says how large it is is read into a slice of exactly that,
// where reading what is not known makes one which grows (and leaves a copy of what it was for each time it does).
func readSendBody(r *http.Request) ([]byte, error) {
	if n := r.ContentLength; n > 0 && n <= sendBudgetBytes {
		body := make([]byte, n)
		if _, err := io.ReadFull(r.Body, body); err != nil {
			return nil, err
		}

		return body, nil
	}

	return io.ReadAll(r.Body)
}

// decodeSendBody decodes the first JSON value of the body. It is decoded from where it is, which does not copy it, and
// if that is not so (the body is not valid, or has more after the value, which has always been allowed) it is
// decoded as it always was, so what is accepted and what is said of what is not are the same.
func decodeSendBody(body []byte, data *sendMessageParams) error {
	if err := json.Unmarshal(body, &data.Body); err == nil {
		return nil
	}

	data.Body = sendMessageParams{}.Body

	return json.NewDecoder(bytes.NewReader(body)).Decode(&data.Body)
}

// httpJSONErrorStatus returns an error message in JSON format with the status code
func httpJSONErrorStatus(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(struct{ Error string }{Error: msg})
}

// Send will validate the message structure and attempt to send to Mailpit.
// It returns a sending summary or an error.
func (d sendMessageParams) Send(remoteAddr string, httpAuthUser *string) (string, error) {
	ipAddr := originAddr(remoteAddr)

	addresses := []string{}

	// enmime.Builder expects the decoded local-part (no quotes) and handles
	// RFC 5322 quoting itself. Use mail.ParseAddress to decode quoted
	// local-parts like "odd user"@example.com into enmime's expected form,
	// while keeping the original Email value for SMTP envelope use.
	parsedFrom, err := mail.ParseAddress(d.Body.From.Email)
	if err != nil {
		// as the recipients are, and the address is the envelope sender and the Return-Path of the stored message
		return "", fmt.Errorf("invalid From address: %s", d.Body.From.Email)
	}

	fromEmail := parsedFrom.Address

	msg := enmime.Builder().
		From(d.Body.From.Name, fromEmail).
		Subject(d.Body.Subject).
		Text([]byte(d.Body.Text))

	if d.Body.HTML != "" {
		msg = msg.HTML([]byte(d.Body.HTML))
	}

	if len(d.Body.To) > 0 {
		for _, a := range d.Body.To {
			if parsed, err := mail.ParseAddress(a.Email); err == nil {
				msg = msg.To(a.Name, parsed.Address)
				addresses = append(addresses, a.Email)
			} else {
				return "", fmt.Errorf("invalid To address: %s", a.Email)
			}
		}
	}

	if len(d.Body.Cc) > 0 {
		for _, a := range d.Body.Cc {
			if parsed, err := mail.ParseAddress(a.Email); err == nil {
				msg = msg.CC(a.Name, parsed.Address)
				addresses = append(addresses, a.Email)
			} else {
				return "", fmt.Errorf("invalid Cc address: %s", a.Email)
			}
		}
	}

	if len(d.Body.Bcc) > 0 {
		for _, e := range d.Body.Bcc {
			if parsed, err := mail.ParseAddress(e); err == nil {
				msg = msg.BCC("", parsed.Address)
				addresses = append(addresses, e)
			} else {
				return "", fmt.Errorf("invalid Bcc address: %s", e)
			}
		}
	}

	if len(d.Body.ReplyTo) > 0 {
		for _, a := range d.Body.ReplyTo {
			if parsed, err := mail.ParseAddress(a.Email); err == nil {
				msg = msg.ReplyTo(a.Name, parsed.Address)
			} else {
				return "", fmt.Errorf("invalid Reply-To address: %s", a.Email)
			}
		}
	}

	restrictedHeaders := []string{"To", "From", "Cc", "Bcc", "Reply-To", "Date", "Subject", "Content-Type", "Mime-Version"}

	if len(d.Body.Tags) > 0 {
		msg = msg.Header("X-Tags", strings.Join(d.Body.Tags, ", "))
		restrictedHeaders = append(restrictedHeaders, "X-Tags")
	}

	if len(d.Body.Headers) > 0 {
		for k, v := range d.Body.Headers {
			// validate header key per RFC 5322: printable US-ASCII (33-126) excluding colon
			if !validHeaderKey(k) {
				return "", fmt.Errorf("invalid header key: %q", k)
			}
			// check header isn't in "restricted" headers
			if tools.InArray(k, restrictedHeaders) {
				return "", fmt.Errorf("cannot overwrite header: \"%s\"", k)
			}
			msg = msg.Header(k, v)
		}
	}

	// what the message is going to be, to make room for it at once rather than a little at a time
	size := 8<<10 + 2*(len(d.Body.Text)+len(d.Body.HTML))

	for i := range d.Body.Attachments {
		a := &d.Body.Attachments[i]

		b, err := decodeAttachment(a.Content)
		if err != nil {
			return "", fmt.Errorf("error decoding base64 attachment \"%s\": %s", a.Filename, err.Error())
		}
		// the base64 is as large as the attachment and more, and it is not needed now
		a.Content = ""

		size += len(b)/3*4 + len(b)/57*2 + 128
		contentType := http.DetectContentType(b)
		if a.ContentType != "" {
			contentType = a.ContentType
		}
		if a.ContentID != "" {
			msg = msg.AddInline(b, contentType, a.Filename, a.ContentID)
		} else {
			msg = msg.AddAttachment(b, contentType, a.Filename)
		}
	}

	part, err := msg.Build()
	if err != nil {
		return "", fmt.Errorf("error building message: %s", err.Error())
	}

	var buff bytes.Buffer
	buff.Grow(size)

	if err := part.Encode(io.Writer(&buff)); err != nil {
		return "", fmt.Errorf("error building message: %s", err.Error())
	}

	// the message is larger than the request was (the attachments are base64 in it as well), and what is accepted is
	// the same as what the SMTP server accepts
	if limit := int64(config.MaxMessageSize) * 1024 * 1024; limit > 0 && int64(buff.Len()) > limit {
		return "", messageTooLargeError{size: int64(buff.Len()), limit: limit}
	}

	return smtpd.SaveToDatabase(ipAddr, d.Body.From.Email, addresses, buff.Bytes(), httpAuthUser)
}

// originAddr returns the address a request came from. A request which is not made over TCP, such as one made over
// a Unix socket (where the address is "@" or the path of the client), has no IP address and port to split, and is
// local.
func originAddr(remoteAddr string) net.Addr {
	ip, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		return &net.IPAddr{IP: net.IPv4(127, 0, 0, 1)}
	}

	return &net.IPAddr{IP: net.ParseIP(ip)}
}

// decodeAttachment decodes the base64 content of an attachment.
//
// This is a workaround for the string which JS readAsDataURL() returns, which has the mime type as a prefix, eg:
// data:image/png;base64,<base64String>, so it is what is after the last comma.
func decodeAttachment(content string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(content[strings.LastIndexByte(content, ',')+1:])
}

// validHeaderKey returns true if s is a valid RFC 5322 header field name.
// Field names must be one or more printable US-ASCII characters (33-126),
// excluding colon (58).
func validHeaderKey(s string) bool {
	if len(s) == 0 {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c < 33 || c > 126 || c == ':' {
			return false
		}
	}
	return true
}
