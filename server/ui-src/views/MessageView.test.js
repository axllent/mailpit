import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";
import { runInNewContext } from "node:vm";
import { parse } from "@vue/compiler-sfc";
import { transform } from "esbuild";

const { descriptor } = parse(readFileSync(new URL("./MessageView.vue", import.meta.url), "utf8"));
const { code } = await transform(descriptor.script.content, { format: "cjs" });

const sideNavRequest = (searching, timeZone) => {
	const mailbox = { searching, timeZone };
	const imports = {
		"../stores/mailbox": { mailbox },
		"../stores/pagination": { pagination: { limit: 50, defaultLimit: 50 } },
	};
	const context = {
		module: { exports: {} },
		require: (id) => imports[id] || { default: {} },
		URLSearchParams,
	};
	runInNewContext(code, context);
	const instance = { resolve: (path) => path };
	context.module.exports.default.methods.initLoadMoreAPIParams.call(instance);
	return { uri: instance.apiSideNavURI, params: instance.apiSideNavParams };
};

for (const query of ["before:2026-11-24", "after:2026-11-24"]) {
	test(`sidebar search for ${query} sends the configured time zone`, () => {
		const { uri, params } = sideNavRequest(query, "America/Los_Angeles");
		assert.equal(uri, "/api/v1/search");
		assert.equal(params.get("query"), query);
		assert.equal(params.get("tz"), "America/Los_Angeles");
	});
}

test("sidebar requests without a date filter do not send a time zone", () => {
	assert.equal(sideNavRequest("subject:test", "America/Los_Angeles").params.has("tz"), false);
	assert.equal(sideNavRequest(false, "America/Los_Angeles").params.has("tz"), false);
	assert.equal(sideNavRequest("before:2026-11-24", "").params.has("tz"), false);
});
