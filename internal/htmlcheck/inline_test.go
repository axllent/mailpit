package htmlcheck

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
)

func TestInlineStyleDetection(t *testing.T) {
	/// tests should contain the HTML test, and expected test results in alphabetical order
	tests := map[string]string{}
	tests[`<h1 style="transform: rotate(20deg)">Heading</h1>`] = "css-transform"
	tests[`<h1 style="color: green; transform:rotate(20deg)">Heading</h1>`] = "css-transform"
	tests[`<h1 style="color:green; transform :rotate(20deg)">Heading</h1>`] = "css-transform"
	tests[`<h1 style="transform:rotate(20deg)">Heading</h1>`] = "css-transform"
	tests[`<h1 style="TRANSFORM:rotate(20deg)">Heading</h1>`] = "css-transform"
	tests[`<h1 style="transform:	rotate(20deg)">Heading</h1>`] = "css-transform"
	tests[`<h1 style="ignore-transform: something">Heading</h1>`] = "" // no match
	tests[`<h1 style="text-transform: uppercase">Heading</h1>`] = "css-text-transform"
	tests[`<h1 style="text-transform: uppercase; text-transform: uppercase">Heading</h1>`] = "css-text-transform"
	tests[`<h1 style="test-transform: uppercase">Heading</h1>`] = "" // no match
	tests[`<h1 style="padding-inline-start: 5rem">Heading</h1>`] = "css-padding-inline-start-end"
	tests[`<h1 style="margin-inline-end: 5rem">Heading</h1>`] = "css-margin-inline-start-end"
	tests[`<h1 style="margin-inline-middle: 5rem">Heading</h1>`] = "" // no match
	tests[`<h1 style="color:green!important">Heading</h1>`] = "css-important"
	tests[`<h1 style="color: green !important">Heading</h1>`] = "css-important"
	tests[`<h1 style="color: green!important;">Heading</h1>`] = "css-important"
	tests[`<h1 style="color:green!important-stuff;">Heading</h1>`] = "" // no match
	tests[`<h1 style="background-image:url('img.jpg')">Heading</h1>`] = "css-background-image"
	tests[`<h1 style="background-image:url('img.jpg'); color: green">Heading</h1>`] = "css-background-image"
	tests[`<h1 style=" color:green; background-image:url('img.jpg');">Heading</h1>`] = "css-background-image"
	tests[`<h1 style="display  : flex ;">Heading</h1>`] = "css-display,css-display-flex"
	tests[`<h1 style="DISPLAY:FLEX;">Heading</h1>`] = "css-display,css-display-flex"
	tests[`<h1 style="display: flexing;">Heading</h1>`] = "css-display" // should not match css-display-flex rule
	tests[`<h1 style="line-height: 1rem;opacity: 0.5; width: calc(10px + 100px)">Heading</h1>`] = "css-line-height,css-opacity,css-unit-calc,css-width"
	tests[`<h1 style="color: rgb(255,255,255);">Heading</h1>`] = "css-rgb"
	tests[`<h1 style="color:rgb(255,255,255);">Heading</h1>`] = "css-rgb"
	tests[`<h1 style="color:rgb(255,255,255);">Heading</h1>`] = "css-rgb"
	tests[`<h1 style="color:rgba(255,255,255, 0);">Heading</h1>`] = "css-rgba"
	tests[`<h1 style="border: solid rgb(255,255,255) 1px; color:rgba(255,255,255, 0);">Heading</h1>`] = "css-border,css-rgb,css-rgba"
	tests[`<h1 border="2">Heading</h1>`] = "css-border"
	tests[`<h1 border="2" background="green">Heading</h1>`] = "css-background,css-border"
	tests[`<h1 BORDER="2" BACKGROUND="GREEN">Heading</h1>`] = "css-background,css-border"
	tests[`<h1 border-something="2" background-something="green">Heading</h1>`] = "" // no match
	tests[`<h1 border="2" style="border: solid green 1px!important">Heading</h1>`] = "css-border,css-important"

	for html, expected := range tests {
		reader := strings.NewReader(html)
		doc, err := goquery.NewDocumentFromReader(reader)
		if err != nil {
			t.Log("error ", err)
			t.Fail()
		}

		results := testInlineStyles(doc)

		matches := []string{}
		uniqMap := make(map[string]bool)
		for key := range results {
			if _, exists := uniqMap[key]; !exists {
				matches = append(matches, key)
			}
		}

		// ensure results are sorted to ensure consistent results
		sort.Strings(matches)

		assertEqual(t, expected, strings.Join(matches, ","), fmt.Sprintf("inline style detection \"%s\"", html))
	}
}

func assertEqual(t *testing.T, a interface{}, b interface{}, message string) {
	if a == b {
		return
	}
	message = fmt.Sprintf("%s: \"%v\" != \"%v\"", message, a, b)
	t.Fatal(message)
}
