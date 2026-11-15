package apiv1

import (
	"bytes"
	"fmt"
	"net/http"
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

// This will rewrite all inline image paths to API URLs.
//
// The HTML is read once, whatever its size and however many images and references it has. It used to be scanned
// again for every Content-ID (with a regular expression of its own), and rewritten again for every reference
// found: a message of 400 inline parts took 11 seconds, and one of 80,000 references to one part 93 seconds, for a
// single request to view it.
func linkInlineImages(msg *storage.Message) string {
	// the URL of the part for each Content-ID (in lower case, as they are not case sensitive), the inline parts
	// coming first and then the attachments, and the first for a Content-ID being the one which is used
	urls := map[string]string{}

	for _, group := range [][]storage.Attachment{msg.Inline, msg.Attachments} {
		for _, a := range group {
			if a.ContentID == "" {
				continue
			}

			key := strings.ToLower(a.ContentID)
			if _, ok := urls[key]; !ok {
				urls[key] = config.Webroot + "api/v1/message/" + msg.ID + "/part/" + a.PartID
			}
		}
	}

	if len(urls) == 0 {
		return msg.HTML
	}

	return rewriteCIDLinks(msg.HTML, urls)
}

// cidTerminators end a Content-ID which is not in quotes: white space, the end of a tag (or of an empty one, as in
// <img src=cid:logo/>), another quote, or a semicolon.
const cidTerminators = " \t\r\n\f>\"'/;"

// rewriteCIDLinks replaces the "cid:<Content-ID>" of attributes (=cid:x, ="cid:x" and ='cid:x', in any case) with
// the URL for the Content-ID, if there is one in urls (by Content-ID in lower case). A Content-ID in quotes is all
// there is up to the closing quote, and one which is not is up to the first of cidTerminators.
//
// It is made so that the time it takes is in proportion to the size of the HTML. After a reference has been looked at
// the HTML is read on from the end of it, so no part of it is read twice by this. (A quote which is not closed is
// the last of its kind, as another would close it, so the search for one to the end of the HTML is made once for
// each kind of quote.)
func rewriteCIDLinks(html string, urls map[string]string) string {
	var out strings.Builder
	last := 0 // what is before this has been written to out (or is not to be changed)
	i := 0    // where the search is

	for i < len(html) {
		eq := strings.IndexByte(html[i:], '=')
		if eq < 0 {
			break
		}
		i += eq + 1 // after the =

		p := i
		var quote byte
		if p < len(html) && (html[p] == '"' || html[p] == '\'') {
			quote = html[p]
			p++
		}

		if len(html)-p < len("cid:") || !strings.EqualFold(html[p:p+len("cid:")], "cid:") {
			continue
		}
		start := p + len("cid:")

		var end int
		if quote != 0 {
			c := strings.IndexByte(html[start:], quote)
			if c < 0 {
				i = start
				continue
			}
			end = start + c
		} else {
			c := strings.IndexAny(html[start:], cidTerminators)
			if c < 0 {
				// there is nothing which ends it from here to the end, so nor for any after this
				break
			}
			end = start + c
		}

		if url, ok := urls[strings.ToLower(html[start:end])]; ok {
			if last == 0 {
				out.Grow(len(html))
			}
			out.WriteString(html[last : start-len("cid:")])
			out.WriteString(url)
			last = end
		}

		i = end
	}

	if last == 0 {
		return html
	}

	out.WriteString(html[last:])

	return out.String()
}
