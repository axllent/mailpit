import assert from "node:assert/strict";
import test from "node:test";
import createDOMPurify from "dompurify";
import { JSDOM } from "jsdom";
import { textToHTML } from "./textToHTML.js";

// An isolated DOMPurify instance per test file run, bound to a jsdom window
const purify = createDOMPurify(new JSDOM("").window);
const convert = (s) => textToHTML(s, purify);
const link = (url, text = url) => `<a href="${url}" target="_blank" rel="noopener">${text}</a>`;

test("a link at the very start of the text is linked", () => {
	assert.equal(convert("https://example.com/start"), link("https://example.com/start"));
	assert.equal(
		convert("https://example.com/start\nhttp://example.com/2"),
		`${link("https://example.com/start")}\n${link("http://example.com/2")}`,
	);
	// control: a link after other text is linked as before
	assert.equal(convert("text\nhttps://example.com/line"), `text\n${link("https://example.com/line")}`);
	assert.equal(convert("see https://example.com/x"), `see ${link("https://example.com/x")}`);
});

test("an angle bracket link at the start is linked once, not twice", () => {
	// the angle brackets themselves are kept (escaped) around the anchor
	assert.equal(convert("<https://example.com/a b>"), `&lt;${link("https://example.com/a b")}&gt;`);
});

test("placeholder characters cannot be used to inject markup", () => {
	// exploit: pre-injected placeholders must be stripped, leaving no attacker-controlled attributes
	const out = convert("˱˱˱a href=ˠˠˠjavascript:alert(1)ˠˠˠ˲˲˲click˱˱˱/a˲˲˲ https://example.com/ok");
	// the leftover text is inert, and the only anchor is the genuine link
	const anchors = [...new JSDOM(out).window.document.querySelectorAll("a")];
	assert.deepEqual(
		anchors.map((a) => a.getAttribute("href")),
		["https://example.com/ok"],
	);
	// control: same shape without placeholders
	assert.equal(convert("a href=javascript:alert(1) click"), "a href=javascript:alert(1) click");
});

test("HTML in the text is escaped", () => {
	assert.equal(convert("<script>alert(1)</script>"), "&lt;script&gt;alert(1)&lt;/script&gt;");
	assert.equal(convert("<b>bold</b>"), "&lt;b&gt;bold&lt;/b&gt;");
});
