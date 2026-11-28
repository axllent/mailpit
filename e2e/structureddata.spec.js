import { expect, test } from "@playwright/test";
import { clearMailbox, row, sendMessage } from "./helpers.js";

// JSON-LD blocks in the HTML body are listed, as written, in the HTML Source tab (the HTML preview removes them).

const page = (...blocks) =>
	`<html><head>${blocks.map((b) => `<script type="application/ld+json">${b}</script>`).join("")}</head><body><p>Hi</p></body></html>`;

const open = async (page, subject) => {
	await page.goto("./");
	await row(page, subject).click();
	await page.locator("#nav-html-source-tab:visible").click();
};

test.beforeEach(async ({ request }) => {
	await clearMailbox(request);
});

test("JSON-LD blocks are listed pretty-printed", async ({ page: p, request }) => {
	await sendMessage(request, {
		subject: "With data",
		html: page('{"@type":"EmailMessage","name":"x"}', '{"@type":"Order"}'),
	});
	await open(p, "With data");

	const blocks = p.getByTestId("structured-data-block");
	await expect(blocks).toHaveCount(2);
	await expect(blocks.nth(0)).toContainText('"@type": "EmailMessage"');
	await expect(blocks.nth(1)).toContainText('"@type": "Order"');
	await expect(p.getByTestId("structured-data")).toContainText("2 blocks");
});

test("nothing is validated: invalid JSON is shown as written with the error", async ({ page: p, request }) => {
	await sendMessage(request, { subject: "Broken", html: page('{"a": oops') });
	await open(p, "Broken");

	await expect(p.getByTestId("structured-data-block")).toContainText('{"a": oops');
	await expect(p.getByTestId("structured-data")).toContainText("Invalid JSON");
});

test("valid but extremely nested JSON is shown as written, and is not reported as invalid", async ({
	page: p,
	request,
}) => {
	const nested = "[".repeat(4000) + "]".repeat(4000);
	await sendMessage(request, { subject: "Deep", html: page(nested) });
	await open(p, "Deep");

	const section = p.getByTestId("structured-data");
	await expect(p.getByTestId("structured-data-block")).toHaveCount(1);
	await expect(section).toContainText("Not formatted");
	await expect(section).not.toContainText("Invalid JSON");
});

test("a message without JSON-LD has no structured data section", async ({ page: p, request }) => {
	await sendMessage(request, { subject: "Plain", html: "<p>nothing</p>" });
	await open(p, "Plain");

	await expect(p.locator("#nav-html-source")).toContainText("nothing");
	await expect(p.getByTestId("structured-data")).toHaveCount(0);
});
