import assert from "node:assert/strict";
import test from "node:test";
import { addToHistory, buildSuggestions, getTokenInfo, parseHistory } from "./search.js";

const labels = (groups) => groups.flatMap((g) => g.items.map((i) => i.label));
const groupNames = (groups) => groups.map((g) => g.name);
const suggest = (search, extra = {}) =>
	buildSuggestions({ search, cursorPosition: search.length, colorHash: () => "#000", ...extra });

test("getTokenInfo: token at the end of the text", () => {
	assert.deepEqual(getTokenInfo("from:a", 6), { token: "from:a", prefix: "from:a", startIndex: 0, endIndex: 6 });
});

test("getTokenInfo: cursor mid-token keeps the full token but a partial prefix", () => {
	assert.deepEqual(getTokenInfo("foo from:bob bar", 8), {
		token: "from:bob",
		prefix: "from",
		startIndex: 4,
		endIndex: 12,
	});
});

test("getTokenInfo: cursor after whitespace or in empty text is an empty token", () => {
	assert.deepEqual(getTokenInfo("foo ", 4), { token: "", prefix: "", startIndex: 4, endIndex: 4 });
	assert.deepEqual(getTokenInfo("", 0), { token: "", prefix: "", startIndex: 0, endIndex: 0 });
	assert.deepEqual(getTokenInfo(undefined, 0), { token: "", prefix: "", startIndex: 0, endIndex: 0 });
});

test("empty token offers the popular filters", () => {
	const out = labels(suggest(""));
	assert.ok(out.includes("is:unread"));
	assert.ok(out.includes("from:"));
	assert.ok(!out.includes("smaller:"), "non-popular filter should not be offered");
});

test("is: completes only matching values, `-` negation is normalised to `!`", () => {
	assert.deepEqual(labels(suggest("is:u")), ["is:unread"]);
	assert.deepEqual(labels(suggest("!is:r")), ["!is:read"]);
	assert.deepEqual(labels(suggest("-is:t")), ["!is:tagged"]);
	// control: a non-matching value yields nothing rather than everything
	assert.deepEqual(labels(suggest("is:zzz")), []);
});

test("has: completes attachment and inline", () => {
	assert.deepEqual(labels(suggest("has:")), ["has:attachment", "has:inline"]);
	assert.deepEqual(labels(suggest("has:i")), ["has:inline"]);
});

test("tag: suggests existing tags, quoting those with spaces, case-insensitively", () => {
	const tags = ["Work", "my tag", "other"];
	assert.deepEqual(labels(suggest("tag:wo", { tags })), ["tag:Work"]);
	assert.deepEqual(labels(suggest("tag:my", { tags })), ['tag:"my tag"']);
	assert.deepEqual(labels(suggest("!tag:oth", { tags })), ["!tag:other"]);
});

test("tag: with no tags falls back to the field hint, but only for an empty value", () => {
	assert.deepEqual(labels(suggest("tag:")), ["tag:"]);
	assert.deepEqual(labels(suggest("tag:x")), []);
});

test("a bare word also matches existing tags", () => {
	const out = labels(suggest("wor", { tags: ["Work"] }));
	assert.ok(out.includes("tag:Work"));
});

test("before:/after: offer dates relative to `now`", () => {
	const now = new Date(2024, 2, 1); // 1 March 2024 (local)
	const out = suggest("before:", { now });
	assert.deepEqual(labels(out), ["before:2024/03/01", "before:2024/02/29", "before:2024/02/23", "before:2024/01/31"]);
	assert.deepEqual(labels(suggest("!after:2024/02/2", { now })), ["!after:2024/02/29", "!after:2024/02/23"]);
});

test("larger:/smaller: offer sizes filtered by prefix", () => {
	assert.deepEqual(labels(suggest("larger:1")), ["larger:100KB", "larger:1MB", "larger:10MB"]);
	assert.deepEqual(labels(suggest("smaller:5m")), ["smaller:5MB"]);
});

test("address filters suggest unique addresses from current messages, capped at 8", () => {
	const messages = [
		{ From: { Address: "bob@example.com" }, To: [{ Address: "amy@example.com" }, { Address: "bob@example.com" }] },
		{ From: { Address: "bob@example.com" }, To: [{ Address: "carol@example.net" }] },
	];
	assert.deepEqual(labels(suggest("from:bo", { messages })), ["from:bob@example.com"]);
	assert.deepEqual(labels(suggest("to:", { messages })), [
		"to:amy@example.com",
		"to:bob@example.com",
		"to:carol@example.net",
	]);
	// addressed: searches every address field
	assert.deepEqual(labels(suggest("addressed:carol", { messages })), ["addressed:carol@example.net"]);

	const many = Array.from({ length: 20 }, (_, i) => ({ From: { Address: `u${i}@example.com` } }));
	assert.equal(labels(suggest("from:", { messages: many })).length, 8);
});

test("address filters fall back to the field hint when nothing matches an empty value", () => {
	assert.deepEqual(labels(suggest("from:")), ["from:"]);
	assert.deepEqual(labels(suggest("from:nobody")), []);
});

test("store values that are not arrays are tolerated", () => {
	assert.doesNotThrow(() => suggest("tag:", { tags: undefined, messages: null }));
	assert.doesNotThrow(() => suggest("from:", { tags: {}, messages: "x" }));
});

test("a lone ! or - offers negations only", () => {
	const out = labels(suggest("!"));
	assert.ok(out.length > 0);
	assert.ok(out.every((l) => l.startsWith("!")));
	// control: the general (non-negated) branch never offers negations
	assert.ok(labels(suggest("is")).every((l) => !l.startsWith("!")));
});

test("results are grouped in a stable order", () => {
	const out = suggest("", { history: ["foo"] });
	assert.equal(groupNames(out)[0], "Recent searches");
	const rest = groupNames(out).slice(1);
	const order = ["Fields", "Addresses", "Status & content", "Dates & sizes", "Tags", "Negations"];
	assert.deepEqual(
		rest,
		order.filter((n) => rest.includes(n)),
	);
});

test("history: five most recent when the query is empty, all matches when typing", () => {
	const history = ["a1", "a2", "a3", "a4", "a5", "a6", "b1"];
	const recent = (s) =>
		suggest(s, { history })
			.find((g) => g.isHistory)
			?.items.map((i) => i.value);
	assert.deepEqual(recent(""), ["a1", "a2", "a3", "a4", "a5"]);
	assert.deepEqual(recent("A"), ["a1", "a2", "a3", "a4", "a5", "a6"]);
	assert.equal(recent("zzz"), undefined);
});

test("parseHistory: valid input is kept, blanks dropped, capped at 50", () => {
	assert.deepEqual(parseHistory('["a"," ","b"]'), ["a", "b"]);
	const long = JSON.stringify(Array.from({ length: 80 }, (_, i) => `q${i}`));
	assert.equal(parseHistory(long).length, 50);
	assert.equal(parseHistory(long)[0], "q0");
});

test("parseHistory: anything malformed yields an empty list", () => {
	for (const raw of [null, "", "not json", "{}", '"str"', "[1,2]", '["a",1]', "null"]) {
		assert.deepEqual(parseHistory(raw), [], `input: ${raw}`);
	}
});

test("addToHistory: newest first, case-insensitive dedupe, trimmed, capped, input untouched", () => {
	const before = ["Foo", "bar"];
	const after = addToHistory(before, "  foo ");
	assert.deepEqual(after, ["foo", "bar"]);
	assert.deepEqual(before, ["Foo", "bar"], "input array must not be mutated");

	const full = Array.from({ length: 50 }, (_, i) => `q${i}`);
	const capped = addToHistory(full, "new");
	assert.equal(capped.length, 50);
	assert.equal(capped[0], "new");
	assert.ok(!capped.includes("q49"));
});

test("addToHistory: blank terms return the same list", () => {
	const h = ["a"];
	assert.equal(addToHistory(h, ""), h);
	assert.equal(addToHistory(h, "   "), h);
	assert.equal(addToHistory(h, undefined), h);
});
