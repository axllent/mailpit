package apiv1

import (
	"fmt"
	"math/rand"
	"net/http"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/axllent/mailpit/config"
	"github.com/axllent/mailpit/internal/storage"
)

// referenceLinkInlineImages is how the links were rewritten before the HTML was read once: a regular expression for
// each Content-ID, which reads all of the HTML, and a replacement of the whole of it for each reference found.
func referenceLinkInlineImages(msg *storage.Message) string {
	html := msg.HTML

	for _, group := range [][]storage.Attachment{msg.Inline, msg.Attachments} {
		for _, a := range group {
			if a.ContentID != "" {
				re := regexp.MustCompile(`(?i)(=["\']?)(cid:` + regexp.QuoteMeta(a.ContentID) + `)(["|\'|\\s|\\/|>|;])`)
				u := config.Webroot + "api/v1/message/" + msg.ID + "/part/" + a.PartID
				matches := re.FindAllStringSubmatch(html, -1)
				for _, m := range matches {
					html = strings.ReplaceAll(html, m[0], m[1]+u+m[3])
				}
			}
		}
	}

	return html
}

func partURL(partID string) string {
	return config.Webroot + "api/v1/message/m1/part/" + partID
}

// message makes a message with the parts, in the HTML given.
func message(html string, inline []storage.Attachment, attachments ...storage.Attachment) *storage.Message {
	return &storage.Message{ID: "m1", HTML: html, Inline: inline, Attachments: attachments}
}

func part(partID, contentID string) storage.Attachment {
	return storage.Attachment{PartID: partID, ContentID: contentID}
}

// Every way of writing the link which was rewritten still is, to the same result, and what was not is not.
func TestInlineImagesRewrites(t *testing.T) {
	msg := message("", []storage.Attachment{part("1", "logo"), part("2", "a@b.test"), part("3", "My.Image-2_x")},
		part("4", "bg"), part("5", "logo"))

	for name, tc := range map[string]struct{ html, want string }{
		"double quotes":             {`<img src="cid:logo">`, `<img src="` + partURL("1") + `">`},
		"single quotes":             {`<img src='cid:logo'>`, `<img src='` + partURL("1") + `'>`},
		"no quotes, end of the tag": {`<img src=cid:logo>`, `<img src=` + partURL("1") + `>`},
		"no quotes, empty tag":      {`<img src=cid:logo/>`, `<img src=` + partURL("1") + `/>`},
		"upper case":                {`<IMG SRC="CID:LOGO">`, `<IMG SRC="` + partURL("1") + `">`},
		"mixed case":                {`<img src="Cid:My.image-2_X">`, `<img src="` + partURL("3") + `">`},
		"an address":                {`<img src="cid:a@b.test">`, `<img src="` + partURL("2") + `">`},
		"background":                {`<td background="cid:bg">`, `<td background="` + partURL("4") + `">`},
		"twice":                     {`<img src="cid:logo"><img src="cid:logo">`, `<img src="` + partURL("1") + `"><img src="` + partURL("1") + `">`},
		"two different":             {`<img src="cid:logo"><img src='cid:bg'>`, `<img src="` + partURL("1") + `"><img src='` + partURL("4") + `'>`},
		"next to each other":        {`<img src="cid:logo"cid:bg">`, `<img src="` + partURL("1") + `"cid:bg">`},
		"not a part":                {`<img src="cid:nobody">`, `<img src="cid:nobody">`},
		"a prefix of an ID":         {`<img src="cid:log">`, `<img src="cid:log">`},
		"more than an ID":           {`<img src="cid:logo.png">`, `<img src="cid:logo.png">`},
		"not an attribute":          {`<p>cid:logo</p> url(cid:logo)`, `<p>cid:logo</p> url(cid:logo)`},
		"nothing after":             {`<img src="cid:`, `<img src="cid:`},
		"no quotes, no end":         {`<p>x</p><img src=cid:logo`, `<p>x</p><img src=cid:logo`},
		"no closing quote":          {`<img src="cid:logo`, `<img src="cid:logo`},
		"empty":                     {``, ``},
		"no images":                 {`<p>hello</p>`, `<p>hello</p>`},
		"a = which is not a link":   {`a=b c="d" e='f' =="cid:logo"`, `a=b c="d" e='f' ==` + `"` + partURL("1") + `"`},
	} {
		t.Run(name, func(t *testing.T) {
			msg.HTML = tc.html
			if got := linkInlineImages(msg); got != tc.want {
				t.Fatalf("got  %s\nwant %s", got, tc.want)
			}
		})
	}

	// the first part with a Content-ID is the one it links to: the inline parts before the attachments, then in order
	msg.HTML = `<img src="cid:logo">`
	if got := linkInlineImages(msg); !strings.Contains(got, partURL("1")) {
		t.Fatalf("the link is not to the first part with the Content-ID: %s", got)
	}
	msg.Inline = nil
	if got := linkInlineImages(msg); !strings.Contains(got, partURL("5")) {
		t.Fatalf("the link is not to the attachment when there is no inline part: %s", got)
	}

	// a message with no Content-IDs is returned as it was
	if got := linkInlineImages(message(`<img src="cid:logo">`, []storage.Attachment{part("1", "")})); got != `<img src="cid:logo">` {
		t.Fatalf("got %s", got)
	}
}

// What was rewritten wrongly (or not at all) is the only difference from what it was.
func TestInlineImagesDifferences(t *testing.T) {
	msg := message("", []storage.Attachment{part("1", "logo")})

	for name, tc := range map[string]struct{ html, want, was string }{
		// a Content-ID not in quotes can be followed by white space (it was not rewritten)
		"unquoted, then a space": {`<img src=cid:logo width=10>`, `<img src=` + partURL("1") + ` width=10>`, `<img src=cid:logo width=10>`},
		"unquoted, then a tab":   {"<img src=cid:logo\theight=10>", `<img src=` + partURL("1") + "\theight=10>", "<img src=cid:logo\theight=10>"},
		// a Content-ID which is longer than one which is a part is not it, whatever it goes on with (the "s" was one of
		// the characters which the Content-ID could be followed by, so the link was to the part and then "s.png")
		"more than the ID, with an s":    {`<img src="cid:logos.png">`, `<img src="cid:logos.png">`, `<img src="` + partURL("1") + `s.png">`},
		"more than the ID, with a pipe":  {`<img src="cid:logo|x">`, `<img src="cid:logo|x">`, `<img src="` + partURL("1") + `|x">`},
		"more than the ID, with a slash": {`<img src="cid:logo/x">`, `<img src="cid:logo/x">`, `<img src="` + partURL("1") + `/x">`},
	} {
		t.Run(name, func(t *testing.T) {
			msg.HTML = tc.html
			if got := linkInlineImages(msg); got != tc.want {
				t.Fatalf("now:  %s\nwant: %s", got, tc.want)
			}
			if was := referenceLinkInlineImages(msg); was != tc.was {
				t.Fatalf("the test is not about what it was: %s", was)
			}
		})
	}
}

// For every message which the old way rewrote properly, the result is the same. The HTML is made of the usual ways of
// linking an image (and of not doing so), in a random order, with IDs which are different in case and look alike.
func TestInlineImagesSameAsBefore(t *testing.T) {
	rnd := rand.New(rand.NewSource(7))

	ids := []string{"a", "B", "logo", "a@b.test", "image001.png@01D6.F3A0", "x-y_z", "Img.2", "c0", "c00"}
	cases, changed := 0, 0

	for range 3000 {
		var inline, attachments []storage.Attachment
		for n, id := range ids {
			if rnd.Intn(3) == 0 {
				continue
			}
			p := part(fmt.Sprint("p", n), id)
			if rnd.Intn(2) == 0 {
				p.ContentID = strings.ToUpper(id)
			}
			if rnd.Intn(2) == 0 {
				inline = append(inline, p)
			} else {
				attachments = append(attachments, p)
			}
		}
		// parts which share a Content-ID, and some which have none
		inline = append(inline, part("dup1", "logo"), part("none", ""))
		attachments = append(attachments, part("dup2", "LOGO"), part("dup3", "a"))

		var html strings.Builder
		for range rnd.Intn(30) {
			id := ids[rnd.Intn(len(ids))]
			if rnd.Intn(5) == 0 {
				id = "none-" + id
			}
			if rnd.Intn(3) == 0 {
				id = strings.ToUpper(id)
			}
			switch rnd.Intn(11) {
			case 0:
				fmt.Fprintf(&html, `<img src="cid:%s">`, id)
			case 1:
				fmt.Fprintf(&html, `<img src='cid:%s'>`, id)
			case 2:
				fmt.Fprintf(&html, `<img src=cid:%s>`, id)
			case 3:
				fmt.Fprintf(&html, `<img src=cid:%s/>`, id)
			case 4:
				fmt.Fprintf(&html, `<td background="CID:%s">`, id)
			case 5:
				fmt.Fprintf(&html, `<p>cid:%s</p> url(cid:%s)`, id, id)
			case 6:
				html.WriteString(`<td a="1" b='2' c=3>text = more</td>`)
			case 7:
				fmt.Fprintf(&html, `<img src="cid:%s%s">`, id, "xy") // an ID and more
			case 8:
				fmt.Fprintf(&html, `<img src="cid:%s">`, id[:len(id)/2]) // less than an ID
			case 9:
				fmt.Fprintf(&html, `<a href="cid:%s"><img alt="x" src="cid:%s"></a>`, id, id)
			default:
				html.WriteString("some text, with = and \"quotes\" and 'more' <b>bold</b>\n")
			}
		}

		msg := message(html.String(), inline, attachments...)
		got, want := linkInlineImages(msg), referenceLinkInlineImages(msg)
		if got != want {
			t.Fatalf("case %d\nhtml: %s\ngot:  %s\nwant: %s", cases, msg.HTML, got, want)
		}
		if got != msg.HTML {
			changed++
		}
		cases++
	}

	// the test is only worth something if most of the messages had links which were rewritten
	if changed < 2500 {
		t.Fatalf("only %d of %d messages had a link rewritten", changed, cases)
	}
}

// A reference to a part from the HTML of a message which is stored and viewed is rewritten as it was.
func TestViewHTMLRewritesInlineImages(t *testing.T) {
	initTestDB(t)

	raw := []byte("From: a@example.test\r\nTo: b@example.test\r\nSubject: inline\r\nMIME-Version: 1.0\r\n" +
		"Content-Type: multipart/related; boundary=\"b1\"\r\n\r\n" +
		"--b1\r\nContent-Type: text/html; charset=utf-8\r\n\r\n" +
		`<html><body><img src="cid:logo@test"><img src=cid:logo@test width=5><img src="cid:other"></body></html>` + "\r\n" +
		"--b1\r\nContent-Type: image/png\r\nContent-ID: <logo@test>\r\nContent-Disposition: inline; filename=\"l.png\"\r\n" +
		"Content-Transfer-Encoding: base64\r\n\r\niVBORw0KGgo=\r\n--b1--\r\n")
	id, err := storage.Store(&raw, nil)
	if err != nil {
		t.Fatal(err)
	}

	msg, err := storage.GetMessage(id)
	if err != nil || len(msg.Inline) != 1 {
		t.Fatalf("%v, %d inline parts", err, len(msg.Inline))
	}
	want := config.Webroot + "api/v1/message/" + id + "/part/" + msg.Inline[0].PartID

	w := viewHTML(t.Context(), id, "")
	if w.Code != http.StatusOK {
		t.Fatalf("status %d", w.Code)
	}
	body := w.Body.String()
	if strings.Count(body, `"`+want+`"`) != 1 || !strings.Contains(body, `src=`+want+` `) || !strings.Contains(body, `"cid:other"`) || strings.Contains(body, "cid:logo") {
		t.Fatalf("the links are not as expected (%s): %s", want, body)
	}
}

// How long it takes is in proportion to the size of the HTML, whatever the HTML is made of. The ways it was not (80,000
// references to one part took 93 seconds, 1500 parts took 15 times as long as 100) are the same size here and take
// milliseconds. The limits are generous (a machine which is slow, or the race detector, is 20 times slower).
func TestInlineImagesTimeIsInProportionToTheHTML(t *testing.T) {
	many := make([]storage.Attachment, 5000)
	for i := range many {
		many[i] = part(fmt.Sprint("p", i), fmt.Sprintf("id-%d", i))
	}

	var allRefs strings.Builder
	for i := range 5000 {
		fmt.Fprintf(&allRefs, `<img src="cid:id-%d">`, i)
	}

	for name, tc := range map[string]*storage.Message{
		"80,000 references to one part":   message(strings.Repeat(`<img src="cid:logo">`, 80000), []storage.Attachment{part("1", "logo")}),
		"5000 parts, each referenced":     message(allRefs.String(), many),
		"5000 parts, and many references": message(strings.Repeat(allRefs.String(), 20), many),
		"many attributes":                 message(strings.Repeat(`<td a="1" b="2" c="3" d="4" e="5">x</td>`, 40000), many),
		// each of these starts what could be a link, and has no end for it
		"= and cid: with no end":               message(strings.Repeat(`=cid:`, 300000), many),
		"quote and cid: with no closing":       message(strings.Repeat(`="cid:`, 250000), many),
		"single quote and cid: with no end":    message(strings.Repeat(`='cid:`, 250000), many),
		"long IDs which are not":               message(strings.Repeat(`<img src="cid:`+strings.Repeat("x", 500)+`">`, 3000), many),
		"long unquoted IDs which are not":      message(strings.Repeat(`<img src=cid:`+strings.Repeat("x", 500)+`>`, 3000), many),
		"unquoted with the end far away":       message(strings.Repeat(`=cid:xxxxxxxxxx`, 100000)+" ", many),
		"a quoted one with many inside":        message(`="cid:`+strings.Repeat(`=cid:`, 300000)+`"`, many),
		"a quoted one with many quoted inside": message(`='cid:`+strings.Repeat(`="cid:`, 200000)+`'`, many),
	} {
		t.Run(name, func(t *testing.T) {
			done := make(chan time.Duration, 1)
			go func() {
				start := time.Now()
				_ = linkInlineImages(tc)
				done <- time.Since(start)
			}()

			// a limit on the wait, so that if it is not in proportion to the HTML this fails rather than going on
			select {
			case took := <-done:
				if took > 5*time.Second {
					t.Fatalf("took %v for %d KiB of HTML", took, len(tc.HTML)>>10)
				}
			case <-time.After(10 * time.Second):
				t.Fatalf("still going after 10 seconds, for %d KiB of HTML", len(tc.HTML)>>10)
			}
		})
	}
}

// And it grows with the HTML, not with the square of it: the time for four times as much is about four times, where
// reading it again for each reference made it sixteen times as long.
func TestInlineImagesTimeGrowsInStep(t *testing.T) {
	timeOf := func(refs int) time.Duration {
		msg := message(strings.Repeat(`<img src="cid:logo">`, refs), []storage.Attachment{part("1", "logo")})

		best := time.Hour
		for range 5 {
			start := time.Now()
			_ = linkInlineImages(msg)
			best = min(best, time.Since(start))
		}

		return best
	}

	small, large := timeOf(50000), timeOf(200000)
	if large > 10*small+50*time.Millisecond {
		t.Fatalf("200,000 references took %v, and 50,000 took %v", large, small)
	}
	t.Logf("50,000 references: %v, 200,000: %v", small, large)
}

// The result is made of what was in the HTML and the URLs, and nothing more: the HTML with the links taken out is the
// HTML with the links taken out of what it was, for any HTML at all.
func TestInlineImagesOnlyChangesTheLinks(t *testing.T) {
	rnd := rand.New(rand.NewSource(11))
	alphabet := []string{"=", `"`, `'`, "cid:", "CID:", "a", "B", "logo", " ", ">", "<", "/", ";", "\n", "é", "x", "=\"", "='", "=cid:", `="cid:`}

	for range 20000 {
		var sb strings.Builder
		for range rnd.Intn(40) {
			sb.WriteString(alphabet[rnd.Intn(len(alphabet))])
		}
		html := sb.String()

		msg := message(html, []storage.Attachment{part("1", "logo"), part("2", "a"), part("3", "B")})
		got := linkInlineImages(msg)

		// put the "cid:<ID>" back where each URL is, and it must be the HTML
		back := got
		for _, p := range []struct{ id, partID string }{{"logo", "1"}, {"a", "2"}, {"B", "3"}} {
			back = strings.ReplaceAll(back, partURL(p.partID), "cid:"+p.id)
		}
		if lower := strings.ToLower(back); lower != strings.ToLower(html) {
			t.Fatalf("html %q\ngot  %q\nback %q", html, got, back)
		}
	}
}

func FuzzInlineImages(f *testing.F) {
	for _, s := range []string{`<img src="cid:logo">`, `<img src=cid:logo/>`, `=cid:`, `="cid:logo`, "='cid:a'='cid:a'", `<td a="1">=`, ""} {
		f.Add(s)
	}

	msg := message("", []storage.Attachment{part("1", "logo"), part("2", "a"), part("3", "Ünï")})

	f.Fuzz(func(t *testing.T, html string) {
		m := *msg
		m.HTML = html

		got := linkInlineImages(&m)

		// it never takes anything away other than the links, and the URLs are all that is added
		if len(got) > len(html)+(strings.Count(html, "cid:")+strings.Count(strings.ToLower(html), "cid:"))*len(partURL("3")) {
			t.Fatalf("the result is much longer than it can be: %d from %d", len(got), len(html))
		}
		if !strings.Contains(html, "=") && got != html {
			t.Fatalf("changed HTML which has no attribute: %q -> %q", html, got)
		}
	})
}

// FuzzInlineImagesSameAsBefore makes HTML from units which are each a complete piece (so what comes before and after
// a link can not change how it is read, which is where the old and the new way are meant to differ) and compares the
// result with what the old way gave.
func FuzzInlineImagesSameAsBefore(f *testing.F) {
	f.Add([]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13})
	f.Add([]byte{3, 3, 3, 9, 9, 12, 12, 5})
	f.Add([]byte{})

	ids := []string{"a", "B", "c.d@e", "x-y_z", "none"}
	units := []func(id string) string{
		func(id string) string { return `<img src="cid:` + id + `">` },
		func(id string) string { return `<img src='cid:` + id + `'>` },
		func(id string) string { return `<img src=cid:` + id + `>` },
		func(id string) string { return `<img src=cid:` + id + `/>` },
		func(id string) string { return `<td background="CID:` + strings.ToUpper(id) + `">` },
		func(id string) string { return `<p>cid:` + id + ` url(cid:` + id + `)</p>` },
		func(id string) string { return `<a a="1" b='2' c=3 href="cid:` + id + `">text = more</a>` },
		func(id string) string { return `<img src="cid:` + id + `xy">` },
		func(id string) string { return `<img src="cid:` + id[:len(id)/2] + `">` },
		func(id string) string { return `<b title="a = b">=cid:</b>` },
		func(id string) string { return `<i x="` + id + `">` },
	}

	msg := message("", []storage.Attachment{part("1", "a"), part("2", "c.d@e")},
		part("3", "b"), part("4", "X-Y_Z"), part("5", ""), part("6", "a"))

	f.Fuzz(func(t *testing.T, in []byte) {
		var html strings.Builder
		for i := 0; i+1 < len(in) && i < 400; i += 2 {
			html.WriteString(units[int(in[i])%len(units)](ids[int(in[i+1])%len(ids)]))
		}

		m := *msg
		m.HTML = html.String()
		if got, want := linkInlineImages(&m), referenceLinkInlineImages(&m); got != want {
			t.Fatalf("html: %s\ngot:  %s\nwant: %s", m.HTML, got, want)
		}
	})
}
