package tools

import (
	"strings"
	"testing"
)

func wrap(j string) string {
	return `<html><head><script type="application/ld+json">` + j + `</script></head><body>x</body></html>`
}

func TestParseEmailActions(t *testing.T) {
	good := wrap(`{"@context":"http://schema.org","@type":"EmailMessage","potentialAction":{"@type":"ConfirmAction","name":"Approve","handler":{},"target":{"@type":"HttpActionHandler","url":"https://example.com/approve"}}}`)
	a := ParseEmailActions(good)
	if len(a) != 1 || a[0].Type != "ConfirmAction" || a[0].Name != "Approve" || a[0].URL != "https://example.com/approve" || len(a[0].Errors) != 0 {
		t.Fatalf("unexpected: %+v", a)
	}

	// @graph + array of actions + string target
	g := wrap(`{"@graph":[{"@type":["EmailMessage"],"potentialAction":[{"@type":"ViewAction","name":"View","target":"https://e.com/1"},{"@type":"SaveAction","name":"Save","target":"https://e.com/2"}]}]}`)
	if a := ParseEmailActions(g); len(a) != 2 || a[1].URL != "https://e.com/2" {
		t.Fatalf("graph: %+v", a)
	}

	// validation errors
	bad := wrap(`{"@type":"EmailMessage","potentialAction":{"@type":"FooAction","target":"http://e.com"}}`)
	a = ParseEmailActions(bad)
	if len(a) != 1 || len(a[0].Errors) != 3 {
		t.Fatalf("expected 3 errors: %+v", a)
	}

	// not on an EmailMessage
	if a := ParseEmailActions(wrap(`{"@type":"Order","potentialAction":{"@type":"ViewAction","name":"V","target":"https://e.com"}}`)); len(a) != 1 || len(a[0].Errors) != 1 {
		t.Fatalf("non-email: %+v", a)
	}

	// case-insensitive script type and schema.org URL types
	ci := `<SCRIPT TYPE="Application/LD+JSON">{"@type":"http://schema.org/EmailMessage","potentialAction":{"@type":"https://schema.org/ViewAction","name":"V","url":"https://e.com"}}</SCRIPT>`
	if a := ParseEmailActions(ci); len(a) != 1 || len(a[0].Errors) != 0 || a[0].Type != "ViewAction" {
		t.Fatalf("case: %+v", a)
	}

	// invalid JSON is reported, no markup is empty
	if a := ParseEmailActions(wrap(`{nope`)); len(a) != 1 || len(a[0].Errors) != 1 {
		t.Fatalf("invalid: %+v", a)
	}
	if a := ParseEmailActions("<p>hello</p>"); len(a) != 0 || a == nil {
		t.Fatalf("none: %+v", a)
	}
}

func TestParseEmailActionsLimits(t *testing.T) {
	one := `{"@type":"ViewAction","name":"V","target":"https://e.com"}`
	many := wrap(`{"@type":"EmailMessage","potentialAction":[` + strings.Repeat(one+",", 50) + one + `]}`)
	if a := ParseEmailActions(many); len(a) != maxEmailActions {
		t.Fatalf("got %d", len(a))
	}
	deep := strings.Repeat(`[`, 5000) + strings.Repeat(`]`, 5000)
	_ = ParseEmailActions(wrap(deep)) // must not blow up
	if a := ParseEmailActions(wrap(`"` + strings.Repeat("a", maxEmailActionsJSON) + `"`)); len(a) != 1 {
		t.Fatalf("large: %+v", a)
	}
}
