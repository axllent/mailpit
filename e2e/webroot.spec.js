import { expect, test } from "@playwright/test";
import { clearMailbox, row, sendMessage } from "./helpers.js";

// Runs against an instance started with `--webroot /mailpit/`. The routing, API calls, static assets
// and websocket must all honour the prefix, which the Go tests cannot verify from the browser's side.

test.beforeEach(async ({ request }) => {
	await clearMailbox(request);
});

test("the UI works end to end under a webroot", async ({ page, request }) => {
	const failed = [];
	page.on("response", (r) => {
		if (r.status() >= 400) failed.push(`${r.status()} ${r.url()}`);
	});

	await page.goto("./");
	await expect(page.getByText("No messages in your mailbox")).toBeVisible();

	// live update proves the websocket connected on the prefixed path
	await sendMessage(request, { subject: "Under webroot", html: "<p>webroot body</p>" });
	const message = row(page, "Under webroot");
	await expect(message).toBeVisible();

	// client-side routing and the message API keep the prefix
	await message.click();
	await expect(page).toHaveURL(/\/mailpit\/view\/[^/]+$/);
	await expect(page.frameLocator("#preview-html").locator("p")).toHaveText("webroot body");

	// the raw source is loaded through a prefixed API URL inside an iframe
	await page.locator("#nav-raw-tab").click();
	await expect(page.frameLocator("#nav-raw iframe").locator("body")).toContainText("Under webroot");

	// a reload on a deep link must be served by the server under the prefix
	await page.reload();
	await expect(page.frameLocator("#preview-html").locator("p")).toHaveText("webroot body");

	expect(failed).toEqual([]);
});
