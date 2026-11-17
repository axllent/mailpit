import assert from "node:assert/strict";
import test from "node:test";
import { screenshotProxyURL } from "./screenshotProxyURL.js";

test("screenshotProxyURL preserves a plus sign in standard Base64", () => {
	const id = "1234567890123456789012";
	const asset = "https://example.test/~asset.png";
	const proxy = new URL("http://localhost:8025/mailpit/proxy");
	const url = new URL(screenshotProxyURL(proxy, id, asset));

	assert.equal(url.pathname, "/mailpit/proxy");
	assert.match(url.search, /%2B/);
	assert.equal(atob(url.searchParams.get("data")), id + ":" + asset);
});
