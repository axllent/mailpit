import { expect, test } from "@playwright/test";
import { clearMailbox, row, sendMessage } from "./helpers.js";

test.beforeEach(async ({ request }) => {
	await clearMailbox(request);
});

test("a new message appears live over the websocket, without reloading", async ({ page, request }) => {
	await page.goto("./");
	await expect(page.getByText("No messages in your mailbox")).toBeVisible();

	await sendMessage(request, { subject: "Live arrival" });

	await expect(page.getByTestId("message-subject")).toHaveText("Live arrival");
	await expect(page.getByText("No messages in your mailbox")).toBeHidden();
});

test("opening a message shows its HTML, text and headers", async ({ page, request }) => {
	await sendMessage(request, {
		subject: "Open me",
		text: "The plain text body",
		html: "<p>The <b>HTML</b> body</p>",
	});

	await page.goto("./");
	await row(page, "Open me").click();
	await expect(page).toHaveURL(/\/view\/[^/]+$/);

	await expect(page.frameLocator("#preview-html").locator("p")).toHaveText("The HTML body");

	await page.locator("#nav-plain-text-tab").click();
	await expect(page.getByText("The plain text body")).toBeVisible();

	await page.locator("#nav-headers-tab").click();
	await expect(page.getByText("sender@example.com").first()).toBeVisible();
});

// Exploit and control share the same shape: the control proves the preview renders normally,
// so the exploit assertions cannot pass simply because nothing rendered.
test("the HTML preview strips scripts and event handlers", async ({ page, request }) => {
	const dialogs = [];
	page.on("dialog", async (d) => {
		dialogs.push(d.message());
		await d.dismiss();
	});

	await sendMessage(request, {
		subject: "Hostile HTML",
		html: `<p id="benign">benign content</p>
			<script>alert("script ran")</script>
			<img id="bad" src="x" onerror="alert('onerror ran')">
			<a id="js" href="javascript:alert('link ran')">link</a>`,
	});

	await page.goto("./");
	await row(page, "Hostile HTML").click();

	const preview = page.frameLocator("#preview-html");
	await expect(preview.locator("#benign")).toHaveText("benign content"); // control
	await expect(preview.locator("script")).toHaveCount(0);
	// the elements themselves survive (controls), only the dangerous parts are removed
	await expect(preview.locator("#bad")).toHaveAttribute("src", "x");
	await expect(preview.locator("#bad")).not.toHaveAttribute("onerror", /.*/);
	await expect(preview.locator("#js")).toHaveText("link");
	await expect(preview.locator("#js")).not.toHaveAttribute("href", /javascript:/i);

	// A second line of defence: the preview iframe is also sandboxed without allow-scripts, so a handler could not
	// run even if it survived. The DOM assertions above are the primary check. The fixed wait gives any unexpected
	// handler time to fire, as there is nothing to wait for when asserting that something does not happen.
	await page.waitForTimeout(500);
	expect(dialogs).toEqual([]);
});
