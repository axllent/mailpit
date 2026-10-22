import assert from "node:assert/strict";
import test from "node:test";
import createDOMPurify from "dompurify";
import { JSDOM } from "jsdom";
import { sanitizeHTML } from "./sanitize.js";

// An isolated DOMPurify instance per test file run, bound to a jsdom window
const purify = createDOMPurify(new JSDOM("").window);
const parse = (html) => new JSDOM(sanitizeHTML(html, purify)).window.document;
const body = (html) => parse(html).body.innerHTML;

// Each case pairs an exploit with a benign control of the same shape, so that a change which
// makes both behave identically (all stripped, or all allowed) fails the suite.

test("script tags are removed, surrounding content is kept", () => {
	assert.equal(body("<p>hi</p><script>alert(1)</script>"), "<p>hi</p>");
	assert.equal(body("<svg><script>alert(1)</script></svg>"), "<svg></svg>");
	// control
	assert.equal(body("<p>hi</p><b>there</b>"), "<p>hi</p><b>there</b>");
});

test("inline event handlers are removed, other attributes are kept", () => {
	assert.equal(body('<img src="x" onerror="alert(1)">'), '<img src="x">');
	assert.equal(body('<div style="color:red" onclick="x()">a</div>'), '<div style="color:red">a</div>');
	// control
	assert.equal(body('<img src="x" alt="logo">'), '<img src="x" alt="logo">');
});

test("javascript: URLs are removed, other protocols are kept", () => {
	for (const href of ["javascript:alert(1)", "JaVaScRiPt:alert(1)", " javascript:alert(1)"]) {
		const a = parse(`<a href="${href}">x</a>`).querySelector("a");
		assert.equal(a.hasAttribute("href"), false, `href survived: ${href}`);
	}
	// controls: allowed protocols, including custom app schemes (ALLOW_UNKNOWN_PROTOCOLS)
	for (const href of ["https://example.com", "mailto:a@example.com", "myapp://open"]) {
		const a = parse(`<a href="${href}">x</a>`).querySelector("a");
		assert.equal(a.getAttribute("href"), href);
	}
});

test("forms and iframes are removed, form fields are kept", () => {
	assert.equal(body('<form action="/steal"><input name="a"></form>'), '<input name="a">');
	assert.equal(body('<iframe src="https://evil.example"></iframe>'), "");
	// control
	assert.equal(
		body("<table><tbody><tr><td>x</td></tr></tbody></table>"),
		"<table><tbody><tr><td>x</td></tr></tbody></table>",
	);
});

test("external links open in a new tab with noopener noreferrer", () => {
	const a = parse('<a href="https://example.com">x</a>').querySelector("a");
	assert.equal(a.getAttribute("target"), "_blank");
	assert.equal(a.getAttribute("rel"), "noopener noreferrer");
});

test("in-page anchors are not given a target", () => {
	const a = parse('<a href="#top">x</a>').querySelector("a");
	assert.equal(a.getAttribute("href"), "#top");
	assert.equal(a.hasAttribute("target"), false);
	assert.equal(a.hasAttribute("rel"), false);
});

test("links with an attacker-set target are overridden", () => {
	const a = parse('<a href="https://example.com" target="_top" rel="opener">x</a>').querySelector("a");
	assert.equal(a.getAttribute("target"), "_blank");
	assert.equal(a.getAttribute("rel"), "noopener noreferrer");
});

test("link hook is registered once per instance, not once per call", () => {
	let hooks = 0;
	const counting = createDOMPurify(new JSDOM("").window);
	const addHook = counting.addHook.bind(counting);
	counting.addHook = (...args) => {
		hooks++;
		return addHook(...args);
	};
	for (let i = 0; i < 5; i++) {
		sanitizeHTML("<a href='https://example.com'>x</a>", counting);
	}
	assert.equal(hooks, 1);
});

test("email-specific head and attributes survive", () => {
	const doc = parse(
		'<html><head><meta charset="utf-8"><style>p{color:red}</style></head><body bgcolor="#fff" vlink="#00f"><p>x</p></body></html>',
	);
	assert.ok(doc.querySelector("meta[charset]"));
	assert.equal(doc.querySelector("style").textContent, "p{color:red}");
	assert.equal(doc.body.getAttribute("vlink"), "#00f");
});

test("meta refresh is removed, other meta tags are kept", () => {
	const head = (html) => parse(html).head.innerHTML;
	for (const v of ["refresh", "Refresh", " REFRESH "]) {
		assert.equal(head(`<meta http-equiv="${v}" content="0;url=https://x.invalid"><p>a</p>`), "");
	}
	// control
	assert.equal(head('<meta http-equiv="Content-Type" content="text/html">'), '<meta http-equiv="Content-Type" content="text/html">');
	assert.equal(head('<meta charset="utf-8">'), '<meta charset="utf-8">');
});
