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
	"github.com/gorilla/mux"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// swagger:parameters GetMessageHTMLParams
type getMessageHTMLParams struct {
	// Message database ID or "latest"
	//
	// in: path
	// required: true
	ID string

	// If this is route is to be embedded in an iframe, set embed to `1` in the URL to add `target="_blank"` and `rel="noreferrer noopener"` to all links.
	//
	// In addition, a small script will be added to the end of the document to post (postMessage()) the height of the document back to the parent window for optional iframe height resizing.
	//
	// Note that this will also *transform* the message into a full HTML document (if it isn't already), so this option is useful for viewing but not programmatic testing.
	//
	// in: query
	// required: false
	// type: string
	Embed string `json:"embed"`
}

// GetMessageHTML (method: GET) returns a rendered version of a message's HTML part
func GetMessageHTML(w http.ResponseWriter, r *http.Request) {
	// swagger:route GET /view/{ID}.html testing GetMessageHTMLParams
	//
	// # Render message HTML part
	//
	// Renders just the message's HTML part which can be used for UI integration testing.
	// Attached inline images are modified to link to the API provided they exist.
	// Note that is the message does not contain a HTML part then an 404 error is returned.
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

	vars := mux.Vars(r)

	id := vars["id"]

	if id == "latest" {
		var err error
		id, err = storage.LatestID(r)
		if err != nil {
			w.WriteHeader(404)
			fmt.Fprint(w, err.Error())
			return
		}
	}

	msg, err := storage.GetMessage(id)
	if err != nil {
		w.WriteHeader(404)
		fmt.Fprint(w, "Message not found")
		return
	}
	if msg.HTML == "" {
		w.WriteHeader(404)
		fmt.Fprint(w, "This message does not contain a HTML part")
		return
	}

	htmlStr := linkInlineImages(msg)

	// If embed=1 is set, then we will add target="_blank" and rel="noreferrer noopener" to all links
	if r.URL.Query().Get("embed") == "1" {
		doc, err := html.Parse(strings.NewReader(htmlStr))
		if err != nil {
			logger.Log().Error(err.Error())
		} else {
			// Walk the entire HTML tree.
			tools.WalkHTML(doc, func(n *html.Node) {
				if n.Type == html.ElementNode && n.DataAtom == atom.A {
					// Set attributes on all anchors with external links.
					tools.SetHTMLAttributeVal(n, "target", "_blank")
					tools.SetHTMLAttributeVal(n, "rel", "noreferrer noopener")
				}
			})

			b := bytes.Buffer{}
			html.Render(&b, doc)
			htmlStr = b.String()

			nonce := r.Header.Get("mp-nonce")

			js := `<script nonce="` + nonce + `">
				if (typeof window.parent == "object") {
					window.addEventListener('load', function () {
						window.parent.postMessage({ messageHeight: document.body.scrollHeight}, "*")
					})
				}
			</script>`

			htmlStr = strings.ReplaceAll(htmlStr, "</body>", js+"</body>")
		}
	}

	w.Header().Add("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(htmlStr))
}

// swagger:parameters GetMessageTextParams
type getMessageTextParams struct {
	// Message database ID or "latest"
	//
	// in: path
	// required: true
	ID string
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

	vars := mux.Vars(r)

	id := vars["id"]

	if id == "latest" {
		var err error
		id, err = storage.LatestID(r)
		if err != nil {
			w.WriteHeader(404)
			fmt.Fprint(w, err.Error())
			return
		}
	}

	msg, err := storage.GetMessage(id)
	if err != nil {
		w.WriteHeader(404)
		fmt.Fprint(w, "Message not found")
		return
	}

	w.Header().Add("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte(msg.Text))
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
