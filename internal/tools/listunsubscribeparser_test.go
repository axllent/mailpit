package tools

import (
	"fmt"
	"math/rand"
	"slices"
	"strings"
	"testing"
	"time"
)

// oldStripComments is how comments used to be removed: each one found is removed from the whole
// value with a separate search.
func oldStripComments(v string) string {
	v = strings.TrimSpace(v)
	for _, c := range luCommentsRe.FindAllStringSubmatch(v, -1) {
		v = strings.ReplaceAll(v, c[0], "")
		v = strings.TrimSpace(v)
	}

	return v
}

// hasNestedComment reports whether a comment found in v has an opening bracket inside it.
func hasNestedComment(v string) bool {
	for _, c := range luCommentsRe.FindAllString(v, -1) {
		if strings.Contains(c[1:], "(") {
			return true
		}
	}

	return false
}

// The single pass gives the same result as before for comments which are not nested. For nested or
// unbalanced brackets the old way removed an inner comment first and left a stray part of the outer
// one behind, where a comment now runs from its "(" to the first ")" that follows.
func TestStripListUnsubscribeCommentsMatchesOldResults(t *testing.T) {
	fixed := []string{
		"", "()", "(a)", "(a)(a)", "<x>(a)", "(a)<x>(b)", "(a\n b)<x>", "(a) , (b)", "(a", "a)", ")(a)(",
		"(Use this command to get off the list) <mailto:unsubscribe@example.com?subject=unsubscribe%20me>",
		"<mailto:unsubscribe@example.com> (Use this command to get off the list)",
		"(Use this) <mailto:unsubscribe@example.com>, (Click this) <http://example.com>",
	}
	for _, v := range fixed {
		if got, want := stripListUnsubscribeComments(v), oldStripComments(v); got != want {
			t.Errorf("%q: got %q, want %q", v, got, want)
		}
	}

	rng := rand.New(rand.NewSource(1))
	alphabet := []string{"(", ")", "a", "b", " ", "<", ">", ",", "\n", "mailto:x@y.test", "(a)", "(b)"}
	for range 50000 {
		var b strings.Builder
		for range rng.Intn(14) {
			b.WriteString(alphabet[rng.Intn(len(alphabet))])
		}
		v := b.String()
		if hasNestedComment(v) {
			continue
		}
		if got, want := stripListUnsubscribeComments(v), oldStripComments(v); got != want {
			t.Fatalf("%q: got %q, want %q", v, got, want)
		}
	}

	// nested brackets: the comment is what the expression matches, from "(" to the first ")"
	if got := stripListUnsubscribeComments("<x>(a (b) c)"); got != "<x> c)" {
		t.Errorf("nested comment: %q", got)
	}
}

// Removing the comments took 48 seconds for a 600 KiB header of different comments, as each one was
// searched for in the whole value.
func TestStripListUnsubscribeCommentsIsLinear(t *testing.T) {
	build := func(distinct bool) string {
		var b strings.Builder
		b.WriteString("<https://example.com/u>")
		for i := range 40000 {
			if distinct {
				fmt.Fprintf(&b, "(c%d)", i)
			} else {
				b.WriteString("(c)")
			}
		}

		return b.String()
	}

	for _, distinct := range []bool{false, true} {
		v := build(distinct)

		start := time.Now()
		got := stripListUnsubscribeComments(v)
		if d := time.Since(start); d > time.Second {
			t.Errorf("distinct=%v: %d KiB took %v", distinct, len(v)>>10, d)
		}
		if got != "<https://example.com/u>" {
			t.Errorf("distinct=%v: comments left in: %.60q", distinct, got)
		}
	}
}

func TestListUnsubscribeParserResults(t *testing.T) {
	links, err := ListUnsubscribeParser("(Use this) <mailto:unsubscribe@example.com>, (Click this) <https://example.com/u?x=1>")
	if err != nil || !slices.Equal(links, []string{"mailto:unsubscribe@example.com", "https://example.com/u?x=1"}) {
		t.Fatalf("links %v, err %v", links, err)
	}
}

// The value is only limited by the size of the message, so a very long one is not parsed.
func TestListUnsubscribeParserLengthLimit(t *testing.T) {
	link := "<https://example.com/u>"

	// control: a value just within the limit, which is mostly comments, is parsed
	within := link + "(" + strings.Repeat("a", maxListUnsubscribeLength-len(link)-2) + ")"
	if len(within) != maxListUnsubscribeLength {
		t.Fatalf("test value is %d bytes", len(within))
	}
	links, err := ListUnsubscribeParser(within)
	if err != nil || len(links) != 1 {
		t.Fatalf("a value at the limit: links %v, err %v", links, err)
	}

	// exploit: one byte more is refused, and without repeating the value in the error
	_, err = ListUnsubscribeParser(within + "a")
	if err == nil {
		t.Fatal("a value over the limit was parsed")
	}
	if len(err.Error()) > 200 {
		t.Errorf("the error is %d bytes long, it should not include the value", len(err.Error()))
	}

	// exploit: the case that took 48 seconds is now refused at once
	var b strings.Builder
	b.WriteString(link)
	for i := range 80000 {
		fmt.Fprintf(&b, "(c%d)", i)
	}
	start := time.Now()
	if _, err := ListUnsubscribeParser(b.String()); err == nil {
		t.Error("a 600 KiB value was parsed")
	}
	if d := time.Since(start); d > time.Second {
		t.Errorf("a 600 KiB value took %v", d)
	}
}

func FuzzListUnsubscribeParser(f *testing.F) {
	f.Add("<mailto:unsubscribe@example.com>, <https://example.com>")
	f.Add("(Use this) <mailto:unsubscribe@example.com> (and this)")
	f.Add("<<>>(()")
	f.Add("")

	f.Fuzz(func(t *testing.T, v string) {
		links, err := ListUnsubscribeParser(v)
		if links == nil {
			t.Fatal("nil links")
		}
		if err == nil && len(links) == 0 {
			t.Fatal("no links and no error")
		}
		for _, l := range links {
			if strings.ContainsAny(l, " \t\r\n") {
				t.Fatalf("link %q contains whitespace", l)
			}
		}
		if got, want := stripListUnsubscribeComments(v), oldStripComments(v); len(v) < 200 && !hasNestedComment(v) && got != want {
			t.Fatalf("%q: got %q, want %q", v, got, want)
		}
	})
}
