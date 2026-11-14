package htmlcheck

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/axllent/mailpit/config"
)

// resetCIE forgets the loaded data so the next load starts from scratch.
// It must only be called while no other goroutine is using the data.
func resetCIE() {
	cieOnce = sync.Once{}
	cieErr = nil
	cie = CanIEmail{}
}

func blockRemoteCSS(t *testing.T) {
	t.Helper()
	config.BlockRemoteCSSAndFonts = true
	t.Cleanup(func() { config.BlockRemoteCSSAndFonts = false })
}

func finite(f float32) bool {
	return !math.IsNaN(float64(f)) && !math.IsInf(float64(f), 0)
}

// Concurrent first use must not race on (or half-read) the shared data.
// Without synchronisation this fails with a fatal concurrent map access.
func TestLoadJSONDataConcurrent(t *testing.T) {
	t.Cleanup(resetCIE)

	for round := 0; round < 10; round++ {
		resetCIE()

		var wg sync.WaitGroup
		errs := make(chan error, 16)
		for i := 0; i < 8; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				if err := loadJSONData(); err != nil {
					errs <- err
					return
				}
				if len(cie.Data) == 0 || cie.APIVersion == "" {
					errs <- fmt.Errorf("data not fully loaded: %d tests", len(cie.Data))
					return
				}
				if _, err := Platforms(); err != nil {
					errs <- err
				}
			}()
		}
		wg.Wait()
		close(errs)

		for err := range errs {
			t.Fatal(err)
		}
	}
}

func TestLoadJSONDataIsLoadedOnce(t *testing.T) {
	t.Cleanup(resetCIE)
	resetCIE()

	if err := loadJSONData(); err != nil {
		t.Fatal(err)
	}
	first := &cie.Data[0]

	if err := loadJSONData(); err != nil {
		t.Fatal(err)
	}
	if first != &cie.Data[0] {
		t.Fatal("data was reloaded by a second call")
	}
}

// Documents with no elements to weight against must still produce a score that can be encoded.
func TestRunTestsScoreIsFinite(t *testing.T) {
	blockRemoteCSS(t)

	tests := []struct {
		name      string
		html      string
		wantNodes int
	}{
		{"control: body with a styled element", `<html><body><p style="display:flex">x</p></body></html>`, 2},
		{"control: svg only", `<svg></svg>`, 1},
		{"control: comment only", `<!-- nothing -->`, 0},
		{"exploit: style only, no elements", `<style>a{color:red}</style>`, 0},
		{"exploit: head with style, no body", `<html><head><style>a{}</style></head></html>`, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := RunTests(context.Background(), tt.html)
			if err != nil {
				t.Fatal(err)
			}

			if r.Total.Nodes != tt.wantNodes {
				t.Fatalf("nodes = %d, want %d", r.Total.Nodes, tt.wantNodes)
			}

			if !finite(r.Total.Supported) || !finite(r.Total.Partial) || !finite(r.Total.Unsupported) {
				t.Fatalf("non-finite totals: %+v", r.Total)
			}

			if _, err := json.Marshal(r); err != nil {
				t.Fatalf("response cannot be encoded: %v", err)
			}

			// no elements means nothing to weight, so nothing is reported as unsupported
			if tt.wantNodes == 0 && r.Total.Supported != 100 {
				t.Fatalf("supported = %v, want 100", r.Total.Supported)
			}
		})
	}
}

func TestNoteNumber(t *testing.T) {
	tests := []struct{ in, want string }{
		{"a #7", "7"},
		{"a #12", "12"}, // used to return "2"
		{"a #123", "123"},
		{"y #11", "11"},
		{"a", ""},
		{"a #", ""},
		{"a #1 b", ""},
	}

	for _, tt := range tests {
		if got := noteNumber(tt.in); got != tt.want {
			t.Errorf("noteNumber(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

// Check every partial result in the real data carries the full note number.
func TestNoteNumbersMatchData(t *testing.T) {
	if err := loadJSONData(); err != nil {
		t.Fatal(err)
	}

	re := regexp.MustCompile(` #(\d+)$`)
	checked, multiDigit := 0, 0

	for _, test := range cie.Data {
		w, err := cie.getTest(test.Slug)
		if err != nil {
			t.Fatal(err)
		}

		want := map[string]string{}
		for family, stats := range test.Stats {
			for platform, clients := range stats.(map[string]any) {
				for version, support := range clients.(map[string]any) {
					if m := re.FindStringSubmatch(fmt.Sprint(support)); m != nil && support != "y" && support != "n" {
						want[family+"/"+platform+"/"+version] = m[1]
					}
				}
			}
		}

		for _, r := range w.Results {
			exp, ok := want[r.Family+"/"+r.Platform+"/"+r.Version]
			if !ok {
				continue
			}
			checked++
			if len(exp) > 1 {
				multiDigit++
			}
			if r.NoteNumber != exp {
				t.Fatalf("%s %s: note number = %q, want %q", test.Slug, r.Name, r.NoteNumber, exp)
			}
		}
	}

	if checked == 0 {
		t.Fatal("no results with note numbers were checked")
	}
	t.Logf("checked %d note numbers, %d with more than one digit", checked, multiDigit)
}

func TestCollectStyleCSS(t *testing.T) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(`<style>a{}</style><p>x</p><style>b{}</style>`))
	if err != nil {
		t.Fatal(err)
	}

	if got := collectStyleCSS(doc); got != "a{}b{}" {
		t.Fatalf("got %q", got)
	}
}

// Collecting the CSS must be linear in the number of <style> elements. With repeated string
// concatenation 80,000 elements took over 10 seconds, this takes a few milliseconds.
func TestCollectStyleCSSIsLinear(t *testing.T) {
	const n = 100000

	for _, tt := range []struct {
		name  string
		count int
		limit time.Duration
	}{
		{"control: few style elements", 10, time.Second},
		{"exploit: many style elements", n, 3 * time.Second},
	} {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := goquery.NewDocumentFromReader(strings.NewReader(
				"<html><head>" + strings.Repeat("<style>a{color:red}</style>", tt.count) + "</head><body>x</body></html>"))
			if err != nil {
				t.Fatal(err)
			}

			start := time.Now()
			css := collectStyleCSS(doc)
			elapsed := time.Since(start)

			if want := len("a{color:red}") * tt.count; len(css) != want {
				t.Fatalf("collected %d bytes, want %d", len(css), want)
			}
			if elapsed > tt.limit {
				t.Fatalf("took %v for %d style elements", elapsed, tt.count)
			}
		})
	}
}

func TestMergeWithinBudget(t *testing.T) {
	tests := []struct {
		name     string
		elements int
		css      int
		want     bool
	}{
		{"control: typical newsletter", 3000, 50 * 1024, true},
		{"control: large newsletter", 10000, 90 * 1024, true},
		{"control: no css", 1000000, 0, true},
		{"exploit: many elements and a lot of css", 100000, 100000, false},
		{"exploit: just over the limit", 1000, int(maxMergeWork/1000) + 1, false},
		{"control: exactly at the limit", 1000, int(maxMergeWork / 1000), true},
	}

	for _, tt := range tests {
		if got := mergeWithinBudget(tt.elements, tt.css); got != tt.want {
			t.Errorf("%s: mergeWithinBudget(%d, %d) = %v, want %v", tt.name, tt.elements, tt.css, got, tt.want)
		}
	}
}

// When merging is skipped for being too large, the checks that do not need it must still run.
func TestRunCSSTestsStillWorksWhenMergeSkipped(t *testing.T) {
	blockRemoteCSS(t)
	if err := loadJSONData(); err != nil {
		t.Fatal(err)
	}

	build := func(elements, rules int) string {
		var sb strings.Builder
		sb.WriteString("<html><head><style>")
		for i := 0; i < rules; i++ {
			fmt.Fprintf(&sb, ".c%d{color:red;margin:0}", i)
		}
		sb.WriteString("@media (min-width:1px){p{color:red}}</style></head><body>")
		for i := 0; i < elements; i++ {
			sb.WriteString(`<p style="display:flex">x</p>`)
		}
		sb.WriteString("</body></html>")

		return sb.String()
	}

	// control: well within the budget so the merge runs
	// exploit: elements * css bytes is over the budget so the merge is skipped
	for _, tt := range []struct {
		name            string
		elements, rules int
		wantMerged      bool
	}{
		{"control: within budget", 50, 100, true},
		{"exploit: over budget", 1500, 30000, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			h := build(tt.elements, tt.rules)

			doc, err := goquery.NewDocumentFromReader(strings.NewReader(h))
			if err != nil {
				t.Fatal(err)
			}
			if got := mergeWithinBudget(len(doc.Find("*").Nodes), len(collectStyleCSS(doc))); got != tt.wantMerged {
				t.Fatalf("test input has mergeWithinBudget = %v, want %v", got, tt.wantMerged)
			}

			results, _, err := runCSSTests(context.Background(), h)
			if err != nil {
				t.Fatal(err)
			}

			found := map[string]int{}
			for _, r := range results {
				found[r.Slug] = r.Score.Found
			}
			if found["css-display-flex"] != tt.elements {
				t.Errorf("css-display-flex found %d times, want %d", found["css-display-flex"], tt.elements)
			}
			if found["css-at-media"] == 0 {
				t.Error("@media in the style block was not detected")
			}
		})
	}
}
