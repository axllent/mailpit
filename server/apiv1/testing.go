package apiv1

import (
	"bytes"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/axllent/mailpit/config"
	"github.com/axllent/mailpit/internal/logger"
	"github.com/axllent/mailpit/internal/storage"
	"github.com/axllent/mailpit/internal/tools"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// maxRenderHTMLBytes is the largest HTML part which is rendered. Rendering parses the whole document, which
// takes about ten times its size in memory (more for a document of many small tags), and the largest HTML part
// in a mailbox of 153,000 real messages was 1 MiB. It is a variable for the tests.
var maxRenderHTMLBytes = 10 << 20

// GetMessageHTML (method: GET) returns a rendered version of a message's HTML part
func GetMessageHTML(w http.ResponseWriter, r *http.Request) {
	// swagger:route GET /view/{ID}.html testing GetMessageHTMLParams
	//
	// # Render message HTML part
	//
	// Renders just the message's HTML part which can be used for UI integration testing.
	// Attached inline images are modified to link to the API provided they exist.
	// Note that is the message does not contain a HTML part then an 404 error is returned.
	// A HTML part which is larger than 10 MiB is not rendered, and a 413 error is returned.
	//
	// The ID can be set to `latest` to return the latest message.
	//
	//	Produces:
	//	- text/html
	//
	//	Schemes: http, https
	//
	//	Responses:
	//		200: HTMLResponse
	//		400: ErrorResponse
	//      404: NotFoundResponse

	path := strings.TrimPrefix(r.URL.Path, config.Webroot+"view/")
	id, ok := resolveMessageID(w, r, strings.TrimSuffix(path, ".html"))
	if !ok {
		return
	}

	msg, err := storage.GetMessage(id)
	if err != nil {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(404)
		_, _ = fmt.Fprint(w, "Message not found")
		return
	}
	if msg.HTML == "" {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(404)
		_, _ = fmt.Fprint(w, "This message does not contain a HTML part")
		return
	}

	if len(msg.HTML) > maxRenderHTMLBytes {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusRequestEntityTooLarge)
		_, _ = fmt.Fprintf(w, "The HTML of this message is too large to render (the limit is %d MiB)", maxRenderHTMLBytes>>20)
		return
	}

	htmlStr := linkInlineImages(msg)

	// If embed=1 is set, then we will add target="_blank" and rel="noreferrer noopener" to all links.
	// This parses the whole document, so it waits for its turn (as removing the redirects does).
	if r.URL.Query().Get("embed") == "1" {
		release, err := tools.AcquireHTMLSlot(r.Context())
		if err != nil {
			// the client has gone, so there is nobody to send anything to
			return
		}

		htmlStr = embedLinks(htmlStr, r.Header.Get("mp-nonce"))

		release()
	}

	// Remove <base> and <meta http-equiv="refresh"> tags, which CSP does not (fully) cover.
	// This is done last so that it applies to exactly what is served, and fails closed.
	htmlStr, err = tools.StripHTMLRedirects(r.Context(), htmlStr)
	if err != nil {
		if r.Context().Err() != nil {
			// the client has gone, so there is nobody to send anything to
			return
		}

		logger.Log().Error(err.Error())
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(500)
		_, _ = fmt.Fprint(w, "Unable to render message")
		return
	}

	w.Header().Add("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(htmlStr)) // #nosec G705 -- intentionally renders stored email HTML for UI integration testing
}

// embedLinks adds target="_blank" and rel="noreferrer noopener" to the links of the HTML, and the script which
// tells the page which embeds it how high it is. The HTML is returned as it was if it can not be parsed.
func embedLinks(htmlStr, nonce string) string {
	doc, err := html.Parse(strings.NewReader(htmlStr))
	if err != nil {
		logger.Log().Error(err.Error())
		return htmlStr
	}

	// Walk the entire HTML tree.
	tools.WalkHTML(doc, func(n *html.Node) {
		if n.Type == html.ElementNode && n.DataAtom == atom.A {
			// Set attributes on all anchors with external links.
			tools.SetHTMLAttributeVal(n, "target", "_blank")
			tools.SetHTMLAttributeVal(n, "rel", "noreferrer noopener")
		}
	})

	b := bytes.Buffer{}
	_ = html.Render(&b, doc)
	htmlStr = b.String()

	js := `<script nonce="` + nonce + `">
				if (typeof window.parent == "object") {
					window.addEventListener('load', function () {
						window.parent.postMessage({ messageHeight: document.body.scrollHeight}, "*")
					})
				}
			</script>`

	return strings.ReplaceAll(htmlStr, "</body>", js+"</body>")
}

// GetMessageText (method: GET) returns a message's text part
func GetMessageText(w http.ResponseWriter, r *http.Request) {
	// swagger:route GET /view/{ID}.txt testing GetMessageTextParams
	//
	// # Render message text part
	//
	// Renders just the message's text part which can be used for UI integration testing.
	//
	// The ID can be set to `latest` to return the latest message.
	//
	//	Produces:
	//	- text/plain
	//
	//	Schemes: http, https
	//
	//	Responses:
	//		200: TextResponse
	//		400: ErrorResponse
	//      404: NotFoundResponse

	path := strings.TrimPrefix(r.URL.Path, config.Webroot+"view/")
	id, ok := resolveMessageID(w, r, strings.TrimSuffix(path, ".txt"))
	if !ok {
		return
	}

	msg, err := storage.GetMessage(id)
	if err != nil {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(404)
		_, _ = fmt.Fprint(w, "Message not found")
		return
	}

	w.Header().Add("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte(msg.Text)) // #nosec G705 -- served as text/plain, not rendered as HTML
}

// This will rewrite all inline image paths to API URLs
func linkInlineImages(msg *storage.Message) string {
	html := msg.HTML

	for _, a := range msg.Inline {
		if a.ContentID != "" {
			re := regexp.MustCompile(`(?i)(=["\']?)(cid:` + regexp.QuoteMeta(a.ContentID) + `)(["|\'|\\s|\\/|>|;])`)
			u := config.Webroot + "api/v1/message/" + msg.ID + "/part/" + a.PartID
			matches := re.FindAllStringSubmatch(html, -1)
			for _, m := range matches {
				html = strings.ReplaceAll(html, m[0], m[1]+u+m[3])
			}
		}
	}

	for _, a := range msg.Attachments {
		if a.ContentID != "" {
			re := regexp.MustCompile(`(?i)(=["\']?)(cid:` + regexp.QuoteMeta(a.ContentID) + `)(["|\'|\\s|\\/|>|;])`)
			u := config.Webroot + "api/v1/message/" + msg.ID + "/part/" + a.PartID
			matches := re.FindAllStringSubmatch(html, -1)
			for _, m := range matches {
				html = strings.ReplaceAll(html, m[0], m[1]+u+m[3])
			}
		}
	}

	return html
}
