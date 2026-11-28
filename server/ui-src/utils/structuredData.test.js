import assert from "node:assert/strict";
import test from "node:test";
import { JSDOM } from "jsdom";
import {
	extractStructuredData,
	MAX_BLOCK_LENGTH,
	MAX_BLOCKS,
	MAX_FORMATTED_LENGTH,
	MAX_TOTAL_LENGTH,
} from "./structuredData.js";

const parser = new new JSDOM("").window.DOMParser();
const extract = (html) => extractStructuredData(html, parser);
const page = (...scripts) => `<html><head>${scripts.join("")}</head><body><p>Hi</p></body></html>`;
const ld = (s, type = "application/ld+json") => `<script type="${type}">${s}</script>`;

test("HTML without structured data has no blocks", () => {
	assert.deepEqual(extract("<p>hello</p>"), { blocks: [], truncated: false });
	assert.deepEqual(extract(""), { blocks: [], truncated: false });
	assert.deepEqual(extract(undefined), { blocks: [], truncated: false });
});

test("a block is pretty-printed", () => {
	const r = extract(page(ld('{"@type":"EmailMessage","a":[1,2]}')));
	assert.equal(r.blocks.length, 1);
	assert.equal(r.blocks[0].error, "");
	assert.equal(r.blocks[0].text, JSON.stringify({ "@type": "EmailMessage", a: [1, 2] }, null, 2));
});

test("the content is not interpreted or validated", () => {
	// not an EmailMessage, no name, an http URL: shown exactly as it is
	const json = { "@type": "Order", potentialAction: { "@type": "Nope", target: "http://example.com" } };
	assert.equal(extract(page(ld(JSON.stringify(json)))).blocks[0].text, JSON.stringify(json, null, 2));
});

test("several blocks keep their order; other scripts are ignored", () => {
	const r = extract(
		page(ld('{"n":1}'), "<script>var x = 1;</script>", ld('{"n":2}', "text/javascript"), ld('{"n":3}')),
	);
	assert.deepEqual(
		r.blocks.map((b) => JSON.parse(b.text).n),
		[1, 3],
	);
});

test("the script type is matched without regard to case", () => {
	assert.equal(extract(page(ld('{"a":1}', " Application/LD+JSON "))).blocks.length, 1);
});

test("invalid JSON is shown as written, with the error", () => {
	const r = extract(page(ld('{"a": oops')));
	assert.equal(r.blocks.length, 1);
	assert.equal(r.blocks[0].text, '{"a": oops');
	assert.notEqual(r.blocks[0].error, "");
});

test("markup in the JSON stays text", () => {
	const r = extract(page(ld('{"name":"x"}')));
	assert.equal(r.blocks[0].text.includes("<"), false);
});

test("the number and size of blocks is limited", () => {
	const many = extract(page(...Array.from({ length: MAX_BLOCKS + 5 }, (_, i) => ld(`{"n":${i}}`))));
	assert.equal(many.blocks.length, MAX_BLOCKS);
	assert.equal(many.truncated, true);

	const big = extract(page(ld(`"${"a".repeat(MAX_BLOCK_LENGTH)}"`), ld('{"ok":true}')));
	assert.equal(big.blocks.length, 1);
	assert.equal(big.truncated, true);
});

const nested = (depth) => "[".repeat(depth) + "]".repeat(depth);

test("deeply nested JSON is not expanded by formatting", () => {
	// 4000 levels is 8 KB of JSON, which formats to 32 MB
	const r = extract(page(ld(nested(4000))));
	assert.equal(r.blocks.length, 1);
	assert.equal(r.blocks[0].text, nested(4000));
	assert.equal(r.blocks[0].error, "");
	assert.notEqual(r.blocks[0].note, "");
	assert.ok(r.blocks[0].text.length <= MAX_FORMATTED_LENGTH);

	// so deep that it cannot be formatted at all: still valid JSON, so it is not reported as an error
	const deep = extract(page(ld(nested(30000))));
	assert.equal(deep.blocks.length, 1);
	assert.equal(deep.blocks[0].text, nested(30000));
	assert.equal(deep.blocks[0].error, "");
	assert.notEqual(deep.blocks[0].note, "");
});

test("the total size shown is limited", () => {
	// each block is just under the size limit and is shown in full, but together they are over the total limit
	const block = ld(JSON.stringify({ a: "x".repeat(MAX_BLOCK_LENGTH - 100) }));
	const r = extract(page(...Array.from({ length: 12 }, () => block)));
	assert.ok(r.blocks.length > 1 && r.blocks.length < 12);
	assert.equal(r.truncated, true);
	assert.ok(r.blocks.reduce((n, b) => n + b.text.length, 0) <= MAX_TOTAL_LENGTH);
});
